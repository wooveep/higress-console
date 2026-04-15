package jobs

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

type stubPortal struct {
	consumers []portalsvc.ConsumerRecord
	accounts  []portalsvc.OrgAccountRecord
}

func (s stubPortal) ListConsumers(ctx context.Context) ([]portalsvc.ConsumerRecord, error) {
	return append([]portalsvc.ConsumerRecord{}, s.consumers...), nil
}

func (s stubPortal) ListAccounts(ctx context.Context) ([]portalsvc.OrgAccountRecord, error) {
	return append([]portalsvc.OrgAccountRecord{}, s.accounts...), nil
}

func TestTriggerPortalConsumerProjectionAndSkipDuplicateSnapshot(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectJobSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE job_name = ? AND idempotency_key = ? AND status IN (?, ?)
		ORDER BY id DESC
		LIMIT 1`)).
		WithArgs("portal-consumer-projection", sqlmock.AnyArg(), RunStatusSuccess, RunStatusSkipped).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO job_run_record (
			job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message, started_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)).
		WithArgs(
			"portal-consumer-projection",
			"manual",
			"manual-1",
			RunStatusRunning,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"job started",
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE job_run_record
		SET status = ?, message = ?, error_text = ?, finished_at = ?, duration_ms = ?
		WHERE id = ?`)).
		WithArgs(RunStatusSuccess, sqlmock.AnyArg(), nil, sqlmock.AnyArg(), sqlmock.AnyArg(), int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE id = ?`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}).AddRow(
			1, "portal-consumer-projection", "manual", "manual-1", RunStatusSuccess, "same", "same", "projected 1 portal consumers, cleaned 0 stale resources",
			nil, time.Now(), time.Now(), int64(1),
		))

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE job_name = ? AND idempotency_key = ? AND status IN (?, ?)
		ORDER BY id DESC
		LIMIT 1`)).
		WithArgs("portal-consumer-projection", sqlmock.AnyArg(), RunStatusSuccess, RunStatusSkipped).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}).AddRow(
			1, "portal-consumer-projection", "manual", "manual-1", RunStatusSuccess, "same", "same", "projected 1 portal consumers, cleaned 0 stale resources",
			nil, time.Now(), time.Now(), int64(1),
		))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO job_run_record (
			job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message, error_text, started_at, finished_at, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)).
		WithArgs(
			"portal-consumer-projection",
			"manual",
			"manual-2",
			RunStatusSkipped,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			"snapshot unchanged",
			nil,
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			int64(0),
		).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, job_name, trigger_source, trigger_id, status, idempotency_key, target_version, message,
			error_text, started_at, finished_at, duration_ms
		FROM job_run_record
		WHERE id = ?`)).
		WithArgs(int64(2)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "job_name", "trigger_source", "trigger_id", "status", "idempotency_key", "target_version", "message",
			"error_text", "started_at", "finished_at", "duration_ms",
		}).AddRow(
			2, "portal-consumer-projection", "manual", "manual-2", RunStatusSkipped, "same", "same", "snapshot unchanged",
			nil, time.Now(), time.Now(), int64(0),
		))

	service := New(
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db),
		stubPortal{
			consumers: []portalsvc.ConsumerRecord{{
				Name:              "demo",
				PortalDisplayName: "Demo",
				PortalEmail:       "demo@example.com",
				PortalStatus:      "active",
				PortalUserLevel:   "pro",
			}},
		},
		nil,
		k8sclient.NewMemoryClient(),
	)

	first, err := service.Trigger(context.Background(), "portal-consumer-projection", TriggerInput{
		Source:    "manual",
		TriggerID: "manual-1",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusSuccess, first.Status)

	second, err := service.Trigger(context.Background(), "portal-consumer-projection", TriggerInput{
		Source:    "manual",
		TriggerID: "manual-2",
	})
	require.NoError(t, err)
	require.Equal(t, RunStatusSkipped, second.Status)

	item, err := service.k8s.GetResource(context.Background(), "consumers", "demo")
	require.NoError(t, err)
	require.Equal(t, "pro", item["userLevel"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func expectJobSchema(mock sqlmock.Sqlmock) {
	query := regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?`)
	for _, table := range []string{
		"portal_user",
		"portal_invite_code",
		"org_department",
		"org_account_membership",
		"asset_grant",
		"quota_policy_user",
		"portal_model_asset",
		"portal_model_binding",
		"portal_agent_catalog",
	} {
		mock.ExpectQuery(query).
			WithArgs(table).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(1))
	}
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_model_binding_price_version").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_detect_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_replace_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_system_config").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_block_audit").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_quota_balance").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_quota_schedule_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS job_run_record").WillReturnResult(sqlmock.NewResult(0, 0))
	for _, table := range []string{
		"portal_user",
		"portal_invite_code",
		"org_department",
		"org_account_membership",
		"asset_grant",
		"quota_policy_user",
		"portal_model_asset",
		"portal_model_binding",
		"portal_agent_catalog",
	} {
		mock.ExpectQuery(query).
			WithArgs(table).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(1))
	}
	for _, table := range []string{
		"portal_users",
		"portal_departments",
		"portal_asset_grant",
		"portal_ai_quota_user_policy",
		"ai_sensitive_detect_rule",
		"ai_sensitive_replace_rule",
		"ai_sensitive_system_config",
		"ai_sensitive_block_audit",
	} {
		mock.ExpectQuery(query).
			WithArgs(table).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(0))
	}
}
