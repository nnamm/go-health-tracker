package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database/mock"
	"github.com/nnamm/go-health-tracker/internal/handlertest"
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
			name:           "1.successful creation",
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
			name: "2.database error",
			setupMock: func(db *mock.MockDB) {
				db.SetSimulateDBError(true)
			},
			requestBody:    handlertest.CreateHealthRecordJSON(t, time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC), 10000),
			expectedStatus: http.StatusInternalServerError,
			wantError:      true,
			errorMessage:   "failed to create health record",
		},
		{
			name:           "3.empty request body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name:           "4.invalid json",
			requestBody:    `{"date": "2024-01-01", "step_count": "Invalid"}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "failed to unmarshal health record",
		},
		{
			name:           "5.missing date",
			requestBody:    `{"step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "unexpected date type",
		},
		{
			name:           "6.zero date",
			requestBody:    `{"date": "0001-01-01", "step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "date is required",
		},
		{
			name:           "7.step count is negative",
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
			name: "1.get by date - successful",
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
			name: "2.get by year - successful",
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
			name: "3.get by year and month - successful",
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
			name: "4.data not exist - successful",
			setupMock: func(t *testing.T) *mock.MockDB {
				records := []models.HealthRecord{
					{Date: handlertest.ParseAPIDateFormat("2025-01-01"), StepCount: 10000},
				}
				return handlertest.SetupMockDBWithRecords(t, records)
			},
			queryParams:    "?date=20240101",
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "5.get by date - invalid format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?date=2024/01/01", // Wrong format
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid date format",
		},
		{
			name: "6.get by date - database error",
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
			name: "7.get by year - invalid year format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?year=invalid",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid year format",
		},
		{
			name: "8.get by year and month - invalid month format",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "?year=2024&month=invalid",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid month format",
		},
		{
			name: "9.missing query parameters",
			setupMock: func(t *testing.T) *mock.MockDB {
				return mock.NewMockDB()
			},
			queryParams:    "",
			expectedStatus: http.StatusBadRequest,
			wantError:      true,
			errorMessage:   "invalid query parameters",
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
