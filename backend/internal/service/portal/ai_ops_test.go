package portal

import (
	"bytes"
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestDownloadOrgTemplateCreatesExpectedSheets(t *testing.T) {
	svc := New(&portaldbclient.FakeClient{})

	content, err := svc.DownloadOrgTemplate(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, content)

	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	require.NoError(t, err)
	defer workbook.Close()

	require.Contains(t, workbook.GetSheetList(), orgDepartmentSheet)
	require.Contains(t, workbook.GetSheetList(), orgAccountSheet)
}

func TestRefreshAIQuota(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_ai_quota_balance (route_name, consumer_name, quota)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE quota = VALUES(quota), updated_at = CURRENT_TIMESTAMP`)).
		WithArgs("route-a", "demo", int64(1200000)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
	item, err := svc.RefreshAIQuota(context.Background(), "route-a", "demo", 1200000)
	require.NoError(t, err)
	require.Equal(t, int64(1200000), item.Quota)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSaveAISensitiveSystemConfigAndStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_ai_sensitive_system_config (config_key, system_deny_enabled, dictionary_text, updated_by)
		VALUES ('default', ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			system_deny_enabled = VALUES(system_deny_enabled),
			dictionary_text = VALUES(dictionary_text),
			updated_by = VALUES(updated_by),
			updated_at = CURRENT_TIMESTAMP`)).
		WithArgs(1, "alpha\nbeta", "tester").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT system_deny_enabled, dictionary_text, updated_by, updated_at
		FROM portal_ai_sensitive_system_config
		WHERE config_key = 'default'`)).
		WillReturnRows(sqlmock.NewRows([]string{"system_deny_enabled", "dictionary_text", "updated_by", "updated_at"}).
			AddRow(1, "alpha\nbeta", "tester", time.Now()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
	item, err := svc.SaveAISensitiveSystemConfig(context.Background(), AISensitiveSystemConfig{
		SystemDenyEnabled: true,
		DictionaryText:    "alpha\nbeta",
		UpdatedBy:         "tester",
	})
	require.NoError(t, err)
	require.True(t, item.SystemDenyEnabled)
	require.Equal(t, "alpha\nbeta", item.DictionaryText)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListAIQuotaRoutesFallsBackForNilFields(t *testing.T) {
	k8s := k8sclient.NewMemoryClient()
	_, err := k8s.UpsertResource(context.Background(), "ai-routes", "doubao", map[string]any{
		"domains":        []any{"ai.local"},
		"pathPredicate":  map[string]any{"matchValue": "/doubao"},
		"redisKeyPrefix": nil,
		"adminConsumer":  nil,
		"adminPath":      nil,
		"quotaUnit":      nil,
	})
	require.NoError(t, err)

	svc := New(&portaldbclient.FakeClient{}, k8s)
	items, err := svc.ListAIQuotaRoutes(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "doubao", items[0].RouteName)
	require.Equal(t, "aigateway:quota:doubao", items[0].RedisKeyPrefix)
	require.Equal(t, builtinQuotaAdminConsumer, items[0].AdminConsumer)
	require.Equal(t, "/v1/ai/quotas/routes/doubao/consumers", items[0].AdminPath)
	require.Equal(t, "amount", items[0].QuotaUnit)
}
