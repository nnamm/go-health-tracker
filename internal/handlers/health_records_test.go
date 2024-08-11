package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

type mockDB struct{}

func (m *mockDB) CreateHealthRecord(hr *models.HealthRecord) error {
	return nil
}

func (m *mockDB) ReadHealthRecord(date time.Time) (*models.HealthRecord, error) {
	return &models.HealthRecord{
		ID:        1,
		Date:      date,
		StepCount: 10000,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
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

func TestCreateHealthRecord(t *testing.T) {
	handler := NewHealthRecordHandler(&mockDB{})

	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
	}{
		{
			name:           "Valid request",
			requestBody:    `{"date": "2023-08-11", "step_count": 10000}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "Empty request body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid JSON",
			requestBody:    `{"date": "2023-08-11", "step_count": "Invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid date",
			requestBody:    `{"date": "invalid-date", "step_count": 10000}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/health", bytes.NewBufferString(tt.requestBody))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.CreateHealthRecord(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestReadHealthRecord(t *testing.T) {
	handler := NewHealthRecordHandler(&mockDB{})

	tests := []struct {
		name           string
		queryDate      string
		expectedStatus int
	}{
		{
			name:           "Valid date",
			queryDate:      "2023-08-11",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid date format",
			queryDate:      "2023/08/11",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/health?date="+tt.queryDate, nil)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler.GetHealthRecord(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tt.expectedStatus)
			}
		})
	}
}

func TestUpdateHealthRecord(t *testing.T) {
	handler := NewHealthRecordHandler(&mockDB{})

	validBody := `{"date": "2023-08-11", "step_count": 12000}`
	req, err := http.NewRequest(http.MethodPut, "/health", bytes.NewBufferString(validBody))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.UpdateHealthRecord(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}

func TestDeleteHealthRecord(t *testing.T) {
	handler := NewHealthRecordHandler(&mockDB{})

	req, err := http.NewRequest(http.MethodDelete, "/health?date=2023-08-11", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.DeleteHealthRecord(rr, req)

	if status := rr.Code; status != http.StatusNoContent {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNoContent)
	}
}
