package database

import (
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

var testDB *DB

func TestMain(m *testing.M) {
	// Set up a database for testing
	var err error
	testDB, err = NewDB(":memory:")
	if err != nil {
		panic(err)
	}

	// Create table
	err = testDB.CreateTable()
	if err != nil {
		panic(err)
	}

	// Run all tests
	code := m.Run()

	// Cleanup after test
	testDB.Close()

	os.Exit(code)
}

func TestCreateTable(t *testing.T) {
	// Create table test
	if err := testDB.CreateTable(); err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Check index existence
	var indexExists bool
	err := testDB.QueryRow(`SELECT EXISTS (
         SELECT 1 FROM sqlite_master
         WHERE type='index' AND name='idx_health_records_date'
         )`).Scan(&indexExists)
	if err != nil {
		t.Fatalf("Failed to check index: %v", err)
	}
	if !indexExists {
		t.Error("Expected index was created")
	}
}

func TestHealthRecordCRUDScenarios(t *testing.T) {
	scenarios := []struct {
		name            string
		initial         *models.HealthRecord // Scenario data - initial data
		update          *models.HealthRecord // Scenerio data - updated data
		wantAfterCreate *models.HealthRecord // Expected value
		wantAfterUpdate *models.HealthRecord //
		wantAfterDelete *models.HealthRecord //
		wantCreateErr   error                // Expected value (error)
		wantUpdateErr   error                //
		wantDeleteErr   error                //
	}{
		{
			name: "Normal scenario - Create, Update, Delete success",
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
			name: "Error scenerio - Update non-existence record",
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
			cleanupDB(t, testDB)

			// Create
			created, err := testDB.CreateHealthRecord(tt.initial)
			if !errors.Is(err, tt.wantCreateErr) {
				t.Errorf("CreateHealthRecord() error = %v, want %v", err, tt.wantAfterCreate)
			}
			if tt.wantAfterCreate != nil {
				assertHealthRecord(t, created, tt.wantAfterCreate)
			}

			// Update
			err = testDB.UpdateHealthRecord(tt.update)
			if !errors.Is(err, tt.wantUpdateErr) {
				t.Errorf("UpdateHealthRecord() error = %v, want %v", err, tt.wantAfterUpdate)
			}
			if tt.wantAfterUpdate != nil {
				retrieved, _ := testDB.ReadHealthRecord(tt.update.Date)
				assertHealthRecord(t, retrieved, tt.wantAfterUpdate)
			}

			// Delete
			if tt.initial != nil {
				err = testDB.DeleteHealthRecord(tt.initial.Date)
				if !errors.Is(err, tt.wantDeleteErr) {
					t.Errorf("DeleteHealthRecord() error = %v, want %v", err, tt.wantAfterDelete)
				}
				retrieved, _ := testDB.ReadHealthRecord(tt.initial.Date)
				if retrieved != tt.wantAfterDelete {
					t.Errorf("After delete, got record = %v, want %v", retrieved, tt.wantAfterDelete)
				}
			}
		})
	}
}

func TestReadHealthRecords(t *testing.T) {
	tests := []struct {
		name     string
		initial  []models.HealthRecord
		year     int
		month    int
		expected []models.HealthRecord
	}{
		{
			name: "Normal - Read records by Year",
			initial: []models.HealthRecord{
				{
					Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					StepCount: 10000,
				},
				{
					Date:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					StepCount: 11000,
				},
				{
					Date:      time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC),
					StepCount: 12000,
				},
			},
			year: 2024,
			expected: []models.HealthRecord{
				{
					Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					StepCount: 10000,
				},
				{
					Date:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					StepCount: 11000,
				},
			},
		},
		{
			name: "Normal - Read records by Year and Month",
			initial: []models.HealthRecord{
				{
					Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					StepCount: 10000,
				},
				{
					Date:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					StepCount: 11000,
				},
				{
					Date:      time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC),
					StepCount: 12000,
				},
			},
			year:  2024,
			month: 1,
			expected: []models.HealthRecord{
				{
					Date:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
					StepCount: 10000,
				},
				{
					Date:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
					StepCount: 11000,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupDB(t, testDB)

			// Preparate health records
			for _, v := range tt.initial {
				_, err := testDB.CreateHealthRecord(&v)
				if err != nil {
					t.Fatalf("Failed to create health_records: %v", err)
				}
			}

			// Read
			var retrieved []models.HealthRecord
			var retErr error
			if tt.month == 0 {
				retrieved, retErr = testDB.ReadHealthRecordsByYear(tt.year)
			} else {
				retrieved, retErr = testDB.ReadHealthRecordsByYearMonth(tt.year, tt.month)
			}
			if retErr != nil {
				t.Errorf("ReadHealthRecords() error = %v", retErr)
			} else {
				assertHealthRecords(t, retrieved, tt.expected)
			}
		})
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

func assertHealthRecords(t *testing.T, got, want []models.HealthRecord) {
	t.Helper()
	for i := 0; i < len(got); i++ {
		if got[i].StepCount != want[i].StepCount {
			t.Errorf("StepCount = %v, want %v", got[i].StepCount, want[i].StepCount)
		}
	}
}
