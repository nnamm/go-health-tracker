package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHealthRecord_MarshalJSON(t *testing.T) {
	date := time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC)
	hr := HealthRecord{
		ID:        1,
		Date:      date,
		StepCount: 10000,
		CreatedAt: date,
		UpdatedAt: date,
	}

	jsonData, err := json.Marshal(hr)
	if err != nil {
		t.Fatalf("Failed to marshal HealthRecord: %v", err)
	}

	expected := `{"id":1,"date":"2024-08-11T00:00:00Z","step_count":10000,"created_at":"2024-08-11T00:00:00Z","updated_at":"2024-08-11T00:00:00Z"}`
	if string(jsonData) != expected {
		t.Errorf("Unexpected JSON output. Got %s, want %s", string(jsonData), expected)
	}
}

func TestHealthRecord_UnmarshalJSON(t *testing.T) {
	jsonData := []byte(`{"id":1,"date":"2024-08-11","step_count":10000,"created_at":"2024-08-11T00:00:00Z","updated_at":"2024-08-11T00:00:00Z"}`)

	var hr HealthRecord
	err := json.Unmarshal(jsonData, &hr)
	if err != nil {
		t.Fatalf("Failed to unmarshal HealthRecord: %v", err)
	}

	expectedDate := time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC)
	if !hr.Date.Equal(expectedDate) {
		t.Errorf("Unexpected Date. Got %v, want %v", hr.Date, expectedDate)
	}
	if hr.ID != 1 {
		t.Errorf("Unexpected ID. Got %d, want 1", hr.ID)
	}
	if hr.StepCount != 10000 {
		t.Errorf("Unexpected StepCount. Got %d, want 10000", hr.StepCount)
	}
}

func TstHealthRecord_UnmarshalJSONInvalidDate(t *testing.T) {
	jsonData := []byte(`{"id":0,"date":"invalid-date","step_count":10000}`)

	var hr HealthRecord
	err := json.Unmarshal(jsonData, &hr)
	if err == nil {
		t.Errorf("Expected an error for invalid date, but got none")
	}
}
