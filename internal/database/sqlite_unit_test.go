package database_test

import (
	"database/sql"
	"regexp"
	"sort"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nnamm/go-health-tracker/internal/database"
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
