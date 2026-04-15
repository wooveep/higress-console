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

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
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

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "mysql", AutoMigrate: true}, db))
	items, err := svc.ListAISensitiveAudits(context.Background(), AISensitiveAuditQuery{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "req-legacy", items[0].RequestID)
	require.Nil(t, items[0].GuardCode)
	require.Empty(t, items[0].BlockedDetails)
	require.NoError(t, mock.ExpectationsWereMet())
}
