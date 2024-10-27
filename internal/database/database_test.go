package database

import (
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

func TestDB_CreateHealthRecord(t *testing.T) {
	dateStr := time.Now().Format("2006-01-02")
	date, _ := time.Parse("2006-01-02", dateStr)
	hr := &models.HealthRecord{
		Date:      date,
		StepCount: 10000,
	}

	// Create a test record
	_, err := testDB.CreateHealthRecord(hr)
	if err != nil {
		t.Errorf("Failed to create health record: %v", err)
	}

	// Read and check the record created
	retrieveHR, err := testDB.ReadHealthRecord(hr.Date)
	if err != nil {
		t.Errorf("Failed to read health record: %v", err)
	}
	if retrieveHR.StepCount != hr.StepCount {
		t.Errorf("Expected step count %d, but got %d", hr.StepCount, retrieveHR.StepCount)
	}
}

func TestDB_UpdateHealthRecord(t *testing.T) {
	dateStr := time.Now().Format("2006-01-02")
	date, _ := time.Parse("2006-01-02", dateStr)
	hrBefore := &models.HealthRecord{
		Date:      date,
		StepCount: 11000,
	}
	hrAfter := &models.HealthRecord{
		Date:      date,
		StepCount: 12000,
	}

	// Create a test record
	_, err := testDB.CreateHealthRecord(hrBefore)
	if err != nil {
		t.Errorf("Failed to create health record: %v", err)
	}

	// Update the test record
	err = testDB.UpdateHealthRecord(hrAfter)
	if err != nil {
		t.Errorf("Failed to update health record: %v", err)
	}

	// Read and check the record created
	retrieveHR, err := testDB.ReadHealthRecord(hrAfter.Date)
	if err != nil {
		t.Errorf("Failed to read health record: %v", err)
	}
	if retrieveHR.StepCount != hrAfter.StepCount {
		t.Errorf("Expected step count %d, but got %d", hrAfter.StepCount, retrieveHR.StepCount)
	}
}

func TestDB_DeleteHealthRecord(t *testing.T) {
	dateStr := time.Now().Format("2006-01-02")
	date, _ := time.Parse("2006-01-02", dateStr)
	hr := &models.HealthRecord{
		Date:      date,
		StepCount: 13000,
	}

	// Create a test record
	_, err := testDB.CreateHealthRecord(hr)
	if err != nil {
		t.Errorf("Failed to create health record: %v", err)
	}

	// Delete a test record
	err = testDB.DeleteHealthRecord(hr.Date)
	if err != nil {
		t.Errorf("Failed to delete health record: %v", err)
	}

	// Read and verify the record does not exist
	record, err := testDB.ReadHealthRecord(hr.Date)
	if record != nil {
		t.Errorf("Failed to delete health record: %v", err)
	}
}
