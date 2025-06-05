package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DatabaseType defines the type of database to use
type DatabaseType string

const (
	DatabaseSQLite     DatabaseType = "sqlite"
	DatabasePostgreSQL DatabaseType = "postgresql"
)

// DatabaseConfig holds all database-related configuration
type DatabaseConfig struct {
	Type     DatabaseType
	Host     string
	Port     int
	Database string
	Username string
	Password string
	SSLMode  string

	// SQLite specific
	SQLitePath string

	// Connection pool settings for PostgreSQL
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// Global database configuration instance
var DBConfig *DatabaseConfig

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig() *DatabaseConfig {
	config := &DatabaseConfig{
		Type:     DatabaseType(getEnv("DB_TYPE", "sqlite")),
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		Database: getEnv("DB_NAME", "health_tracker"),
		Username: getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),

		// SQLite configuration
		SQLitePath: getEnv("DB_PATH", "./health_tracker.db"),

		// PostgreSQL connection pool settings
		MaxConns:        int32(getEnvAsInt("DB_MAX_CONNS", 25)),
		MinConns:        int32(getEnvAsInt("DB_MIN_CONNS", 5)),
		MaxConnLifetime: time.Duration(getEnvAsInt("DB_MAX_CONN_LIFETIME_MINUTES", 60)) * time.Minute,
		MaxConnIdleTime: time.Duration(getEnvAsInt("DB_MAX_CONN_IDLE_MINUTES", 30)) * time.Minute,
	}

	return config
}

// GetConnectionString returns the appropriate connection string based on database type
func (c *DatabaseConfig) GetConnectionString() string {
	switch c.Type {
	case DatabasePostgreSQL:
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			c.Username, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
	case DatabaseSQLite:
		return c.SQLitePath
	default:
		return c.SQLitePath
	}
}

// IsPostgreSQL returns true if PostgreSQL is configured
func (c *DatabaseConfig) IsPostgreSQL() bool {
	return c.Type == DatabasePostgreSQL
}

// IsSQLite returns true if SQLite is configured
func (c *DatabaseConfig) IsSQLite() bool {
	return c.Type == DatabaseSQLite
}

// getEnv retrieves the value of an environment variable by key.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvAsInt retrieves the value of an environment variable by key and converts it to an integer.
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

// init function to initialize database configuration
func init() {
	DBConfig = LoadDatabaseConfig()
}
