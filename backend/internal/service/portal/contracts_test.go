package portal_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/stretchr/testify/require"

	portalcontroller "github.com/wooveep/aigateway-console/backend/internal/controller/portal"
	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

type contractFixture struct {
	Request struct {
		Method string          `json:"method"`
		Path   string          `json:"path"`
		Body   json.RawMessage `json:"body"`
	} `json:"request"`
	Response json.RawMessage `json:"response"`
}

func TestPortalContracts(t *testing.T) {
	t.Run("consumers", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, d.department_id, d.name, d.path
		FROM portal_user u
		LEFT JOIN org_account_membership m ON m.consumer_name = u.consumer_name
		LEFT JOIN org_department d ON d.department_id = m.department_id AND d.status = 'active'
		WHERE COALESCE(u.is_deleted, 0) = 0
		ORDER BY u.consumer_name ASC`)).
			WillReturnRows(sqlmock.NewRows([]string{
				"consumer_name", "display_name", "email", "status", "user_level", "source", "department_id", "name", "path",
			}).AddRow("alice", "Alice", "alice@example.com", "active", "pro", "console", "dept-a", "AI Team", "root / AI Team"))

		assertPortalFixture(t, db, "consumers/list-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("org-tree", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT department_id, name, parent_department_id, admin_consumer_name
		FROM org_department
		WHERE status = 'active' AND department_id <> ?
		ORDER BY name ASC`)).
			WithArgs("root").
			WillReturnRows(sqlmock.NewRows([]string{
				"department_id", "name", "parent_department_id", "admin_consumer_name",
			}).AddRow("dept-a", "AI Team", nil, "alice").AddRow("dept-b", "Agents", "dept-a", nil))
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT m.department_id, COUNT(1)
		FROM org_account_membership m
		INNER JOIN portal_user u ON u.consumer_name = m.consumer_name
		WHERE COALESCE(u.is_deleted, 0) = 0 AND m.department_id IS NOT NULL AND m.department_id <> '' AND m.department_id <> ?
		GROUP BY m.department_id`)).
			WithArgs("root").
			WillReturnRows(sqlmock.NewRows([]string{"department_id", "count"}).AddRow("dept-a", 2).AddRow("dept-b", 1))

		assertPortalFixture(t, db, "org/tree-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("invite-codes", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT invite_code, status, expires_at, used_by_consumer, used_at, created_at
		FROM portal_invite_code
		ORDER BY created_at DESC`)).
			WillReturnRows(sqlmock.NewRows([]string{
				"invite_code", "status", "expires_at", "used_by_consumer", "used_at", "created_at",
			}).AddRow("INVITE001", "active", now.Add(7*24*time.Hour), "", nil, now))

		assertPortalFixture(t, db, "portal-invite/list-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("model-assets", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at
		FROM portal_model_asset
		ORDER BY asset_id`)).
			WillReturnRows(sqlmock.NewRows([]string{
				"asset_id", "canonical_name", "display_name", "intro", "tags_json", "modalities_json", "features_json", "request_kinds_json", "created_at", "updated_at",
			}).AddRow("model-gpt-4o", "openai.gpt-4o", "GPT-4o", "flagship", `["premium"]`, `["text","image"]`, `["vision"]`, `["chat_completions"]`, now, now))
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		ORDER BY asset_id, binding_id`)).
			WillReturnRows(sqlmock.NewRows([]string{
				"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
				"published_at", "unpublished_at", "pricing_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
			}).AddRow("binding-gpt-4o", "model-gpt-4o", "gpt-4o", "openai", "gpt-4o", "openai/v1", "https://api.openai.com/v1", "published",
				now, nil, `{"currency":"USD","inputCostPerToken":0.00001}`, 6000, 120000, 128000, now, now))

		assertPortalFixture(t, db, "model-assets/list-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("agent-catalog", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, mcp_server_name,
			status, tool_count, transport_types_json, resource_summary, prompt_summary, published_at, unpublished_at, created_at, updated_at
		FROM portal_agent_catalog
		ORDER BY agent_id`)).
			WillReturnRows(sqlmock.NewRows([]string{
				"agent_id", "canonical_name", "display_name", "intro", "description", "icon_url", "tags_json", "mcp_server_name",
				"status", "tool_count", "transport_types_json", "resource_summary", "prompt_summary", "published_at", "unpublished_at", "created_at", "updated_at",
			}).AddRow("agent-researcher", "researcher", "Research Assistant", "research", "multi-step agent", "https://example.com/icon.png", `["research"]`, "research-mcp",
				"published", 3, `["http","sse"]`, "knowledge base", "3 tools exposed", now, nil, now, now))

		assertPortalFixture(t, db, "agent-catalog/list-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ai-quota", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name
		FROM portal_user
		WHERE COALESCE(is_deleted, 0) = 0
		ORDER BY consumer_name ASC`)).
			WillReturnRows(sqlmock.NewRows([]string{"consumer_name"}).AddRow("alice").AddRow("bob"))
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, quota
		FROM portal_ai_quota_balance
		WHERE route_name = ?`)).
			WithArgs("chat-route").
			WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "quota"}).AddRow("alice", 1200))
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT consumer_name, available_micro_yuan
		FROM billing_wallet
		WHERE consumer_name IN (?,?,?)`)).
			WithArgs("administrator", "alice", "bob").
			WillReturnRows(sqlmock.NewRows([]string{"consumer_name", "available_micro_yuan"}).AddRow("alice", 1200))

		assertPortalFixture(t, db, "ai-quota/consumers-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ai-sensitive", func(t *testing.T) {
		db, mock := newPortalContractMock(t)
		now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
		mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
			WillReturnRows(sqlmock.NewRows([]string{"system_deny_enabled", "dictionary_text", "updated_by", "updated_at"}).
				AddRow(1, "alpha\nbeta", "admin", now))
		mock.ExpectQuery("SELECT COUNT\\(1\\) FROM portal_ai_sensitive_detect_rule").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
		mock.ExpectQuery("SELECT COUNT\\(1\\) FROM portal_ai_sensitive_replace_rule").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
		mock.ExpectQuery("SELECT COUNT\\(1\\) FROM portal_ai_sensitive_block_audit").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		assertPortalFixture(t, db, "ai-sensitive/status-success.json")
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func newPortalContractMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	expectSchema(mock)
	return db, mock
}

func assertPortalFixture(t *testing.T, db *sql.DB, relativePath string) {
	t.Helper()

	fixturePath := filepath.Join("..", "..", "..", "test", "contracts", relativePath)
	raw, err := os.ReadFile(fixturePath)
	require.NoError(t, err)

	var fixture contractFixture
	require.NoError(t, json.Unmarshal(raw, &fixture))

	svc := portalsvc.New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
	serverName := fmt.Sprintf("portal-contract-%d", time.Now().UnixNano())
	server := ghttp.GetServer(serverName)
	server.SetPort(0)
	server.Group("/", func(group *ghttp.RouterGroup) {
		portalcontroller.Bind(group, svc)
	})
	require.NoError(t, server.Start())
	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	url := fmt.Sprintf("http://127.0.0.1:%d%s", server.GetListenedPort(), fixture.Request.Path)
	var bodyReader *bytes.Reader
	if len(fixture.Request.Body) > 0 {
		bodyReader = bytes.NewReader(fixture.Request.Body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(context.Background(), fixture.Request.Method, url, bodyReader)
	require.NoError(t, err)
	if len(fixture.Request.Body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	actual, err := json.Marshal(readJSONBody(t, resp))
	require.NoError(t, err)
	require.JSONEq(t, string(fixture.Response), string(actual))
}

func readJSONBody(t *testing.T, resp *http.Response) any {
	t.Helper()
	var payload any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	return payload
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
