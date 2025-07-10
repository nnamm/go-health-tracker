// Package database provides database abstraction interfaces and implementations
// for the health tracker application.
package database

import (
	"context"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
)

type DBInterface interface {
	CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error)
	ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error)
	ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error)
	ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error)
	UpdateHealthRecord(ctx context.Context, hr *models.HealthRecord) error
	DeleteHealthRecord(ctx context.Context, date time.Time) error
	Close() error
}
