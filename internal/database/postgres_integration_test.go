package database_test

import (
	"context"
	"sync"
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
			name:        "successfully_update_existing_record",
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
			name:        "successfully_update_to_zero_steps",
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
			name:        "successfully_update_to_maximum_int_value",
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
			name:         "fail_update_non_existing_record",
			setupRecord:  nil, // No initial record
			updateRecord: testutils.CreateHealthRecord("2024-06-04", 8500),
			wantError:    true,
			errorMsg:     "record not found for date",
			description:  "Should fail when trying to update non-existing record",
		},
		{
			name:        "fail_update_with_negative_step_count",
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
			name:        "successfully_update_same_value",
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

	// Create a context that gets cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	updateRecord := &models.HealthRecord{
		Date:      testutils.CreateDate("2024-08-01"),
		StepCount: 12000,
	}

	// Update should fail due to cancelled context
	err = ptc.DB.UpdateHealthRecord(ctx, updateRecord)
	assert.Error(t, err, "update should fail with cancelled context")
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
			name:        "successfully_delete_existing_record",
			setupRecord: testutils.CreateHealthRecord("2024-06-01", 8500),
			deleteDate:  testutils.CreateDate("2024-06-01"),
			wantError:   false,
			description: "Should successfully delete an existing record",
		},
		{
			name:        "successfully_delete_record_with_zero_steps",
			setupRecord: testutils.CreateHealthRecord("2024-06-02", 0),
			deleteDate:  testutils.CreateDate("2024-06-02"),
			wantError:   false,
			description: "Should successfully delete record with zero step count",
		},
		{
			name:        "successfully_delete_record_with_maximum_steps",
			setupRecord: testutils.CreateHealthRecord("2024-06-03", 2147483647),
			deleteDate:  testutils.CreateDate("2024-06-03"),
			wantError:   false,
			description: "Should successfully delete record with maximum step count",
		},
		{
			name:        "fail_delete_non_existing_record",
			setupRecord: nil, // No initial record
			deleteDate:  testutils.CreateDate("2024-06-04"),
			wantError:   true,
			errorMsg:    "record not found for date",
			description: "Should fail when trying to delete non-existing record",
		},
		{
			name:        "fail_delete_record_after_already_deleted",
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
			if tt.name == "fail_delete_record_after_already_deleted" {
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
