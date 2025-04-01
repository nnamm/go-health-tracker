package dbtest

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

// NewTestDB creates a new in-memory database for testing
func NewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("db connection error: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

// CreateDate returns a time.Time from a string
func CreateDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}
	return t
}

// MonthOf returns a pointer to an int
func MonthOf(m int) *int {
	return &m
}

// assertHealthRecordEqual compares two HealthRecord
func AssertHelathRecordEqual(t *testing.T, got, want *models.HealthRecord) {
	t.Helper()
	if got.StepCount != want.StepCount {
		t.Errorf("StepCount = %v, want %v", got.StepCount, want.StepCount)
	}
}

// AssertHealthRecordEqual compares two slices of HealthRecord
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
