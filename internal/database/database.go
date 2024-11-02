package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nnamm/go-health-tracker/internal/models"
)

type DB struct {
	*sql.DB
}

type DBInterface interface {
	CreateHealthRecord(hr *models.HealthRecord) (*models.HealthRecord, error)
	ReadHealthRecord(date time.Time) (*models.HealthRecord, error)
	ReadHealthRecordsByYear(year int) ([]models.HealthRecord, error)
	ReadHealthRecordsByYearMonth(year, month int) ([]models.HealthRecord, error)
	UpdateHealthRecord(hr *models.HealthRecord) error
	DeleteHealthRecord(date time.Time) error
}

// NewDB opens the DB
func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return &DB{db}, nil
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
func (db *DB) CreateHealthRecord(hr *models.HealthRecord) (*models.HealthRecord, error) {
	var createdRecord *models.HealthRecord

	err := db.withTx(func(tx *sql.Tx) error {
		query := `INSERT INTO health_records (date, step_count, created_at, updated_at) VALUES (?, ?, ?, ?)`
		now := time.Now()
		result, err := tx.Exec(query, hr.Date, hr.StepCount, now, now)
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
func (db *DB) ReadHealthRecord(date time.Time) (*models.HealthRecord, error) {
	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date = ?`
	hr := &models.HealthRecord{}
	err := db.QueryRow(query, date).Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return hr, nil
}

// ReadHealthRecordsByYear retrieves record(s) by year
func (db *DB) ReadHealthRecordsByYear(year int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, 0)

	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date >= ? AND date < ? ORDER BY date`
	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.HealthRecord
	for rows.Next() {
		var hr models.HealthRecord
		if err := rows.Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, hr)
	}

	return records, nil
}

// ReadHealthRecordsByYearMonth retrieves record(s) by year and month
func (db *DB) ReadHealthRecordsByYearMonth(year, month int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)

	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date >= ? AND date < ? ORDER BY date`
	rows, err := db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.HealthRecord
	for rows.Next() {
		var hr models.HealthRecord
		if err := rows.Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, hr)
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
