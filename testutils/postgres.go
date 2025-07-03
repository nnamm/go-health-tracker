package testutils

import (
	"context"
	"testing"

	"github.com/nnamm/go-health-tracker/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type PostgresTestContainer struct {
	Container *postgres.PostgresContainer
	DB        *database.PostgresDB
	ConnStr   string
}

func SetupPostgresContainer(ctx context.Context, t *testing.T) *PostgresTestContainer {
	t.Helper()

	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("health_tracker_test"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start PostgreSQL container")

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	db, err := database.NewPostgresDB(connStr)
	require.NoError(t, err, "failed to connect to test database")

	return &PostgresTestContainer{
		Container: container,
		DB:        db,
		ConnStr:   connStr,
	}
}

func (ptc *PostgresTestContainer) Cleanup(ctx context.Context, t *testing.T) {
	t.Helper()

	if ptc.DB != nil {
		require.NoError(t, ptc.DB.Close(), "failed to close database connection")
	}
	if ptc.Container != nil {
		assert.NoError(t, ptc.Container.Terminate(ctx), "failed to terminate container")
	}
}

func (ptc *PostgresTestContainer) CleanupTestData(ctx context.Context, t *testing.T) {
	t.Helper()

	_, err := ptc.DB.Exec(ctx, "TRUNCATE TABLE health_records RESTART IDENTITY")
	require.NoError(t, err, "failed to cleanup test data")
}
