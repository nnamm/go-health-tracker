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

type DB struct {
	*sql.DB
	stmts map[string]*sql.Stmt
	mu    sync.RWMutex
}

type DBInterface interface {
	CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error)
	ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error)
	ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error)
	ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error)
	UpdateHealthRecord(hr *models.HealthRecord) error
	DeleteHealthRecord(date time.Time) error
}

// NewDB opens the DB
func NewDB(dataSourceName string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = sqlDB.Ping(); err != nil {
		return nil, err
	}

	db := &DB{
		DB:    sqlDB,
		stmts: make(map[string]*sql.Stmt),
		mu:    sync.RWMutex{},
	}

	queries := map[string]string{
		"insert_health_record":       `INSERT INTO health_records (date, step_count, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		"select_health_record":       `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date = ?`,
		"select_range_health_record": `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date >= ? AND date < ? ORDER BY date`,
		"update_health_record":       `UPDATE health_records SET step_count = ?, updated_at = ? WHERE date = ?`,
		"delete_health_record":       `DELETE FROM health_records WHERE date = ?`,
	}

	for name, query := range queries {
		stmt, err := db.Prepare(query)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("prepare statement %s: %w", name, err)
		}
		db.stmts[name] = stmt
	}

	return db, nil
}

// Close closes the DB
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// close all prepared statements
	for name, stmt := range db.stmts {
		if err := stmt.Close(); err != nil {
			return fmt.Errorf("closing statement %s: %w", name, err)
		}
	}

	// close the original database connection
	return db.DB.Close()
}

// getStmt is helper function to get a prepared statement
func (db *DB) getStmt(name string) (*sql.Stmt, error) {
	db.mu.RLock()
	stmt, ok := db.stmts[name]
	db.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("statement %s not found", name)
	}
	return stmt, nil
}

// withTxContext executes a function with a transaction and context
func (db *DB) withTxContext(ctx context.Context, fn func(*sql.Tx) error) error {
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

// CreateTable inisializes the table
func (db *DB) CreateTable() error {
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

// CreateHealthRecord inserts a new record
// func (db *DB) CreateHealthRecord(hr *models.HealthRecord) (*models.HealthRecord, error) {
// 	var createdRecord *models.HealthRecord
//
// 	insertStmt, err := db.getStmt("insert_health_record")
// 	if err != nil {
// 		return nil, fmt.Errorf("getting insert statment: %w", err)
// 	}
//
// 	err = db.withTx(func(tx *sql.Tx) error {
// 		// stmt := tx.Stmt(db.stmts["insert_health_record"])
// 		stmt := tx.Stmt(insertStmt)
//
// 		now := time.Now()
// 		result, err := stmt.Exec(hr.Date, hr.StepCount, now, now)
// 		if err != nil {
// 			return fmt.Errorf("insert record: %w", err)
// 		}
//
// 		id, err := result.LastInsertId()
// 		if err != nil {
// 			return fmt.Errorf("get last insert id: %w", err)
// 		}
// 		createdRecord = &models.HealthRecord{
// 			ID:        id,
// 			Date:      hr.Date,
// 			StepCount: hr.StepCount,
// 			CreatedAt: now,
// 			UpdatedAt: now,
// 		}
//
// 		return nil
// 	})
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	return createdRecord, nil
// }

// ReadHealthRecord retrieves a health record by date
// func (db *DB) ReadHealthRecord(date time.Time) (*models.HealthRecord, error) {
// 	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date = ?`
// 	hr := &models.HealthRecord{}
// 	err := db.QueryRow(query, date).Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, nil
// 		}
// 		return nil, err
// 	}
//
// 	return hr, nil
// }

// CreateHealthRecord inserts a new record
func (db *DB) CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error) {
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
func (db *DB) ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error) {
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
func (db *DB) ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

// ReadHealthRecordsByYearMonth retrieves record(s) by year and month
func (db *DB) ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

// readHealthRecordsByRange retrieves records between startDate and endDate
func (db *DB) readHealthRecordsByRange(ctx context.Context, startDate, endDate time.Time) ([]models.HealthRecord, error) {
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
func (db *DB) UpdateHealthRecord(hr *models.HealthRecord) error {
	return db.withTx(func(tx *sql.Tx) error {
		// Check existing a record
		var exists bool
		err := tx.QueryRow("SELECT 1 FROM health_records WHERE date = ?", hr.Date).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check existence: %w", err)
		}
		if !exists {
			return sql.ErrNoRows
		}

		// Update
		query := `UPDATE health_records SET step_count = ?, updated_at = ? WHERE date = ?`
		_, err = tx.Exec(query, hr.StepCount, time.Now(), hr.Date)
		if err != nil {
			return fmt.Errorf("execute update: %w", err)
		}

		return nil
	})
}

// DeleteHealthRecord removes a health record by date
func (db *DB) DeleteHealthRecord(date time.Time) error {
	return db.withTx(func(tx *sql.Tx) error {
		// Check existing a record
		var exists bool
		err := tx.QueryRow("SELECT 1 FROM health_records WHERE date = ?", date).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check existence: %w", err)
		}
		if !exists {
			return sql.ErrNoRows
		}

		// Delete
		query := `DELETE FROM health_records WHERE date = ?`
		_, err = tx.Exec(query, date)
		if err != nil {
			return fmt.Errorf("execute delete: %w", err)
		}

		return nil
	})
}

type TxFn func(*sql.Tx) error

func (db *DB) withTx(fn TxFn) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rallback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}
