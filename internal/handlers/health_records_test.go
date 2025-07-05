package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			name:           "successful - normal creation",
			requestBody:    handlertest.CreateHealthRecordJSON(t, time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC), 10000),
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)
				require.Len(t, result.Records, 1)

				record := result.Records[0]
				assert.Equal(t, 10000, record.StepCount)
				assert.Equal(t, "2024-07-10", record.Date.Format("2006-01-02"))
			},
		},
		{
			name:           "error - empty request body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name:           "error - invalid json",
			requestBody:    `{"date": "2024-01-01", "step_count": "Invalid"}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name:           "error - missing date",
			requestBody:    `{"step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "unexpected date type",
		},
		{
			name:           "error - zero date",
			requestBody:    `{"date": "0001-01-01", "step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "date is required",
		},
		{
			name:           "error - step count is negative",
			requestBody:    `{"date": "2024-01-01", "step_count": -5000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "step count must not be negative",
		},
		{
			name: "error - database error",
			setupMock: func(db *mock.MockDB) {
				db.SetSimulateDBError(true)
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC), 10000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to create health record",
		},
		{
			name: "error - timeout error",
			setupMock: func(db *mock.MockDB) {
				db.SetSimulateTimeout(true)
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC), 10000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to create health record",
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
			req := handlertest.CreateRequestContext(ctx, http.MethodPost, "/health/records", tt.requestBody)

			// Act
			rr := handlertest.ExecuteHandlerRequest(t, handler.CreateHealthRecord, req)

			// Assert
			handlertest.AssertHTTPStatusCode(t, rr.Code, tt.expectedStatus)

			if tt.wantError {
				handlertest.AssertErrorResponse(t, rr.Body.Bytes(), tt.errorMessage)
			} else if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

func TestGetHealthRecord(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*testing.T) *mock.MockDB
		queryParams    string
		expectedStatus int
		wantError      bool
		errorMessage   string
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful - get by date",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2024-01-01"), StepCount: 10000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			queryParams:    "?date=20240101",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 1)
				record := result.Records[0]
				assert.Equal(t, "2024-01-01", record.Date.Format("2006-01-02"))
				assert.Equal(t, 10000, record.StepCount)
			},
		},
		{
			name: "successful - data not exist",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2025-01-01"), StepCount: 10000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			queryParams:    "?date=20240101",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 0)
			},
		},
		{
			name: "successful - get by year",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2024-01-01"), StepCount: 10000},
					{Date: handlertest.ParseAPIDateFormat("2024-02-01"), StepCount: 11000},
					{Date: handlertest.ParseAPIDateFormat("2025-12-01"), StepCount: 12000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			queryParams:    "?year=2024",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 2)
				assert.Equal(t, "2024-01-01", result.Records[0].Date.Format("2006-01-02"))
				assert.Equal(t, 10000, result.Records[0].StepCount)
				assert.Equal(t, "2024-02-01", result.Records[1].Date.Format("2006-01-02"))
				assert.Equal(t, 11000, result.Records[1].StepCount)
			},
		},
		{
			name: "successful - get by year and month",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2024-01-01"), StepCount: 10000},
					{Date: handlertest.ParseAPIDateFormat("2024-01-15"), StepCount: 11000},
					{Date: handlertest.ParseAPIDateFormat("2025-12-01"), StepCount: 12000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			queryParams:    "?year=2024&month=01",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 2)
				assert.Equal(t, "2024-01-01", result.Records[0].Date.Format("2006-01-02"))
				assert.Equal(t, 10000, result.Records[0].StepCount)
				assert.Equal(t, "2024-01-15", result.Records[1].Date.Format("2006-01-02"))
				assert.Equal(t, 11000, result.Records[1].StepCount)
			},
		},
		{
			name: "error - invalid date format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?date=2024/01/01", // Wrong format
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid date format",
		},
		{
			name: "error - invalid year format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?year=invalid",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid year format",
		},
		{
			name: "error - invalid month format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?year=2024&month=invalid",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid month format",
		},
		{
			name: "error - missing query parameters",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid query parameters",
		},
		{
			name: "error - database error",
			setupMock: func(t *testing.T) *mock.MockDB {
				mockDB := mock.NewMockDB()
				mockDB.SetSimulateDBError(true)
				return mockDB
			},
			queryParams:    "?date=20250101",
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to read health record",
		},
		{
			name: "error - timeout",
			setupMock: func(t *testing.T) *mock.MockDB {
				mockDB := mock.NewMockDB()
				mockDB.SetSimulateTimeout(true)
				return mockDB
			},
			queryParams:    "?date=20250101",
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to read health record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.setupMock(t)
			handler := NewHealthRecordHandler(mockDB)
			ctx := context.Background()
			req := handlertest.CreateRequestContext(ctx, http.MethodGet, "/health/records"+tt.queryParams, "")

			// Act
			rr := handlertest.ExecuteHandlerRequest(t, handler.GetHealthRecords, req)

			// Assert
			handlertest.AssertHTTPStatusCode(t, rr.Code, tt.expectedStatus)

			if tt.wantError {
				handlertest.AssertErrorResponse(t, rr.Body.Bytes(), tt.errorMessage)
			} else if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

func TestUpdateHealthRecord(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*testing.T) *mock.MockDB
		requestBody    string
		expectedStatus int
		wantError      bool
		errorMessage   string
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful - normal update",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2025-01-01"), StepCount: 10000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-01-01"), 15000),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 1)
				assert.Equal(t, 15000, result.Records[0].StepCount)
			},
		},
		{
			name: "successful - zero step count",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2025-01-01"), StepCount: 10000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-01-01"), 0),
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 1)
				assert.Equal(t, 0, result.Records[0].StepCount)
			},
		},
		{
			name: "successful - no existing record (no update)",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-01-01"), 15000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "record not found",
		},
		{
			name: "error - invalid request body",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name: "error - validation error (negative step count)",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-01-01"), -10000),
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "step count must not be negative",
		},
		{
			name: "error - validation error (too many step count)",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-01-01"), 1000001),
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "step count is unrealistically high",
		},
		{
			name: "error - database error",
			setupMock: func(t *testing.T) *mock.MockDB {
				mockDB := mock.NewMockDB()
				mockDB.SetSimulateDBError(true)
				return mockDB
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-03-01"), 5000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to update health record",
		},
		{
			name: "error - timeout",
			setupMock: func(t *testing.T) *mock.MockDB {
				mockDB := mock.NewMockDB()
				mockDB.SetSimulateTimeout(true)
				return mockDB
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, handlertest.ParseAPIDateFormat("2025-03-01"), 5000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to update health record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.setupMock(t)
			handler := NewHealthRecordHandler(mockDB)
			ctx := context.Background()
			req := handlertest.CreateRequestContext(ctx, http.MethodGet, "/health/records", tt.requestBody)

			// Act
			rr := handlertest.ExecuteHandlerRequest(t, handler.UpdateHealthRecord, req)

			// Assert
			handlertest.AssertHTTPStatusCode(t, rr.Code, tt.expectedStatus)

			if tt.wantError {
				handlertest.AssertErrorResponse(t, rr.Body.Bytes(), tt.errorMessage)
			} else if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}

func TestDeleteHealthRecord(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*testing.T) *mock.MockDB
		queryParams    string
		expectedStatus int
		wantError      bool
		errorMessage   string
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful - normal delete",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2025-01-01"), StepCount: 10000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			queryParams:    "?date=20250101",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var result HealthRecordResult
				handlertest.ParseJSONResponse(t, rr.Body.Bytes(), &result)

				require.Len(t, result.Records, 0)
			},
		},
		{
			name: "error - record not found",
			setupMock: func(t *testing.T) *mock.MockDB {
				// Setup an empty mock DB without the requested record
				return mock.NewMockDB()
			},
			queryParams:    "?date=20250101",
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "record not found",
		},
		{
			name: "error - invalid date format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?date=2025/01/01", // Wrong format
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid date format",
		},
		{
			name: "error - missing query parameters",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "date parameter is required",
		},
		{
			name: "error - database error",
			setupMock: func(t *testing.T) *mock.MockDB {
				mockDB := mock.NewMockDB()
				mockDB.SetSimulateDBError(true)
				return mockDB
			},
			queryParams:    "?date=20250101",
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to delete health record",
		},
		{
			name: "error - timeout",
			setupMock: func(t *testing.T) *mock.MockDB {
				mockDB := mock.NewMockDB()
				mockDB.SetSimulateTimeout(true)
				return mockDB
			},
			queryParams:    "?date=20250101",
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to delete health record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := tt.setupMock(t)
			handler := NewHealthRecordHandler(mockDB)
			ctx := context.Background()
			req := handlertest.CreateRequestContext(ctx, http.MethodDelete, "/health/records"+tt.queryParams, "")

			// Act
			rr := handlertest.ExecuteHandlerRequest(t, handler.DeleteHealthRecord, req)

			// Assert
			handlertest.AssertHTTPStatusCode(t, rr.Code, tt.expectedStatus)

			if tt.wantError {
				handlertest.AssertErrorResponse(t, rr.Body.Bytes(), tt.errorMessage)
			} else if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}
