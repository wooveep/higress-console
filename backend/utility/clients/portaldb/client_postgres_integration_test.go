package portaldb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	portalshared "higress-portal-backend/schema/shared"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestEnsureSchemaAndMigrateLegacyDataAgainstPostgres(t *testing.T) {
	ctx := context.Background()
	db := openPostgresForTest(t, ctx, "console_portaldb_pg_it")

	require.NoError(t, portalshared.ApplyToSQLWithDriver(ctx, db, "postgres"))

	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS portal_users (
			consumer_name VARCHAR(128) PRIMARY KEY,
			display_name VARCHAR(128) NULL,
			email VARCHAR(255) NULL,
			password_hash VARCHAR(255) NULL,
			status VARCHAR(16) NULL,
			source VARCHAR(16) NULL,
			user_level VARCHAR(16) NULL,
			department_id VARCHAR(64) NULL,
			parent_consumer_name VARCHAR(128) NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			last_login_at TIMESTAMP NULL
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS portal_ai_quota_user_policy (
			route_name VARCHAR(255) NOT NULL,
			consumer_name VARCHAR(128) NOT NULL,
			limit_total BIGINT NOT NULL DEFAULT 0,
			limit_5h BIGINT NOT NULL DEFAULT 0,
			limit_daily BIGINT NOT NULL DEFAULT 0,
			daily_reset_mode VARCHAR(16) NOT NULL DEFAULT 'fixed',
			daily_reset_time VARCHAR(5) NOT NULL DEFAULT '00:00',
			limit_weekly BIGINT NOT NULL DEFAULT 0,
			limit_monthly BIGINT NOT NULL DEFAULT 0,
			cost_reset_at TIMESTAMP NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO portal_users (
			consumer_name, display_name, email, password_hash, status, source, user_level, deleted
		) VALUES ('demo', 'Demo', 'demo@example.com', 'hash', 'active', 'console', 'pro', FALSE)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO portal_ai_quota_user_policy (
			route_name, consumer_name, limit_total, limit_5h, limit_daily, daily_reset_mode, daily_reset_time, limit_weekly, limit_monthly
		) VALUES ('route-a', 'demo', 1000, 200, 300, 'fixed', '08:00', 400, 500)`)
	require.NoError(t, err)

	client := NewFromDB(Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db)
	require.NoError(t, client.EnsureSchema(ctx))
	require.NoError(t, client.MigrateLegacyData(ctx))

	var (
		displayName string
		userLevel   string
		quotaTotal  int64
	)
	err = db.QueryRowContext(ctx, `SELECT display_name, user_level FROM portal_user WHERE consumer_name = 'demo'`).Scan(&displayName, &userLevel)
	require.NoError(t, err)
	require.Equal(t, "Demo", displayName)
	require.Equal(t, "pro", userLevel)

	err = db.QueryRowContext(ctx, `SELECT limit_total_micro_yuan FROM quota_policy_user WHERE consumer_name = 'demo'`).Scan(&quotaTotal)
	require.NoError(t, err)
	require.EqualValues(t, 1000, quotaTotal)
}

func openPostgresForTest(t *testing.T, ctx context.Context, databaseName string) *sql.DB {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:16-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_USER":     "postgres",
				"POSTGRES_DB":       databaseName,
			},
			WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("host=%s port=%s user=postgres password=postgres dbname=%s sslmode=disable", host, port.Port(), databaseName)
	db, err := sql.Open(pgxRebindDriverName, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.Eventually(t, func() bool {
		return db.PingContext(ctx) == nil
	}, 60*time.Second, 500*time.Millisecond)
	return db
}
