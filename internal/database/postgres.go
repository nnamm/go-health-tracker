package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnamm/go-health-tracker/internal/config"
	"github.com/nnamm/go-health-tracker/internal/models"
)

// PostgresDB represents a PostgreSQL database connection pool
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB creates a new PostgreSQL database connection pool using configuration
func NewPostgresDB(dataSourceName string) (*PostgresDB, error) {
	return NewPostgresDBWithConfig(dataSourceName, config.DBConfig)
}

// NewPostgresDBWithConfig creates a new PostgreSQL database connection pool with explicit configuration
// This function is useful for testing or when you need to override the global config
func NewPostgresDBWithConfig(dataSourceName string, dbConfig *config.DatabaseConfig) (*PostgresDB, error) {
	if dbConfig == nil {
		return nil, fmt.Errorf("database configuration cannot be nil")
	}

	// Use existing validation from factory.go instead of duplicating logic
	if err := ValidateConfiguration(dbConfig); err != nil {
		return nil, fmt.Errorf("invalid database configuration: %w", err)
	}

	// Parse the pool configuration from connection string
	poolConfig, err := pgxpool.ParseConfig(dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to parse postgres config: %w", err)
	}

	// Apply configuration values from config instead of hardcoded values
	poolConfig.MaxConns = dbConfig.MaxConns
	poolConfig.MinConns = dbConfig.MinConns
	poolConfig.MaxConnLifetime = dbConfig.MaxConnLifetime
	poolConfig.MaxConnIdleTime = dbConfig.MaxConnIdleTime

	// Set health check period based on best practices
	// Health check period should be shorter than max idle time for optimal performance
	healthCheckPeriod := time.Minute
	if dbConfig.MaxConnIdleTime > 2*time.Minute {
		healthCheckPeriod = dbConfig.MaxConnIdleTime / 2
	}
	poolConfig.HealthCheckPeriod = healthCheckPeriod

	// Create connection pool with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres pool: %w", err)
	}

	// Test the connection with a timeout
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	if err = pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	db := &PostgresDB{
		pool: pool,
	}

	// Create table with timeout context
	if err := db.createTable(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}

// createTable creates the health_records table if it doesn't exist
func (db *PostgresDB) createTable() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS health_records (
			id SERIAL PRIMARY KEY,
			date DATE NOT NULL UNIQUE,
			step_count INTEGER NOT NULL CHECK (step_count >= 0),
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
	    )`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_health_records_date
         ON health_records(date)`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, query := range queries {
		if _, err := db.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}
	return nil
}

// CreateHealthRecord creates a new health record
func (db *PostgresDB) CreateHealthRecord(ctx context.Context, hr *models.HealthRecord) (*models.HealthRecord, error) {
	query := `
		INSERT INTO health_records (date, step_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	var createdRecord models.HealthRecord

	// Copy input values to result
	createdRecord.Date = hr.Date
	createdRecord.StepCount = hr.StepCount

	err := db.pool.QueryRow(ctx, query, hr.Date, hr.StepCount, now, now).Scan(
		&createdRecord.ID,
		&createdRecord.CreatedAt,
		&createdRecord.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create health record: %w", err)
	}

	return &createdRecord, nil
}

// ReadHealthRecord reads a health record by date
func (db *PostgresDB) ReadHealthRecord(ctx context.Context, date time.Time) (*models.HealthRecord, error) {
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
			return nil, nil // No record found, return nil without error
		}
		return nil, fmt.Errorf("failed to read health record: %w", err)
	}

	return &hr, nil
}

// ReadHealthRecordsByYear reads health records for a specific year
func (db *PostgresDB) ReadHealthRecordsByYear(ctx context.Context, year int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(1, 0, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

// ReadHealthRecordsByYearMonth reads health records for a specific year and month
func (db *PostgresDB) ReadHealthRecordsByYearMonth(ctx context.Context, year, month int) ([]models.HealthRecord, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0)
	return db.readHealthRecordsByRange(ctx, startDate, endDate)
}

// readHealthRecordsByRange reads health records within a date range
func (db *PostgresDB) readHealthRecordsByRange(ctx context.Context, startDate, endDate time.Time) ([]models.HealthRecord, error) {
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

// UpdateHealthRecord updates an existing health record
func (db *PostgresDB) UpdateHealthRecord(ctx context.Context, hr *models.HealthRecord) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				err = fmt.Errorf("update failed: %v, rollback failed: %w", err, rollbackErr)
			}
		}
	}()

	// Check if record exists
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM health_records WHERE date = $1)"
	err = tx.QueryRow(ctx, checkQuery, hr.Date).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check record existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("record not found for date: %v", hr.Date)
	}

	// Update the record
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

// DeleteHealthRecord deletes a health record
func (db *PostgresDB) DeleteHealthRecord(ctx context.Context, date time.Time) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
				err = fmt.Errorf("delete failed: %v, rollback failed: %w", err, rollbackErr)
			}
		}
	}()

	// Check if record exists
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM health_records WHERE date = $1)"
	err = tx.QueryRow(ctx, checkQuery, date).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check record existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("record not found for date: %v", date)
	}

	// Delete the record
	deleteQuery := `DELETE FROM health_records WHERE date = $1`
	_, err = tx.Exec(ctx, deleteQuery, date)
	if err != nil {
		return fmt.Errorf("failed to delete health record: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection pool
func (db *PostgresDB) Close() error {
	if db.pool != nil {
		db.pool.Close()
	}
	return nil
}

// Ping checks if the database connection is alive
func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (db *PostgresDB) Stats() *pgxpool.Stat {
	return db.pool.Stat()
}

// HealthCheck performs a comprehensive health check of the database connection
func (db *PostgresDB) HealthCheck(ctx context.Context) error {
	// Check if pool is available
	if db.pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	// Ping the database
	if err := db.pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Check pool statistics for potential issues
	stats := db.pool.Stat()
	if stats.TotalConns() == 0 {
		return fmt.Errorf("no database connections available")
	}

	// Verify we can execute a simple query
	var result int
	err := db.pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	return nil
}

// Exec executes a query that doesn't return rows
func (db *PostgresDB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

// GetPoolInfo returns formatted pool information for monitoring/debugging
func (db *PostgresDB) GetPoolInfo() map[string]any {
	if db.pool == nil {
		return map[string]any{
			"status": "not_initialized",
		}
	}

	stats := db.pool.Stat()
	return map[string]any{
		"status":               "active",
		"total_connections":    stats.TotalConns(),
		"acquired_connections": stats.AcquiredConns(),
		"idle_connections":     stats.IdleConns(),
		"max_connections":      stats.MaxConns(),
		"acquire_count":        stats.AcquireCount(),
		"acquire_duration":     stats.AcquireDuration(),
		"new_conns_count":      stats.NewConnsCount(),
	}
}
