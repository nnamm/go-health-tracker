package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/nnamm/go-health-tracker/testutils"
	"github.com/pashagolub/pgxmock/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testPostgres *database.PostgresDB

func NewPostgresDBWithMock(t *testing.T) (*database.PostgresDB, pgxmock.PgxPoolIface) {
	t.Helper()

	mockPool, err := pgxmock.NewPool(pgxmock.QueryMatcherOption(pgxmock.QueryMatcherRegexp))
	require.NoError(t, err)

	db := database.NewPostgresDBWithPool(mockPool)
	return db, mockPool
}

func TestPosgres_GetPoolInfo(t *testing.T) {
	tests := []struct {
		name     string
		setupDB  func() *database.PostgresDB
		expected map[string]any
	}{
		{
			name: "pool not initialized returns not_initialized status",
			setupDB: func() *database.PostgresDB {
				return testutils.NewPostgresDBForTest()
			},
			expected: map[string]any{
				"status": "not_initialized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.setupDB()
			result := db.GetPoolInfo()

			assert.Equal(t, tt.expected, result, "GetPoolInfo() should return expected status")
		})
	}
}

func TestPosgres_Close(t *testing.T) {
	tests := []struct {
		name    string
		setupDB func() *database.PostgresDB
		wantErr bool
	}{
		{
			name: "nil pool closes without error",
			setupDB: func() *database.PostgresDB {
				return testutils.NewPostgresDBForTest()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.setupDB()
			err := db.Close()

			if tt.wantErr {
				assert.Error(t, err, "Close() should return an error")
			} else {
				assert.NoError(t, err, "Close() should not return an error")
			}
		})
	}
}

func TestPostgres_RollbackScenarios(t *testing.T) {
	db, mock := NewPostgresDBWithMock(t)
	runCreateHealthRecordPostgresRollbackTests(t, db, mock)
	runUpdateHealthRecordPostgresRollbackTests(t, db, mock)
	runDeleteHealthRecordPostgresRollbackTests(t, db, mock)
}

func runCreateHealthRecordPostgresRollbackTests(t *testing.T, db database.DBInterface, mock pgxmock.PgxPoolIface) {
	t.Helper()

	tests := []struct {
		name        string
		record      *models.HealthRecord
		buildStubs  func(mock pgxmock.PgxPoolIface)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "create rollback on context cancellation",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-01"),
				StepCount: 12000,
			},
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO health_records").
					WithArgs(
						pgxmock.AnyArg(), // date
						pgxmock.AnyArg(), // step_count
						pgxmock.AnyArg(), // created_at
						pgxmock.AnyArg(), // updated_at
					).
					WillReturnError(context.Canceled)
			},
			checkResult: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, but got %v", err)
				}
			},
		},
		{
			name: "create rollback on other database error during exec",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-02"),
				StepCount: 8500,
			},
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO health_records").
					WithArgs(
						pgxmock.AnyArg(), // date
						pgxmock.AnyArg(), // step_count
						pgxmock.AnyArg(), // created_at
						pgxmock.AnyArg(), // updated_at
					).
					WillReturnError(errors.New("some database error"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
		// Only SQLite3 test (update and delete as well)
		// {
		// 	name: "create rollback on commit failure",
		// },
		{
			name: "create rollback on unique constraint violation",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-04"),
				StepCount: 9000,
			},
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery("INSERT INTO health_records").
					WithArgs(
						pgxmock.AnyArg(), // date
						pgxmock.AnyArg(), // step_count
						pgxmock.AnyArg(), // created_at
						pgxmock.AnyArg(), // updated_at
					).
					WillReturnError(errors.New("duplicate key value violation unique constraint"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected unique-constraint error, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.buildStubs(mock)
			_, err := db.CreateHealthRecord(context.Background(), tt.record)
			tt.checkResult(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}

func runUpdateHealthRecordPostgresRollbackTests(t *testing.T, db database.DBInterface, mock pgxmock.PgxPoolIface) {
	t.Helper()

	record := &models.HealthRecord{
		Date:      testutils.CreateDate("2025-02-01"),
		StepCount: 15000,
	}

	tests := []struct {
		name        string
		buildStubs  func(mock pgxmock.PgxPoolIface)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "update rollback on context cancellation",
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE health_records").
					WithArgs(
						record.StepCount,
						pgxmock.AnyArg(), // updated_at
						record.Date,
					).
					WillReturnError(context.Canceled)
			},
			checkResult: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, but got %v", err)
				}
			},
		},
		{
			name: "update rollback on other database error during exec",
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("UPDATE health_records").
					WithArgs(
						record.StepCount,
						pgxmock.AnyArg(), // updated_at
						record.Date,
					).
					WillReturnError(errors.New("some database error"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.buildStubs(mock)
			err := db.UpdateHealthRecord(context.Background(), record)
			tt.checkResult(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}

func runDeleteHealthRecordPostgresRollbackTests(t *testing.T, db database.DBInterface, mock pgxmock.PgxPoolIface) {
	t.Helper()

	date := testutils.CreateDate("2025-03-01")

	tests := []struct {
		name        string
		buildStubs  func(mock pgxmock.PgxPoolIface)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "delete rollback on context cancellation",
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM health_records").
					WithArgs(date).
					WillReturnError(context.Canceled)
			},
			checkResult: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, but got %v", err)
				}
			},
		},
		{
			name: "delete rollback on other database error during exec",
			buildStubs: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec("DELETE FROM health_records").
					WithArgs(date).
					WillReturnError(errors.New("some database error"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.buildStubs(mock)
			err := db.DeleteHealthRecord(context.Background(), date)
			tt.checkResult(t, err)
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}
