package portal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/os/gtime"

	"github.com/wooveep/aigateway-console/backend/internal/dao"
	"github.com/wooveep/aigateway-console/backend/internal/model/do"
	"github.com/wooveep/aigateway-console/backend/internal/model/entity"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

type portalStore struct {
	db     *sql.DB
	driver string
}

func newPortalStore(db *sql.DB, drivers ...string) *portalStore {
	driver := "postgres"
	if len(drivers) > 0 && strings.TrimSpace(drivers[0]) != "" {
		driver = drivers[0]
	}
	return &portalStore{db: db, driver: driver}
}

func (s *portalStore) insertInviteCode(ctx context.Context, invite do.PortalInviteCode) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+dao.PortalInviteCode.Name+` (invite_code, status, expires_at)
		VALUES (?, ?, ?)`,
		invite.InviteCode,
		invite.Status,
		invite.ExpiresAt,
	)
	return err
}

func (s *portalStore) listInviteCodes(ctx context.Context) ([]entity.PortalInviteCode, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT invite_code, status, expires_at, used_by_consumer, used_at, created_at
		FROM `+dao.PortalInviteCode.Name+`
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalInviteCode, 0)
	for rows.Next() {
		item, err := scanInviteCodeEntity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) getInviteCode(ctx context.Context, inviteCode string) (*entity.PortalInviteCode, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT invite_code, status, expires_at, used_by_consumer, used_at, created_at
		FROM `+dao.PortalInviteCode.Name+`
		WHERE invite_code = ?`, inviteCode)
	item, err := scanInviteCodeEntity(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *portalStore) updateInviteCodeStatus(ctx context.Context, inviteCode, status string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE `+dao.PortalInviteCode.Name+` SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE invite_code = ?`, status, inviteCode)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *portalStore) listAssetGrants(ctx context.Context, assetType, assetID string) ([]entity.AssetGrant, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT asset_type, asset_id, subject_type, subject_id
		FROM `+dao.AssetGrant.Name+`
		WHERE asset_type = ? AND asset_id = ?
		ORDER BY subject_type, subject_id`, assetType, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.AssetGrant, 0)
	for rows.Next() {
		var item entity.AssetGrant
		if err := rows.Scan(&item.AssetType, &item.AssetId, &item.SubjectType, &item.SubjectId); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) replaceAssetGrants(ctx context.Context, assetType, assetID string, grants []do.AssetGrant) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM `+dao.AssetGrant.Name+` WHERE asset_type = ? AND asset_id = ?`, assetType, assetID); err != nil {
		return err
	}
	for _, item := range grants {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO `+dao.AssetGrant.Name+` (asset_type, asset_id, subject_type, subject_id)
			VALUES (?, ?, ?, ?)`,
			item.AssetType, item.AssetId, item.SubjectType, item.SubjectId,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *portalStore) listAIQuotaScheduleCounts(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT route_name, COUNT(1)
		FROM `+dao.PortalAIQuotaScheduleRule.Name+`
		GROUP BY route_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(map[string]int)
	for rows.Next() {
		var (
			routeName string
			count     int
		)
		if err := rows.Scan(&routeName, &count); err != nil {
			return nil, err
		}
		items[routeName] = count
	}
	return items, rows.Err()
}

func (s *portalStore) listAIQuotaBalances(ctx context.Context, routeName string) ([]entity.PortalAIQuotaBalance, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT consumer_name, quota
		FROM `+dao.PortalAIQuotaBalance.Name+`
		WHERE route_name = ?`, routeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalAIQuotaBalance, 0)
	for rows.Next() {
		var item entity.PortalAIQuotaBalance
		item.RouteName = routeName
		if err := rows.Scan(&item.ConsumerName, &item.Quota); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) listPortalBillingBalances(ctx context.Context, consumerNames []string) (map[string]int64, error) {
	normalized := make([]string, 0, len(consumerNames))
	seen := map[string]struct{}{}
	for _, consumerName := range consumerNames {
		trimmed := strings.TrimSpace(consumerName)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return map[string]int64{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(normalized)), ",")
	args := make([]any, 0, len(normalized))
	for _, item := range normalized {
		args = append(args, item)
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT consumer_name, available_micro_yuan
		FROM billing_wallet
		WHERE consumer_name IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := map[string]int64{}
	for rows.Next() {
		var (
			consumerName string
			balance      int64
		)
		if err := rows.Scan(&consumerName, &balance); err != nil {
			return nil, err
		}
		items[consumerName] = balance
	}
	return items, rows.Err()
}

func (s *portalStore) listActivePortalUsers(ctx context.Context) ([]entity.PortalUser, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT consumer_name
		FROM `+dao.PortalUser.Name+`
		WHERE COALESCE(is_deleted, FALSE) = FALSE
		ORDER BY consumer_name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalUser, 0)
	for rows.Next() {
		var item entity.PortalUser
		if err := rows.Scan(&item.ConsumerName); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) saveAIQuotaBalance(ctx context.Context, balance do.PortalAIQuotaBalance) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+dao.PortalAIQuotaBalance.Name+` (route_name, consumer_name, quota)
		VALUES (?, ?, ?)
		`+portaldbclient.UpsertClause(s.driver, []string{"route_name", "consumer_name"},
		portaldbclient.AssignValue(s.driver, "quota"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		balance.RouteName,
		balance.ConsumerName,
		balance.Quota,
	)
	return err
}

func (s *portalStore) refreshPortalBillingBalance(
	ctx context.Context,
	consumerName string,
	balanceMicroYuan int64,
	sourceHint string,
) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	current, err := queryCurrentPortalBillingBalance(ctx, tx, s.driver, consumerName)
	if err != nil {
		return 0, err
	}
	if err := upsertPortalBillingWallet(ctx, tx, s.driver, consumerName, balanceMicroYuan); err != nil {
		return 0, err
	}
	if err := insertPortalBillingAdjustTransaction(ctx, tx, s.driver, consumerName, balanceMicroYuan-current, "console_ai_quota_refresh", sourceHint); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return balanceMicroYuan, nil
}

func (s *portalStore) deltaPortalBillingBalance(
	ctx context.Context,
	consumerName string,
	deltaMicroYuan int64,
	sourceHint string,
) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	current, err := queryCurrentPortalBillingBalance(ctx, tx, s.driver, consumerName)
	if err != nil {
		return 0, err
	}
	next := current + deltaMicroYuan
	if err := upsertPortalBillingWallet(ctx, tx, s.driver, consumerName, next); err != nil {
		return 0, err
	}
	if err := insertPortalBillingAdjustTransaction(ctx, tx, s.driver, consumerName, deltaMicroYuan, "console_ai_quota_delta", sourceHint); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return next, nil
}

func (s *portalStore) getAIQuotaBalance(ctx context.Context, routeName, consumerName string) (int64, error) {
	var quota int64
	err := s.db.QueryRowContext(ctx, `
		SELECT quota
		FROM `+dao.PortalAIQuotaBalance.Name+`
		WHERE route_name = ? AND consumer_name = ?`,
		routeName,
		consumerName,
	).Scan(&quota)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return quota, nil
}

func (s *portalStore) getPortalBillingBalance(ctx context.Context, consumerName string) (int64, error) {
	var balance int64
	err := s.db.QueryRowContext(ctx, `
		SELECT available_micro_yuan
		FROM billing_wallet
		WHERE consumer_name = ?
		LIMIT 1`, strings.TrimSpace(consumerName)).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return balance, nil
}

func (s *portalStore) getAIQuotaUserPolicy(ctx context.Context, consumerName string) (*entity.QuotaPolicyUser, error) {
	var (
		item        entity.QuotaPolicyUser
		costResetAt sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT limit_total_micro_yuan, limit_5h_micro_yuan, limit_daily_micro_yuan, daily_reset_mode, daily_reset_time,
			limit_weekly_micro_yuan, limit_monthly_micro_yuan, cost_reset_at
		FROM `+dao.QuotaPolicyUser.Name+`
		WHERE consumer_name = ?`,
		consumerName,
	).Scan(
		&item.LimitTotalMicroYuan,
		&item.Limit5hMicroYuan,
		&item.LimitDailyMicroYuan,
		&item.DailyResetMode,
		&item.DailyResetTime,
		&item.LimitWeeklyMicroYuan,
		&item.LimitMonthlyMicroYuan,
		&costResetAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	item.ConsumerName = consumerName
	item.CostResetAt = gtimePointerFromNullTime(costResetAt)
	return &item, nil
}

func (s *portalStore) saveAIQuotaUserPolicy(ctx context.Context, policy do.QuotaPolicyUser) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+dao.QuotaPolicyUser.Name+` (
			consumer_name, limit_total_micro_yuan, limit_5h_micro_yuan, limit_daily_micro_yuan,
			daily_reset_mode, daily_reset_time, limit_weekly_micro_yuan, limit_monthly_micro_yuan, cost_reset_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`+portaldbclient.UpsertClause(s.driver, []string{"consumer_name"},
		portaldbclient.AssignValue(s.driver, "limit_total_micro_yuan"),
		portaldbclient.AssignValue(s.driver, "limit_5h_micro_yuan"),
		portaldbclient.AssignValue(s.driver, "limit_daily_micro_yuan"),
		portaldbclient.AssignValue(s.driver, "daily_reset_mode"),
		portaldbclient.AssignValue(s.driver, "daily_reset_time"),
		portaldbclient.AssignValue(s.driver, "limit_weekly_micro_yuan"),
		portaldbclient.AssignValue(s.driver, "limit_monthly_micro_yuan"),
		portaldbclient.AssignValue(s.driver, "cost_reset_at"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		policy.ConsumerName,
		policy.LimitTotalMicroYuan,
		policy.Limit5hMicroYuan,
		policy.LimitDailyMicroYuan,
		policy.DailyResetMode,
		policy.DailyResetTime,
		policy.LimitWeeklyMicroYuan,
		policy.LimitMonthlyMicroYuan,
		policy.CostResetAt,
	)
	return err
}

func (s *portalStore) listAIQuotaScheduleRules(ctx context.Context, routeName, consumerName string) ([]entity.PortalAIQuotaScheduleRule, error) {
	statement := `
		SELECT id, consumer_name, action, cron, value, enabled, created_at, updated_at, last_applied_at, last_error
		FROM ` + dao.PortalAIQuotaScheduleRule.Name + `
		WHERE route_name = ?`
	args := []any{routeName}
	if strings.TrimSpace(consumerName) != "" {
		statement += ` AND consumer_name = ?`
		args = append(args, consumerName)
	}
	statement += ` ORDER BY consumer_name ASC, created_at DESC`

	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalAIQuotaScheduleRule, 0)
	for rows.Next() {
		var (
			item          entity.PortalAIQuotaScheduleRule
			enabled       bool
			createdAt     sql.NullTime
			updatedAt     sql.NullTime
			lastAppliedAt sql.NullTime
			lastError     sql.NullString
		)
		if err := rows.Scan(
			&item.Id,
			&item.ConsumerName,
			&item.Action,
			&item.Cron,
			&item.Value,
			&enabled,
			&createdAt,
			&updatedAt,
			&lastAppliedAt,
			&lastError,
		); err != nil {
			return nil, err
		}
		item.RouteName = routeName
		if enabled {
			item.Enabled = 1
		}
		item.CreatedAt = gtimePointerFromNullTime(createdAt)
		item.UpdatedAt = gtimePointerFromNullTime(updatedAt)
		item.LastAppliedAt = gtimePointerFromNullTime(lastAppliedAt)
		item.LastError = lastError.String
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) saveAIQuotaScheduleRule(ctx context.Context, rule do.PortalAIQuotaScheduleRule) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+dao.PortalAIQuotaScheduleRule.Name+` (
			id, route_name, consumer_name, action, cron, value, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		`+portaldbclient.UpsertClause(s.driver, []string{"id"},
		portaldbclient.AssignValue(s.driver, "consumer_name"),
		portaldbclient.AssignValue(s.driver, "action"),
		portaldbclient.AssignValue(s.driver, "cron"),
		portaldbclient.AssignValue(s.driver, "value"),
		portaldbclient.AssignValue(s.driver, "enabled"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		rule.Id, rule.RouteName, rule.ConsumerName, rule.Action, rule.Cron, rule.Value, rule.Enabled,
	)
	return err
}

func (s *portalStore) deleteAIQuotaScheduleRule(ctx context.Context, routeName, ruleID string) (bool, error) {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM `+dao.PortalAIQuotaScheduleRule.Name+`
		WHERE route_name = ? AND id = ?`,
		routeName, ruleID,
	)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *portalStore) listAISensitiveDetectRules(ctx context.Context) ([]entity.PortalAISensitiveDetectRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pattern, match_type, description, priority, enabled, created_at, updated_at
		FROM `+dao.PortalAISensitiveDetectRule.Name+`
		ORDER BY priority DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalAISensitiveDetectRule, 0)
	for rows.Next() {
		item, err := scanAISensitiveDetectRuleEntity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) getAISensitiveDetectRule(ctx context.Context, id int64) (*entity.PortalAISensitiveDetectRule, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, pattern, match_type, description, priority, enabled, created_at, updated_at
		FROM `+dao.PortalAISensitiveDetectRule.Name+`
		WHERE id = ?`, id)
	item, err := scanAISensitiveDetectRuleEntity(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *portalStore) saveAISensitiveDetectRule(ctx context.Context, rule do.PortalAISensitiveDetectRule, id int64) (int64, error) {
	if id > 0 {
		_, err := s.db.ExecContext(ctx, `
			UPDATE `+dao.PortalAISensitiveDetectRule.Name+`
			SET pattern = ?, match_type = ?, description = ?, priority = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			rule.Pattern, rule.MatchType, rule.Description, rule.Priority, rule.Enabled, id,
		)
		return id, err
	}
	return portaldbclient.InsertReturningID(ctx, s.db, s.driver, `
		INSERT INTO `+dao.PortalAISensitiveDetectRule.Name+` (pattern, match_type, description, priority, enabled)
		VALUES (?, ?, ?, ?, ?)`,
		rule.Pattern, rule.MatchType, rule.Description, rule.Priority, rule.Enabled,
	)
}

func (s *portalStore) deleteAISensitiveDetectRule(ctx context.Context, id int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM `+dao.PortalAISensitiveDetectRule.Name+` WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *portalStore) listAISensitiveReplaceRules(ctx context.Context) ([]entity.PortalAISensitiveReplaceRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		FROM `+dao.PortalAISensitiveReplaceRule.Name+`
		ORDER BY priority DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalAISensitiveReplaceRule, 0)
	for rows.Next() {
		item, err := scanAISensitiveReplaceRuleEntity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) getAISensitiveReplaceRule(ctx context.Context, id int64) (*entity.PortalAISensitiveReplaceRule, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		FROM `+dao.PortalAISensitiveReplaceRule.Name+`
		WHERE id = ?`, id)
	item, err := scanAISensitiveReplaceRuleEntity(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *portalStore) saveAISensitiveReplaceRule(ctx context.Context, rule do.PortalAISensitiveReplaceRule, id int64) (int64, error) {
	if id > 0 {
		_, err := s.db.ExecContext(ctx, `
			UPDATE `+dao.PortalAISensitiveReplaceRule.Name+`
			SET pattern = ?, replace_type = ?, replace_value = ?, restore = ?, description = ?, priority = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?`,
			rule.Pattern, rule.ReplaceType, rule.ReplaceValue, rule.Restore, rule.Description, rule.Priority, rule.Enabled, id,
		)
		return id, err
	}
	return portaldbclient.InsertReturningID(ctx, s.db, s.driver, `
		INSERT INTO `+dao.PortalAISensitiveReplaceRule.Name+` (
			pattern, replace_type, replace_value, restore, description, priority, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rule.Pattern, rule.ReplaceType, rule.ReplaceValue, rule.Restore, rule.Description, rule.Priority, rule.Enabled,
	)
}

func (s *portalStore) deleteAISensitiveReplaceRule(ctx context.Context, id int64) (bool, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM `+dao.PortalAISensitiveReplaceRule.Name+` WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, _ := result.RowsAffected()
	return affected > 0, nil
}

func (s *portalStore) listAISensitiveAudits(ctx context.Context, query AISensitiveAuditQuery) ([]entity.PortalAISensitiveBlockAudit, error) {
	statement := `
		SELECT id, request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, request_phase,
			blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd
		FROM ` + dao.PortalAISensitiveBlockAudit.Name + `
		WHERE 1 = 1`
	args := make([]any, 0)
	if strings.TrimSpace(query.ConsumerName) != "" {
		statement += ` AND consumer_name = ?`
		args = append(args, strings.TrimSpace(query.ConsumerName))
	}
	if strings.TrimSpace(query.DisplayName) != "" {
		statement += ` AND display_name = ?`
		args = append(args, strings.TrimSpace(query.DisplayName))
	}
	if strings.TrimSpace(query.RouteName) != "" {
		statement += ` AND route_name = ?`
		args = append(args, strings.TrimSpace(query.RouteName))
	}
	if strings.TrimSpace(query.MatchType) != "" {
		statement += ` AND match_type = ?`
		args = append(args, strings.TrimSpace(query.MatchType))
	}
	if strings.TrimSpace(query.StartTime) != "" {
		statement += ` AND blocked_at >= ?`
		args = append(args, strings.TrimSpace(query.StartTime))
	}
	if strings.TrimSpace(query.EndTime) != "" {
		statement += ` AND blocked_at <= ?`
		args = append(args, strings.TrimSpace(query.EndTime))
	}
	statement += ` ORDER BY blocked_at DESC`
	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	statement += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]entity.PortalAISensitiveBlockAudit, 0)
	for rows.Next() {
		item, err := scanAISensitiveAuditEntity(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *portalStore) getAISensitiveSystemConfig(ctx context.Context) (*entity.PortalAISensitiveSystemConfig, error) {
	var (
		item      entity.PortalAISensitiveSystemConfig
		enabled   bool
		updatedBy sql.NullString
		updatedAt sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM `+dao.PortalAISensitiveSystemConfig.Name+`
		WHERE config_key = 'default'`,
	).Scan(&enabled, &item.DictionaryText, &updatedBy, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	item.ConfigKey = "default"
	if enabled {
		item.SystemDenyEnabled = 1
	}
	item.UpdatedBy = updatedBy.String
	item.UpdatedAt = gtimePointerFromNullTime(updatedAt)
	return &item, nil
}

func (s *portalStore) saveAISensitiveSystemConfig(ctx context.Context, config do.PortalAISensitiveSystemConfig) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO `+dao.PortalAISensitiveSystemConfig.Name+` (config_key, system_deny_enabled, dictionary_text, updated_by)
		VALUES ('default', ?, ?, ?)
		`+portaldbclient.UpsertClause(s.driver, []string{"config_key"},
		portaldbclient.AssignValue(s.driver, "system_deny_enabled"),
		portaldbclient.AssignValue(s.driver, "dictionary_text"),
		portaldbclient.AssignValue(s.driver, "updated_by"),
		`updated_at = CURRENT_TIMESTAMP`)+``,
		config.SystemDenyEnabled,
		config.DictionaryText,
		config.UpdatedBy,
	)
	return err
}

func (s *portalStore) countTable(ctx context.Context, table string) int {
	var count int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM `+table).Scan(&count)
	return count
}

func scanInviteCodeEntity(scanner interface{ Scan(...any) error }) (entity.PortalInviteCode, error) {
	var (
		item           entity.PortalInviteCode
		expiresAt      sql.NullTime
		usedAt         sql.NullTime
		createdAt      sql.NullTime
		usedByConsumer sql.NullString
	)
	err := scanner.Scan(
		&item.InviteCode,
		&item.Status,
		&expiresAt,
		&usedByConsumer,
		&usedAt,
		&createdAt,
	)
	if err != nil {
		return entity.PortalInviteCode{}, err
	}
	item.ExpiresAt = gtimePointerFromNullTime(expiresAt)
	item.UsedByConsumer = usedByConsumer.String
	item.UsedAt = gtimePointerFromNullTime(usedAt)
	item.CreatedAt = gtimePointerFromNullTime(createdAt)
	return item, nil
}

func scanAISensitiveDetectRuleEntity(scanner interface{ Scan(...any) error }) (entity.PortalAISensitiveDetectRule, error) {
	var (
		item      entity.PortalAISensitiveDetectRule
		enabled   bool
		desc      sql.NullString
		createdAt sql.NullTime
		updatedAt sql.NullTime
	)
	if err := scanner.Scan(
		&item.Id,
		&item.Pattern,
		&item.MatchType,
		&desc,
		&item.Priority,
		&enabled,
		&createdAt,
		&updatedAt,
	); err != nil {
		return entity.PortalAISensitiveDetectRule{}, err
	}
	if enabled {
		item.Enabled = 1
	}
	item.Description = desc.String
	item.CreatedAt = gtimePointerFromNullTime(createdAt)
	item.UpdatedAt = gtimePointerFromNullTime(updatedAt)
	return item, nil
}

func scanAISensitiveReplaceRuleEntity(scanner interface{ Scan(...any) error }) (entity.PortalAISensitiveReplaceRule, error) {
	var (
		item         entity.PortalAISensitiveReplaceRule
		restore      bool
		enabled      bool
		replaceValue sql.NullString
		desc         sql.NullString
		createdAt    sql.NullTime
		updatedAt    sql.NullTime
	)
	if err := scanner.Scan(
		&item.Id,
		&item.Pattern,
		&item.ReplaceType,
		&replaceValue,
		&restore,
		&desc,
		&item.Priority,
		&enabled,
		&createdAt,
		&updatedAt,
	); err != nil {
		return entity.PortalAISensitiveReplaceRule{}, err
	}
	if restore {
		item.Restore = 1
	}
	if enabled {
		item.Enabled = 1
	}
	item.ReplaceValue = replaceValue.String
	item.Description = desc.String
	item.CreatedAt = gtimePointerFromNullTime(createdAt)
	item.UpdatedAt = gtimePointerFromNullTime(updatedAt)
	return item, nil
}

func scanAISensitiveAuditEntity(scanner interface{ Scan(...any) error }) (entity.PortalAISensitiveBlockAudit, error) {
	var (
		item              entity.PortalAISensitiveBlockAudit
		requestID         sql.NullString
		routeName         sql.NullString
		consumerName      sql.NullString
		displayName       sql.NullString
		blockedAt         sql.NullTime
		blockedBy         sql.NullString
		requestPhase      sql.NullString
		blockedReasonJSON sql.NullString
		matchType         sql.NullString
		matchedRule       sql.NullString
		matchedExcerpt    sql.NullString
		providerID        sql.NullInt64
		costUSD           sql.NullString
	)
	if err := scanner.Scan(
		&item.Id,
		&requestID,
		&routeName,
		&consumerName,
		&displayName,
		&blockedAt,
		&blockedBy,
		&requestPhase,
		&blockedReasonJSON,
		&matchType,
		&matchedRule,
		&matchedExcerpt,
		&providerID,
		&costUSD,
	); err != nil {
		return entity.PortalAISensitiveBlockAudit{}, err
	}
	item.RequestId = requestID.String
	item.RouteName = routeName.String
	item.ConsumerName = consumerName.String
	item.DisplayName = displayName.String
	item.BlockedAt = gtimePointerFromNullTime(blockedAt)
	item.BlockedBy = blockedBy.String
	item.RequestPhase = requestPhase.String
	item.BlockedReasonJson = blockedReasonJSON.String
	item.MatchType = matchType.String
	item.MatchedRule = matchedRule.String
	item.MatchedExcerpt = matchedExcerpt.String
	if providerID.Valid {
		item.ProviderId = providerID.Int64
	}
	item.CostUsd = costUSD.String
	return item, nil
}

func gtimePointerFromNullTime(value sql.NullTime) *gtime.Time {
	if !value.Valid {
		return nil
	}
	return gtime.NewFromTime(value.Time)
}

func queryCurrentPortalBillingBalance(ctx context.Context, tx *sql.Tx, driver string, consumerName string) (int64, error) {
	var balance int64
	err := tx.QueryRowContext(ctx, `
		SELECT available_micro_yuan
		FROM billing_wallet
		WHERE consumer_name = ?
		LIMIT 1`, strings.TrimSpace(consumerName)).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return balance, nil
}

func upsertPortalBillingWallet(ctx context.Context, tx *sql.Tx, driver string, consumerName string, balanceMicroYuan int64) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO billing_wallet (consumer_name, currency, available_micro_yuan, version)
		VALUES (?, 'CNY', ?, 1)
		`+portaldbclient.UpsertClause(driver, []string{"consumer_name"},
		portaldbclient.AssignValue(driver, "available_micro_yuan"),
		portaldbclient.UpsertAdd(driver, "billing_wallet", "version"))+``,
		strings.TrimSpace(consumerName),
		balanceMicroYuan,
	)
	return err
}

func insertPortalBillingAdjustTransaction(
	ctx context.Context,
	tx *sql.Tx,
	driver string,
	consumerName string,
	deltaMicroYuan int64,
	sourceType string,
	sourceHint string,
) error {
	if deltaMicroYuan == 0 {
		return nil
	}
	sourceID := buildPortalBillingSourceID(sourceHint, consumerName, deltaMicroYuan)
	currentTimeExpr := portaldbclient.UTCCurrentTimestamp(driver)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO billing_transaction (
			tx_id, consumer_name, tx_type, amount_micro_yuan, currency, source_type, source_id, occurred_at, created_at
		)
		VALUES (?, ?, 'adjust', ?, 'CNY', ?, ?, `+currentTimeExpr+`, `+currentTimeExpr+`)`,
		buildPortalBillingTransactionID(sourceType, sourceID),
		strings.TrimSpace(consumerName),
		deltaMicroYuan,
		sourceType,
		sourceID,
	)
	return err
}

func buildPortalBillingSourceID(sourceHint, consumerName string, amount int64) string {
	return strings.Join([]string{
		firstNonEmpty(strings.TrimSpace(sourceHint), "ai-quota"),
		strings.TrimSpace(consumerName),
		fmt.Sprint(amount),
		fmt.Sprint(gtime.Now().UnixNano()),
	}, ":")
}

func buildPortalBillingTransactionID(sourceType, sourceID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(sourceType) + ":" + strings.TrimSpace(sourceID)))
	return "a" + hex.EncodeToString(sum[:16])
}
