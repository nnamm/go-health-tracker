package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

type mockDB struct{}

var fixedTime = time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC)

func (m *mockDB) CreateHealthRecord(hr *models.HealthRecord) error {
	return nil
}

func (m *mockDB) ReadHealthRecord(date time.Time) (*models.HealthRecord, error) {
	return &models.HealthRecord{
		ID:        1,
		Date:      date,
		StepCount: 10000,
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}, nil
}

func (m *mockDB) ReadHealthRecordsByYear(year int) ([]models.HealthRecord, error) {
	return []models.HealthRecord{
		{
			ID:        1,
			Date:      time.Date(year, 8, 11, 0, 0, 0, 0, time.UTC),
			StepCount: 10000,
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
	}, nil
}

func (m *mockDB) ReadHealthRecordsByYearMonth(year, month int) ([]models.HealthRecord, error) {
	return []models.HealthRecord{
		{
			ID:        1,
			Date:      time.Date(year, 8, 11, 0, 0, 0, 0, time.UTC),
			StepCount: 10000,
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		},
	}, nil
}

func (m *mockDB) UpdateHealthRecord(hr *models.HealthRecord) error {
	// todo
	return nil
}

func (m *mockDB) DeleteHealthRecord(date time.Time) error {
	// todo
	return nil
}

// func TestCreateHealthRecord(t *testing.T) {
// 	handler := NewHealthRecordHandler(&mockDB{})
//
// 	tests := []struct {
// 		name           string
// 		requestBody    string
// 		expectedStatus int
// 	}{
// 		{
// 			name:           "Valid request",
// 			requestBody:    `{"date": "2023-08-11", "step_count": 10000}`,
// 			expectedStatus: http.StatusCreated,
// 		},
// 		{
// 			name:           "Empty request body",
// 			requestBody:    "",
// 			expectedStatus: http.StatusBadRequest,
// 		},
// 		{
// 			name:           "Invalid JSON",
// 			requestBody:    `{"date": "2023-08-11", "step_count": "Invalid"}`,
// 			expectedStatus: http.StatusBadRequest,
// 		},
// 		{
// 			name:           "Invalid date",
// 			requestBody:    `{"date": "invalid-date", "step_count": 10000}`,
// 			expectedStatus: http.StatusBadRequest,
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			req, err := http.NewRequest(http.MethodPost, "/health", bytes.NewBufferString(tt.requestBody))
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			rr := httptest.NewRecorder()
// 			handler.CreateHealthRecord(rr, req)
//
// 			if status := rr.Code; status != tt.expectedStatus {
// 				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
// 			}
// 		})
// 	}
// }

func TestGetHealthRecords(t *testing.T) {
	mockDB := &mockDB{}
	handler := NewHealthRecordHandler(mockDB)

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedBody   string
		expectError    bool
	}{
		{
			name:           "Get by date",
			url:            "/health/records?date=20240811",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":1, "date":"2024-08-11","step_count":10000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"}]`,
			expectError:    false,
		},
		{
			name:           "Get by year",
			url:            "/health/records?year=2024",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":1, "date":"2024-08-11","step_count":10000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"}]`,
			expectError:    false,
		},
		{
			name:           "Get by year and month",
			url:            "/health/records?year=2024&month=08",
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"id":1, "date":"2024-08-11","step_count":10000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"}]`,
			expectError:    false,
		},
		{
			name:           "Invalid date format",
			url:            "/health/records?date=2024-08-11",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `[{"error":"Invalid date format. Use YYYYMMDD"}]`,
			expectError:    true,
		},
		// {
		// 	name:           "Invalid year format",
		// 	url:            "/health/records?year=24",
		// 	expectedStatus: http.StatusBadRequest,
		// 	expectedBody:   `[{"error":"Invalid year format. Use YYYY"}]`,
		// },
		// {
		// 	name:           "Invalid month format",
		// 	url:            "/health/records?year=2024&month=8",
		// 	expectedStatus: http.StatusBadRequest,
		// 	expectedBody:   `[{"error":"Invalid month format. Use MM"}]`,
		// },
		// {
		// 	name:           "Month without year",
		// 	url:            "/health/records?month=08",
		// 	expectedStatus: http.StatusBadRequest,
		// 	expectedBody:   `[{"error":"Year is required when month is specified"}]`,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tt.url, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.GetHealthRecords(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedBody)
			}

			if tt.expectError {
				// Error cases
				var errResponse map[string]string
				err = json.Unmarshal(rr.Body.Bytes(), &errResponse)
				if err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}
				if errResponse["error"] != tt.expectedBody[10:len(tt.expectedBody)-2] { // Remove {"error": and }
					t.Errorf("handler returned unexpected error: got %v want %v", errResponse["error"], tt.expectedBody[10:len(tt.expectedBody)-2])
				}
			} else {
				// Normal cases
				var gotRecords []models.HealthRecord
				err = json.Unmarshal(rr.Body.Bytes(), &gotRecords)
				if err != nil {
					t.Fatalf("Faild to unmarshal response body: %v", err)
				}

				var expectedRecords []models.HealthRecord
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedRecords)
				if err != nil {
					t.Fatalf("Failed to unmarshal expected body: %v", err)
				}

				if !reflect.DeepEqual(gotRecords, expectedRecords) {
					t.Errorf("handler returned unexpected body: got %+v want %+v", gotRecords, expectedRecords)
				}

			}
		})
	}
}

// func TestReadHealthRecord(t *testing.T) {
// 	handler := NewHealthRecordHandler(&mockDB{})
//
// 	tests := []struct {
// 		name           string
// 		queryDate      string
// 		expectedStatus int
// 	}{
// 		{
// 			name:           "Valid date",
// 			queryDate:      "2023-08-11",
// 			expectedStatus: http.StatusOK,
// 		},
// 		{
// 			name:           "Invalid date format",
// 			queryDate:      "2023/08/11",
// 			expectedStatus: http.StatusBadRequest,
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			req, err := http.NewRequest(http.MethodGet, "/health?date="+tt.queryDate, nil)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			rr := httptest.NewRecorder()
// 			handler.GetHealthRecord(rr, req)
//
// 			if status := rr.Code; status != tt.expectedStatus {
// 				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
// 			}
// 		})
// 	}
// }

// func TestUpdateHealthRecord(t *testing.T) {
// 	handler := NewHealthRecordHandler(&mockDB{})
//
// 	validBody := `{"date": "2023-08-11", "step_count": 12000}`
// 	req, err := http.NewRequest(http.MethodPut, "/health", bytes.NewBufferString(validBody))
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	rr := httptest.NewRecorder()
// 	handler.UpdateHealthRecord(rr, req)
//
// 	if status := rr.Code; status != http.StatusOK {
// 		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
// 	}
// }

// func TestDeleteHealthRecord(t *testing.T) {
// 	handler := NewHealthRecordHandler(&mockDB{})
//
// 	req, err := http.NewRequest(http.MethodDelete, "/health?date=2023-08-11", nil)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	rr := httptest.NewRecorder()
// 	handler.DeleteHealthRecord(rr, req)
//
// 	if status := rr.Code; status != http.StatusNoContent {
// 		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
// 	}
// }
