package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type envVars struct {
	dbType            string
	dbHost            string
	dbPort            string
	dbName            string
	dbUser            string
	dbPassword        string
	dbSSLMode         string
	dbPath            string
	dbMaxConns        string
	dbMinConns        string
	dbMaxConnLifetime string
	dbMaxConnIdle     string
}

func TestLoadDatabaseConfig(t *testing.T) {
	tests := []struct {
		name     string
		envVars  envVars
		expected *DatabaseConfig
	}{
		{
			name:    "all default values",
			envVars: envVars{}, // all empty strings (not set)
			expected: &DatabaseConfig{
				Type:            DatabaseSQLite,
				Host:            "localhost",
				Port:            5432,
				Database:        "health_tracker",
				Username:        "postgres",
				Password:        "",
				SSLMode:         "disable",
				SQLitePath:      "./health_tracker.db",
				MaxConns:        25,
				MinConns:        5,
				MaxConnLifetime: 60 * time.Minute,
				MaxConnIdleTime: 30 * time.Minute,
			},
		},
		{
			name: "postgresql configuration",
			envVars: envVars{
				dbType:     "postgresql",
				dbHost:     "db.example.com",
				dbPort:     "5433",
				dbName:     "test_db",
				dbUser:     "test_user",
				dbPassword: "secret123",
				dbSSLMode:  "require",
			},
			expected: &DatabaseConfig{
				Type:            DatabasePostgreSQL,
				Host:            "db.example.com",
				Port:            5433,
				Database:        "test_db",
				Username:        "test_user",
				Password:        "secret123",
				SSLMode:         "require",
				SQLitePath:      "./health_tracker.db", // default value
				MaxConns:        25,                    // default value
				MinConns:        5,                     // default value
				MaxConnLifetime: 60 * time.Minute,      // default value
				MaxConnIdleTime: 30 * time.Minute,      // default value
			},
		},
		{
			name: "sqlite configuration with custom path",
			envVars: envVars{
				dbType: "sqlite",
				dbPath: "/tmp/test.db",
			},
			expected: &DatabaseConfig{
				Type:            DatabaseSQLite,
				Host:            "localhost",
				Port:            5432,
				Database:        "health_tracker",
				Username:        "postgres",
				Password:        "",
				SSLMode:         "disable",
				SQLitePath:      "/tmp/test.db",
				MaxConns:        25,
				MinConns:        5,
				MaxConnLifetime: 60 * time.Minute,
				MaxConnIdleTime: 30 * time.Minute,
			},
		},
		{
			name: "connection pool settings",
			envVars: envVars{
				dbType:            "postgresql",
				dbMaxConns:        "50",
				dbMinConns:        "10",
				dbMaxConnLifetime: "120",
				dbMaxConnIdle:     "60",
			},
			expected: &DatabaseConfig{
				Type:            DatabasePostgreSQL,
				Host:            "localhost",
				Port:            5432,
				Database:        "health_tracker",
				Username:        "postgres",
				Password:        "",
				SSLMode:         "disable",
				SQLitePath:      "./health_tracker.db",
				MaxConns:        50,
				MinConns:        10,
				MaxConnLifetime: 120 * time.Minute,
				MaxConnIdleTime: 60 * time.Minute,
			},
		},
		{
			name: "invalid port falls back to default",
			envVars: envVars{
				dbType: "postgresql",
				dbPort: "invalid_port",
			},
			expected: &DatabaseConfig{
				Type:            DatabasePostgreSQL,
				Host:            "localhost",
				Port:            5432, // default value
				Database:        "health_tracker",
				Username:        "postgres",
				Password:        "",
				SSLMode:         "disable",
				SQLitePath:      "./health_tracker.db",
				MaxConns:        25,
				MinConns:        5,
				MaxConnLifetime: 60 * time.Minute,
				MaxConnIdleTime: 30 * time.Minute,
			},
		},
	}

	envKeys := []string{
		"DB_TYPE", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "DB_SSL_MODE",
		"DB_PATH", "DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME_MINUTES", "DB_MAX_CONN_IDLE_MINUTES",
	}

	originalEnv := make(map[string]string)
	for _, key := range envKeys {
		if val, exists := os.LookupEnv(key); exists {
			originalEnv[key] = val
		}
	}

	t.Cleanup(func() {
		clearTestEnvVars()
		for key, val := range originalEnv {
			os.Setenv(key, val)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnvVars()
			setTestEnvVars(tt.envVars)

			result := LoadDatabaseConfig()

			assert.Equal(t, tt.expected, result, "LoadDatabaseConfig() should return expected configuration")
		})
	}
}

func setTestEnvVars(envVars envVars) {
	envMap := map[string]string{
		"DB_TYPE":                      envVars.dbType,
		"DB_HOST":                      envVars.dbHost,
		"DB_PORT":                      envVars.dbPort,
		"DB_NAME":                      envVars.dbName,
		"DB_USER":                      envVars.dbUser,
		"DB_PASSWORD":                  envVars.dbPassword,
		"DB_SSL_MODE":                  envVars.dbSSLMode,
		"DB_PATH":                      envVars.dbPath,
		"DB_MAX_CONNS":                 envVars.dbMaxConns,
		"DB_MIN_CONNS":                 envVars.dbMinConns,
		"DB_MAX_CONN_LIFETIME_MINUTES": envVars.dbMaxConnLifetime,
		"DB_MAX_CONN_IDLE_MINUTES":     envVars.dbMaxConnIdle,
	}

	for key, value := range envMap {
		if value != "" {
			os.Setenv(key, value)
		}
	}
}

func clearTestEnvVars() {
	envKeys := []string{
		"DB_TYPE", "DB_HOST", "DB_PORT", "DB_NAME", "DB_USER", "DB_PASSWORD", "DB_SSL_MODE",
		"DB_PATH", "DB_MAX_CONNS", "DB_MIN_CONNS", "DB_MAX_CONN_LIFETIME_MINUTES", "DB_MAX_CONN_IDLE_MINUTES",
	}

	for _, key := range envKeys {
		os.Unsetenv(key)
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue string
		setEnv       bool
		expected     string
	}{
		{
			name:         "environment variable exists",
			envKey:       "TEST_KEY_1",
			envValue:     "test_value_1",
			defaultValue: "default_value",
			setEnv:       true,
			expected:     "test_value_1",
		},
		{
			name:         "environment variable does not exist",
			envKey:       "TEST_KEY_2",
			envValue:     "",
			defaultValue: "default_value",
			setEnv:       false,
			expected:     "default_value",
		},
		{
			name:         "environment variable is empty",
			envKey:       "TEST_KEY_3",
			envValue:     "",
			defaultValue: "default_value",
			setEnv:       true,
			expected:     "",
		},
		{
			name:         "special characters in value",
			envKey:       "TEST_KEY_4",
			envValue:     "user:password@host:5432/db?sslmode=disable",
			defaultValue: "default",
			setEnv:       true,
			expected:     "user:password@host:5432/db?sslmode=disable",
		},
		{
			name:         "japanese characters in value",
			envKey:       "TEST_KEY_5",
			envValue:     "データベース設定",
			defaultValue: "default",
			setEnv:       true,
			expected:     "データベース設定",
		},
	}

	originalEnv := make(map[string]string)
	envKeysToCleanup := make([]string, 0, len(tests))

	for _, tt := range tests {
		if val, exists := os.LookupEnv(tt.envKey); exists {
			originalEnv[tt.envKey] = val
		}
		envKeysToCleanup = append(envKeysToCleanup, tt.envKey)
	}

	t.Cleanup(func() {
		for _, key := range envKeysToCleanup {
			if originalVal, existed := originalEnv[key]; existed {
				os.Setenv(key, originalVal)
			} else {
				os.Unsetenv(key)
			}
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.setEnv {
				os.Setenv(tt.envKey, tt.envValue)
			} else {
				os.Unsetenv(tt.envKey)
			}

			result := getEnv(tt.envKey, tt.defaultValue)

			assert.Equal(t, tt.expected, result,
				"getEnv(%q, %q) should return %q", tt.envKey, tt.defaultValue, tt.expected)
		})
	}
}

func TestGetEnvAsInt(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		defaultValue int
		setEnv       bool
		expected     int
	}{
		{
			name:         "valid integer string exists",
			envKey:       "TEST_INT_KEY_1",
			envValue:     "42",
			defaultValue: 10,
			setEnv:       true,
			expected:     42,
		},
		{
			name:         "empty string returns default",
			envKey:       "TEST_INT_KEY_2",
			envValue:     "",
			defaultValue: 100,
			setEnv:       true,
			expected:     100,
		},
		{
			name:         "environment variable not set returns default",
			envKey:       "TEST_INT_KEY_3",
			envValue:     "",
			defaultValue: 200,
			setEnv:       false,
			expected:     200,
		},
		{
			name:         "invalid string returns default",
			envKey:       "TEST_INT_KEY_4",
			envValue:     "not_a_number",
			defaultValue: 300,
			setEnv:       true,
			expected:     300,
		},
		{
			name:         "negative integer",
			envKey:       "TEST_INT_KEY_5",
			envValue:     "-123",
			defaultValue: 50,
			setEnv:       true,
			expected:     -123,
		},
		{
			name:         "zero value",
			envKey:       "TEST_INT_KEY_6",
			envValue:     "0",
			defaultValue: 999,
			setEnv:       true,
			expected:     0,
		},
		{
			name:         "large integer",
			envKey:       "TEST_INT_KEY_7",
			envValue:     "2147483647", // int32 max
			defaultValue: 1,
			setEnv:       true,
			expected:     2147483647,
		},
		{
			name:         "integer with leading zeros",
			envKey:       "TEST_INT_KEY_8",
			envValue:     "0042",
			defaultValue: 1,
			setEnv:       true,
			expected:     42,
		},
		{
			name:         "float string returns default",
			envKey:       "TEST_INT_KEY_9",
			envValue:     "42.5",
			defaultValue: 15,
			setEnv:       true,
			expected:     15,
		},
		{
			name:         "integer overflow returns default",
			envKey:       "TEST_INT_KEY_10",
			envValue:     "9223372036854775808", // int64 max + 1
			defaultValue: 999,
			setEnv:       true,
			expected:     999,
		},
		{
			name:         "whitespace around number returns default",
			envKey:       "TEST_INT_KEY_11",
			envValue:     " 42 ",
			defaultValue: 100,
			setEnv:       true,
			expected:     100, // strconv.Atoi not allow whitespace around number
		},
	}

	originalEnv := make(map[string]string)
	envKeysToCleanup := make([]string, 0, len(tests))

	for _, tt := range tests {
		if val, exists := os.LookupEnv(tt.envKey); exists {
			originalEnv[tt.envKey] = val
		}
		envKeysToCleanup = append(envKeysToCleanup, tt.envKey)
	}

	t.Cleanup(func() {
		for _, key := range envKeysToCleanup {
			if originalVal, existed := originalEnv[key]; existed {
				os.Setenv(key, originalVal)
			} else {
				os.Unsetenv(key)
			}
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.setEnv {
				os.Setenv(tt.envKey, tt.envValue)
			} else {
				os.Unsetenv(tt.envKey)
			}

			result := getEnvAsInt(tt.envKey, tt.defaultValue)

			assert.Equal(t, tt.expected, result,
				"getEnvAsInt(%q, %d) should return %d", tt.envKey, tt.defaultValue, tt.expected)
		})
	}
}
