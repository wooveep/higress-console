package portal

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestListUsageStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	rows := sqlmock.NewRows([]string{
		"consumer_name", "model_id", "request_count", "input_tokens", "output_tokens", "total_tokens",
		"cache_creation_input_tokens", "cache_creation_5m_input_tokens", "cache_creation_1h_input_tokens",
		"cache_read_input_tokens", "input_image_tokens", "output_image_tokens", "input_image_count", "output_image_count",
	}).AddRow(
		"alice", "qwen", 3, 120, 80, 0, 20, 10, 5, 7, 0, 0, 0, 0,
	)
	mock.ExpectQuery("FROM billing_usage_event").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(rows)

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ListUsageStats(context.Background(), UsageStatsQuery{})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(227), items[0].TotalTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsageEventsUnavailable(t *testing.T) {
	svc := New(portaldbclient.New(portaldbclient.Config{}))
	_, err := svc.ListUsageEvents(context.Background(), UsageEventsQuery{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unavailable")
}

func TestListUsageEventsIncludesAlignedFields(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	startedAt := time.Now().Add(-6 * time.Second)
	finishedAt := startedAt.Add(5500 * time.Millisecond)
	mock.ExpectQuery("SELECT event_id, request_id, trace_id, consumer_name, department_id").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 50, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"event_id", "request_id", "trace_id", "consumer_name", "department_id", "department_path", "api_key_id",
			"model_id", "price_version_id", "route_name", "request_path", "request_kind", "request_status",
			"usage_status", "http_status", "error_code", "error_message", "input_tokens", "output_tokens",
			"total_tokens", "request_count", "cost_micro_yuan", "started_at", "finished_at", "occurred_at",
		}).AddRow(
			"evt-1", "req-1", "trace-1", "alice", "dept-a", "root/dept-a", "ak-1",
			"qwen", 9, "chat-route", "/v1/chat/completions", "chat", "failed",
			"parsed", 500, "upstream_error", "timeout", 10, 5,
			15, 1, 3200, startedAt, finishedAt, finishedAt,
		).AddRow(
			"evt-2", "req-2", "trace-2", "alice", "dept-a", "root/dept-a", nil,
			"qwen", nil, "chat-route", "/v1/chat/completions", "chat", "success",
			"missing", 200, "", "", 0, 0,
			0, 0, 0, startedAt, finishedAt, finishedAt,
		))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ListUsageEvents(context.Background(), UsageEventsQuery{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "/v1/chat/completions", items[0].RequestPath)
	require.Equal(t, "upstream_error", items[0].ErrorCode)
	require.Equal(t, "timeout", items[0].ErrorMessage)
	require.Equal(t, int64(5500), items[0].ServiceDurationMs)
	require.NotEmpty(t, items[0].StartedAt)
	require.NotEmpty(t, items[0].FinishedAt)
	require.Empty(t, items[1].APIKeyID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListDepartmentBillsWithDepartmentScope(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery("SELECT path FROM org_department WHERE department_id = \\? LIMIT 1").
		WithArgs("dept-a").
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow("root/dept-a"))
	mock.ExpectQuery("SELECT department_id FROM org_department").
		WithArgs("dept-a", "root/dept-a", "root/dept-a/%").
		WillReturnRows(sqlmock.NewRows([]string{"department_id"}).AddRow("dept-a").AddRow("dept-b"))
	mock.ExpectQuery("FROM billing_usage_event e").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{
			"department_id", "department_name", "department_path", "request_count", "total_tokens", "total_cost", "active_consumers",
		}).AddRow("dept-a", "Dept A", "root/dept-a", 10, 1000, 12.5, 3))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	includeChildren := true
	items, err := svc.ListDepartmentBills(context.Background(), DepartmentBillsQuery{
		DepartmentIDs:   []string{"dept-a"},
		IncludeChildren: &includeChildren,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(3), items[0].ActiveConsumers)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsageTrend(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	from := now.Add(-time.Hour).UnixMilli()
	to := now.UnixMilli()
	mock.ExpectQuery("TO_TIMESTAMP\\(FLOOR\\(EXTRACT\\(EPOCH FROM occurred_at\\) / 300\\) \\* 300\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_label", "request_count", "input_tokens", "output_tokens", "total_tokens", "cost_micro_yuan", "active_consumers",
		}).AddRow("2026-04-14 10:05:00", 12, 2400, 800, 3650, 3200, 3))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	svc.schemaChecked = true
	items, err := svc.ListUsageTrend(context.Background(), UsageTrendQuery{
		From: &from,
		To:   &to,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(3200), items[0].CostMicroYuan)
	require.Equal(t, int64(3650), items[0].TotalTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsageTrendPostgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	now := time.Now()
	from := now.Add(-time.Hour).UnixMilli()
	to := now.UnixMilli()
	mock.ExpectQuery("TO_TIMESTAMP\\(FLOOR\\(EXTRACT\\(EPOCH FROM occurred_at\\) / 300\\) \\* 300\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_label", "request_count", "input_tokens", "output_tokens", "total_tokens", "cost_micro_yuan", "active_consumers",
		}).AddRow("2026-04-14 10:05:00", 12, 2400, 800, 3650, 3200, 3))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	svc.schemaChecked = true
	items, err := svc.ListUsageTrend(context.Background(), UsageTrendQuery{
		From: &from,
		To:   &to,
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, int64(3200), items[0].CostMicroYuan)
	require.Equal(t, int64(3650), items[0].TotalTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsageEventFilterOptions(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery("SELECT DISTINCT e.department_id").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "label"}).AddRow("dept-a", "Dept A"))
	mock.ExpectQuery("SELECT DISTINCT consumer_name AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("alice"))
	mock.ExpectQuery("SELECT DISTINCT api_key_id AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("ak-1"))
	mock.ExpectQuery("SELECT DISTINCT model_id AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("qwen"))
	mock.ExpectQuery("SELECT DISTINCT route_name AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("chat-route"))
	mock.ExpectQuery("SELECT DISTINCT request_status AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("failed"))
	mock.ExpectQuery("SELECT DISTINCT usage_status AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("parsed"))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.ListUsageEventFilterOptions(context.Background(), UsageEventsQuery{})
	require.NoError(t, err)
	require.Equal(t, "alice", item.Consumers[0].Value)
	require.Equal(t, "Dept A", item.Departments[0].Label)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsageEventFilterOptionsNarrowByDepartmentScope(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery("SELECT path FROM org_department WHERE department_id = \\? LIMIT 1").
		WithArgs("dept-a").
		WillReturnRows(sqlmock.NewRows([]string{"path"}).AddRow("root/dept-a"))
	mock.ExpectQuery("SELECT department_id FROM org_department").
		WithArgs("dept-a", "root/dept-a", "root/dept-a/%").
		WillReturnRows(sqlmock.NewRows([]string{"department_id"}).AddRow("dept-a").AddRow("dept-b"))
	mock.ExpectQuery("SELECT DISTINCT e.department_id").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "alice").
		WillReturnRows(sqlmock.NewRows([]string{"department_id", "label"}).AddRow("dept-a", "Dept A"))
	mock.ExpectQuery("SELECT DISTINCT consumer_name AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("alice"))
	mock.ExpectQuery("SELECT DISTINCT api_key_id AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "alice", "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("ak-1"))
	mock.ExpectQuery("SELECT DISTINCT model_id AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "alice", "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("qwen"))
	mock.ExpectQuery("SELECT DISTINCT route_name AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "alice", "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("chat-route"))
	mock.ExpectQuery("SELECT DISTINCT request_status AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "alice", "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("success"))
	mock.ExpectQuery("SELECT DISTINCT usage_status AS value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "alice", "dept-a", "dept-b").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("parsed"))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	includeChildren := true
	item, err := svc.ListUsageEventFilterOptions(context.Background(), UsageEventsQuery{
		ConsumerNames:   []string{"alice"},
		DepartmentIDs:   []string{"dept-a"},
		IncludeChildren: &includeChildren,
	})
	require.NoError(t, err)
	require.Equal(t, "alice", item.Consumers[0].Value)
	require.Equal(t, "Dept A", item.Departments[0].Label)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNormalizeStatsRangeDefaults(t *testing.T) {
	from, to := normalizeStatsRange(nil, nil, time.Hour)
	require.True(t, to.After(from))
	require.WithinDuration(t, to.Add(-time.Hour), from, 2*time.Second)
	require.Equal(t, time.UTC, to.Location())
	require.Equal(t, time.UTC, from.Location())
}

func TestNormalizeStatsRangeUsesUTCMillis(t *testing.T) {
	fromMillis := time.Date(2026, time.April, 16, 1, 0, 0, 0, time.FixedZone("CST", 8*60*60)).UnixMilli()
	toMillis := time.Date(2026, time.April, 16, 2, 0, 0, 0, time.FixedZone("CST", 8*60*60)).UnixMilli()

	from, to := normalizeStatsRange(&fromMillis, &toMillis, time.Hour)

	require.Equal(t, time.UTC, from.Location())
	require.Equal(t, time.UTC, to.Location())
	require.Equal(t, "2026-04-15T17:00:00Z", from.Format(time.RFC3339))
	require.Equal(t, "2026-04-15T18:00:00Z", to.Format(time.RFC3339))
}
