package testutils

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/stretchr/testify/require"
)

// SetupSQLiteTester sets up a SQLite database for testing
func SetupSQLiteTester(t *testing.T) (*database.SQLiteDB, func()) {
	t.Helper()

	// Set up a database for testing
	db, err := database.NewSQLiteDB(":memory:")
	require.NoError(t, err)

	// Create table. Assumes CreateTable() is now a public method.
	err = db.CreateTable()
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
	}
	return db, cleanup
}

// MonthOf returns a pointer to an int
func MonthOf(m int) *int {
	return &m
}

// AssertHealthRecordEqual compares two HealthRecord
func AssertHealthRecordEqual(t *testing.T, got, want *models.HealthRecord) {
	t.Helper()
	if got.StepCount != want.StepCount {
		t.Errorf("StepCount = %v, want %v", got.StepCount, want.StepCount)
	}
}

// AssertHealthRecordsEqual compares two slices of HealthRecord
func AssertHealthRecordsEqual(t *testing.T, got, want []models.HealthRecord) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Record count = %d, want %d", len(got), len(want))
		return
	}

	for i := range got {
		if i > len(want) {
			break
		}
		if !got[i].Date.Equal(want[i].Date) {
			t.Errorf("Date[%d] = %v, want %v", i, got[i].Date, want[i].Date)
		}
		if got[i].StepCount != want[i].StepCount {
			t.Errorf("StepCount[%d] = %v, want %v", i, got[i].StepCount, want[i].StepCount)
		}
	}
}

// CreateTestRecords creates records in the test table
func CreateTestRecords(ctx context.Context, t *testing.T, db *sql.DB, records []models.HealthRecord) {
	t.Helper()
	stmt, err := db.PrepareContext(ctx, "INSERT INTO health_records (date, step_count, created_at, updated_at) VALUES (?, ?, ?, ?)")
	if err != nil {
		t.Fatalf("statement preparation error: %v", err)
	}
	defer stmt.Close()

	for _, r := range records {
		now := time.Now()
		_, err := stmt.ExecContext(ctx, r.Date, r.StepCount, now, now)
		if err != nil {
			t.Fatalf("failed to create records: %v", err)
		}
	}
}

// CleanupDB removes all records from the test table
func CleanupDB(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM health_records")
	if err != nil {
		t.Fatalf("failed to cleanup db: %v", err)
	}
}
