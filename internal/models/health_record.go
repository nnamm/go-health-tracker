package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// HealthRecord represents a single health tracking record.
// it contains step count data for a specific date.
type HealthRecord struct {
	ID        int64     `json:"id"`
	Date      time.Time `json:"date"`
	StepCount int       `json:"step_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// MarshalJSON implements the json.Marshaler interface.
// converts the record's date to YYYY-MM-DD format JSON output.
func (hr *HealthRecord) MarshalJSON() ([]byte, error) {
	type Alias HealthRecord
	return json.Marshal(&struct {
		Date string `json:"date"`
		*Alias
	}{
		Date:  hr.Date.Format("2006-01-02"),
		Alias: (*Alias)(hr),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// supports multipul formats: RFC3339 and YYYY-MM-DD.
func (hr *HealthRecord) UnmarshalJSON(date []byte) error {
	type Alias HealthRecord
	aux := &struct {
		Date any `json:"date"`
		*Alias
	}{
		Alias: (*Alias)(hr),
	}
	if err := json.Unmarshal(date, &aux); err != nil {
		return fmt.Errorf("failed to unmarshal health record: %w", err)
	}

	switch v := aux.Date.(type) {
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			hr.Date = t
			return nil
		}

		if t, err := time.Parse("2006-01-02", v); err == nil {
			hr.Date = t
			return nil
		}

		return fmt.Errorf("invalid date format: %s", v)
	case float64:
		hr.Date = time.Unix(int64(v), 0)
		return nil
	default:
		return fmt.Errorf("unexpected date type: %T", aux.Date)
	}
}
