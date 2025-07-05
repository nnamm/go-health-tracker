package database_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/nnamm/go-health-tracker/testutils"
)

var testDB *database.SQLiteDB

func TestSQLite_HealthRecordCRUDScenarios(t *testing.T) {
	testDB, cleanup := testutils.SetupSQLiteTester(t)
	defer cleanup()

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
			name: "normal scenario - create, Update, Delete success",
			initial: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 10000,
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 12000,
			},
			wantAfterCreate: &models.HealthRecord{StepCount: 10000},
			wantAfterUpdate: &models.HealthRecord{StepCount: 12000},
			wantAfterDelete: nil,
		},
		{
			name: "error scenerio - update non-existence record",
			initial: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 10000,
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-02"),
				StepCount: 15000,
			},
			wantUpdateErr: sql.ErrNoRows,
		},
		{
			name:          "error scenerio - delete non-existence record",
			wantDeleteErr: sql.ErrNoRows,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testutils.CleanupDB(t, testDB.DB)

			// create
			if tt.initial != nil {
				created, err := testDB.CreateHealthRecord(ctx, tt.initial)
				if !errors.Is(err, tt.wantCreateErr) {
					t.Errorf("CreateHealthRecord() error = %v, want %v", err, tt.wantCreateErr)
				}
				if tt.wantAfterCreate != nil && created != nil {
					testutils.AssertHealthRecordEqual(t, created, tt.wantAfterCreate)
				}
			}

			// update
			if tt.update != nil {
				err := testDB.UpdateHealthRecord(ctx, tt.update)
				if !errors.Is(err, tt.wantUpdateErr) {
					t.Errorf("UpdateHealthRecord() error = %v, want %v", err, tt.wantUpdateErr)
				}
				if tt.wantAfterUpdate != nil && err == nil {
					retrieved, _ := testDB.ReadHealthRecord(ctx, tt.update.Date)
					testutils.AssertHealthRecordEqual(t, retrieved, tt.wantAfterUpdate)
				}
			}

			// delete
			if tt.initial != nil {
				err := testDB.DeleteHealthRecord(ctx, tt.initial.Date)
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

func TestSQLite_ReadHealthRecords(t *testing.T) {
	testDB, cleanup := testutils.SetupSQLiteTester(t)
	defer cleanup()

	tests := []struct {
		name    string
		setup   func(*testing.T, context.Context, *database.SQLiteDB)
		year    int
		month   *int // optional
		want    []models.HealthRecord
		wantErr error
	}{
		{
			name: "successful yearly query - returns all records for 2024",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				records := []models.HealthRecord{
					{Date: testutils.CreateDate("2024-01-01"), StepCount: 10000},
					{Date: testutils.CreateDate("2024-12-31"), StepCount: 11000},
					{Date: testutils.CreateDate("2025-01-01"), StepCount: 12000},
				}
				testutils.CreateTestRecords(ctx, t, db.DB, records)
			},
			year:  2024,
			month: nil, // yearly query
			want: []models.HealthRecord{
				{Date: testutils.CreateDate("2024-01-01"), StepCount: 10000},
				{Date: testutils.CreateDate("2024-12-31"), StepCount: 11000},
			},
			wantErr: nil,
		},
		{
			name: "successful monthly query - returns only Jan 2024 records",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				records := []models.HealthRecord{
					{Date: testutils.CreateDate("2024-01-01"), StepCount: 10000},
					{Date: testutils.CreateDate("2024-01-31"), StepCount: 11000},
					{Date: testutils.CreateDate("2024-02-01"), StepCount: 12000},
				}
				testutils.CreateTestRecords(ctx, t, db.DB, records)
			},
			year:  2024,
			month: testutils.MonthOf(1),
			want: []models.HealthRecord{
				{Date: testutils.CreateDate("2024-01-01"), StepCount: 10000},
				{Date: testutils.CreateDate("2024-01-31"), StepCount: 11000},
			},
			wantErr: nil,
		},
		{
			name: "empty result - no records for year",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				records := []models.HealthRecord{
					{Date: testutils.CreateDate("2023-01-01"), StepCount: 10000},
					{Date: testutils.CreateDate("2025-01-01"), StepCount: 11000},
				}
				testutils.CreateTestRecords(ctx, t, db.DB, records)
			},
			year:    2024,
			want:    []models.HealthRecord{},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testutils.CleanupDB(t, testDB.DB)
			if tt.setup != nil {
				tt.setup(t, ctx, testDB)
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
				testutils.AssertHealthRecordsEqual(t, got, tt.want)
			}
		})
	}
}

func TestSQLite_UpdateHealthRecord(t *testing.T) {
	testDB, cleanup := testutils.SetupSQLiteTester(t)
	defer cleanup()

	tests := []struct {
		name      string
		setup     func(*testing.T, context.Context, *database.SQLiteDB)
		update    *models.HealthRecord
		nonUpdate *models.HealthRecord
		wantErr   error
	}{
		{
			name: "successful update",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 12000,
			},
			wantErr: nil,
		},
		{
			name: "successful update - max step count",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 100000,
			},
			wantErr: nil,
		},
		{
			name: "successful update - zero step count",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 0,
			},
			wantErr: nil,
		},
		{
			name: "verify update affects only specified record",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				records := []models.HealthRecord{
					{Date: testutils.CreateDate("2024-01-01"), StepCount: 10000},
					{Date: testutils.CreateDate("2024-01-02"), StepCount: 20000},
				}
				testutils.CreateTestRecords(ctx, t, db.DB, records)
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 15000,
			},
			nonUpdate: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-02"),
				StepCount: 20000,
			},
			wantErr: nil,
		},
		{
			name: "error - update non-existence record",
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-01"),
				StepCount: 10000,
			},
			wantErr: sql.ErrNoRows,
		},
		{
			name: "error - update with different date (future)",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-02-01"),
				StepCount: 12000,
			},
			wantErr: sql.ErrNoRows,
		},
		{
			name: "error - update with different date (past)",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2020-01-01"),
				StepCount: 12000,
			},
			wantErr: sql.ErrNoRows,
		},
		{
			name: "error - update with improbable step count",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			update: &models.HealthRecord{
				Date:      testutils.CreateDate("2020-01-01"),
				StepCount: 100001,
			},
			wantErr: sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testutils.CleanupDB(t, testDB.DB)

			if tt.setup != nil {
				tt.setup(t, ctx, testDB)
			}

			err := testDB.UpdateHealthRecord(ctx, tt.update)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				retrieved, _ := testDB.ReadHealthRecord(ctx, tt.update.Date)
				testutils.AssertHealthRecordEqual(t, retrieved, tt.update)
			}
			if tt.nonUpdate != nil {
				nonAffectRecord, _ := testDB.ReadHealthRecord(ctx, tt.nonUpdate.Date)
				testutils.AssertHealthRecordEqual(t, nonAffectRecord, tt.nonUpdate)
			}
		})
	}
}

func TestSQLite_DeleteHealthRecord(t *testing.T) {
	testDB, cleanup := testutils.SetupSQLiteTester(t)
	defer cleanup()

	tests := []struct {
		name       string
		setup      func(*testing.T, context.Context, *database.SQLiteDB)
		deleteDate time.Time
		nonDelete  *models.HealthRecord
		wantErr    error
	}{
		{
			name: "successful delete",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			deleteDate: testutils.CreateDate("2024-01-01"),
			wantErr:    nil,
		},
		{
			name: "verify delete affects only specified record",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				records := []models.HealthRecord{
					{Date: testutils.CreateDate("2024-01-01"), StepCount: 10000},
					{Date: testutils.CreateDate("2024-01-02"), StepCount: 20000},
				}
				testutils.CreateTestRecords(ctx, t, db.DB, records)
			},
			deleteDate: testutils.CreateDate("2024-01-01"),
			nonDelete: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-01-02"),
				StepCount: 20000,
			},
			wantErr: nil,
		},
		{
			name:       "error - delete non-existence record",
			setup:      nil,
			deleteDate: testutils.CreateDate("2024-01-01"),
			wantErr:    sql.ErrNoRows,
		},
		{
			name: "error - delete with different date (future)",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			deleteDate: testutils.CreateDate("2025-02-01"),
			wantErr:    sql.ErrNoRows,
		},
		{
			name: "error - delete with different date (past)",
			setup: func(t *testing.T, ctx context.Context, db *database.SQLiteDB) {
				record := &models.HealthRecord{
					Date:      testutils.CreateDate("2024-01-01"),
					StepCount: 10000,
				}
				testutils.CreateTestRecords(ctx, t, db.DB, []models.HealthRecord{*record})
			},
			deleteDate: testutils.CreateDate("2023-12-31"),
			wantErr:    sql.ErrNoRows,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			testutils.CleanupDB(t, testDB.DB)

			if tt.setup != nil {
				tt.setup(t, ctx, testDB)
			}

			err := testDB.DeleteHealthRecord(ctx, tt.deleteDate)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				retrieved, _ := testDB.ReadHealthRecord(ctx, tt.deleteDate)
				if retrieved != nil {
					t.Errorf("record still exists after deletion")
				}
			}
			if tt.nonDelete != nil {
				nonAffectRecord, _ := testDB.ReadHealthRecord(ctx, tt.nonDelete.Date)
				testutils.AssertHealthRecordEqual(t, nonAffectRecord, tt.nonDelete)
			}
		})
	}
}

func TestSQLite_ContextCancellation(t *testing.T) {
	testDB, cleanup := testutils.SetupSQLiteTester(t)
	defer cleanup()

	date := testutils.CreateDate("2024-07-01")
	record := &models.HealthRecord{
		Date:      date,
		StepCount: 10000,
	}
	ctx := context.Background()
	_, err := testDB.CreateHealthRecord(ctx, record)
	if err != nil {
		t.Fatalf("failed to create test record: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cancel()

	err = testDB.UpdateHealthRecord(ctx, record)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}

	err = testDB.DeleteHealthRecord(ctx, date)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
