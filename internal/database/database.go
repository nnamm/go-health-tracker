package database

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nnamm/go-health-tracker/internal/models"
)

type DB struct {
	*sql.DB
}

type DBInterface interface {
	CreateHealthRecord(hr *models.HealthRecord) error
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
	query := `CREATE TABLE IF NOT EXISTS health_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE NOT NULL,
			step_count INTEGER NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
	)`
	_, err := db.Exec(query)
	return err
}

// CreateHealthRecord inserts a new record
func (db *DB) CreateHealthRecord(hr *models.HealthRecord) error {
	query := `INSERT INTO health_records (date, step_count, created_at, updated_at) VALUES (?, ?, ?, ?)`
	now := time.Now()
	_, err := db.Exec(query, hr.Date, hr.StepCount, now, now)
	return err
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
	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE strftime('%Y', date) = ?`
	rows, err := db.Query(query, year)
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
	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE strftime('%Y', date) = ? AND strftime('%m', date) = ?`
	rows, err := db.Query(query, year, month)
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
	query := `UPDATE health_records SET step_count = ?, updated_at = ? WHERE date = ?`
	_, err := db.Exec(query, hr.StepCount, time.Now(), hr.Date)
	return err
}

// DeleteHealthRecord removes a health record by date
func (db *DB) DeleteHealthRecord(date time.Time) error {
	query := `DELETE FROM health_records WHERE date = ?`
	_, err := db.Exec(query, date)
	return err
}
