package models

import (
	"encoding/json"
	"time"
)

type HealthRecord struct {
	ID        int64     `json:"id"`
	Date      time.Time `json:"date"`
	StepCount int       `json:"step_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

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

func (hr *HealthRecord) UnmarshalJSON(date []byte) error {
	type Alias HealthRecord
	aux := &struct {
		Date interface{} `json:"date"`
		*Alias
	}{
		Alias: (*Alias)(hr),
	}
	if err := json.Unmarshal(date, &aux); err != nil {
		return err
	}

	switch v := aux.Date.(type) {
	case string:
		// Try parsing as RFC3339 first (includes time and timezone)
		t, err := time.Parse(time.RFC3339, v)
		if err == nil {
			hr.Date = t
			return nil
		}

		// If that fails, try parsing as "2006-01-02"
		t, err = time.Parse("2006-01-02", v)
		if err == nil {
			hr.Date = t
			return nil
		}

		return err
	case float64:
		hr.Date = time.Unix(int64(v), 0)
	default:
		return json.Unmarshal(date, &hr.Date)
	}

	return nil
}
