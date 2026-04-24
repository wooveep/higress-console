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

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
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
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`)).
		WithArgs("disabled", "demo").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT department_id, name, parent_department_id, path
		FROM org_department
		WHERE status = 'active'`)).
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "name", "parent_department_id", "path"}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, m.department_id,
			CASE WHEN d.admin_consumer_name = u.consumer_name THEN 1 ELSE 0 END, u.last_login_at
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE
		ORDER BY consumer_name ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"consumer_name", "display_name", "email", "status", "user_level", "source",
			"department_id", "case", "last_login_at",
		}).AddRow("demo", "Demo", "demo@example.com", "disabled", "normal", "console", nil, false, nil))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.UpdateAccountStatus(context.Background(), "demo", "disabled")
	require.NoError(t, err)
	require.Equal(t, "disabled", item.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateAccountHandlesRootDepartmentWithNullParent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1) FROM portal_user WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`)).
		WithArgs("demo").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(0))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT department_id, name, parent_department_id, admin_consumer_name
		FROM org_department
		WHERE status = 'active' AND department_id <> ?
		ORDER BY name ASC`)).
		WithArgs("root").
		WillReturnRows(sqlmock.NewRows([]string{
			"department_id", "name", "parent_department_id", "admin_consumer_name",
		}).AddRow("dept-root-level", "Platform", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT m.department_id, COUNT(1)
		FROM org_account_membership m
		INNER JOIN portal_user u ON u.consumer_name = m.consumer_name
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE AND m.department_id IS NOT NULL AND m.department_id <> '' AND m.department_id <> ?
		GROUP BY m.department_id`)).
		WithArgs("root").
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "count"}))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_user (
			consumer_name, display_name, email, status, user_level, source, password_hash
		) VALUES (?, ?, ?, ?, ?, 'console', ?)`)).
		WithArgs("demo", "Demo", "demo@example.com", "active", "normal", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO org_account_membership (consumer_name, department_id)
		VALUES (?, ?)
		ON CONFLICT (consumer_name) DO UPDATE SET department_id = EXCLUDED.department_id, updated_at = CURRENT_TIMESTAMP`)).
		WithArgs("demo", "dept-root-level").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT department_id, name, parent_department_id, path
		FROM org_department
		WHERE status = 'active' AND department_id <> ?`)).
		WithArgs("root").
		WillReturnRows(sqlmock.NewRows([]string{
			"department_id", "name", "parent_department_id", "path",
		}).AddRow("dept-root-level", "Platform", nil, "Platform"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, m.department_id,
			CASE WHEN d.admin_consumer_name = u.consumer_name THEN 1 ELSE 0 END, u.last_login_at
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE
		ORDER BY consumer_name ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"consumer_name", "display_name", "email", "status", "user_level", "source",
			"department_id", "case", "last_login_at",
		}).AddRow("demo", "Demo", "demo@example.com", "active", "normal", "console", "dept-root-level", false, nil))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.CreateAccount(context.Background(), AccountMutation{
		ConsumerName: "demo",
		DisplayName:  "Demo",
		Email:        "demo@example.com",
		UserLevel:    "normal",
		Status:       "active",
		DepartmentID: "dept-root-level",
		Password:     "secret",
	})
	require.NoError(t, err)
	require.Equal(t, "demo", item.ConsumerName)
	require.Equal(t, "dept-root-level", item.DepartmentID)
	require.Equal(t, "Platform", item.DepartmentName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateDepartmentWithExistingAdmin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, source, status
		FROM portal_user
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE
		LIMIT 1`)).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "source", "status"}).AddRow("alice", "console", "active"))
	mock.ExpectExec(regexp.QuoteMeta(`
			UPDATE org_department
			SET admin_consumer_name = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE admin_consumer_name = ? AND status = 'active'`)).
		WithArgs("alice").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO org_department (department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status)
		VALUES (?, ?, ?, ?, ?, ?, 0, 'active')`)).
		WithArgs(sqlmock.AnyArg(), "AI Team", nil, "alice", "AI Team", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO org_account_membership (consumer_name, department_id)
		VALUES (?, ?)
		ON CONFLICT (consumer_name) DO UPDATE SET department_id = EXCLUDED.department_id, updated_at = CURRENT_TIMESTAMP`)).
		WithArgs("alice", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT department_id, name, parent_department_id, admin_consumer_name
		FROM org_department
		WHERE status = 'active' AND department_id <> ?
		ORDER BY name ASC`)).
		WithArgs("root").
		WillReturnRows(sqlmock.NewRows([]string{
			"department_id", "name", "parent_department_id", "admin_consumer_name",
		}).AddRow("dept-new", "AI Team", nil, "alice"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT m.department_id, COUNT(1)
		FROM org_account_membership m
		INNER JOIN portal_user u ON u.consumer_name = m.consumer_name
		WHERE COALESCE(u.is_deleted, FALSE) = FALSE AND m.department_id IS NOT NULL AND m.department_id <> '' AND m.department_id <> ?
		GROUP BY m.department_id`)).
		WithArgs("root").
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "count"}).AddRow("dept-new", 1))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	_, err = svc.CreateDepartment(context.Background(), DepartmentMutation{
		Name:              "AI Team",
		AdminMode:         "existing",
		AdminConsumerName: "alice",
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRebindAccountSSOIdentity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT COUNT(1) FROM org_department
			WHERE admin_consumer_name = ? AND status = 'active'`)).
		WithArgs("sso-user").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, source, status
		FROM portal_user
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE
		LIMIT 1`)).
		WithArgs("sso-user").
		WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "source", "status"}).AddRow("sso-user", "sso", "pending"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, source, status
		FROM portal_user
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE
		LIMIT 1`)).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "source", "status"}).AddRow("alice", "console", "active"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT provider_key, issuer, subject
		FROM portal_user_sso_identity
		WHERE consumer_name = ?
		ORDER BY linked_at DESC
		LIMIT 1`)).
		WithArgs("sso-user").
		WillReturnRows(sqlmock.NewRows([]string{"provider_key", "issuer", "subject"}).AddRow("portal-oidc", "https://issuer.example.com", "sub-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM portal_user_sso_identity
		WHERE provider_key = ? AND consumer_name = ?`)).
		WithArgs("portal-oidc", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_user_sso_identity
		SET consumer_name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE provider_key = ? AND issuer = ? AND subject = ?`)).
		WithArgs("alice", "portal-oidc", "https://issuer.example.com", "sub-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_user
		SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`)).
		WithArgs("sso-user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	result, err := svc.RebindAccountSSOIdentity(context.Background(), "sso-user", "alice")
	require.NoError(t, err)
	require.Equal(t, "sso-user", result.SourceConsumerName)
	require.Equal(t, "alice", result.TargetConsumerName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRebindAccountSSOIdentityAllowsDisabledSSOAccount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT COUNT(1) FROM org_department
			WHERE admin_consumer_name = ? AND status = 'active'`)).
		WithArgs("sso-user").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, source, status
		FROM portal_user
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE
		LIMIT 1`)).
		WithArgs("sso-user").
		WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "source", "status"}).AddRow("sso-user", "sso", "disabled"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, source, status
		FROM portal_user
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE
		LIMIT 1`)).
		WithArgs("alice").
		WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "source", "status"}).AddRow("alice", "console", "active"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT provider_key, issuer, subject
		FROM portal_user_sso_identity
		WHERE consumer_name = ?
		ORDER BY linked_at DESC
		LIMIT 1`)).
		WithArgs("sso-user").
		WillReturnRows(sqlmock.NewRows([]string{"provider_key", "issuer", "subject"}).AddRow("portal-oidc", "https://issuer.example.com", "sub-1"))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM portal_user_sso_identity
		WHERE provider_key = ? AND consumer_name = ?`)).
		WithArgs("portal-oidc", "alice").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_user_sso_identity
		SET consumer_name = ?, updated_at = CURRENT_TIMESTAMP
		WHERE provider_key = ? AND issuer = ? AND subject = ?`)).
		WithArgs("alice", "portal-oidc", "https://issuer.example.com", "sub-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_user
		SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`)).
		WithArgs("sso-user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	result, err := svc.RebindAccountSSOIdentity(context.Background(), "sso-user", "alice")
	require.NoError(t, err)
	require.Equal(t, "sso-user", result.SourceConsumerName)
	require.Equal(t, "alice", result.TargetConsumerName)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAccountSoftDeletesPortalUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
			SELECT COUNT(1) FROM org_department
			WHERE admin_consumer_name = ? AND status = 'active'`)).
		WithArgs("demo").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_user SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE consumer_name = ? AND COALESCE(is_deleted, FALSE) = FALSE`)).
		WithArgs("demo").
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	require.NoError(t, svc.DeleteAccount(context.Background(), "demo"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHashPasswordUsesBcrypt(t *testing.T) {
	hash, err := hashPassword("8DC563ED")
	require.NoError(t, err)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(hash), []byte("8DC563ED")))
}

func expectSchema(mock sqlmock.Sqlmock) {
	expectSharedSchemaMigration(mock)
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_model_binding_price_version").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_detect_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_replace_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_system_config").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_sensitive_block_audit").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_quota_balance").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS portal_ai_quota_schedule_rule").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("CREATE TABLE IF NOT EXISTS job_run_record").WillReturnResult(sqlmock.NewResult(0, 0))
	expectSharedSchemaCheck(mock)
	expectLegacyTablesAbsent(mock)
}

func expectSharedSchemaMigration(mock sqlmock.Sqlmock) {
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
		"portal_sso_config",
		"portal_user_sso_identity",
	} {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS " + table).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	columnQuery := regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = current_schema()
		  AND TABLE_NAME = $1
		  AND COLUMN_NAME = $2`)
	for _, item := range [][2]string{
		{"portal_user", "user_level"},
		{"portal_user", "is_deleted"},
		{"portal_user", "deleted_at"},
		{"portal_model_asset", "request_kinds_json"},
		{"portal_model_asset", "model_type"},
		{"portal_model_asset", "input_modalities_json"},
		{"portal_model_asset", "output_modalities_json"},
		{"portal_model_asset", "feature_flags_json"},
		{"portal_model_binding", "limits_json"},
		{"org_department", "admin_consumer_name"},
	} {
		mock.ExpectQuery(columnQuery).
			WithArgs(item[0], item[1]).
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(1))
	}
	mock.ExpectExec("ALTER TABLE org_department ADD CONSTRAINT uk_org_department_admin_consumer").WillReturnResult(sqlmock.NewResult(0, 0))
	expectSharedSchemaCheck(mock)
}

func expectSharedSchemaCheck(mock sqlmock.Sqlmock) {
	query := regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = current_schema()
		  AND TABLE_NAME = $1`)
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
		"portal_sso_config",
		"portal_user_sso_identity",
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
		WHERE TABLE_SCHEMA = current_schema()
		  AND TABLE_NAME = $1`)
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
