package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/nnamm/go-health-tracker/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateHealthRecord(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	tests := []struct {
		name      string
		input     *models.HealthRecord
		wantError bool
		errorMsg  string
		validate  bool
	}{
		{
			name:      "valid health record",
			input:     testutils.CreateHealthRecord("2024-06-01", 8500),
			wantError: false,
			validate:  true,
		},
		{
			name:      "valid health record (maximum step count)",
			input:     testutils.CreateHealthRecord("2024-06-02", 100000),
			wantError: false,
			validate:  true,
		},
		{
			name:      "valid health record with maximum integer value",
			input:     testutils.CreateHealthRecord("2024-06-05", 2147483647),
			wantError: false,
			validate:  true,
		},
		{
			name:      "valid health record (zero step count)",
			input:     testutils.CreateHealthRecord("2024-06-03", 0),
			wantError: false,
			validate:  true,
		},
		{
			name:      "invalid step count (minus value)",
			input:     testutils.CreateHealthRecord("2024-06-04", -1),
			wantError: true,
			errorMsg:  "failed to create health record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ptc.CleanupTestData(ctx, t)

			result, err := ptc.DB.CreateHealthRecord(ctx, tt.input)

			if tt.wantError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)

				if tt.validate {
					testutils.AssertHealthRecord(t, result, tt.input)
				}
			}
		})
	}
}

// TestCreateHealthRecord_DuplicateConstraint tests UNIQUE constraint violation
func TestCreateHealthRecord_DuplicateConstraint(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	// First insertion should succeed
	firstRecord := testutils.CreateHealthRecord("2024-07-01", 8500)
	result1, err := ptc.DB.CreateHealthRecord(ctx, firstRecord)
	require.NoError(t, err)
	require.NotNil(t, result1)

	// Second insertion with same date should fail
	duplicateRecord := testutils.CreateHealthRecord("2024-07-01", 9000)
	result2, err := ptc.DB.CreateHealthRecord(ctx, duplicateRecord)

	require.Error(t, err)
	assert.Nil(t, result2)
	assert.Contains(t, err.Error(), "duplicate key value violates unique constraint")
}

func TestReadHealthRecord(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	testRecords := testutils.CreateHealthRecords()
	cleanup := testutils.SetupTestData(ctx, t, ptc, testRecords)
	defer cleanup()

	tests := []struct {
		name           string
		date           time.Time
		expectFound    bool
		expectedRecord *models.HealthRecord
	}{
		{
			name:           "existing record found - 2024-01-01",
			date:           testutils.CreateDate("2024-01-01"),
			expectFound:    true,
			expectedRecord: testutils.FindHealthRecordByDate(testRecords, "2024-01-01"),
		},
		{
			name:           "existing record found - 2024-02-14",
			date:           testutils.CreateDate("2024-02-14"),
			expectFound:    true,
			expectedRecord: testutils.FindHealthRecordByDate(testRecords, "2024-02-14"),
		},
		{
			name:           "record not found",
			date:           testutils.CreateDate("2024-06-01"),
			expectFound:    false,
			expectedRecord: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ptc.DB.ReadHealthRecord(ctx, tt.date)
			require.NoError(t, err, "ReadHealthRecord should not return error for any valid date")

			if tt.expectFound {
				require.NotNil(t, got)
				testutils.AssertHealthRecord(t, got, tt.expectedRecord)
			} else {
				assert.Nil(t, got)
			}
		})
	}
}

func TestReadHealthRecorsByYear(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	testRecords := testutils.CreateHealthRecords()
	cleanup := testutils.SetupTestData(ctx, t, ptc, testRecords)
	defer cleanup()

	tests := []struct {
		name            string
		year            int
		expectFound     bool
		expectedRecords []models.HealthRecord
		expectedCount   int
	}{
		{
			name:            "existing records found - 2024",
			year:            2024,
			expectFound:     true,
			expectedRecords: testutils.FindHealthRecordByYear(testRecords, 2024),
			expectedCount:   8,
		},
		{
			name:            "existing records found - 2025",
			year:            2025,
			expectFound:     true,
			expectedRecords: testutils.FindHealthRecordByYear(testRecords, 2025),
			expectedCount:   1,
		},
		{
			name:            "record not found",
			year:            2026,
			expectFound:     false,
			expectedRecords: nil,
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ptc.DB.ReadHealthRecordsByYear(ctx, tt.year)
			require.NoError(t, err, "ReadHealthRecordsByYear should not return error for any valid year")

			if tt.expectFound {
				require.NotNil(t, got)
				assert.Len(t, got, tt.expectedCount)
				testutils.AssertHealthRecords(t, got, tt.expectedRecords)
			} else {
				assert.Empty(t, got)
			}
		})
	}
}
