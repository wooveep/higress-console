package portaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	portalshared "higress-portal-backend/schema/shared"
)

var ErrUnavailable = errors.New("portal database is unavailable")

type Config struct {
	Enabled     bool
	Driver      string
	DSN         string
	AutoMigrate bool
}

type Client interface {
	Healthy(ctx context.Context) error
	Enabled() bool
	DB() *sql.DB
	Driver() string
	EnsureSchema(ctx context.Context) error
	MigrateLegacyData(ctx context.Context) error
}

type FakeClient struct {
	config Config
}

type SQLClient struct {
	config Config
	db     *sql.DB
	err    error
}

func New(cfg Config) Client {
	if !cfg.Enabled || strings.TrimSpace(cfg.DSN) == "" {
		return &FakeClient{config: cfg}
	}

	driver := normalizeDriver(cfg.Driver, cfg.DSN)
	db, err := sql.Open(driver, cfg.DSN)
	if err != nil {
		return &SQLClient{config: cfg, err: err}
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxIdleConns(4)
	db.SetMaxOpenConns(8)

	client := &SQLClient{
		config: cfg,
		db:     db,
	}
	client.config.Driver = driver
	if cfg.AutoMigrate {
		client.err = client.EnsureSchema(context.Background())
	}
	return client
}

func NewFromDB(cfg Config, db *sql.DB) Client {
	return &SQLClient{
		config: Config{
			Enabled:     true,
			Driver:      normalizeDriver(cfg.Driver, cfg.DSN),
			DSN:         cfg.DSN,
			AutoMigrate: cfg.AutoMigrate,
		},
		db: db,
	}
}

func (c *FakeClient) Healthy(ctx context.Context) error { return ErrUnavailable }
func (c *FakeClient) Enabled() bool                     { return false }
func (c *FakeClient) DB() *sql.DB                       { return nil }
func (c *FakeClient) Driver() string                    { return normalizeDriver(c.config.Driver, c.config.DSN) }
func (c *FakeClient) EnsureSchema(ctx context.Context) error {
	return ErrUnavailable
}
func (c *FakeClient) MigrateLegacyData(ctx context.Context) error { return ErrUnavailable }

func (c *SQLClient) Healthy(ctx context.Context) error {
	if c.err != nil {
		return c.err
	}
	if c.db == nil {
		return ErrUnavailable
	}
	return c.db.PingContext(ctx)
}

func (c *SQLClient) Enabled() bool {
	return c.db != nil && (c.config.Enabled || strings.TrimSpace(c.config.DSN) != "")
}

func (c *SQLClient) DB() *sql.DB {
	return c.db
}

func (c *SQLClient) Driver() string {
	return c.config.Driver
}

func (c *SQLClient) EnsureSchema(ctx context.Context) error {
	if c.db == nil {
		return ErrUnavailable
	}
	if err := c.ensureSharedSchemaAvailable(ctx); err != nil {
		return err
	}
	if !c.config.AutoMigrate {
		return nil
	}

	ddl := []string{
		`CREATE TABLE IF NOT EXISTS portal_model_binding_price_version (
			version_id BIGINT PRIMARY KEY AUTO_INCREMENT,
			asset_id VARCHAR(255) NOT NULL,
			binding_id VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			active TINYINT NOT NULL DEFAULT 0,
			effective_from TIMESTAMP NULL,
			effective_to TIMESTAMP NULL,
			pricing_json TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_detect_rule (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			pattern TEXT NOT NULL,
			match_type VARCHAR(32) NOT NULL,
			description TEXT NULL,
			priority INT NOT NULL DEFAULT 0,
			enabled TINYINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_replace_rule (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			pattern TEXT NOT NULL,
			replace_type VARCHAR(32) NOT NULL,
			replace_value TEXT NULL,
			restore TINYINT NOT NULL DEFAULT 0,
			description TEXT NULL,
			priority INT NOT NULL DEFAULT 0,
			enabled TINYINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_system_config (
			config_key VARCHAR(64) PRIMARY KEY,
			system_deny_enabled TINYINT NOT NULL DEFAULT 0,
			dictionary_text LONGTEXT NULL,
			updated_by VARCHAR(255) NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_block_audit (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			request_id VARCHAR(255) NULL,
			route_name VARCHAR(255) NULL,
			consumer_name VARCHAR(255) NULL,
			display_name VARCHAR(255) NULL,
			blocked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			blocked_by VARCHAR(255) NULL,
			request_phase VARCHAR(64) NULL,
			blocked_reason_json LONGTEXT NULL,
			match_type VARCHAR(32) NULL,
			matched_rule TEXT NULL,
			matched_excerpt TEXT NULL,
			provider_id BIGINT NULL,
			cost_usd VARCHAR(64) NULL
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_quota_balance (
			route_name VARCHAR(255) NOT NULL,
			consumer_name VARCHAR(255) NOT NULL,
			quota BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (route_name, consumer_name)
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_quota_schedule_rule (
			id VARCHAR(64) PRIMARY KEY,
			route_name VARCHAR(255) NOT NULL,
			consumer_name VARCHAR(255) NOT NULL,
			action VARCHAR(32) NOT NULL,
			cron VARCHAR(255) NOT NULL,
			value BIGINT NOT NULL DEFAULT 0,
			enabled TINYINT NOT NULL DEFAULT 1,
			last_applied_at TIMESTAMP NULL,
			last_error TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS job_run_record (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			job_name VARCHAR(255) NOT NULL,
			trigger_source VARCHAR(64) NOT NULL,
			trigger_id VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			idempotency_key VARCHAR(255) NULL,
			target_version VARCHAR(255) NULL,
			message TEXT NULL,
			error_text LONGTEXT NULL,
			started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			finished_at TIMESTAMP NULL,
			duration_ms BIGINT NULL
		)`,
	}

	for _, statement := range ddl {
		if _, err := c.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return c.MigrateLegacyData(ctx)
}

func (c *SQLClient) MigrateLegacyData(ctx context.Context) error {
	if c.db == nil {
		return ErrUnavailable
	}
	if err := c.ensureSharedSchemaAvailable(ctx); err != nil {
		return err
	}

	if exists, err := c.tableExists(ctx, "portal_users"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyUsers(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "portal_departments"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyDepartments(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "portal_asset_grant"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAssetGrants(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "portal_ai_quota_user_policy"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyQuotaPolicies(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_detect_rule"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveDetectRules(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_replace_rule"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveReplaceRules(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_system_config"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveSystemConfig(ctx); err != nil {
			return err
		}
	}
	if exists, err := c.tableExists(ctx, "ai_sensitive_block_audit"); err != nil {
		return err
	} else if exists {
		if err := c.migrateLegacyAISensitiveAudits(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLClient) ensureSharedSchemaAvailable(ctx context.Context) error {
	for _, table := range portalshared.RequiredTables() {
		ok, err := c.tableExists(ctx, table)
		if err != nil {
			return WrapExecError("check shared schema", err)
		}
		if !ok {
			return fmt.Errorf("shared portal schema table is missing: %s", table)
		}
	}
	return nil
}

func (c *SQLClient) tableExists(ctx context.Context, table string) (bool, error) {
	var count int
	err := c.db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?`, table).Scan(&count)
	return count > 0, err
}

func (c *SQLClient) migrateLegacyUsers(ctx context.Context) error {
	statement := `
		INSERT INTO portal_user (
			consumer_name, display_name, email, password_hash, status, source, user_level, is_deleted, deleted_at, last_login_at
		)
		SELECT
			u.consumer_name,
			COALESCE(NULLIF(u.display_name, ''), u.consumer_name),
			COALESCE(u.email, ''),
			COALESCE(u.password_hash, ''),
			COALESCE(NULLIF(u.status, ''), 'pending'),
			COALESCE(NULLIF(u.source, ''), 'console'),
			COALESCE(NULLIF(u.user_level, ''), 'normal'),
			COALESCE(u.deleted, 0),
			CASE WHEN COALESCE(u.deleted, 0) = 1 THEN CURRENT_TIMESTAMP ELSE NULL END,
			u.last_login_at
		FROM portal_users u
		ON DUPLICATE KEY UPDATE
			display_name = VALUES(display_name),
			email = VALUES(email),
			password_hash = VALUES(password_hash),
			status = VALUES(status),
			source = VALUES(source),
			user_level = VALUES(user_level),
			is_deleted = VALUES(is_deleted),
			deleted_at = VALUES(deleted_at),
			last_login_at = VALUES(last_login_at)`
	if _, err := c.db.ExecContext(ctx, statement); err != nil {
		return WrapExecError("migrate legacy users", err)
	}

	membershipStatement := `
		INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name)
		SELECT consumer_name, NULLIF(department_id, ''), NULLIF(parent_consumer_name, '')
		FROM portal_users
		ON DUPLICATE KEY UPDATE
			department_id = VALUES(department_id),
			parent_consumer_name = VALUES(parent_consumer_name)`
	_, err := c.db.ExecContext(ctx, membershipStatement)
	return WrapExecError("migrate legacy memberships", err)
}

func (c *SQLClient) migrateLegacyDepartments(ctx context.Context) error {
	statement := `
		INSERT INTO org_department (
			department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status
		)
		SELECT
			d.department_id,
			d.name,
			NULLIF(d.parent_department_id, ''),
			NULLIF(d.admin_consumer_name, ''),
			COALESCE(NULLIF(d.name, ''), d.department_id),
			1,
			0,
			CASE WHEN COALESCE(d.deleted, 0) = 1 THEN 'deleted' ELSE 'active' END
		FROM portal_departments d
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			parent_department_id = VALUES(parent_department_id),
			admin_consumer_name = VALUES(admin_consumer_name),
			status = VALUES(status),
			updated_at = CURRENT_TIMESTAMP`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy departments", err)
}

func (c *SQLClient) migrateLegacyAssetGrants(ctx context.Context) error {
	statement := `
		INSERT INTO asset_grant (asset_type, asset_id, subject_type, subject_id)
		SELECT asset_type, asset_id, subject_type, subject_id
		FROM portal_asset_grant
		GROUP BY asset_type, asset_id, subject_type, subject_id
		ON DUPLICATE KEY UPDATE
			updated_at = CURRENT_TIMESTAMP`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy asset grants", err)
}

func (c *SQLClient) migrateLegacyQuotaPolicies(ctx context.Context) error {
	statement := `
		INSERT INTO quota_policy_user (
			consumer_name, limit_total_micro_yuan, limit_5h_micro_yuan, limit_daily_micro_yuan,
			daily_reset_mode, daily_reset_time, limit_weekly_micro_yuan, limit_monthly_micro_yuan, cost_reset_at
		)
		SELECT
			consumer_name,
			MAX(limit_total),
			MAX(limit_5h),
			MAX(limit_daily),
			SUBSTRING_INDEX(GROUP_CONCAT(daily_reset_mode ORDER BY updated_at DESC), ',', 1),
			SUBSTRING_INDEX(GROUP_CONCAT(daily_reset_time ORDER BY updated_at DESC), ',', 1),
			MAX(limit_weekly),
			MAX(limit_monthly),
			MAX(cost_reset_at)
		FROM portal_ai_quota_user_policy
		GROUP BY consumer_name
		ON DUPLICATE KEY UPDATE
			limit_total_micro_yuan = VALUES(limit_total_micro_yuan),
			limit_5h_micro_yuan = VALUES(limit_5h_micro_yuan),
			limit_daily_micro_yuan = VALUES(limit_daily_micro_yuan),
			daily_reset_mode = VALUES(daily_reset_mode),
			daily_reset_time = VALUES(daily_reset_time),
			limit_weekly_micro_yuan = VALUES(limit_weekly_micro_yuan),
			limit_monthly_micro_yuan = VALUES(limit_monthly_micro_yuan),
			cost_reset_at = VALUES(cost_reset_at),
			updated_at = CURRENT_TIMESTAMP`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy quota policies", err)
}

func (c *SQLClient) migrateLegacyAISensitiveDetectRules(ctx context.Context) error {
	statement := `
		INSERT INTO portal_ai_sensitive_detect_rule (
			id, pattern, match_type, description, priority, enabled, created_at, updated_at
		)
		SELECT
			id,
			pattern,
			match_type,
			description,
			priority,
			COALESCE(is_enabled, 1),
			created_at,
			updated_at
		FROM ai_sensitive_detect_rule
		ON DUPLICATE KEY UPDATE
			pattern = VALUES(pattern),
			match_type = VALUES(match_type),
			description = VALUES(description),
			priority = VALUES(priority),
			enabled = VALUES(enabled),
			created_at = VALUES(created_at),
			updated_at = VALUES(updated_at)`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy ai sensitive detect rules", err)
}

func (c *SQLClient) migrateLegacyAISensitiveReplaceRules(ctx context.Context) error {
	statement := `
		INSERT INTO portal_ai_sensitive_replace_rule (
			id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		)
		SELECT
			id,
			pattern,
			replace_type,
			replace_value,
			restore,
			description,
			priority,
			COALESCE(is_enabled, 1),
			created_at,
			updated_at
		FROM ai_sensitive_replace_rule
		ON DUPLICATE KEY UPDATE
			pattern = VALUES(pattern),
			replace_type = VALUES(replace_type),
			replace_value = VALUES(replace_value),
			restore = VALUES(restore),
			description = VALUES(description),
			priority = VALUES(priority),
			enabled = VALUES(enabled),
			created_at = VALUES(created_at),
			updated_at = VALUES(updated_at)`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy ai sensitive replace rules", err)
}

func (c *SQLClient) migrateLegacyAISensitiveSystemConfig(ctx context.Context) error {
	statement := `
		INSERT INTO portal_ai_sensitive_system_config (
			config_key, system_deny_enabled, dictionary_text, updated_by, updated_at
		)
		SELECT
			'default',
			system_deny_enabled,
			COALESCE(dictionary_text, ''),
			updated_by,
			updated_at
		FROM ai_sensitive_system_config
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
		ON DUPLICATE KEY UPDATE
			system_deny_enabled = VALUES(system_deny_enabled),
			dictionary_text = VALUES(dictionary_text),
			updated_by = VALUES(updated_by),
			updated_at = VALUES(updated_at)`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy ai sensitive system config", err)
}

func (c *SQLClient) migrateLegacyAISensitiveAudits(ctx context.Context) error {
	statement := `
		INSERT INTO portal_ai_sensitive_block_audit (
			id, request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, request_phase,
			blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd
		)
		SELECT
			id,
			request_id,
			route_name,
			consumer_name,
			display_name,
			blocked_at,
			blocked_by,
			request_phase,
			blocked_reason_json,
			match_type,
			matched_rule,
			matched_excerpt,
			provider_id,
			CAST(cost_usd AS CHAR)
		FROM ai_sensitive_block_audit
		ON DUPLICATE KEY UPDATE
			request_id = VALUES(request_id),
			route_name = VALUES(route_name),
			consumer_name = VALUES(consumer_name),
			display_name = VALUES(display_name),
			blocked_at = VALUES(blocked_at),
			blocked_by = VALUES(blocked_by),
			request_phase = VALUES(request_phase),
			blocked_reason_json = VALUES(blocked_reason_json),
			match_type = VALUES(match_type),
			matched_rule = VALUES(matched_rule),
			matched_excerpt = VALUES(matched_excerpt),
			provider_id = VALUES(provider_id),
			cost_usd = VALUES(cost_usd)`
	_, err := c.db.ExecContext(ctx, statement)
	return WrapExecError("migrate legacy ai sensitive audits", err)
}

func normalizeDriver(driver, dsn string) string {
	normalized := strings.ToLower(strings.TrimSpace(driver))
	if normalized != "" {
		return normalized
	}
	return "mysql"
}

func MustDB(client Client) (*sql.DB, error) {
	if client == nil || !client.Enabled() || client.DB() == nil {
		return nil, ErrUnavailable
	}
	return client.DB(), nil
}

func WrapExecError(action string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", action, err)
}
