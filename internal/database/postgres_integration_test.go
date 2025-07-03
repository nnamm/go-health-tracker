package database_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/database"
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
			expectedCount:   9,
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

func TestReadHealthRecorsByYearMonth(t *testing.T) {
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
		month           int
		expectFound     bool
		expectedRecords []models.HealthRecord
		expectedCount   int
	}{
		{
			name:            "existing records found - 202401",
			year:            2024,
			month:           1,
			expectFound:     true,
			expectedRecords: testutils.FindHealthRecordByYearMonth(testRecords, 2024, 1),
			expectedCount:   5,
		},
		{
			name:            "existing records found - 2025",
			year:            2024,
			month:           3,
			expectFound:     true,
			expectedRecords: testutils.FindHealthRecordByYearMonth(testRecords, 2024, 3),
			expectedCount:   1,
		},
		{
			name:            "record not found",
			year:            2026,
			month:           12,
			expectFound:     false,
			expectedRecords: nil,
			expectedCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ptc.DB.ReadHealthRecordsByYearMonth(ctx, tt.year, tt.month)
			require.NoError(t, err, "ReadHealthRecordsByRange should not return error for any valid year")

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

func TestUpdateHealthRecord(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Share a single container for all test cases
	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	tests := []struct {
		name         string
		setupRecord  *models.HealthRecord // Initial record to create (nil if no setup needed)
		updateRecord *models.HealthRecord // Record data to update with
		wantError    bool
		errorMsg     string // Expected error substring
		validate     bool
		description  string // Additional context for the test case
	}{
		{
			name:        "successfully update existing record",
			setupRecord: testutils.CreateHealthRecord("2024-06-01", 8500),
			updateRecord: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-06-01"),
				StepCount: 12000,
			},
			wantError:   false,
			validate:    true,
			description: "Should successfully update step count of existing record",
		},
		{
			name:        "successfully update to zero steps",
			setupRecord: testutils.CreateHealthRecord("2024-06-02", 8500),
			updateRecord: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-06-02"),
				StepCount: 0,
			},
			wantError:   false,
			validate:    true,
			description: "Should allow updating to zero step count",
		},
		{
			name:        "successfully update to maximum int value",
			setupRecord: testutils.CreateHealthRecord("2024-06-03", 8500),
			updateRecord: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-06-03"),
				StepCount: 2147483647, // Maximum int32 value
			},
			wantError:   false,
			validate:    true,
			description: "Should handle maximum integer step count",
		},
		{
			name:         "fail update non existing record",
			setupRecord:  nil, // No initial record
			updateRecord: testutils.CreateHealthRecord("2024-06-04", 8500),
			wantError:    true,
			errorMsg:     "record not found for date",
			description:  "Should fail when trying to update non-existing record",
		},
		{
			name:        "fail update with negative step count",
			setupRecord: testutils.CreateHealthRecord("2024-06-05", 8500),
			updateRecord: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-06-05"),
				StepCount: -1,
			},
			wantError:   true,
			errorMsg:    "failed to update health record",
			description: "Should fail with negative step count due to CHECK constraint",
		},
		{
			name:        "successfully update same value",
			setupRecord: testutils.CreateHealthRecord("2024-06-06", 8500),
			updateRecord: &models.HealthRecord{
				Date:      testutils.CreateDate("2024-06-06"),
				StepCount: 8500, // Same value as initial
			},
			wantError:   false,
			validate:    true,
			description: "Should successfully update even with same value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up data before each test case
			ptc.CleanupTestData(ctx, t)

			// Setup initinal record if needed
			if tt.setupRecord != nil {
				_, err := ptc.DB.CreateHealthRecord(ctx, tt.setupRecord)
				require.NoError(t, err, "failed to setup initial record for test: %s", tt.description)
			}

			// Record the time before update for timestamp validation
			beforeUpdate := time.Now()

			// Perform the update opration
			err := ptc.DB.UpdateHealthRecord(ctx, tt.updateRecord)

			if tt.wantError {
				require.Error(t, err, "expected error for test case: %s", tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "error message should contain expected substring for test: %s", tt.description)
				}
			} else {
				require.NoError(t, err, "unexpected error for test case: %s", tt.description)

				if tt.validate {
					// Verify the record was updated correctly
					updatedRecord, err := ptc.DB.ReadHealthRecord(ctx, tt.updateRecord.Date)
					require.NoError(t, err, "failed to read updated record")
					require.NotNil(t, updatedRecord, "updated record should not be nil")

					// Validate core fields
					assert.Equal(t,
						tt.updateRecord.StepCount,
						updatedRecord.StepCount,
						"Step count should match expected value")
					assert.Equal(t,
						tt.updateRecord.Date.Format("2006-01-02"),
						updatedRecord.Date.Format("2006-01-02"),
						"Date should remain unchanged")

					// Validate timestamp fields
					assert.NotZero(t, updatedRecord.ID, "ID should be set")
					assert.NotZero(t, updatedRecord.CreatedAt, "CreatedAt should be set")
					assert.NotZero(t, updatedRecord.UpdatedAt, "UpdatedAt should be set")

					// Verify UpdatedAt was modified
					assert.True(t,
						updatedRecord.UpdatedAt.After(beforeUpdate) ||
							updatedRecord.UpdatedAt.Equal(beforeUpdate),
						"UpdatedAto should be recent")

					// For updates, UpdatedAt should be >= CreatedAt
					assert.True(t,
						updatedRecord.UpdatedAt.After(updatedRecord.CreatedAt) ||
							updatedRecord.UpdatedAt.Equal(updatedRecord.CreatedAt),
						"UpdatedAt should be >= CreatedAt for update records")
				}
			}
		})
	}
}

func TestUpdateHealthRecord_ConcurrentUpdates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	// Setup initial record for concurrent testing
	initialRecord := testutils.CreateHealthRecord("2024-07-01", 8500)
	_, err := ptc.DB.CreateHealthRecord(ctx, initialRecord)
	require.NoError(t, err, "failed to setup initial record for concurrent test")

	// Create multiple update records with different step counts
	updates := []*models.HealthRecord{
		{
			Date:      testutils.CreateDate("2024-07-01"),
			StepCount: 10000,
		},
		{
			Date:      testutils.CreateDate("2024-07-01"),
			StepCount: 12000,
		},
		{
			Date:      testutils.CreateDate("2024-07-01"),
			StepCount: 15000,
		},
	}
	// Execute concurrent updates
	var wg sync.WaitGroup
	errors := make([]error, len(updates))

	for i, update := range updates {
		wg.Add(1)
		go func(index int, updateRecord *models.HealthRecord) {
			defer wg.Done()
			errors[index] = ptc.DB.UpdateHealthRecord(ctx, updateRecord)
		}(i, update)
	}

	// Wait for all updates to complete
	wg.Wait()

	// All updates should succeed due to proper transaction handling
	for i, err := range errors {
		assert.NoError(t, err, "Concurrent update %d should succeed", i+1)
	}

	// Verify final state - one of the update values should be the final value
	finalRecord, err := ptc.DB.ReadHealthRecord(ctx, testutils.CreateDate("2024-07-01"))
	require.NoError(t, err, "failed to read final record state")
	require.NotNil(t, finalRecord, "final record should exist")

	// The final step count should be one of the updated values (last write wins)
	possibleValues := []int{10000, 12000, 15000}
	assert.Contains(t, possibleValues, finalRecord.StepCount,
		"final step count should be one of the concurrently updated values, got: %d",
		finalRecord.StepCount)

	// Verify UpdatedAt timestamp was modified
	assert.True(t, finalRecord.UpdatedAt.After(finalRecord.CreatedAt),
		"UpdatedAt should be more recent than CreatedAt after concurrent updates")
}

func TestUpdateHealthRecord_ContextCancellation(t *testing.T) {
	ptc := testutils.SetupPostgresContainer(context.Background(), t)
	defer ptc.Cleanup(context.Background(), t)

	// Setup initial record
	initialRecord := testutils.CreateHealthRecord("2024-08-01", 8500)
	_, err := ptc.DB.CreateHealthRecord(context.Background(), initialRecord)
	require.NoError(t, err, "failed to setup initial record")

	// Create a context that gets canceled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	updateRecord := &models.HealthRecord{
		Date:      testutils.CreateDate("2024-08-01"),
		StepCount: 12000,
	}

	// Update should fail due to canceled context
	err = ptc.DB.UpdateHealthRecord(ctx, updateRecord)
	assert.Error(t, err, "update should fail with canceled context")
	assert.Contains(t, err.Error(), "context canceled",
		"error should indicate context cancellation")
}

func TestDeleteHealthRecord(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Share a single container for all test cases
	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	tests := []struct {
		name        string
		setupRecord *models.HealthRecord
		deleteDate  time.Time
		wantError   bool
		errorMsg    string
		description string
	}{
		{
			name:        "successfully delete existing record",
			setupRecord: testutils.CreateHealthRecord("2024-06-01", 8500),
			deleteDate:  testutils.CreateDate("2024-06-01"),
			wantError:   false,
			description: "Should successfully delete an existing record",
		},
		{
			name:        "successfully delete record with zero steps",
			setupRecord: testutils.CreateHealthRecord("2024-06-02", 0),
			deleteDate:  testutils.CreateDate("2024-06-02"),
			wantError:   false,
			description: "Should successfully delete record with zero step count",
		},
		{
			name:        "successfully delete record with maximum steps",
			setupRecord: testutils.CreateHealthRecord("2024-06-03", 2147483647),
			deleteDate:  testutils.CreateDate("2024-06-03"),
			wantError:   false,
			description: "Should successfully delete record with maximum step count",
		},
		{
			name:        "fail delete non existing record",
			setupRecord: nil, // No initial record
			deleteDate:  testutils.CreateDate("2024-06-04"),
			wantError:   true,
			errorMsg:    "record not found for date",
			description: "Should fail when trying to delete non-existing record",
		},
		{
			name:        "fail delete record after already deleted",
			setupRecord: testutils.CreateHealthRecord("2024-06-05", 8500),
			deleteDate:  testutils.CreateDate("2024-06-05"),
			wantError:   true,
			errorMsg:    "record not found for date",
			description: "Should fail when trying to delete already deleted record (second attempt)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up data before each test case
			ptc.CleanupTestData(ctx, t)

			// Setup initial record if needed
			if tt.setupRecord != nil {
				_, err := ptc.DB.CreateHealthRecord(ctx, tt.setupRecord)
				// _, err := ptc.DB.CreateHealthRecord(opCtx, tt.setupRecord)
				require.NoError(t, err, "failed to setup initial record for test: %s", tt.description)

				// Verify the record exists before description
				existingRecord, err := ptc.DB.ReadHealthRecord(ctx, tt.setupRecord.Date)
				require.NoError(t, err, "failed to verify setup record exists")
				require.NotNil(t, existingRecord, "setup record should exist before deletion")
			}

			// For the "already deleted" test case, perform the first deletion
			if tt.name == "fail delete record after already deleted" {
				err := ptc.DB.DeleteHealthRecord(ctx, tt.deleteDate)
				require.NoError(t, err, "first deletion should succeed")

				// Verify record was deleted
				deletedRecord, err := ptc.DB.ReadHealthRecord(ctx, tt.deleteDate)
				require.NoError(t, err, "should be able to query for deleted record")
				assert.Nil(t, deletedRecord, "record should not exist after first deletion")
			}

			// Perform the delete operation
			err := ptc.DB.DeleteHealthRecord(ctx, tt.deleteDate)

			if tt.wantError {
				require.Error(t, err, "expected error for test case: %s", tt.description)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"error message should contain expected substring for test: %s", tt.description)
				}

				// Verify record strill doesn't exist (for non-existing record case)
				if tt.setupRecord != nil {
					nonExistentRecord, err := ptc.DB.ReadHealthRecord(ctx, tt.deleteDate)
					require.NoError(t, err, "should be able to query for non-existent record")
					assert.Nil(t, nonExistentRecord, "record should not exist")
				}
			} else {
				require.NoError(t, err, "unexpected error for test case: %s", tt.description)

				// Verify the record was actually deleted
				deletedRecord, err := ptc.DB.ReadHealthRecord(ctx, tt.deleteDate)
				require.NoError(t, err, "failed to verify record deletion")
				assert.Nil(t, deletedRecord,
					"record should not exist after successful deletion for test: %s", tt.description)

				// Verify other records are not affected (if any exist)
				allRecords, err := ptc.DB.ReadHealthRecordsByYear(ctx, tt.deleteDate.Year())
				require.NoError(t, err, "failed to read remaining records")

				// Ensure the deleted record is not in the results
				for _, record := range allRecords {
					assert.NotEqual(t,
						tt.deleteDate.Format("2006-01-02"),
						record.Date.Format("2006-01-2"),
						"deleted record should not appear in year results")
				}
			}
		})
	}
}

func TestDeleteHealthRecord_ConcurrentDeletes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	// Setup initial records for concurrent testing
	testDates := []string{"2024-07-01", "2024-07-02", "2024-07-03"}
	for _, dateStr := range testDates {
		record := testutils.CreateHealthRecord(dateStr, 8500)
		_, err := ptc.DB.CreateHealthRecord(ctx, record)
		require.NoError(t, err, "failed to setup initial record for date: %s", dateStr)
	}

	// Attempt to delete the same record concurrently
	var wg sync.WaitGroup
	errors := make([]error, 3)
	deleteDate := testutils.CreateDate("2024-07-01")

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors[index] = ptc.DB.DeleteHealthRecord(ctx, deleteDate)
		}(i)
	}

	// Wait for all deletions to complete
	wg.Wait()

	// Only one deletion should succeed, others should fail
	successCount := 0
	errorCount := 0
	for i, err := range errors {
		if err == nil {
			successCount++
		} else {
			errorCount++
			assert.Contains(t, err.Error(), "record not found for date",
				"concurrent deletion %d should fail with appropriate error", i+1)
		}
	}

	assert.Equal(t, 1, successCount, "exactly one concurrent deletion should succeed")
	assert.Equal(t, 2, errorCount, "two concurrent deletions should fail")

	// Verify the record was actually deleted
	deletedRecord, err := ptc.DB.ReadHealthRecord(ctx, deleteDate)
	require.NoError(t, err, "should be able to query for deleted record")
	assert.Nil(t, deletedRecord, "record should not exist after concurrent deletions")

	// Verify other records are unaffected
	remainingRecords, err := ptc.DB.ReadHealthRecordsByYear(ctx, 2024)
	require.NoError(t, err, "should be able to read remaining records")
	assert.Len(t, remainingRecords, 2, "should have 2 remaining records")
}

func TestDeleteHealthRecord_ContextCancellation(t *testing.T) {
	ptc := testutils.SetupPostgresContainer(context.Background(), t)
	defer ptc.Cleanup(context.Background(), t)

	// Setup initial record
	initialRecord := testutils.CreateHealthRecord("2024-08-01", 8500)
	_, err := ptc.DB.CreateHealthRecord(context.Background(), initialRecord)
	require.NoError(t, err, "failed to setup initial record")

	// Create a context that gets canceled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Delete should fail due to canceled context
	err = ptc.DB.DeleteHealthRecord(ctx, initialRecord.Date)
	assert.Error(t, err, "delete should fail with canceled context")
	assert.Contains(t, err.Error(), "context canceled",
		"error should indicate context cancelation")

	// Verify record still exists after failed deletion
	existingRecord, err := ptc.DB.ReadHealthRecord(context.Background(), initialRecord.Date)
	require.NoError(t, err, "should be able to read record after failed deletion")
	require.NotNil(t, existingRecord, "record should still exist after canceled deletion")
	testutils.AssertHealthRecord(t, existingRecord, initialRecord)
}

func TestDeleteHealthRecord_MultipulRecords(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	testRecords := []*models.HealthRecord{
		testutils.CreateHealthRecord("2024-09-01", 8500),
		testutils.CreateHealthRecord("2024-09-02", 9000),
		testutils.CreateHealthRecord("2024-09-03", 7500),
		testutils.CreateHealthRecord("2024-09-04", 10000),
		testutils.CreateHealthRecord("2024-09-05", 6500),
	}

	for _, record := range testRecords {
		_, err := ptc.DB.CreateHealthRecord(ctx, record)
		require.NoError(t, err, "failed to setup record for date: %s",
			record.Date.Format("2006-01-02"))
	}

	// Verify all records exist
	allRecords, err := ptc.DB.ReadHealthRecordsByYearMonth(ctx, 2024, 9)
	require.NoError(t, err, "failed to read initial records")
	assert.Len(t, allRecords, 5, "should have 5 initial records")

	// Delete records on by one
	for i, record := range testRecords {
		err := ptc.DB.DeleteHealthRecord(ctx, record.Date)
		require.NoError(t, err, "failed to delete record %d", i+1)

		// Verify this specific recorde was deleted
		deletedRecord, err := ptc.DB.ReadHealthRecord(ctx, record.Date)
		require.NoError(t, err, "should be able to query for deleted record")
		assert.Nil(t, deletedRecord, "record %d should be deleted", i+1)

		// Verify remaining count
		remainingRecords, err := ptc.DB.ReadHealthRecordsByYearMonth(ctx, 2024, 9)
		require.NoError(t, err, "failed to read remaining records")
		expectedCount := 5 - (i + 1)
		assert.Len(t, remainingRecords, expectedCount,
			"should have %d remaining records after deleting record %d", expectedCount, i+1)
	}

	// Verify no records remain
	finalRecords, err := ptc.DB.ReadHealthRecordsByYearMonth(ctx, 2024, 9)
	require.NoError(t, err, "failed to read final records")
	assert.Empty(t, finalRecords, "should have no remaining records")
}

func TestPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	canceledCtx, cancelCtx := context.WithCancel(context.Background())
	cancelCtx()

	tests := []struct {
		name      string
		ctx       context.Context
		wantError bool
		errorMsg  string
	}{
		{
			name:      "successful ping with active connection",
			ctx:       ctx,
			wantError: false,
		},
		{
			name:      "ping fails with canceled context",
			ctx:       canceledCtx,
			wantError: true,
			errorMsg:  "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ptc.DB.Ping(tt.ctx)

			if tt.wantError {
				require.Error(t, err, "expected an error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg,
						"error message for '%s' should contain '%s'", tt.name, tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "did not expect an error for test case: %s", tt.name)
			}
		})
	}
}

func TestHealthCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	canceledCtx, cancelCtx := context.WithCancel(context.Background())
	cancelCtx()

	tests := []struct {
		name      string
		db        *database.PostgresDB
		ctx       context.Context
		wantError bool
		errorMsg  string
	}{
		{
			name:      "successful health check with active connection",
			db:        ptc.DB,
			ctx:       ctx,
			wantError: false,
		},
		{
			name:      "health check fails with canceled context",
			db:        ptc.DB,
			ctx:       canceledCtx,
			wantError: true,
			errorMsg:  "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.db.HealthCheck(tt.ctx)
			if tt.wantError {
				require.Error(t, err, "Expected an error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message for '%s' should contain '%s'", tt.name, tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Did not expect an error for test case: %s", tt.name)
			}
		})
	}
}

func TestExec(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ptc := testutils.SetupPostgresContainer(ctx, t)
	defer ptc.Cleanup(ctx, t)

	// Setup initial record for update/delete tests
	initialRecord := testutils.CreateHealthRecord("2024-10-01", 1000)
	_, err := ptc.DB.CreateHealthRecord(ctx, initialRecord)
	require.NoError(t, err, "failed to setup initial record for exec test")

	canceledCtx, cancelCtx := context.WithCancel(context.Background())
	cancelCtx()

	tests := []struct {
		name      string
		sql       string
		args      []any
		ctx       context.Context
		wantError bool
		errorMsg  string
	}{
		{
			name:      "successful DDL (CREATE TABLE)",
			sql:       "CREATE TABLE IF NOT EXISTS test_exec_table (id INT);",
			args:      nil,
			ctx:       ctx,
			wantError: false,
		},
		{
			name:      "successful DML (UPDATE)",
			sql:       "UPDATE health_records SET step_count = $1 WHERE date = $2",
			args:      []any{2000, initialRecord.Date},
			ctx:       ctx,
			wantError: false,
		},
		{
			name:      "failed DML (invalid syntax)",
			sql:       "UPDATE health_records SET step_count = 2000 WHERE date = ",
			args:      nil,
			ctx:       ctx,
			wantError: true,
			errorMsg:  "syntax error",
		},
		{
			name:      "failed with canceled context",
			sql:       "DELETE FROM health_records WHERE date = $1",
			args:      []any{initialRecord.Date},
			ctx:       canceledCtx,
			wantError: true,
			errorMsg:  "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ptc.DB.Exec(tt.ctx, tt.sql, tt.args...)

			if tt.wantError {
				require.Error(t, err, "Expected an error for test case: %s", tt.name)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message for '%s' should contain '%s'", tt.name, tt.errorMsg)
				}
			} else {
				require.NoError(t, err, "Did not expect an error for test case: %s", tt.name)

				// Additional verification for successful DML
				if tt.name == "successful DML (UPDATE)" {
					updatedRecord, readErr := ptc.DB.ReadHealthRecord(ctx, initialRecord.Date)
					require.NoError(t, readErr, "Failed to read back record after update for verification")
					assert.Equal(t, 2000, updatedRecord.StepCount, "Step count should be updated to 2000")
				}
			}
		})
	}
}
