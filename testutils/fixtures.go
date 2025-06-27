package testutils

import (
	"context"
	"testing"

	"github.com/nnamm/go-health-tracker/internal/models"
)

func CreateHealthRecord(date string, stepCount int) *models.HealthRecord {
	return &models.HealthRecord{
		Date:      CreateDate(date),
		StepCount: stepCount,
	}
}

func CreateHealthRecords() []*models.HealthRecord {
	return []*models.HealthRecord{
		CreateHealthRecord("2024-01-01", 8500),
		CreateHealthRecord("2024-01-02", 9200),
		CreateHealthRecord("2024-01-03", 7800),
		CreateHealthRecord("2024-01-15", 10500),
		CreateHealthRecord("2024-01-31", 10500),
		CreateHealthRecord("2024-02-01", 6500),
		CreateHealthRecord("2024-02-14", 11000),
		CreateHealthRecord("2024-03-01", 9800),
		CreateHealthRecord("2024-12-31", 12000),
		CreateHealthRecord("2025-01-01", 5000),
	}
}

func CreateHealthRecordsByRange(startDate, endDate string, baseStepCount int) []*models.HealthRecord {
	start := CreateDate(startDate)
	end := CreateDate(endDate)

	var records []*models.HealthRecord
	current := start
	stepVariation := 0

	for !current.After(end) {
		records = append(records, &models.HealthRecord{
			Date:      current,
			StepCount: baseStepCount + stepVariation,
		})
		current = current.AddDate(0, 0, 1)
		stepVariation = (stepVariation + 500) % 2000
	}

	return records
}

// SetupTestData sets up test data in the database and returns cleanup function
func SetupTestData(ctx context.Context, t *testing.T, ptc *PostgresTestContainer, records []*models.HealthRecord) func() {
	t.Helper()

	for _, record := range records {
		_, err := ptc.DB.CreateHealthRecord(ctx, record)
		if err != nil {
			t.Fatalf("failed to setup test data: %v", err)
		}
	}

	return func() {
		ptc.CleanupTestData(ctx, t)
	}
}
