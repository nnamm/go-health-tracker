package models

import (
	"encoding/json"
	"strings"
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

	got, err := json.Marshal(hr)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	want := `{"id":1,"date":"2024-08-11T00:00:00Z","step_count":10000,"created_at":"2024-08-11T00:00:00Z","updated_at":"2024-08-11T00:00:00Z"}`
	if string(got) != want {
		t.Errorf("marshal result mismatch\ngot: %s\nwaant: %s", got, want)
	}
}

func TestHealthRecord_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
		errMsg  string
	}{
		{
			name:    "simple date format",
			input:   `{"id":1,"date":"2024-08-11","step_count":10000}`,
			want:    time.Date(2024, 8, 11, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "RFC3339 format",
			input:   `{"id":1,"date":"2024-08-11T15:04:05Z","step_count":10000}`,
			want:    time.Date(2024, 8, 11, 15, 4, 5, 0, time.UTC),
			wantErr: false,
		},
		{
			name:    "Invalid date format",
			input:   `{"id":1,"date":"invalid-date","step_count":10000}`,
			want:    time.Time{},
			wantErr: true,
			errMsg:  "invalid date format: invalid-date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hr HealthRecord
			err := json.Unmarshal([]byte(tt.input), &hr)

			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errMsg, err)
				}
				return
			}

			if !hr.Date.Equal(tt.want) {
				t.Errorf("date mismatch\ngot: %v\nwant: %v", hr.Date, tt.want)
			}
		})
	}
}

func TestHealthRecord_UnmarshalJSON_SpecialCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
		errMsg  string
	}{
		{
			name:    "unix timestamp",
			input:   `{"date": 1691712000, "step_count": 10000}`,
			want:    time.Unix(1691712000, 0),
			wantErr: false,
		},
		{
			name:    "null date",
			input:   `{"date": null, "step_count": 10000}`,
			wantErr: true,
			errMsg:  "unexpected date type: <nil>",
		},
		{
			name:    "boolean date",
			input:   `{"date": true, "step_count": 10000}`,
			wantErr: true,
			errMsg:  "unexpected date type: bool",
		},
		{
			name:    "array date",
			input:   `{"date": [], "step_count": 10000}`,
			wantErr: true,
			errMsg:  "unexpected date type: []interface {}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hr HealthRecord
			err := json.Unmarshal([]byte(tt.input), &hr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message mismatch\ngot: %v\nwant: %v", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !hr.Date.Equal(tt.want) {
				t.Errorf("date mismatch\ngot: %v\nwant: %v", hr.Date, tt.want)
			}
		})
	}
}
