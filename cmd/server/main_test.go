package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

	// Set up server for testing
	healthHandler := handlers.NewHealthRecordHandler(db)

	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routeHandler(healthHandler)(w, r)
	}))

	// Run all tests
	code := m.Run()

	// Clean up
	testServer.Close()
	db.Close()

	os.Exit(code)
}

func TestHealthRecordIntegration(t *testing.T) {
	// Create a test record
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
		t.Fatalf("failed to marshal record: %v", err)
	}

	t.Logf("sending create JSON: %s", string(body))

	// TEST: create a health record
	req, err := http.NewRequest("POST", testServer.URL+"/health/records", bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to send request: %v", err)
	}
	defer res.Body.Close()

	// Check: status code
	if res.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(res.Body)
		t.Fatalf("expected status created, got %v. response body: %s", res.Status, string(bodyBytes))
	}

	// Check: retrieve the record
	queryParam := now.Format("20060102")
	res, err = http.Get(testServer.URL + "/health/records?date=" + queryParam)
	if err != nil {
		t.Fatalf("failed to get health record: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %v", res.Status)
	}

	// Check: verify response body
	var result handlers.HealthRecordResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Records) == 0 {
		t.Fatalf("expected at least one record in response")
	}
	retrievedRecord := result.Records[0]
	if retrievedRecord.StepCount != record.StepCount {
		t.Errorf("expected step count %d, got %d", record.StepCount, retrievedRecord.StepCount)
	}

	// TEST: update the record
	updatedStepCount := 12500
	updatedRecord := JSONRecord{
		Date:      now.Format("2006-01-02"),
		StepCount: updatedStepCount,
	}

	updatedBody, err := json.Marshal(updatedRecord)
	if err != nil {
		t.Fatalf("failed to marshal updated record: %v", err)
	}

	t.Logf("sending update JSON: %s", string(updatedBody))

	updateReq, err := http.NewRequest("PUT", testServer.URL+"/health/records", bytes.NewBuffer(updatedBody))
	if err != nil {
		t.Fatalf("failed to create update request: %v", err)
	}
	updateReq.Header.Set("Content-Type", "application/json")

	updateRes, err := client.Do(updateReq)
	if err != nil {
		t.Fatalf("failed to send update request: %v", err)
	}
	defer updateRes.Body.Close()

	// Check: status code
	if updateRes.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(updateRes.Body)
		t.Fatalf("expected status OK, got %v. response body: %s", updateRes.Status, string(bodyBytes))
	}

	// Check: verify the update worked
	verifyRes, err := http.Get(testServer.URL + "/health/records?date=" + queryParam)
	if err != nil {
		t.Fatalf("failed to get updated health record: %v", err)
	}
	defer verifyRes.Body.Close()

	if verifyRes.StatusCode != http.StatusOK {
		t.Errorf("expected status OK after update, got %v", verifyRes.Status)
	}

	var updatedResult handlers.HealthRecordResult
	if err := json.NewDecoder(verifyRes.Body).Decode(&updatedResult); err != nil {
		t.Fatalf("failed to decode updated response: %v", err)
	}

	if len(updatedResult.Records) == 0 {
		t.Fatalf("expected at least one record in updated response")
	}

	verifiedRecord := updatedResult.Records[0]
	if verifiedRecord.StepCount != updatedStepCount {
		t.Errorf("expected updated step count %d, got %d", updatedStepCount, verifiedRecord.StepCount)
	}

	// TEST: delete the record
	t.Logf("deleting record for date: %s", queryParam)
	deleteReq, err := http.NewRequest("DELETE", testServer.URL+"/health/records?date="+queryParam, nil)
	if err != nil {
		t.Fatalf("failed to create delete request: %v", err)
	}

	deleteRes, err := client.Do(deleteReq)
	if err != nil {
		t.Fatalf("failed to send delete request: %v", err)
	}
	defer deleteRes.Body.Close()

	// Check: status code
	if deleteRes.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(deleteRes.Body)
		t.Fatalf("expected status OK for delete, got %v. response body: %s", deleteRes.Status, string(bodyBytes))
	}

	// Check: cerify the record was deleted
	checkDeletedRes, err := http.Get(testServer.URL + "/health/records?date=" + queryParam)
	if err != nil {
		t.Fatalf("failed to check deleted record: %v", err)
	}
	defer checkDeletedRes.Body.Close()

	if checkDeletedRes.StatusCode != http.StatusNotFound {
		t.Errorf("expected status NotFound after delete, got %v", checkDeletedRes.Status)
	}
}

func TestHealthRecordInvalidPattern(t *testing.T) {
	// TEST: invalid path
	invalidReq, err := http.NewRequest("GET", testServer.URL+"/health/record", nil)
	if err != nil {
		t.Fatalf("failed to create invalid request: %v", err)
	}

	client := &http.Client{}
	invalidRes, err := client.Do(invalidReq)
	if err != nil {
		t.Fatalf("failed to send invalid request: %v", err)
	}
	defer invalidRes.Body.Close()

	// Check: status code
	if invalidRes.StatusCode != http.StatusNotFound {
		t.Errorf("expected status NotFound for invalid path, got %v", invalidRes.Status)
	}

	// TEST: invalid method
	invalidMethodReq, err := http.NewRequest("PATCH", testServer.URL+"/health/records", nil)
	if err != nil {
		t.Fatalf("failed to create method request: %v", err)
	}

	invalidMethodRes, err := client.Do(invalidMethodReq)
	if err != nil {
		t.Fatalf("failed to send method request: %v", err)
	}

	// Check: status code
	if invalidMethodRes.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("expected status MethodNotAllowed for invalid method, got %v", invalidMethodRes.Status)
	}
}

func TestRouting(t *testing.T) {
	tests := []struct {
		name             string
		method           string
		path             string
		requestHeaders   map[string]string
		requestBody      string
		wantStatus       int
		wantHeaders      map[string]string
		wantBodyContains []string
	}{
		{
			name:        "successful - create health record",
			method:      "POST",
			path:        "/health/records",
			requestBody: `{"date":"2024-05-01","step_count":10000}`,
			requestHeaders: map[string]string{
				"Content-Type": "application/json",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "successful - get health record",
			method:     "GET",
			path:       "/health/records?date=20240501",
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			name:       "invalid path",
			method:     "GET",
			path:       "/invalid/path",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid method",
			method:     "PATCH",
			path:       "/health/records",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "CORS preflight request",
			method: "OPTIONS",
			path:   "/health/records",
			requestHeaders: map[string]string{
				"Origin":                        "http://localhost:3000",
				"Access-Control-Request-Method": "POST",
			},
			wantStatus: http.StatusOK,
			wantHeaders: map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
			},
		},
		{
			name:       "path normalization - trailing slash",
			method:     "GET",
			path:       "/health/records/",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, testServer.URL+tt.path,
				bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatalf("リクエスト作成失敗: %v", err)
			}

			for k, v := range tt.requestHeaders {
				req.Header.Set(k, v)
			}

			client := &http.Client{}
			res, err := client.Do(req)
			if err != nil {
				t.Fatalf("リクエスト送信失敗: %v", err)
			}
			defer res.Body.Close()

			if res.StatusCode != tt.wantStatus {
				body, _ := io.ReadAll(res.Body)
				t.Errorf("ステータスコード不一致: got=%d, want=%d, body=%s",
					res.StatusCode, tt.wantStatus, string(body))
			}

			for k, want := range tt.wantHeaders {
				if got := res.Header.Get(k); got != want {
					t.Errorf("ヘッダー %s の値が不一致: got=%s, want=%s", k, got, want)
				}
			}

			if len(tt.wantBodyContains) > 0 {
				body, _ := io.ReadAll(res.Body)
				bodyStr := string(body)
				for _, substr := range tt.wantBodyContains {
					if !strings.Contains(bodyStr, substr) {
						t.Errorf("レスポンスボディに期待文字列が含まれていない: %s", substr)
					}
				}
			}
		})
	}
}

func TestServerConfiguration(t *testing.T) {
	// backup original emv data
	originalDBPath := os.Getenv("DB_PATH")
	originalPort := os.Getenv("PORT")

	// restore
	defer func() {
		os.Setenv("DB_PATH", originalDBPath)
		os.Setenv("PORT", originalPort)
	}()

	tests := []struct {
		name       string
		envVars    map[string]string
		wantDBPath string
		wantPort   string
	}{
		{
			name: "default settings",
			envVars: map[string]string{
				"DB_PATH": "",
				"PORT":    "",
			},
			wantDBPath: "./health_tracker.db",
			wantPort:   "8000",
		},
		{
			name: "custom settings",
			envVars: map[string]string{
				"DB_PATH": "/custom/path.db",
				"PORT":    "9000",
			},
			wantDBPath: "/custom/path.db",
			wantPort:   "9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set env variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			dbPath, port := getServerConfig()

			if dbPath != tt.wantDBPath {
				t.Errorf("db path mismatch: got=%s, want=%s", dbPath, tt.wantDBPath)
			}

			if port != tt.wantPort {
				t.Errorf("port number mismatch: got=%s, want=%s", port, tt.wantPort)
			}
		})
	}
}

// getServerConfig retrieves the database path and port from environment variables.
func getServerConfig() (dbPath, port string) {
	dbPath = os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./health_tracker.db"
	}

	port = os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	return dbPath, port
}
