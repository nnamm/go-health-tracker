package handlers

import (
	"bytes"
	"encoding/json"
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

	dateStr := time.Now().Format("2006-01-02")
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		t.Fatal(err)
	}
	record := &models.HealthRecord{
		Date:      date,
		StepCount: 10000,
	}
	body, _ := json.Marshal(record)

	req, err := http.NewRequest("POST", "/health", bytes.NewBuffer(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler.CreateHealthRecord(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}
}

func TestReadHealthRecord(t *testing.T) {
	// todo
}

func TestUpdateHealthRecord(t *testing.T) {
	// todo
}

func TestDeleteHealthRecord(t *testing.T) {
	// todo
}
