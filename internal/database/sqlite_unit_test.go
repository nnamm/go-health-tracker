package database_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/internal/models"
	"github.com/nnamm/go-health-tracker/testutils"
)

func NewSQLiteDBWithMock(t *testing.T) (*database.SQLiteDB, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	queries := map[string]string{
		"insert_health_record":       `INSERT INTO health_records (date, step_count, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"select_health_record":       `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date = ?`,
		"select_range_health_record": `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date >= ? AND date < ? ORDER BY date`,
		"update_health_record":       `UPDATE health_records SET step_count = ?, updated_at = ? WHERE date = ?`,
		"delete_health_record":       `DELETE FROM health_records WHERE date = ?`,
	}

	var sortedKeys []string
	for k := range queries {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	for _, name := range sortedKeys {
		query := queries[name]
		mock.ExpectPrepare(regexp.QuoteMeta(query))
	}

	stmts := make(map[string]*sql.Stmt)
	for _, name := range sortedKeys {
		query := queries[name]
		stmt, err := mockDB.Prepare(query)
		if err != nil {
			t.Fatalf("mock db failed to prepare statement %q: %v", name, err)
		}
		stmts[name] = stmt
	}

	sqliteDB := &database.SQLiteDB{
		DB:    mockDB,
		Stmts: stmts,
		Mu:    sync.RWMutex{},
	}

	return sqliteDB, mock
}

func TestSQLite_RollbackScenarios(t *testing.T) {
	db, mock := NewSQLiteDBWithMock(t)
	runCreateHealthRecordSQLiteRollbackTests(t, db, mock)
	runUpdateHealthRecordSQLiteRollbackTests(t, db, mock)
	runDeleteHealthRecordSQLiteRollbackTests(t, db, mock)
}

func runCreateHealthRecordSQLiteRollbackTests(t *testing.T, db database.DBInterface, mock sqlmock.Sqlmock) {
	t.Helper()

	tests := []struct {
		name        string
		record      *models.HealthRecord
		buildStubs  func(mock sqlmock.Sqlmock)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "create rollback on context cancellation",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-01"),
				StepCount: 12000,
			},
			buildStubs: func(mock sqlmock.Sqlmock) {
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
			name: "create rollback on other database error during exec",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-02"),
				StepCount: 8500,
			},
			buildStubs: func(mock sqlmock.Sqlmock) {
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
			name: "create rollback on commit failure",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-03"),
				StepCount: 10000,
			},
			buildStubs: func(mock sqlmock.Sqlmock) {
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
		{
			name: "create rollback on unique constraint violation",
			record: &models.HealthRecord{
				Date:      testutils.CreateDate("2025-01-04"),
				StepCount: 9000,
			},
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO health_records").
					WillReturnError(errors.New("UNIQUE constraint failed: health_records.date"))
				mock.ExpectRollback()
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

func runUpdateHealthRecordSQLiteRollbackTests(t *testing.T, db database.DBInterface, mock sqlmock.Sqlmock) {
	t.Helper()

	record := &models.HealthRecord{
		Date:      testutils.CreateDate("2025-02-01"),
		StepCount: 15000,
	}

	tests := []struct {
		name        string
		buildStubs  func(mock sqlmock.Sqlmock)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "update rollback on context cancellation",
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1 FROM health_records WHERE date = ?").
					WithArgs(record.Date).
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
				mock.ExpectExec("UPDATE health_records").
					WithArgs(record.StepCount, sqlmock.AnyArg(), record.Date).
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
			name: "update rollback on other database error during exec",
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1 FROM health_records WHERE date = ?").
					WithArgs(record.Date).
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
				mock.ExpectExec("UPDATE health_records").
					WithArgs(record.StepCount, sqlmock.AnyArg(), record.Date).
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
			name: "update rollback on commit failure",
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1 FROM health_records WHERE date = ?").
					WithArgs(record.Date).
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
				mock.ExpectExec("UPDATE health_records").
					WithArgs(record.StepCount, sqlmock.AnyArg(), record.Date).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected commit failure error, but got nil")
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

func runDeleteHealthRecordSQLiteRollbackTests(t *testing.T, db database.DBInterface, mock sqlmock.Sqlmock) {
	t.Helper()

	date := testutils.CreateDate("2025-03-01")

	tests := []struct {
		name        string
		buildStubs  func(mock sqlmock.Sqlmock)
		checkResult func(t *testing.T, err error)
	}{
		{
			name: "delete rollback on context cancellation",
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1 FROM health_records WHERE date = ?").
					WithArgs(date).
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
				mock.ExpectExec("DELETE FROM health_records").
					WithArgs(date).
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
			name: "delete rollback on other database error during exec",
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1 FROM health_records WHERE date = ?").
					WithArgs(date).
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
				mock.ExpectExec("DELETE FROM health_records").
					WithArgs(date).
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
			name: "delete rollback on commit failure",
			buildStubs: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery("SELECT 1 FROM health_records WHERE date = ?").
					WithArgs(date).
					WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
				mock.ExpectExec("DELETE FROM health_records").
					WithArgs(date).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
			},
			checkResult: func(t *testing.T, err error) {
				if err == nil {
					t.Error("expected commit failure error, but got nil")
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
