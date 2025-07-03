//go:build ignore_test

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPostgresSQLContainer_HelloWorld(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start PostgreSQL container")

	defer func() {
		assert.NoError(t, container.Terminate(ctx))
	}()

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	assert.Contains(t, connStr, "postgres://")
	assert.Contains(t, connStr, "test_user")
	assert.Contains(t, connStr, "test_db")

	t.Logf("PostgreSQL container started successfully!")
	t.Logf("Connection string: %s", connStr)
}
