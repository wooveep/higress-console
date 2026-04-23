package portal

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestListAISensitiveAuditsParsesStructuredBlockedReason(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, request_phase,
			blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd
		FROM portal_ai_sensitive_block_audit
		WHERE 1 = 1 ORDER BY blocked_at DESC LIMIT 100`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "request_id", "route_name", "consumer_name", "display_name", "blocked_at", "blocked_by", "request_phase",
			"blocked_reason_json", "match_type", "matched_rule", "matched_excerpt", "provider_id", "cost_usd",
		}).AddRow(
			1, "", "route-a", "demo", "Demo", time.Now(), "ai-security-guard", "response",
			`{"blockedDetails":[{"type":"contentModeration","level":"high","suggestion":"block"}],"requestId":"req-1","guardCode":200}`,
			"contains", "rule-a", "敏感词", 12, "0.12",
		))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ListAISensitiveAudits(context.Background(), AISensitiveAuditQuery{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "req-1", items[0].RequestID)
	require.NotNil(t, items[0].GuardCode)
	require.Equal(t, 200, *items[0].GuardCode)
	require.Len(t, items[0].BlockedDetails, 1)
	require.Equal(t, "contentModeration", items[0].BlockedDetails[0].Type)
	require.Equal(t, "high", items[0].BlockedDetails[0].Level)
	require.Equal(t, "block", items[0].BlockedDetails[0].Suggestion)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAISensitiveAuditsIgnoresInvalidBlockedReasonJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, request_phase,
			blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd
		FROM portal_ai_sensitive_block_audit
		WHERE 1 = 1 ORDER BY blocked_at DESC LIMIT 100`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "request_id", "route_name", "consumer_name", "display_name", "blocked_at", "blocked_by", "request_phase",
			"blocked_reason_json", "match_type", "matched_rule", "matched_excerpt", "provider_id", "cost_usd",
		}).AddRow(
			2, "req-legacy", "route-b", "demo", "Demo", time.Now(), "ai-security-guard", "request",
			"{bad json", "contains", "rule-b", "历史数据", 0, "0.00",
		))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ListAISensitiveAudits(context.Background(), AISensitiveAuditQuery{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "req-legacy", items[0].RequestID)
	require.Nil(t, items[0].GuardCode)
	require.Empty(t, items[0].BlockedDetails)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAISensitiveReplaceRulesAcceptsBooleanColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		FROM portal_ai_sensitive_replace_rule
		ORDER BY priority DESC, id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pattern", "replace_type", "replace_value", "restore", "description", "priority", "enabled", "created_at", "updated_at",
		}).AddRow(1, "secret", "mask", "***", true, "demo", 10, true, time.Now(), time.Now()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ListAISensitiveReplaceRules(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].Restore)
	require.True(t, items[0].Enabled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAISensitiveSystemConfigAcceptsBooleanColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
		WillReturnRows(sqlmock.NewRows([]string{"system_deny_enabled", "dictionary_text", "updated_by", "updated_at"}).
			AddRow(true, "alpha\nbeta", "tester", time.Now()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.GetAISensitiveSystemConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, item)
	require.True(t, item.SystemDenyEnabled)
	require.Equal(t, "alpha\nbeta", item.DictionaryText)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAISensitiveSystemConfigFallsBackToDefaultDictionaryWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	defaultDictionary := installDefaultAISensitiveDictionaryFixture(t, "alpha\nbeta")

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
		WillReturnError(sql.ErrNoRows)

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.GetAISensitiveSystemConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, item)
	require.False(t, item.SystemDenyEnabled)
	require.Equal(t, defaultDictionary, item.DictionaryText)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAISensitiveSystemConfigFallsBackToDefaultDictionaryWhenStoredTextBlank(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	defaultDictionary := installDefaultAISensitiveDictionaryFixture(t, "alpha\nbeta")

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
		WillReturnRows(sqlmock.NewRows([]string{"system_deny_enabled", "dictionary_text", "updated_by", "updated_at"}).
			AddRow(true, "", "tester", time.Now()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.GetAISensitiveSystemConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, item)
	require.True(t, item.SystemDenyEnabled)
	require.Equal(t, defaultDictionary, item.DictionaryText)
	require.Equal(t, "tester", item.UpdatedBy)
	require.NoError(t, mock.ExpectationsWereMet())
}

func installDefaultAISensitiveDictionaryFixture(t *testing.T, content string) string {
	t.Helper()

	fixtureDir := t.TempDir()
	fixturePath := filepath.Join(fixtureDir, "sensitive_word_dict.txt")
	require.NoError(t, os.WriteFile(fixturePath, []byte(content), 0o644))

	previous := aiSensitiveDefaultDictionaryCandidates
	aiSensitiveDefaultDictionaryCandidates = []string{fixturePath}
	t.Cleanup(func() {
		aiSensitiveDefaultDictionaryCandidates = previous
	})

	return content
}

func TestGetAISensitiveRuntimeConfigFallsBackToDefaults(t *testing.T) {
	svc := New(&portaldbclient.FakeClient{}, k8sclient.NewMemoryClient())

	item, err := svc.GetAISensitiveRuntimeConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, item)
	require.True(t, item.DenyOpenai)
	require.Equal(t, []string{"$.messages[*].content"}, item.DenyJsonpath)
	require.Equal(t, 200, item.DenyCode)
	require.Equal(t, 2000, item.AuditSink.TimeoutMs)
}

func TestBuildAISensitiveProjectionPayloadOmitsDictionaryText(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, pattern, match_type, description, priority, enabled, created_at, updated_at
		FROM portal_ai_sensitive_detect_rule
		ORDER BY priority DESC, id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pattern", "match_type", "description", "priority", "enabled", "created_at", "updated_at",
		}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, pattern, replace_type, replace_value, restore, description, priority, enabled, created_at, updated_at
		FROM portal_ai_sensitive_replace_rule
		ORDER BY priority DESC, id ASC`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pattern", "replace_type", "replace_value", "restore", "description", "priority", "enabled", "created_at", "updated_at",
		}))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
		WillReturnRows(sqlmock.NewRows([]string{"system_deny_enabled", "dictionary_text", "updated_by", "updated_at"}).
			AddRow(true, "不应该进入 projection", "tester", time.Now()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db), k8sclient.NewMemoryClient())
	payload, err := svc.buildAISensitiveProjectionPayload(context.Background(), defaultAISensitiveRuntimeConfig())
	require.NoError(t, err)

	systemConfig := mapStringAny(payload["systemConfig"])
	require.Equal(t, true, systemConfig["systemDenyEnabled"])
	require.NotContains(t, systemConfig, "dictionaryText")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAISensitiveStatusIncludesBoundRoutes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
		WillReturnRows(sqlmock.NewRows([]string{"system_deny_enabled", "dictionary_text", "updated_by", "updated_at"}).
			AddRow(false, "alpha\nbeta", "tester", time.Now()))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(1) FROM portal_ai_sensitive_detect_rule`)).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(1))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(1) FROM portal_ai_sensitive_replace_rule`)).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(2))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(1) FROM portal_ai_sensitive_block_audit`)).
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(3))

	k8s := k8sclient.NewMemoryClient()
	_, err = k8s.UpsertResource(context.Background(), "ai-routes", "doubao", map[string]any{
		"name": "doubao",
	})
	require.NoError(t, err)
	_, err = k8s.UpsertResource(context.Background(), "route-plugin-instances:ai-route-doubao.internal", "ai-data-masking", map[string]any{
		"name":       "ai-data-masking",
		"pluginName": "ai-data-masking",
		"enabled":    true,
	})
	require.NoError(t, err)

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db), k8s)
	item, err := svc.GetAISensitiveStatus(context.Background())
	require.NoError(t, err)
	require.NotNil(t, item)
	require.Equal(t, 1, item.EnabledRouteCount)
	require.Equal(t, []string{"doubao"}, item.EnabledRoutes)
	require.NoError(t, mock.ExpectationsWereMet())
}
