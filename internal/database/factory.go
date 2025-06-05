package database

import (
	"fmt"

	"github.com/nnamm/go-health-tracker/internal/config"
)

// NewDatabase creates a new database instance based on configuration
// It returns a DBInterface implementation that can be either SQLite or PostgreSQL
func NewDatabase() (DBInterface, error) {
	dbConfig := config.DBConfig
	if dbConfig == nil {
		return nil, fmt.Errorf("database configuration is not initialized")
	}

	connectionString := dbConfig.GetConnectionString()
	if connectionString == "" {
		return nil, fmt.Errorf("database connection string is empty")
	}

	switch dbConfig.Type {
	case config.DatabasePostgreSQL:
		return NewPostgresDB(connectionString)
	case config.DatabaseSQLite:
		return NewDB(connectionString)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}
}

// NewDatabaseWithConfig creates a database instance with explicit configuration
// This functions is useful for testing or when you need to override the global config
func NewDatabaseWithConfig(dbConfig *config.DatabaseConfig) (DBInterface, error) {
	if dbConfig == nil {
		return nil, fmt.Errorf("database configuration cannot be nil")
	}

	connectionString := dbConfig.GetConnectionString()
	if connectionString == "" {
		return nil, fmt.Errorf("database connection string is empty")
	}

	switch dbConfig.Type {
	case config.DatabasePostgreSQL:
		return NewPostgresDB(connectionString)
	case config.DatabaseSQLite:
		return NewDB(connectionString)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}
}

// GetDatabaseType returns the currently configured database type
// This is useful for conditional logic or logging purposes
func GetDatabaseType() config.DatabaseType {
	if config.DBConfig == nil {
		return config.DatabaseSQLite
	}
	return config.DBConfig.Type
}

// ValidateConfiguration validates the database configuration
// Returns an error if the configuration is invalid
func ValidateConfiguration(dbConfig *config.DatabaseConfig) error {
	if dbConfig == nil {
		return fmt.Errorf("database configuration cannot be nil")
	}

	switch dbConfig.Type {
	case config.DatabasePostgreSQL:
		if dbConfig.Host == "" {
			return fmt.Errorf("PostgreSQL host cannot be empty")
		}
		if dbConfig.Database == "" {
			return fmt.Errorf("PostgreSQL database name cannot be empty")
		}
		if dbConfig.Username == "" {
			return fmt.Errorf("PostgreSQL username cannot be empty")
		}
		if dbConfig.Port <= 0 || dbConfig.Port > 65535 {
			return fmt.Errorf("PostgreSQL port must be between 1 and 65535, got: %d", dbConfig.Port)
		}
		if dbConfig.MaxConns <= 0 {
			return fmt.Errorf("PostgreSQL max connections must be greater than 0, got: %d", dbConfig.MaxConns)
		}
		if dbConfig.MinConns < 0 {
			return fmt.Errorf("PostgreSQL min connections cannot be negative, got: %d", dbConfig.MinConns)
		}
		if dbConfig.MinConns > dbConfig.MaxConns {
			return fmt.Errorf("PostgreSQL min connections (%d) cannot exceed max connections (%d)",
				dbConfig.MinConns, dbConfig.MaxConns)
		}
	case config.DatabaseSQLite:
		if dbConfig.SQLitePath == "" {
			return fmt.Errorf("SQLite database path cannot be empty")
		}
	default:
		return fmt.Errorf("unsupported database type: %s", dbConfig.Type)
	}

	return nil
}

// NewTestDatabase creates a database instance specifically for testing
// It uses in-memory SQLite by default for fast test execution
func NewTestDatabase() (DBInterface, error) {
	testConfig := &config.DatabaseConfig{
		Type:       config.DatabaseSQLite,
		SQLitePath: ":memory:",
	}
	return NewDatabaseWithConfig(testConfig)
}

// NewTestDatabaseWithType creates a test database with specific type
// Useful for testing both SQLite and PostgreSQL implementations
func NewTestDatabaseWithType(dbType config.DatabaseType) (DBInterface, error) {
	var testConfig *config.DatabaseConfig

	switch dbType {
	case config.DatabaseSQLite:
		testConfig = &config.DatabaseConfig{
			Type:       config.DatabaseSQLite,
			SQLitePath: ":memory:",
		}
	case config.DatabasePostgreSQL:
		// Test PostgreSQL configuration (requires Testcontainer in actual tests)
		testConfig = &config.DatabaseConfig{
			Type:            config.DatabasePostgreSQL,
			Host:            "localhost",
			Port:            5432,
			Database:        "test_health_tracker",
			Username:        "test_user",
			Password:        "test_password",
			SSLMode:         "disable",
			MaxConns:        5,
			MinConns:        1,
			MaxConnLifetime: config.DBConfig.MaxConnLifetime,
			MaxConnIdleTime: config.DBConfig.MaxConnIdleTime,
		}
	default:
		return nil, fmt.Errorf("unsupported test database type: %s", dbType)
	}

	return NewDatabaseWithConfig(testConfig)
}
