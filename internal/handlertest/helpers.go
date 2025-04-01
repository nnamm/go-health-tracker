package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database/mock"
	"github.com/nnamm/go-health-tracker/internal/models"
)

// CreateTestContext returns the context and cancel function for testing
func CreateTestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// CreateRequestContext creates an HTTP request with JSON content type for testing
func CreateRequestContext(ctx context.Context, method, url, body string) *http.Request {
	req, _ := http.NewRequestWithContext(ctx, method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

// AssertHTTPStatusCode checks if the status code is as expected
func AssertHTTPStatusCode(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("Status code = %d, want %d", got, want)
	}
}

// CreateTestHealthRecord creates a new health record for testing
func CreateTestHealthRecord(date time.Time, stepCount int) *models.HealthRecord {
	return &models.HealthRecord{
		Date:      date,
		StepCount: stepCount,
	}
}

// FormatDateForAPI formats the date for API request(YYYYMMDD)
func FormatDateForAPI(t time.Time) string {
	return t.Format("20060102")
}

// ParseAPIDateFormat parses the date string(YYYYMMDD) from API request
func ParseAPIDateFormat(dateStr string) (time.Time, error) {
	return time.Parse("20060102", dateStr)
}

// ParseJSONResponse parses a JSON response body into the given target
func ParseJSONResponse(t *testing.T, body []byte, target any) {
	t.Helper()
	err := json.Unmarshal(body, target)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
}

// AssertErrorResponse checks if the error response contains the expected message
func AssertErrorResponse(t *testing.T, body []byte, expectedMessage string) {
	t.Helper()
	var errResponse map[string]string
	err := json.Unmarshal(body, &errResponse)
	if err != nil {
		t.Fatalf("Failed to parse error response: %v", err)
	}

	errMsg, ok := errResponse["error"]
	if !ok {
		t.Errorf("Error response does not contain 'error' field")
		return
	}

	if !strings.Contains(errMsg, expectedMessage) {
		t.Errorf("Error message '%s' does not contain expected message '%s'", errMsg, expectedMessage)
	}
}

// SetupMockDBWithRecords sets up a mock DB with the given records
func SetupMockDBWithRecords(t *testing.T, records ...*models.HealthRecord) *mock.MockDB {
	t.Helper()
	mockDB := mock.NewMockDB()
	ctx := context.Background()

	for _, record := range records {
		_, err := mockDB.CreateHealthRecord(ctx, record)
		if err != nil {
			t.Fatalf("Failed to create record: %v", err)
		}
	}

	return mockDB
}

// ExecuteHandlerRequest executes a handler with the given request and returns the response
func ExecuteHandlerRequest(t *testing.T, handler http.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	handler(rr, req)
	return rr
}

// CreateHealthRecordJSON creates a JSON representation of a health record
func CreateHealthRecordJSON(t *testing.T, date time.Time, stepCount int) string {
	t.Helper()
	return fmt.Sprintf(`{"date": "%s", "step_count": %d}`, date.Format("2006-01-02"), stepCount)
}
