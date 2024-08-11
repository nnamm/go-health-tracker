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
	tests := []struct {
		name     string
		jsonData string
		wantDate time.Time
		wantErr  bool
	}{
		{
			name:     "Valid date (2006-01-02)",
			jsonData: `{"id":1,"date":"2024-08-11","step_count":10000}`,
			wantDate: time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "Valid date (RFC3339)",
			jsonData: `{"id":1,"date":"2024-08-11T15:04:05Z","step_count":10000}`,
			wantDate: time.Date(2024, 8, 11, 15, 4, 5, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "Invalid date",
			jsonData: `{"id":1,"date":"invalid-date","step_count":10000}`,
			wantDate: time.Time{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hr HealthRecord
			err := json.Unmarshal([]byte(tt.jsonData), &hr)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !hr.Date.Equal(tt.wantDate) {
				t.Errorf("UnmarshalJSON() got Date = %v, want %v", hr.Date, tt.wantDate)
			}

			if !tt.wantErr && hr.StepCount != 10000 {
				t.Errorf("UnmarshalJSON() got StepCount = %d, want 10000", hr.StepCount)
			}
		})
	}
}

func TstHealthRecord_UnmarshalJSONInvalidDate(t *testing.T) {
	jsonData := []byte(`{"id":0,"date":"1691712000","step_count":10000}`)

	var hr HealthRecord
	err := json.Unmarshal(jsonData, &hr)
	if err == nil {
		t.Fatalf("Failed to unmarshal HealthRecord with timestamp: %v", err)
	}

	expectedDate := time.Date(2023, 8, 11, 0, 0, 0, 0, time.UTC)
	if !hr.Date.Equal(expectedDate) {
		t.Errorf("Unexpected Date. Got %v, want %v", hr.Date, expectedDate)
	}
}
