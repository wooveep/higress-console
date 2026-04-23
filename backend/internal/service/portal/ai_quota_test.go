package portal

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestListAIQuotaScheduleRulesAcceptsBooleanColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT id, consumer_name, action, cron, value, enabled, created_at, updated_at, last_applied_at, last_error
		FROM portal_ai_quota_schedule_rule
		WHERE route_name = ?
		ORDER BY consumer_name ASC, created_at DESC`)).
		WithArgs("route-a").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "consumer_name", "action", "cron", "value", "enabled", "created_at", "updated_at", "last_applied_at", "last_error",
		}).AddRow("rule-a", "demo", "REFRESH", "0 8 * * *", int64(100), true, time.Now(), time.Now(), nil, nil))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ListAIQuotaScheduleRules(context.Background(), "route-a", "")
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.True(t, items[0].Enabled)
	require.NoError(t, mock.ExpectationsWereMet())
}
