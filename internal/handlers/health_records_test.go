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

var (
	fixedTime0710 = time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)
	fixedTime0811 = time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC)
	fixedTime0812 = time.Date(2024, 8, 12, 0, 0, 0, 0, time.UTC)
)

func (m *mockDB) CreateHealthRecord(hr *models.HealthRecord) error {
	return nil
}

func (m *mockDB) ReadHealthRecord(date time.Time) (*models.HealthRecord, error) {
	return &models.HealthRecord{
		ID:        1,
		Date:      date,
		StepCount: 10000,
		CreatedAt: fixedTime0710,
		UpdatedAt: fixedTime0710,
	}, nil
}

func (m *mockDB) ReadHealthRecordsByYear(year int) ([]models.HealthRecord, error) {
	return []models.HealthRecord{
		{
			ID:        1,
			Date:      time.Date(year, 7, 10, 0, 0, 0, 0, time.UTC),
			StepCount: 10000,
			CreatedAt: fixedTime0710,
			UpdatedAt: fixedTime0710,
		},
		{
			ID:        2,
			Date:      time.Date(year, 8, 11, 0, 0, 0, 0, time.UTC),
			StepCount: 11000,
			CreatedAt: fixedTime0811,
			UpdatedAt: fixedTime0811,
		},
		{
			ID:        3,
			Date:      time.Date(year, 8, 12, 0, 0, 0, 0, time.UTC),
			StepCount: 12000,
			CreatedAt: fixedTime0812,
			UpdatedAt: fixedTime0812,
		},
	}, nil
}

func (m *mockDB) ReadHealthRecordsByYearMonth(year, month int) ([]models.HealthRecord, error) {
	return []models.HealthRecord{
		{
			ID:        2,
			Date:      time.Date(year, 8, 11, 0, 0, 0, 0, time.UTC),
			StepCount: 11000,
			CreatedAt: fixedTime0811,
			UpdatedAt: fixedTime0811,
		},
		{
			ID:        3,
			Date:      time.Date(year, 8, 12, 0, 0, 0, 0, time.UTC),
			StepCount: 12000,
			CreatedAt: fixedTime0812,
			UpdatedAt: fixedTime0812,
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
			url:            "/health/records?date=20240710",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"records": [{"id":1, "date":"2024-07-10","step_count":10000, "created_at":"2024-07-10T00:00:00Z", "updated_at":"2024-07-10T00:00:00Z"}]}`,
			expectError:    false,
		},
		{
			name:           "Get by year",
			url:            "/health/records?year=2024",
			expectedStatus: http.StatusOK,
			expectedBody: `{"records": [
                              {"id":1, "date":"2024-07-10","step_count":10000, "created_at":"2024-07-10T00:00:00Z", "updated_at":"2024-07-10T00:00:00Z"},
                              {"id":2, "date":"2024-08-11","step_count":11000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"},
                              {"id":3, "date":"2024-08-12","step_count":12000, "created_at":"2024-08-12T00:00:00Z", "updated_at":"2024-08-12T00:00:00Z"}
                           ]}`,
			expectError: false,
		},
		{
			name:           "Get by year and month",
			url:            "/health/records?year=2024&month=08",
			expectedStatus: http.StatusOK,
			expectedBody: `{"records": [
                              {"id":2, "date":"2024-08-11","step_count":11000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"},
                              {"id":3, "date":"2024-08-12","step_count":12000, "created_at":"2024-08-12T00:00:00Z", "updated_at":"2024-08-12T00:00:00Z"}
                           ]}`,
			expectError: false,
		},
		{
			name:           "Invalid date format",
			url:            "/health/records?date=2024-07-10",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `[{"error":Invalid date format: 2024-07-10 (Use YYYYMMDD)}]`,
			expectError:    true,
		},
		{
			name:           "Invalid year format",
			url:            "/health/records?year=24",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `[{"error":Invalid year format: 24 (Use YYYY)}]`,
			expectError:    true,
		},
		{
			name:           "Invalid month format",
			url:            "/health/records?year=2024&month=8",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `[{"error":Invalid month format: 8 (Use MM)}]`,
			expectError:    true,
		},
		{
			name:           "Invalid query parameters",
			url:            "/health/records?month=08",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `[{"error":Invalid query parameters: expected date or year}]`,
			expectError:    true,
		},
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
				var gotResult HealthRecordResult
				err = json.Unmarshal(rr.Body.Bytes(), &gotResult)
				if err != nil {
					t.Fatalf("Faild to unmarshal response body: %v", err)
				}

				var expectedResult HealthRecordResult
				err = json.Unmarshal([]byte(tt.expectedBody), &expectedResult)
				if err != nil {
					t.Fatalf("Failed to unmarshal expected body: %v", err)
				}

				if !reflect.DeepEqual(gotResult, expectedResult) {
					t.Errorf("handler returned unexpected body: got %+v want %+v", gotResult, expectedResult)
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
