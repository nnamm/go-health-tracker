package models

import "time"

type HealthRecord struct {
	ID int64 `json:"id"`
	//Date      string    `json:"date"`
	Date      time.Time `json:"date"`
	StepCount int       `json:"step_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
