package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/handlers"
	"github.com/nnamm/go-health-tracker/internal/models"
)

var testServer *httptest.Server

func TestMain(m *testing.M) {
	// Set up a database for testing
	db, err := database.NewDB(":memory:")
	if err != nil {
		panic(err)
	}
	err = db.CreateTable()
	if err != nil {
		panic(err)
	}

	// Set up server for testing
	healthHandler := handlers.NewHealthRecordHandler(db)
	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			healthHandler.GetHealthRecord(w, r)
		case http.MethodPost:
			healthHandler.CreateHealthRecord(w, r)
		case http.MethodPut:
			healthHandler.UpdateHealthRecord(w, r)
		case http.MethodDelete:
			healthHandler.DeleteHealthRecord(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// Run all tests
	code := m.Run()

	// Clean up
	testServer.Close()
	db.Close()

	os.Exit(code)
}

func TestHealthRecordIntegration(t *testing.T) {
	// Create a record
	dateStr := time.Now().Format("2006-01-02")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Fatal(err)
	}
	record := models.HealthRecord{
		Date:      date,
		StepCount: 10000,
	}
	body, _ := json.Marshal(record)

	res, err := http.Post(testServer.URL+"/health", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create health record: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Errorf("Expected status Create, got %v", res.Status)
	}

	// Retrieve the record
	res, err = http.Get(testServer.URL + "/health?date=" + dateStr)
	if err != nil {
		t.Fatalf("Failed to get health record: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	// Verify response body
	var retrievedRecord models.HealthRecord
	json.NewDecoder(res.Body).Decode(&retrievedRecord)
	if retrievedRecord.StepCount != record.StepCount {
		t.Errorf("Expected step count %d, got %d", record.StepCount, retrievedRecord.StepCount)
	}

	// todo: Separate test cases for update, delete etc.
}
