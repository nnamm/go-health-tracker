package testutils

import (
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/stretchr/testify/assert"
)

func CreateDate(dateStr string) time.Time {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}
	return t
}

func FindHealthRecordByDate(records []*models.HealthRecord, dateStr string) *models.HealthRecord {
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}
	for _, record := range records {
		if record.Date.Equal(date) {
			return record
		}
	}
	return nil
}

func AssertHealthRecord(t *testing.T, got, want *models.HealthRecord) {
	t.Helper()

	assert.Equal(t, want.Date.Format("2006-01-02"), got.Date.Format("2006-01-02"))
	assert.Equal(t, want.StepCount, got.StepCount)
	assert.NotZero(t, got.ID)
	assert.NotZero(t, got.CreatedAt)
	assert.NotZero(t, got.UpdatedAt)
}
