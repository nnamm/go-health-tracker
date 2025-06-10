package database

import (
	"testing"
	"time"

	"github.com/nnamm/go-health-tracker/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDatabaseType(t *testing.T) {
	tests := []struct {
		name     string
		dbConfig *config.DatabaseConfig
		expected config.DatabaseType
	}{
		{
			name:     "nil config returns sqlite default",
			dbConfig: nil,
			expected: config.DatabaseSQLite,
		},
		{
			name: "postgresql config with valid values returns postgresql type",
			dbConfig: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "test_db",
				Username: "test_user",
			},
			expected: config.DatabasePostgreSQL,
		},
		{
			name: "sqlite config returns sqlite",
			dbConfig: &config.DatabaseConfig{
				Type:       config.DatabaseSQLite,
				SQLitePath: "/tmp/test.db",
			},
			expected: config.DatabaseSQLite,
		},
	}

	originalDBConfig := config.DBConfig
	t.Cleanup(func() {
		config.DBConfig = originalDBConfig
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.DBConfig = tt.dbConfig
			result := GetDatabaseType()
			assert.Equal(t, tt.expected, result, "GetDatabaseType() should return %v", tt.expected)
		})
	}
}

func TestValidateConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		config    *config.DatabaseConfig
		wantError bool
		errorMsg  string
	}{
		{
			name:      "nil config returns error",
			config:    nil,
			wantError: true,
			errorMsg:  "database configuration cannot be nil",
		},
		{
			name:      "valid postgresql config",
			config:    newValidPostgreSQLConfig(),
			wantError: false,
		},
		{
			name: "postgresql with empty host",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "",
				Port:     5432,
				Database: "test_db",
				Username: "test_user",
				MaxConns: 10,
				MinConns: 2,
			},
			wantError: true,
			errorMsg:  "PostgreSQL host cannot be empty",
		},
		{
			name: "postgresql with empty database name",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "",
				Username: "test_user",
				MaxConns: 10,
				MinConns: 2,
			},
			wantError: true,
			errorMsg:  "PostgreSQL database name cannot be empty",
		},
		{
			name: "postgresql with empty username",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "test_db",
				Username: "",
				MaxConns: 10,
				MinConns: 2,
			},
			wantError: true,
			errorMsg:  "PostgreSQL username cannot be empty",
		},
		{
			name: "postgresql with invalid port zero",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     0,
				Database: "test_db",
				Username: "test_user",
				MaxConns: 10,
				MinConns: 2,
			},
			wantError: true,
			errorMsg:  "PostgreSQL port must be between 1 and 65535, got: 0",
		},
		{
			name: "postgresql with invalid port too high",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     65536,
				Database: "test_db",
				Username: "test_user",
				MaxConns: 10,
				MinConns: 2,
			},
			wantError: true,
			errorMsg:  "PostgreSQL port must be between 1 and 65535, got: 65536",
		},
		{
			name: "postgresql with zero max connections",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "test_db",
				Username: "test_user",
				MaxConns: 0,
				MinConns: 2,
			},
			wantError: true,
			errorMsg:  "PostgreSQL max connections must be greater than 0, got: 0",
		},
		{
			name: "postgresql with negative min connections",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "test_db",
				Username: "test_user",
				MaxConns: 10,
				MinConns: -1,
			},
			wantError: true,
			errorMsg:  "PostgreSQL min connections cannot be negative, got: -1",
		},
		{
			name: "postgresql min connections exceeds max connections",
			config: &config.DatabaseConfig{
				Type:     config.DatabasePostgreSQL,
				Host:     "localhost",
				Port:     5432,
				Database: "test_db",
				Username: "test_user",
				MaxConns: 5,
				MinConns: 10,
			},
			wantError: true,
			errorMsg:  "PostgreSQL min connections (10) cannot exceed max connections (5)",
		},
		{
			name: "valid sqlite config",
			config: &config.DatabaseConfig{
				Type:       config.DatabaseSQLite,
				SQLitePath: "/tmp/test.db",
			},
			wantError: false,
		},
		{
			name: "sqlite with empty path",
			config: &config.DatabaseConfig{
				Type:       config.DatabaseSQLite,
				SQLitePath: "",
			},
			wantError: true,
			errorMsg:  "SQLite database path cannot be empty",
		},
		{
			name: "unsupported database type",
			config: &config.DatabaseConfig{
				Type: config.DatabaseType("unsupported"),
			},
			wantError: true,
			errorMsg:  "unsupported database type: unsupported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateConfiguration(tt.config)

			if tt.wantError {
				require.Error(t, err, "ValidateConfiguration() should return an error")
				assert.Contains(t, err.Error(), tt.errorMsg, "error message should contain expected text")
			} else {
				require.NoError(t, err, "ValidateConfiguration() should not return an error")
			}
		})
	}
}

func newValidPostgreSQLConfig() *config.DatabaseConfig {
	return &config.DatabaseConfig{
		Type:            config.DatabasePostgreSQL,
		Host:            "localhost",
		Port:            5432,
		Database:        "test_db",
		Username:        "test_user",
		Password:        "password",
		SSLMode:         "disable",
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 15 * time.Minute,
	}
}
