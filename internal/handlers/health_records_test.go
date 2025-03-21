package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database/mock"
	"github.com/nnamm/go-health-tracker/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// type mockDB struct {
// 	mu            sync.RWMutex
// 	records       map[time.Time]*models.HealthRecord
// 	createFunc    func(*models.HealthRecord) (*models.HealthRecord, error)
// 	readFunc      func(time.Time) (*models.HealthRecord, error)
// 	readYearFunc  func(int) ([]models.HealthRecord, error)
// 	readMonthFunc func(int, int) ([]models.HealthRecord, error)
// 	updateFunc    func(*models.HealthRecord) error
// 	deleteFunc    func(time.Time) error
// }
//
// func newMockDB() *mockDB {
// 	return &mockDB{
// 		records: make(map[time.Time]*models.HealthRecord),
// 	}
// }

// var (
// 	date0710          = time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)
// 	date0811          = time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC)
// 	date0812          = time.Date(2024, 8, 12, 0, 0, 0, 0, time.UTC)
// 	fixedDateTime0710 = time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC)
// 	fixedDateTime0811 = time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC)
// 	fixedDateTime0812 = time.Date(2024, 8, 12, 0, 0, 0, 0, time.UTC)
// )
//
// func (m *mockDB) CreateHealthRecord(hr *models.HealthRecord) (*models.HealthRecord, error) {
// 	m.mu.Lock()
// 	defer m.mu.Unlock()
//
// 	if m.createFunc != nil {
// 		return m.createFunc(hr)
// 	}
//
// 	if hr.Date.IsZero() {
// 		return nil, fmt.Errorf("date is required")
// 	}
//
// 	if _, exists := m.records[hr.Date]; exists {
// 		return nil, fmt.Errorf("record already exists for date: %v", hr.Date)
// 	}
//
// 	record := &models.HealthRecord{
// 		ID:        int64(len(m.records) + 1),
// 		Date:      hr.Date,
// 		StepCount: hr.StepCount,
// 		CreatedAt: time.Now(),
// 		UpdatedAt: time.Now(),
// 	}
//
// 	m.records[hr.Date] = record
// 	return record, nil
// }
//
// func (m *mockDB) ReadHealthRecord(date time.Time) (*models.HealthRecord, error) {
// 	return &models.HealthRecord{
// 		ID:        1,
// 		Date:      date0710,
// 		StepCount: 10000,
// 		CreatedAt: fixedDateTime0710,
// 		UpdatedAt: fixedDateTime0710,
// 	}, nil
// }
//
// func (m *mockDB) ReadHealthRecordsByYear(year int) ([]models.HealthRecord, error) {
// 	return []models.HealthRecord{
// 		{
// 			ID:        1,
// 			Date:      date0710,
// 			StepCount: 10000,
// 			CreatedAt: fixedDateTime0710,
// 			UpdatedAt: fixedDateTime0710,
// 		},
// 		{
// 			ID:        2,
// 			Date:      date0811,
// 			StepCount: 11000,
// 			CreatedAt: fixedDateTime0811,
// 			UpdatedAt: fixedDateTime0811,
// 		},
// 		{
// 			ID:        3,
// 			Date:      date0812,
// 			StepCount: 12000,
// 			CreatedAt: fixedDateTime0812,
// 			UpdatedAt: fixedDateTime0812,
// 		},
// 	}, nil
// }
//
// func (m *mockDB) ReadHealthRecordsByYearMonth(year, month int) ([]models.HealthRecord, error) {
// 	return []models.HealthRecord{
// 		{
// 			ID:        2,
// 			Date:      date0811,
// 			StepCount: 11000,
// 			CreatedAt: fixedDateTime0811,
// 			UpdatedAt: fixedDateTime0811,
// 		},
// 		{
// 			ID:        3,
// 			Date:      date0812,
// 			StepCount: 12000,
// 			CreatedAt: fixedDateTime0812,
// 			UpdatedAt: fixedDateTime0812,
// 		},
// 	}, nil
// }
//
// func (m *mockDB) UpdateHealthRecord(hr *models.HealthRecord) error {
// 	// todo
// 	return nil
// }
//
// func (m *mockDB) DeleteHealthRecord(date time.Time) error {
// 	// todo
// 	return nil
// }

func TestCreateHealthRecord(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mock.MockDB)
		requestBody    string
		expectedStatus int
		wantError      bool
		errorMessage   string
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "successful creation",
			requestBody:    testutil.CreateHealthRecordJSON(t, time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC), 10000),
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				testutil.ParseJSONResponse(t, rr.Body.Bytes(), &result)
				require.Len(t, result.Records, 1)

				record := result.Records[0]
				assert.Equal(t, 10000, record.StepCount)
				assert.Equal(t, "2024-07-10", record.Date.Format("2006-01-02"))
			},
		},
		{
			name: "database error",
			setupMock: func(db *mock.MockDB) {
				db.SetSimulateDBError(true)
			},
			requestBody:    testutil.CreateHealthRecordJSON(t, time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC), 10000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to create health record",
		},
		{
			name:           "empty request body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name:           "invalid json",
			requestBody:    `{"date": "2024-01-01", "step_count": "Invalid"}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name:           "missing date",
			requestBody:    `{"step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "unexpected date type",
		},
		{
			name:           "zero date",
			requestBody:    `{"date": "0001-01-01", "step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "date is required",
		},
		{
			name:           "step count is negative",
			requestBody:    `{"date": "2024-01-01", "step_count": -5000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "step count must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := mock.NewMockDB()
			if tt.setupMock != nil {
				tt.setupMock(mockDB)
			}

			handler := NewHealthRecordHandler(mockDB)
			ctx := context.Background()
			req := testutil.CreateRequestContext(ctx, http.MethodPost, "/health", tt.requestBody)

			// Act
			rr := testutil.ExecuteHandlerRequest(t, handler.CreateHealthRecord, req)

			// Assert
			testutil.AssertHTTPStatusCode(t, rr.Code, tt.expectedStatus)

			if tt.wantError {
				testutil.AssertErrorResponse(t, rr.Body.Bytes(), tt.errorMessage)
			} else if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

// func TestGetHealthRecords(t *testing.T) {
// 	mockDB := &mockDB{}
// 	handler := NewHealthRecordHandler(mockDB)
//
// 	tests := []struct {
// 		name           string
// 		url            string
// 		expectedStatus int
// 		expectedBody   string
// 		expectError    bool
// 	}{
// 		{
// 			name:           "Get by date",
// 			url:            "/health/records?date=20240710",
// 			expectedStatus: http.StatusOK,
// 			expectedBody:   `{"records": [{"id":1, "date":"2024-07-10","step_count":10000, "created_at":"2024-07-10T00:00:00Z", "updated_at":"2024-07-10T00:00:00Z"}]}`,
// 			expectError:    false,
// 		},
// 		{
// 			name:           "Get by year",
// 			url:            "/health/records?year=2024",
// 			expectedStatus: http.StatusOK,
// 			expectedBody: `{"records": [
//                               {"id":1, "date":"2024-07-10","step_count":10000, "created_at":"2024-07-10T00:00:00Z", "updated_at":"2024-07-10T00:00:00Z"},
//                               {"id":2, "date":"2024-08-11","step_count":11000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"},
//                               {"id":3, "date":"2024-08-12","step_count":12000, "created_at":"2024-08-12T00:00:00Z", "updated_at":"2024-08-12T00:00:00Z"}
//                            ]}`,
// 			expectError: false,
// 		},
// 		{
// 			name:           "Get by year and month",
// 			url:            "/health/records?year=2024&month=08",
// 			expectedStatus: http.StatusOK,
// 			expectedBody: `{"records": [
//                               {"id":2, "date":"2024-08-11","step_count":11000, "created_at":"2024-08-11T00:00:00Z", "updated_at":"2024-08-11T00:00:00Z"},
//                               {"id":3, "date":"2024-08-12","step_count":12000, "created_at":"2024-08-12T00:00:00Z", "updated_at":"2024-08-12T00:00:00Z"}
//                            ]}`,
// 			expectError: false,
// 		},
// 		{
// 			name:           "Invalid date format",
// 			url:            "/health/records?date=2024-07-10",
// 			expectedStatus: http.StatusBadRequest,
// 			expectedBody:   `[{"error":Invalid date format: 2024-07-10 (Use YYYYMMDD)}]`,
// 			expectError:    true,
// 		},
// 		{
// 			name:           "Invalid year format",
// 			url:            "/health/records?year=24",
// 			expectedStatus: http.StatusBadRequest,
// 			expectedBody:   `[{"error":Invalid year format: 24 (Use YYYY)}]`,
// 			expectError:    true,
// 		},
// 		{
// 			name:           "Invalid month format",
// 			url:            "/health/records?year=2024&month=8",
// 			expectedStatus: http.StatusBadRequest,
// 			expectedBody:   `[{"error":Invalid month format: 8 (Use MM)}]`,
// 			expectError:    true,
// 		},
// 		{
// 			name:           "Invalid query parameters",
// 			url:            "/health/records?month=08",
// 			expectedStatus: http.StatusBadRequest,
// 			expectedBody:   `[{"error":Invalid query parameters: expected date or year}]`,
// 			expectError:    true,
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			req, err := http.NewRequest("GET", tt.url, nil)
// 			if err != nil {
// 				t.Fatal(err)
// 			}
//
// 			rr := httptest.NewRecorder()
// 			handler.GetHealthRecords(rr, req)
//
// 			if status := rr.Code; status != tt.expectedStatus {
// 				t.Errorf("Handler returned wrong status code: got %v want %v", status, tt.expectedBody)
// 			}
//
// 			if tt.expectError {
// 				// Error cases
// 				var errResponse map[string]string
// 				err = json.Unmarshal(rr.Body.Bytes(), &errResponse)
// 				if err != nil {
// 					t.Fatalf("Failed to unmarshal error response: %v", err)
// 				}
// 				if errResponse["error"] != tt.expectedBody[10:len(tt.expectedBody)-2] { // Remove {"error": and }
// 					t.Errorf("handler returned unexpected error: got %v want %v", errResponse["error"], tt.expectedBody[10:len(tt.expectedBody)-2])
// 				}
// 			} else {
// 				// Normal cases
// 				var gotResult HealthRecordResult
// 				err = json.Unmarshal(rr.Body.Bytes(), &gotResult)
// 				if err != nil {
// 					t.Fatalf("Faild to unmarshal response body: %v", err)
// 				}
//
// 				var expectedResult HealthRecordResult
// 				err = json.Unmarshal([]byte(tt.expectedBody), &expectedResult)
// 				if err != nil {
// 					t.Fatalf("Failed to unmarshal expected body: %v", err)
// 				}
//
// 				if !reflect.DeepEqual(gotResult, expectedResult) {
// 					t.Errorf("handler returned unexpected body: got %+v want %+v", gotResult, expectedResult)
// 				}
//
// 			}
// 		})
// 	}
// }
