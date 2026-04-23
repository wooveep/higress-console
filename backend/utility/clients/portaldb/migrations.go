package portaldb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

func ensureSchemaDDLs(driver string) []string {
	_ = driver
	return []string{
		`CREATE TABLE IF NOT EXISTS portal_model_binding_price_version (
			version_id BIGSERIAL PRIMARY KEY,
			asset_id VARCHAR(255) NOT NULL,
			binding_id VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			active BOOLEAN NOT NULL DEFAULT FALSE,
			effective_from TIMESTAMP NULL,
			effective_to TIMESTAMP NULL,
			pricing_json TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_detect_rule (
			id BIGSERIAL PRIMARY KEY,
			pattern TEXT NOT NULL,
			match_type VARCHAR(32) NOT NULL,
			description TEXT NULL,
			priority INT NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_replace_rule (
			id BIGSERIAL PRIMARY KEY,
			pattern TEXT NOT NULL,
			replace_type VARCHAR(32) NOT NULL,
			replace_value TEXT NULL,
			restore BOOLEAN NOT NULL DEFAULT FALSE,
			description TEXT NULL,
			priority INT NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_system_config (
			config_key VARCHAR(64) PRIMARY KEY,
			system_deny_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			dictionary_text TEXT NULL,
			updated_by VARCHAR(255) NULL,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS portal_ai_sensitive_block_audit (
			id BIGSERIAL PRIMARY KEY,
			request_id VARCHAR(255) NULL,
			route_name VARCHAR(255) NULL,
			consumer_name VARCHAR(255) NULL,
			display_name VARCHAR(255) NULL,
			blocked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			blocked_by VARCHAR(255) NULL,
			request_phase VARCHAR(64) NULL,
			blocked_reason_json TEXT NULL,
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
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			last_applied_at TIMESTAMP NULL,
			last_error TEXT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS job_run_record (
			id BIGSERIAL PRIMARY KEY,
			job_name VARCHAR(255) NOT NULL,
			trigger_source VARCHAR(64) NOT NULL,
			trigger_id VARCHAR(255) NOT NULL,
			status VARCHAR(32) NOT NULL,
			idempotency_key VARCHAR(255) NULL,
			target_version VARCHAR(255) NULL,
			message TEXT NULL,
			error_text TEXT NULL,
			started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			finished_at TIMESTAMP NULL,
			duration_ms BIGINT NULL
		)`,
	}
}

func tableExistenceQuery(driver string) string {
	_ = driver
	return `
		SELECT COUNT(1)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = current_schema()
		  AND TABLE_NAME = ?`
}

func legacyUsersMigrationSQL(driver string) string {
	_ = driver
	return fmt.Sprintf(`
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
			COALESCE(u.deleted, FALSE),
			CASE WHEN COALESCE(u.deleted, FALSE) THEN CURRENT_TIMESTAMP ELSE NULL END,
			u.last_login_at
		FROM portal_users u
		%s`,
		upsertClause(driver, []string{"consumer_name"}, []string{
			assign(driver, "display_name"),
			assign(driver, "email"),
			assign(driver, "password_hash"),
			assign(driver, "status"),
			assign(driver, "source"),
			assign(driver, "user_level"),
			assign(driver, "is_deleted"),
			assign(driver, "deleted_at"),
			assign(driver, "last_login_at"),
		}),
	)
}

func legacyMembershipMigrationSQL(driver string) string {
	return fmt.Sprintf(`
		INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name)
		SELECT consumer_name, NULLIF(department_id, ''), NULLIF(parent_consumer_name, '')
		FROM portal_users
		%s`,
		upsertClause(driver, []string{"consumer_name"}, []string{
			assign(driver, "department_id"),
			assign(driver, "parent_consumer_name"),
		}),
	)
}

func legacyDepartmentMigrationSQL(driver string) string {
	_ = driver
	return fmt.Sprintf(`
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
			CASE WHEN COALESCE(d.deleted, FALSE) THEN 'deleted' ELSE 'active' END
		FROM portal_departments d
		%s`,
		upsertClause(driver, []string{"department_id"}, []string{
			assign(driver, "name"),
			assign(driver, "parent_department_id"),
			assign(driver, "admin_consumer_name"),
			assign(driver, "path"),
			assign(driver, "level"),
			assign(driver, "sort_order"),
			assign(driver, "status"),
			"updated_at = CURRENT_TIMESTAMP",
		}),
	)
}

func legacyAssetGrantMigrationSQL(driver string) string {
	return fmt.Sprintf(`
		INSERT INTO asset_grant (asset_type, asset_id, subject_type, subject_id)
		SELECT asset_type, asset_id, subject_type, subject_id
		FROM portal_asset_grant
		GROUP BY asset_type, asset_id, subject_type, subject_id
		%s`,
		upsertClause(driver, []string{"asset_type", "asset_id", "subject_type", "subject_id"}, []string{
			"updated_at = CURRENT_TIMESTAMP",
		}),
	)
}

func legacyAISensitiveDetectRuleMigrationSQL(driver string) string {
	_ = driver
	return fmt.Sprintf(`
		INSERT INTO portal_ai_sensitive_detect_rule (
			id, pattern, match_type, description, priority, enabled, created_at, updated_at
		)
		SELECT
			id,
			pattern,
			match_type,
			description,
			priority,
			COALESCE(is_enabled, TRUE),
			created_at,
			updated_at
		FROM ai_sensitive_detect_rule
		%s`,
		upsertClause(driver, []string{"id"}, []string{
			assign(driver, "pattern"),
			assign(driver, "match_type"),
			assign(driver, "description"),
			assign(driver, "priority"),
			assign(driver, "enabled"),
			assign(driver, "created_at"),
			assign(driver, "updated_at"),
		}),
	)
}

func legacyAISensitiveReplaceRuleMigrationSQL(driver string) string {
	_ = driver
	return fmt.Sprintf(`
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
			COALESCE(is_enabled, TRUE),
			created_at,
			updated_at
		FROM ai_sensitive_replace_rule
		%s`,
		upsertClause(driver, []string{"id"}, []string{
			assign(driver, "pattern"),
			assign(driver, "replace_type"),
			assign(driver, "replace_value"),
			assign(driver, "restore"),
			assign(driver, "description"),
			assign(driver, "priority"),
			assign(driver, "enabled"),
			assign(driver, "created_at"),
			assign(driver, "updated_at"),
		}),
	)
}

func legacyAISensitiveAuditMigrationSQL(driver string) string {
	return fmt.Sprintf(`
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
			%s
		FROM ai_sensitive_block_audit
		%s`,
		CastToText(driver, "cost_usd"),
		upsertClause(driver, []string{"id"}, []string{
			assign(driver, "request_id"),
			assign(driver, "route_name"),
			assign(driver, "consumer_name"),
			assign(driver, "display_name"),
			assign(driver, "blocked_at"),
			assign(driver, "blocked_by"),
			assign(driver, "request_phase"),
			assign(driver, "blocked_reason_json"),
			assign(driver, "match_type"),
			assign(driver, "matched_rule"),
			assign(driver, "matched_excerpt"),
			assign(driver, "provider_id"),
			assign(driver, "cost_usd"),
		}),
	)
}

func (c *SQLClient) migrateLegacyQuotaPoliciesByRows(ctx context.Context) error {
	rows, err := QueryContext(ctx, c.db, c.config.Driver, `
		SELECT consumer_name, limit_total, limit_5h, limit_daily, daily_reset_mode, daily_reset_time,
			limit_weekly, limit_monthly, cost_reset_at, updated_at
		FROM portal_ai_quota_user_policy
		ORDER BY consumer_name ASC, updated_at DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type policyAggregate struct {
		consumerName string
		limitTotal   int64
		limit5h      int64
		limitDaily   int64
		dailyMode    string
		dailyTime    string
		limitWeekly  int64
		limitMonthly int64
		costResetAt  sql.NullTime
		initialized  bool
	}

	items := map[string]*policyAggregate{}
	for rows.Next() {
		var (
			consumerName string
			limitTotal   sql.NullInt64
			limit5h      sql.NullInt64
			limitDaily   sql.NullInt64
			dailyMode    sql.NullString
			dailyTime    sql.NullString
			limitWeekly  sql.NullInt64
			limitMonthly sql.NullInt64
			costResetAt  sql.NullTime
			updatedAt    sql.NullTime
		)
		if err := rows.Scan(&consumerName, &limitTotal, &limit5h, &limitDaily, &dailyMode, &dailyTime, &limitWeekly, &limitMonthly, &costResetAt, &updatedAt); err != nil {
			return err
		}
		if strings.TrimSpace(consumerName) == "" {
			continue
		}
		item := items[consumerName]
		if item == nil {
			item = &policyAggregate{
				consumerName: consumerName,
				dailyMode:    "fixed",
				dailyTime:    "00:00",
			}
			items[consumerName] = item
		}
		if limitTotal.Valid && limitTotal.Int64 > item.limitTotal {
			item.limitTotal = limitTotal.Int64
		}
		if limit5h.Valid && limit5h.Int64 > item.limit5h {
			item.limit5h = limit5h.Int64
		}
		if limitDaily.Valid && limitDaily.Int64 > item.limitDaily {
			item.limitDaily = limitDaily.Int64
		}
		if limitWeekly.Valid && limitWeekly.Int64 > item.limitWeekly {
			item.limitWeekly = limitWeekly.Int64
		}
		if limitMonthly.Valid && limitMonthly.Int64 > item.limitMonthly {
			item.limitMonthly = limitMonthly.Int64
		}
		if !item.initialized {
			if strings.TrimSpace(dailyMode.String) != "" {
				item.dailyMode = strings.TrimSpace(dailyMode.String)
			}
			if strings.TrimSpace(dailyTime.String) != "" {
				item.dailyTime = strings.TrimSpace(dailyTime.String)
			}
			if costResetAt.Valid {
				item.costResetAt = costResetAt
			}
			item.initialized = true
		}
		if !item.costResetAt.Valid && costResetAt.Valid {
			item.costResetAt = costResetAt
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	statement := fmt.Sprintf(`
		INSERT INTO quota_policy_user (
			consumer_name, limit_total_micro_yuan, limit_5h_micro_yuan, limit_daily_micro_yuan,
			daily_reset_mode, daily_reset_time, limit_weekly_micro_yuan, limit_monthly_micro_yuan, cost_reset_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		%s`,
		upsertClause(c.config.Driver, []string{"consumer_name"}, []string{
			assign(c.config.Driver, "limit_total_micro_yuan"),
			assign(c.config.Driver, "limit_5h_micro_yuan"),
			assign(c.config.Driver, "limit_daily_micro_yuan"),
			assign(c.config.Driver, "daily_reset_mode"),
			assign(c.config.Driver, "daily_reset_time"),
			assign(c.config.Driver, "limit_weekly_micro_yuan"),
			assign(c.config.Driver, "limit_monthly_micro_yuan"),
			assign(c.config.Driver, "cost_reset_at"),
			"updated_at = CURRENT_TIMESTAMP",
		}),
	)
	for _, item := range items {
		if _, err := ExecContext(ctx, c.db, c.config.Driver, statement,
			item.consumerName,
			item.limitTotal,
			item.limit5h,
			item.limitDaily,
			item.dailyMode,
			item.dailyTime,
			item.limitWeekly,
			item.limitMonthly,
			nullTimeArg(item.costResetAt),
		); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLClient) migrateLegacyAISensitiveSystemConfigByRow(ctx context.Context) error {
	row := QueryRowContext(ctx, c.db, c.config.Driver, `
		SELECT system_deny_enabled, COALESCE(dictionary_text, ''), updated_by, updated_at
		FROM ai_sensitive_system_config
		ORDER BY updated_at DESC, id DESC
		LIMIT 1`)
	var (
		systemDenyEnabled sql.NullBool
		dictionaryText    sql.NullString
		updatedBy         sql.NullString
		updatedAt         sql.NullTime
	)
	if err := row.Scan(&systemDenyEnabled, &dictionaryText, &updatedBy, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	statement := fmt.Sprintf(`
		INSERT INTO portal_ai_sensitive_system_config (
			config_key, system_deny_enabled, dictionary_text, updated_by, updated_at
		)
		VALUES (?, ?, ?, ?, ?)
		%s`,
		upsertClause(c.config.Driver, []string{"config_key"}, []string{
			assign(c.config.Driver, "system_deny_enabled"),
			assign(c.config.Driver, "dictionary_text"),
			assign(c.config.Driver, "updated_by"),
			assign(c.config.Driver, "updated_at"),
		}),
	)
	_, err := ExecContext(ctx, c.db, c.config.Driver, statement,
		"default",
		systemDenyEnabled.Bool,
		dictionaryText.String,
		nullStringArg(updatedBy),
		nullTimeArg(updatedAt),
	)
	return err
}

func upsertClause(driver string, conflictColumns []string, assignments []string) string {
	_ = driver
	return fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET %s", strings.Join(conflictColumns, ", "), strings.Join(assignments, ", "))
}

func assign(driver, column string) string {
	_ = driver
	return fmt.Sprintf("%s = EXCLUDED.%s", column, column)
}

func nullTimeArg(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func nullStringArg(value sql.NullString) any {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	return value.String
}
