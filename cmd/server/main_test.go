package main

import (
	"bytes"
	"encoding/json"
	"io"
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
	now := time.Now().Truncate(24 * time.Hour)
	record := models.HealthRecord{
		Date:      now,
		StepCount: 10000,
	}

	// Use a custom struct for JSON marshaling to ensure consistent date format
	type JSONRecord struct {
		Date      string `json:"date"`
		StepCount int    `json:"step_count"`
	}

	jsonRecord := JSONRecord{
		Date:      now.Format("2006-01-02"),
		StepCount: record.StepCount,
	}

	body, err := json.Marshal(jsonRecord)
	if err != nil {
		t.Fatalf("Failed to marshal record: %v", err)
	}

	// for Debug
	t.Logf("Sending JSON: %s", string(body))

	req, err := http.NewRequest("POST", testServer.URL+"/health", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(res.Body)
		t.Fatalf("Expected status Created, got %v. Response body: %s", res.Status, string(bodyBytes))
	}

	// Retrieve the record
	dateStr := now.Format("2006-01-02")
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
