package database

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

var testDB *DB

func TestMain(m *testing.M) {
	// set up a database for testing
	var err error
	testDB, err = NewDB(":memory:")
	if err != nil {
		panic(err)
	}

	// create table
	err = testDB.createTable()
	if err != nil {
		panic(err)
	}

	// run all tests
	code := m.Run()

	// cleanup after test
	testDB.Close()

	os.Exit(code)
}

func TestCreateTable(t *testing.T) {
	// create table test
	if err := testDB.createTable(); err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// check index existence
	var indexExists bool
	err := testDB.QueryRow(`SELECT EXISTS (
         SELECT 1 FROM sqlite_master
         WHERE type='index' AND name='idx_health_records_date'
         )`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("failed to check index: %v", err)
	}
	if !indexExists {
		t.Error("expected index was created")
	}
}

func TestHealthRecordCRUDScenarios(t *testing.T) {
	scenarios := []struct {
		name            string
		initial         *models.HealthRecord // scenario data - initial data
		update          *models.HealthRecord // scenerio data - updated data
		wantAfterCreate *models.HealthRecord // expected value
		wantAfterUpdate *models.HealthRecord //
		wantAfterDelete *models.HealthRecord //
		wantCreateErr   error                // expected value (error)
		wantUpdateErr   error                //
		wantDeleteErr   error                //
	}{
		{
			name: "normal scenario - Create, Update, Delete success",
			initial: &models.HealthRecord{
				Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StepCount: 10000,
			},
			update: &models.HealthRecord{
				Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StepCount: 12000,
			},
			wantAfterCreate: &models.HealthRecord{StepCount: 10000},
			wantAfterUpdate: &models.HealthRecord{StepCount: 12000},
			wantAfterDelete: nil,
		},
		{
			name: "error scenerio - Update non-existence record",
			initial: &models.HealthRecord{
				Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				StepCount: 10000,
			},
			update: &models.HealthRecord{
				Date:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				StepCount: 15000,
			},
			wantUpdateErr: sql.ErrNoRows,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cleanupDB(t, testDB)

			// create
			created, err := testDB.CreateHealthRecord(ctx, tt.initial)
			if !errors.Is(err, tt.wantCreateErr) {
				t.Errorf("CreateHealthRecord() error = %v, want %v", err, tt.wantCreateErr)
			}
			if tt.wantAfterCreate != nil && created != nil {
				assertHealthRecord(t, created, tt.wantAfterCreate)
			}

			// update
			err = testDB.UpdateHealthRecord(ctx, tt.update)
			if !errors.Is(err, tt.wantUpdateErr) {
				t.Errorf("UpdateHealthRecord() error = %v, want %v", err, tt.wantUpdateErr)
			}
			if tt.wantAfterUpdate != nil && err == nil {
				retrieved, _ := testDB.ReadHealthRecord(ctx, tt.update.Date)
				assertHealthRecord(t, retrieved, tt.wantAfterUpdate)
			}

			// delete
			if tt.initial != nil {
				err = testDB.DeleteHealthRecord(ctx, tt.initial.Date)
				if !errors.Is(err, tt.wantDeleteErr) {
					t.Errorf("DeleteHealthRecord() error = %v, want %v", err, tt.wantDeleteErr)
				}
				retrieved, _ := testDB.ReadHealthRecord(ctx, tt.initial.Date)
				if retrieved != tt.wantAfterDelete {
					t.Errorf("after delete, got record = %v, want %v", retrieved, tt.wantAfterDelete)
				}
			}
		})
	}
}

func TestReadHealthRecords(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*testing.T, *DB, context.Context) // setup func
		year    int
		month   *int // optional
		want    []models.HealthRecord
		wantErr error
	}{
		{
			name: "successful yearly query - returns all records for 2024",
			setup: func(t *testing.T, db *DB, ctx context.Context) {
				mustCreateRecords(t, db, ctx, []models.HealthRecord{
					{Date: date("2024-01-01"), StepCount: 10000},
					{Date: date("2024-12-31"), StepCount: 11000},
					{Date: date("2025-01-01"), StepCount: 12000},
				})
			},
			year:  2024,
			month: nil, // yearly query
			want: []models.HealthRecord{
				{Date: date("2024-01-01"), StepCount: 10000},
				{Date: date("2024-12-31"), StepCount: 11000},
			},
			wantErr: nil,
		},
		{
			name: "successful monthly query - returns only Jan 2024 records",
			setup: func(t *testing.T, db *DB, ctx context.Context) {
				mustCreateRecords(t, db, ctx, []models.HealthRecord{
					{Date: date("2024-01-01"), StepCount: 10000},
					{Date: date("2024-01-31"), StepCount: 11000},
					{Date: date("2024-02-01"), StepCount: 12000},
				})
			},
			year:  2024,
			month: monthOf(1),
			want: []models.HealthRecord{
				{Date: date("2024-01-01"), StepCount: 10000},
				{Date: date("2024-01-31"), StepCount: 11000},
			},
			wantErr: nil,
		},
		{
			name: "empty result - no records for year",
			setup: func(t *testing.T, db *DB, ctx context.Context) {
				mustCreateRecords(t, db, ctx, []models.HealthRecord{
					{Date: date("2023-01-01"), StepCount: 10000},
					{Date: date("2025-01-01"), StepCount: 11000},
				})
			},
			year:    2024,
			want:    []models.HealthRecord{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cleanupDB(t, testDB)
			if tt.setup != nil {
				tt.setup(t, testDB, ctx)
			}

			var got []models.HealthRecord
			var err error
			if tt.month == nil {
				got, err = testDB.ReadHealthRecordsByYear(ctx, tt.year)
			} else {
				got, err = testDB.ReadHealthRecordsByYearMonth(ctx, tt.year, *tt.month)
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				assertHealthRecordsEqual(t, got, tt.want)
			}
		})
	}
}

// date converts a date string to time.Time.
func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// monthOf returns pointer to the argument.
func monthOf(m int) *int {
	return &m
}

// mustCreateRecords creates multiple health records.
func mustCreateRecords(t *testing.T, db *DB, ctx context.Context, records []models.HealthRecord) {
	t.Helper()
	for _, r := range records {
		_, err := db.CreateHealthRecord(ctx, &r)
		if err != nil {
			t.Fatalf("failed to create record: %v", err)
		}
	}
}

func cleanupDB(t *testing.T, db *DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM health_records")
	if err != nil {
		t.Fatalf("Failed to cleanup database: %v", err)
	}
}

func assertHealthRecord(t *testing.T, got, want *models.HealthRecord) {
	t.Helper()
	if got.StepCount != want.StepCount {
		t.Errorf("StepCount = %v, want %v", got.StepCount, want.StepCount)
	}
}

func assertHealthRecordsEqual(t *testing.T, got, want []models.HealthRecord) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %d records, want %d records", len(got), len(want))
	}

	for i := range got {
		if i >= len(want) {
			break
		}
		if !got[i].Date.Equal(want[i].Date) {
			t.Errorf("Date = %v, want %v", got[i].Date, want[i].Date)
		}

		if got[i].StepCount != want[i].StepCount {
			t.Errorf("StepCount = %v, want %v", got[i].StepCount, want[i].StepCount)
		}
	}
}
