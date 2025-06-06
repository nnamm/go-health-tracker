package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
