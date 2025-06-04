package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnamm/go-health-tracker/internal/models"
)

type PostgresDB struct {
	pool *pgxpool.Pool
	mu   sync.RWMutex
}

func NewPostgresDB(dataSourceName string) (*PostgresDB, error) {
	config, err := pgxpool.ParseConfig(dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	if err = pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	db := &PostgresDB{
		pool: pool,
		mu:   sync.RWMutex{},
	}

	if err := db.createTable(); err != nil {
		return nil, fmt.Errorf("creating table: %w", err)
	}

	return db, nil
}

func (db *PostgresDB) createTable() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS health_records (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL UNIQUE,
			step_count INTEGER NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL
	    )`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_health_records_date
         ON health_records(date)`,
	}

	ctx := context.Background()
	for _, query := range queries {
		if _, err := db.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}
	return nil
}

func (db *PostgresDB) CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error) {
	db.mu.Lock()
	defer db.mu.RUnlock()

	query := `
		INSERT INTO health_records (date, step_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	var createdRecord models.HealthRecord

	err := db.pool.QueryRow(ctx, query, hr.Date, hr.StepCount, now, now).Scan(
		&createdRecord.ID,
		&createdRecord.Date,
		&createdRecord.StepCount,
		&createdRecord.CreatedAt,
		&createdRecord.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create health record: %w", err)
	}
	return &createdRecord, nil
}

func (db *PostgresDB) ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error) {
	db.mu.Lock()
	defer db.mu.RUnlock()

	query := `SELECT id, date, step_count, created_at, updated_at FROM health_records WHERE date = $1`

	var hr models.HealthRecord
	err := db.pool.QueryRow(ctx, query, date).Scan(
		&hr.ID,
		&hr.Date,
		&hr.StepCount,
		&hr.CreatedAt,
		&hr.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No record found, return nil. without error
		}
		return nil, fmt.Errorf("failed to read health record: %w", err)
	}

	return &hr, nil
}

func (db *PostgresDB) ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

func (db *PostgresDB) ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

func (db *PostgresDB) readHealthRecordsByRange(ctx context.Context, startDate, endDate time.Time) ([]models.HealthRecord, error) {
	db.mu.Lock()
	defer db.mu.RUnlock()

	query := `
		SELECT id, date, step_count, created_at, updated_at
		FROM health_records
		WHERE date >= $1 AND date < $2
		ORDER BY date`

	rows, err := db.pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query health records: %w", err)
	}
	defer rows.Close()

	var records []models.HealthRecord
	for rows.Next() {
		var hr models.HealthRecord
		if err := rows.Scan(&hr.ID, &hr.Date, &hr.StepCount, &hr.CreatedAt, &hr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan record: %w", err)
		}
		records = append(records, hr)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating through rows: %w", err)
	}

	return records, nil
}

func (db *PostgresDB) UpdateHealthRecord(ctx context.Context, hr *models.HealthRecord) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM health_records WHERE date = $1"
	err = tx.QueryRow(ctx, checkQuery, hr.Date).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check record existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("record not found for date: %v", hr.Date)
	}

	updateQuery := `UPDATE health_records
	                SET step_count = $1, updated_at = $2
	                WHERE date = $3`

	now := time.Now()
	_, err = tx.Exec(ctx, updateQuery, hr.StepCount, now, hr.Date)
	if err != nil {
		return fmt.Errorf("failed to update health record: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *PostgresDB) DeleteHealthRecord(ctx context.Context, hr *models.HealthRecord) error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM health_records WHERE date = $1"
	err = tx.QueryRow(ctx, checkQuery, hr.Date).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check record existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("record not found for date: %v", hr.Date)
	}

	deleteQuery := `DELETE FROM health_records WHERE date = $1`
	_, err = tx.Exec(ctx, deleteQuery)
	if err != nil {
		return fmt.Errorf("failed to delete health record: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (db *PostgresDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.pool != nil {
		db.pool.Close()
	}
	return nil
}
