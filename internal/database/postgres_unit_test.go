package database_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/nnamm/go-health-tracker/testutils"
	"github.com/stretchr/testify/assert"
)

var testPostgres *database.PostgresDB

func NewPostgresDBWithMock(t *testing.T) (*database.PostgresDB, sqlmock.Sqlmock) {
	t.Helper()

	_, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	poolCfg, err := pgxpool.ParseConfig("")
	if err != nil {
		t.Fatalf("failed to parse pgxpool config: %v", err)
	}
	poolCfg.ConnConfig.Host = "mock_host"

	t.Log("Note: A proper mock for pgxpool is required. The current setup is a placeholder.")

	mockDB := &database.PostgresDB{} // This won't work as-is.

	return mockDB, mock
}

func TestPosgres_GetPoolInfo(t *testing.T) {
	tests := []struct {
		name     string
		setupDB  func() *database.PostgresDB
		expected map[string]any
	}{
		{
			name: "pool not initialized returns not_initialized status",
			setupDB: func() *database.PostgresDB {
				return testutils.NewPostgresDBForTest()
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

func TestPosgres_Close(t *testing.T) {
	tests := []struct {
		name    string
		setupDB func() *database.PostgresDB
		wantErr bool
	}{
		{
			name: "nil pool closes without error",
			setupDB: func() *database.PostgresDB {
				return testutils.NewPostgresDBForTest()
			},
			wantErr: false,
		},
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
