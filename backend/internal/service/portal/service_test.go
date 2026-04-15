package portal

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestCreateInviteCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_invite_code (invite_code, status, expires_at)
		VALUES (?, ?, ?)`)).
		WithArgs(sqlmock.AnyArg(), "active", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT invite_code, status, expires_at, used_by_consumer, used_at, created_at
		FROM portal_invite_code
		WHERE invite_code = ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"invite_code", "status", "expires_at", "used_by_consumer", "used_at", "created_at",
		}).AddRow("ABCD1234", "active", time.Now(), nil, nil, time.Now()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
	item, err := svc.CreateInviteCode(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "active", item.Status)
	require.NotEmpty(t, item.InviteCode)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAccountStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_user
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, 0) = 0`)).
		WithArgs("disabled", "demo").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT department_id, name, parent_department_id, path
		FROM org_department
		WHERE status = 'active'`)).
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "name", "parent_department_id", "path"}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, m.department_id,
			m.parent_consumer_name, CASE WHEN d.admin_consumer_name = u.consumer_name THEN 1 ELSE 0 END, u.last_login_at
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id
		WHERE COALESCE(u.is_deleted, 0) = 0
		ORDER BY consumer_name ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"consumer_name", "display_name", "email", "status", "user_level", "source",
			"department_id", "parent_consumer_name", "case", "last_login_at",
		}).AddRow("demo", "Demo", "demo@example.com", "disabled", "normal", "console", nil, nil, false, nil))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
	item, err := svc.UpdateAccountStatus(context.Background(), "demo", "disabled")
	require.NoError(t, err)
	require.Equal(t, "disabled", item.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHashPasswordUsesBcrypt(t *testing.T) {
	hash, err := hashPassword("8DC563ED")
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("8DC563ED")))
}

func expectSchema(mock sqlmock.Sqlmock) {
	expectSharedSchema(mock)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_model_binding_price_version").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_detect_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_replace_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_system_config").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_block_audit").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_quota_balance").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_quota_schedule_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS job_run_record").WillReturnResult(sqlmock.NewResult(0, 0))
	expectSharedSchema(mock)
	expectLegacyTablesAbsent(mock)
}

func expectSharedSchema(mock sqlmock.Sqlmock) {
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
}

func expectLegacyTablesAbsent(mock sqlmock.Sqlmock) {
	query := regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = DATABASE()
		  AND TABLE_NAME = ?`)
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
