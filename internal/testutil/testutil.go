package testutil

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

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
