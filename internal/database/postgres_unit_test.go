package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var testPostgres PostgresDB

func TestGetPoolInfo(t *testing.T) {
	tests := []struct {
		name     string
		setupDB  func() *PostgresDB
		expected map[string]any
	}{
		{
			name: "pool not initialized returns not_initialized status",
			setupDB: func() *PostgresDB {
				return &PostgresDB{pool: nil}
			},
			expected: map[string]any{
				"status": "not_initialized",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.setupDB()
			result := db.GetPoolInfo()

			assert.Equal(t, tt.expected, result, "GetPoolInfo() should return expected status")
		})
	}
}

func TestClose(t *testing.T) {
	tests := []struct {
		name    string
		setupDB func() *PostgresDB
		wantErr bool
	}{
		{
			name: "nil pool closes without error",
			setupDB: func() *PostgresDB {
				return &PostgresDB{pool: nil}
			},
			wantErr: false,
		},
		// {
		// 	name: "already closed pool handles gracegully",
		// 	setupDB: func() *PostgresDB {
		// 		// simulates the state in which Close() is acutually called
		// 		return &PostgresDB{pool: nil}
		// 	},
		// 	wantErr: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.setupDB()
			err := db.Close()

			if tt.wantErr {
				assert.Error(t, err, "Close() should return an error")
			} else {
				assert.NoError(t, err, "Close() should not return an error")
			}
		})
	}
}
