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

// CreateHealthRecord insert a new record
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
		return nil, err
	}
	return hr, nil
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
