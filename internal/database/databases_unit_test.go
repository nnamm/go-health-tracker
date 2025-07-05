package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/nnamm/go-health-tracker/testutils"
)

type dbTestRuner struct {
	dbType      string
	setupMockDB func(t *testing.T) (database.DBInterface, sqlmock.Sqlmock)
}

func TestDatabaseRollbackScenarios(t *testing.T) {
	// Define test runners for each database
	runners := []dbTestRuner{
		//{
		//	dbType: "postgres",
		//	setupMockDB: func(t *testing.T) (database.DBInterface, sqlmock.Sqlmock) {
		//		return NewPostgresDBWithMock(t)
		//	},
		//},
		{
			dbType: "sqlite",
			setupMockDB: func(t *testing.T) (database.DBInterface, sqlmock.Sqlmock) {
				return NewSQLiteDBWithMock(t)
			},
		},
	}

	for _, r := range runners {
		t.Run(r.dbType, func(t *testing.T) {
			runCreateHealthRecordRollbackTests(t, r.setupMockDB)
		})
	}
}

func runCreateHealthRecordRollbackTests(t *testing.T, setupMockDB func(t *testing.T) (database.DBInterface, sqlmock.Sqlmock)) {
	t.Helper()

	tests := []struct {
		name        string
		record      *models.HealthRecord
		buildStabs  func(mock sqlmock.Sqlmock)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "rollback on context cancellation",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-01"),
				StepCount: 12000,
			},
			buildStabs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO health_records").
					WillReturnError(context.Canceled)
				mock.ExpectRollback()
			},
			checkResult: func(t *testing.T, err error) {
				if !errors.Is(err, context.Canceled) {
					t.Errorf("expected context.Canceled error, but got %v", err)
				}
			},
		},
		{
			name: "rollback on other database error during exec",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-02"),
				StepCount: 8500,
			},
			buildStabs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO health_records").
					WillReturnError(errors.New("some database error"))
				mock.ExpectRollback()
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error, but got nil")
				}
			},
		},
		{
			name: "rollback on commit failure",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-03"),
				StepCount: 10000,
			},
			buildStabs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO health_records").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected an error on commit failure, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)

			tt.buildStabs(mock)

			_, err := db.CreateHealthRecord(context.Background(), tt.record)

			tt.checkResult(t, err)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %v", err)
			}
		})
	}
}
