package portaldb

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	portalshared "higress-portal-backend/schema/shared"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestEnsureSchemaAndMigrateLegacyDataAgainstSharedPortalSchema(t *testing.T) {
	ctx := context.Background()
	db := openMySQLForTest(t, ctx, "console_portaldb_it")

	require.NoError(t, portalshared.ApplyToSQL(ctx, db))

	_, err := db.ExecContext(ctx, `
		INSERT INTO org_department (department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status)
		VALUES ('root', 'Root', NULL, NULL, 'Root', 0, 0, 'active')`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
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
			deleted TINYINT(1) NOT NULL DEFAULT 0,
			last_login_at DATETIME NULL
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS portal_departments (
			department_id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			parent_department_id VARCHAR(64) NULL,
			admin_consumer_name VARCHAR(128) NULL,
			deleted TINYINT(1) NOT NULL DEFAULT 0
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS portal_asset_grant (
			asset_type VARCHAR(32) NOT NULL,
			asset_id VARCHAR(128) NOT NULL,
			subject_type VARCHAR(32) NOT NULL,
			subject_id VARCHAR(128) NOT NULL
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
			cost_reset_at DATETIME NULL,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ai_sensitive_detect_rule (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			pattern VARCHAR(1024) NOT NULL,
			match_type VARCHAR(32) NOT NULL DEFAULT 'contains',
			signature_hash VARCHAR(64) NOT NULL DEFAULT '',
			description TEXT NULL,
			priority INT NOT NULL DEFAULT 0,
			is_enabled TINYINT(1) NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ai_sensitive_replace_rule (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			pattern VARCHAR(1024) NOT NULL,
			replace_type VARCHAR(32) NOT NULL DEFAULT 'replace',
			replace_value TEXT NULL,
			restore TINYINT(1) NOT NULL DEFAULT 0,
			signature_hash VARCHAR(64) NOT NULL DEFAULT '',
			description TEXT NULL,
			priority INT NOT NULL DEFAULT 0,
			is_enabled TINYINT(1) NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ai_sensitive_system_config (
			id BIGINT PRIMARY KEY,
			system_deny_enabled TINYINT(1) NOT NULL DEFAULT 0,
			dictionary_text LONGTEXT NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_by VARCHAR(255) NULL
		)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS ai_sensitive_block_audit (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			request_id VARCHAR(128) NULL,
			route_name VARCHAR(255) NULL,
			consumer_name VARCHAR(255) NULL,
			display_name VARCHAR(255) NULL,
			blocked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			blocked_by VARCHAR(64) NOT NULL DEFAULT 'sensitive_word',
			request_phase VARCHAR(32) NULL,
			blocked_reason_json TEXT NULL,
			match_type VARCHAR(32) NULL,
			matched_rule VARCHAR(1024) NULL,
			matched_excerpt TEXT NULL,
			provider_id BIGINT NOT NULL DEFAULT 0,
			cost_usd DECIMAL(18,6) NOT NULL DEFAULT 0.000000
		)`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `
		INSERT INTO portal_departments (department_id, name, parent_department_id, admin_consumer_name, deleted)
		VALUES ('dept-eng', 'Engineering', 'root', 'demo', 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO portal_users (
			consumer_name, display_name, email, password_hash, status, source, user_level, department_id, parent_consumer_name, deleted
		) VALUES ('demo', 'Demo', 'demo@example.com', 'hash', 'active', 'console', 'pro', 'dept-eng', NULL, 0)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO portal_asset_grant (asset_type, asset_id, subject_type, subject_id)
		VALUES ('model_binding', 'binding-1', 'consumer', 'demo')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO portal_ai_quota_user_policy (
			route_name, consumer_name, limit_total, limit_5h, limit_daily, daily_reset_mode, daily_reset_time, limit_weekly, limit_monthly, cost_reset_at
		) VALUES ('route-a', 'demo', 1000, 200, 300, 'fixed', '08:00', 400, 500, NULL)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO ai_sensitive_detect_rule (
			id, pattern, match_type, signature_hash, description, priority, is_enabled
		) VALUES (7, '南京', 'contains', 'detect-7', 'legacy detect', 2, 1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO ai_sensitive_replace_rule (
			id, pattern, replace_type, replace_value, restore, signature_hash, description, priority, is_enabled
		) VALUES (8, '%{MOBILE}', 'replace', '***', 0, 'replace-8', 'legacy replace', 3, 1)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO ai_sensitive_system_config (
			id, system_deny_enabled, dictionary_text, updated_by
		) VALUES (1, 1, 'alpha\nbeta', 'tester')`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `
		INSERT INTO ai_sensitive_block_audit (
			id, request_id, route_name, consumer_name, display_name, blocked_by, request_phase, blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd
		) VALUES (9, 'req-9', 'route-a', 'demo', 'Demo', 'ai-security-guard', 'request', '{}', 'contains', 'rule-a', '南京', 12, 0.120000)`)
	require.NoError(t, err)

	client := NewFromDB(Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db)
	require.NoError(t, client.EnsureSchema(ctx))
	require.NoError(t, client.MigrateLegacyData(ctx))

	assertTableExists(t, ctx, db, "portal_model_binding_price_version")
	assertTableExists(t, ctx, db, "portal_ai_sensitive_detect_rule")
	assertTableExists(t, ctx, db, "portal_ai_quota_balance")
	assertTableExists(t, ctx, db, "job_run_record")

	var (
		displayName string
		email       string
		userLevel   string
	)
	err = db.QueryRowContext(ctx, `
		SELECT display_name, email, user_level
		FROM portal_user
		WHERE consumer_name = 'demo'`).Scan(&displayName, &email, &userLevel)
	require.NoError(t, err)
	require.Equal(t, "Demo", displayName)
	require.Equal(t, "demo@example.com", email)
	require.Equal(t, "pro", userLevel)

	var departmentID string
	err = db.QueryRowContext(ctx, `
		SELECT department_id
		FROM org_account_membership
		WHERE consumer_name = 'demo'`).Scan(&departmentID)
	require.NoError(t, err)
	require.Equal(t, "dept-eng", departmentID)

	var grantCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM asset_grant
		WHERE asset_type = 'model_binding' AND asset_id = 'binding-1' AND subject_type = 'consumer' AND subject_id = 'demo'`).Scan(&grantCount)
	require.NoError(t, err)
	require.Equal(t, 1, grantCount)

	var quotaTotal int64
	err = db.QueryRowContext(ctx, `
		SELECT limit_total_micro_yuan
		FROM quota_policy_user
		WHERE consumer_name = 'demo'`).Scan(&quotaTotal)
	require.NoError(t, err)
	require.EqualValues(t, 1000, quotaTotal)

	var detectCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM portal_ai_sensitive_detect_rule
		WHERE id = 7 AND pattern = '南京' AND enabled = 1`).Scan(&detectCount)
	require.NoError(t, err)
	require.Equal(t, 1, detectCount)

	var replaceCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM portal_ai_sensitive_replace_rule
		WHERE id = 8 AND pattern = '%{MOBILE}' AND enabled = 1`).Scan(&replaceCount)
	require.NoError(t, err)
	require.Equal(t, 1, replaceCount)

	var (
		systemDeny int
		dictText   string
	)
	err = db.QueryRowContext(ctx, `
		SELECT system_deny_enabled, dictionary_text
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`).Scan(&systemDeny, &dictText)
	require.NoError(t, err)
	require.Equal(t, 1, systemDeny)
	require.Equal(t, "alpha\nbeta", dictText)

	var auditCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM portal_ai_sensitive_block_audit
		WHERE id = 9 AND request_id = 'req-9'`).Scan(&auditCount)
	require.NoError(t, err)
	require.Equal(t, 1, auditCount)
}

func openMySQLForTest(t *testing.T, ctx context.Context, databaseName string) *sql.DB {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "mysql:8.4",
			ExposedPorts: []string{"3306/tcp"},
			Env: map[string]string{
				"MYSQL_ROOT_PASSWORD": "root",
				"MYSQL_DATABASE":      databaseName,
			},
			WaitingFor: wait.ForListeningPort("3306/tcp").WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)

	dsn := fmt.Sprintf("root:root@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=UTC", host, port.Port(), databaseName)
	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	require.Eventually(t, func() bool {
		return db.PingContext(ctx) == nil
	}, 30*time.Second, 500*time.Millisecond)
	return db
}

func assertTableExists(t *testing.T, ctx context.Context, db *sql.DB, table string) {
	t.Helper()

	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?`, table).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "expected table %s to exist", table)
}
