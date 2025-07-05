package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nnamm/go-health-tracker/internal/models"
)

type SQLiteDB struct {
	*sql.DB
	Stmts map[string]*sql.Stmt
	Mu    sync.RWMutex
}

// NewSQLiteDB opens the DB
func NewSQLiteDB(dataSourceName string) (*SQLiteDB, error) {
	sqlDB, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}

	db := &SQLiteDB{
		DB:    sqlDB,
		Stmts: make(map[string]*sql.Stmt),
		Mu:    sync.RWMutex{},
	}

	if err := db.CreateTable(); err != nil {
		return nil, fmt.Errorf("creating table: %w", err)
	}

	if err := db.prepareStatements(); err != nil {
		return nil, fmt.Errorf("preparing statements: %w", err)
	}

	return db, nil
}

// CreateTable inisializes the table
func (db *SQLiteDB) CreateTable() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS health_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE NOT NULL UNIQUE,
			step_count INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
	    )`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_health_records_date
         on health_records(date)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

// PrepareStatements prepares SQL statements
func (db *SQLiteDB) prepareStatements() error {
	queries := map[string]string{
		"insert_health_record":       `INSERT INTO health_records (date, step_count, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"select_health_record":       `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date = ?`,
		"select_range_health_record": `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date >= ? AND date < ? ORDER BY date`,
		"update_health_record":       `UPDATE health_records SET step_count = ?, updated_at = ? WHERE date = ?`,
		"delete_health_record":       `DELETE FROM health_records WHERE date = ?`,
	}

	db.Mu.Lock()
	defer db.Mu.Unlock()

	for name, query := range queries {
		stmt, err := db.Prepare(query)
		if err != nil {
			return fmt.Errorf("prepare statement %s: %w", name, err)
		}
		db.Stmts[name] = stmt
	}

	return nil
}

// getStmt is helper function to get a prepared statement
func (db *SQLiteDB) getStmt(name string) (*sql.Stmt, error) {
	db.Mu.RLock()
	stmt, ok := db.Stmts[name]
	db.Mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("statement %s not found", name)
	}
	return stmt, nil
}

// Close closes the DB
func (db *SQLiteDB) Close() error {
	db.Mu.Lock()
	defer db.Mu.Unlock()

	// Close all prepared statements
	for name, stmt := range db.Stmts {
		if err := stmt.Close(); err != nil {
			return fmt.Errorf("closing statement %s: %w", name, err)
		}
	}

	// Close the original database connection
	return db.DB.Close()
}

// withTxContext executes a function with a transaction and context
func (db *SQLiteDB) withTxContext(ctx context.Context, fn func(*sql.Tx) error) error {
	// Start a transaction for the context
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transactin: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	// Rollback if the context is canceled
	select {
	case <-ctx.Done():
		tx.Rollback()
		return ctx.Err()
	default:
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}
		return nil
	}
}

// CreateHealthRecord inserts a new record
func (db *SQLiteDB) CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error) {
	insertStmt, err := db.getStmt("insert_health_record")
	if err != nil {
		return nil, fmt.Errorf("getting insert statement: %w", err)
	}

	var createdRecord *models.HealthRecord
	err = db.withTxContext(ctx, func(tx *sql.Tx) error {
		stmt := tx.StmtContext(ctx, insertStmt)

		now := time.Now()
		result, err := stmt.ExecContext(ctx, hr.Date, hr.StepCount, now, now)
		if err != nil {
			return fmt.Errorf("insert record: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get last insert id: %w", err)
		}

		createdRecord = &models.HealthRecord{
			ID:        id,
			Date:      hr.Date,
			StepCount: hr.StepCount,
			CreatedAt: now,
			UpdatedAt: now,
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return createdRecord, nil
}

// ReadHealthRecord retrieves a health record by date
func (db *SQLiteDB) ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error) {
	selectStmt, err := db.getStmt("select_health_record")
	if err != nil {
		return nil, fmt.Errorf("getting select statement: %w", err)
	}

	hr := &models.HealthRecord{}
	err = selectStmt.QueryRowContext(ctx, date).Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No error, but no record found
		}
		return nil, fmt.Errorf("scan record: %w", err)
	}

	return hr, nil
}

// ReadHealthRecordsByYear retrieves record(s) by year
func (db *SQLiteDB) ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

// ReadHealthRecordsByYearMonth retrieves record(s) by year and month
func (db *SQLiteDB) ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

// readHealthRecordsByRange retrieves records between startDate and endDate
func (db *SQLiteDB) readHealthRecordsByRange(ctx context.Context, startDate, endDate time.Time) ([]models.HealthRecord, error) {
	selectStmt, err := db.getStmt("select_range_health_record")
	if err != nil {
		return nil, fmt.Errorf("getting select_range statement: %w", err)
	}

	rows, err := selectStmt.QueryContext(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query records: %w", err)
	}
	defer rows.Close()

	var records []models.HealthRecord
	for rows.Next() {
		var hr models.HealthRecord
		if err := rows.Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, hr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating through rows: %w", err)
	}

	return records, nil
}

// UpdateHealthRecord updates an existing health record
func (db *SQLiteDB) UpdateHealthRecord(ctx context.Context, hr *models.HealthRecord) error {
	updateStmt, err := db.getStmt("update_health_record")
	if err != nil {
		return fmt.Errorf("getting update statement: %w", err)
	}

	return db.withTxContext(ctx, func(tx *sql.Tx) error {
		// check if record exists
		var exists bool
		err := tx.QueryRowContext(ctx, "SELECT 1 FROM health_records WHERE date = ?", hr.Date).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check existence: %w", err)
		}
		if !exists {
			return sql.ErrNoRows
		}

		// Update
		stmt := tx.StmtContext(ctx, updateStmt)
		now := time.Now()
		_, err = stmt.ExecContext(ctx, hr.StepCount, now, hr.Date)
		if err != nil {
			return fmt.Errorf("execute update %w", err)
		}

		return nil
	})
}

// DeleteHealthRecord deletes a health record by date
func (db *SQLiteDB) DeleteHealthRecord(ctx context.Context, date time.Time) error {
	dleleteStmt, err := db.getStmt("delete_health_record")
	if err != nil {
		return fmt.Errorf("getting delete statement: %w", err)
	}

	return db.withTxContext(ctx, func(tx *sql.Tx) error {
		// Check if record exists
		var exists bool
		err := tx.QueryRowContext(ctx, "SELECT 1 FROM health_records WHERE date = ?", date).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check existence: %w", err)
		}
		if !exists {
			return sql.ErrNoRows
		}

		// Delete
		stmt := tx.StmtContext(ctx, dleleteStmt)
		_, err = stmt.ExecContext(ctx, date)
		if err != nil {
			return fmt.Errorf("execute delete: %w", err)
		}

		return nil
	})
}
