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
		Date string `json:"date"`
		*Alias
	}{
		Alias: (*Alias)(hr),
	}
	if err := json.Unmarshal(date, &aux); err != nil {
		return err
	}
	var err error
	hr.Date, err = time.Parse("2006-01-02", aux.Date)
	return err
}
