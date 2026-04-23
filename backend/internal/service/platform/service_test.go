package platform

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	"github.com/wooveep/aigateway-console/backend/internal/model/response"
	grafanaclient "github.com/wooveep/aigateway-console/backend/utility/clients/grafana"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestServiceHealthz(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(), grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	status, err := svc.Healthz(context.Background())

	require.NoError(t, err)
	require.Equal(t, "ok", status.Status)
	require.Equal(t, consts.ServiceName, status.Service)
	require.Equal(t, consts.LegacyBackendDir, status.LegacyBackend)
}

func TestServiceSystemInfo(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(), grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	info, err := svc.SystemInfo(context.Background())

	require.NoError(t, err)
	require.Equal(t, consts.ServiceName, info.Service)
	require.Equal(t, consts.PreferredProduct, info.PreferredNaming)
	require.Contains(t, info.BusinessLines, "gateway")
	require.Contains(t, info.BusinessLines, "portal")
}

func TestServiceInitializeAndLogin(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(), grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	err := svc.InitializeSystem(context.Background(), &response.User{
		Username:    "admin",
		Password:    "secret",
		DisplayName: "Admin",
	}, map[string]any{
		"login.prompt": "Hello",
	})
	require.NoError(t, err)
	require.True(t, svc.IsSystemInitialized(context.Background()))

	user, token, err := svc.Login(context.Background(), "admin", "secret")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Equal(t, "admin", user.Username)

	validated, err := svc.ValidateSessionToken(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "admin", validated.Username)
}

func TestServiceLoadsPersistedAdminState(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	hash, err := hashAdminPassword("secret")
	require.NoError(t, err)

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS console_system_state").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT initialized, admin_username, admin_display_name, admin_password_hash, configs_json").
		WithArgs(consts.DefaultAdminStateKey).
		WillReturnRows(sqlmock.NewRows([]string{
			"initialized", "admin_username", "admin_display_name", "admin_password_hash", "configs_json",
		}).AddRow(true, "admin", "Admin", hash, `{"system.initialized":true,"login.prompt":"hello"}`))

	svc := New(
		k8sclient.NewMemoryClient(),
		grafanaclient.New(grafanaclient.Config{}),
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres"}, db),
	)

	require.True(t, svc.IsSystemInitialized(context.Background()))
	configs := svc.GetConfigs(context.Background())
	require.Equal(t, "hello", configs["login.prompt"])

	user, token, err := svc.Login(context.Background(), "admin", "secret")
	require.NoError(t, err)
	require.Equal(t, "admin", user.Username)

	validated, err := svc.ValidateSessionToken(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "admin", validated.Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceLoadsPersistedAdminStateEnsuresDefaultGatewayResources(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	hash, err := hashAdminPassword("secret")
	require.NoError(t, err)

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS console_system_state").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT initialized, admin_username, admin_display_name, admin_password_hash, configs_json").
		WithArgs(consts.DefaultAdminStateKey).
		WillReturnRows(sqlmock.NewRows([]string{
			"initialized", "admin_username", "admin_display_name", "admin_password_hash", "configs_json",
		}).AddRow(true, "admin", "Admin", hash, `{"system.initialized":true,"route.default.initialized":true}`))

	client := k8sclient.NewMemoryClient()
	svc := New(
		client,
		grafanaclient.New(grafanaclient.Config{}),
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres"}, db),
	)

	require.True(t, svc.IsSystemInitialized(context.Background()))

	route, err := client.GetResource(context.Background(), "routes", consts.DefaultRouteName)
	require.NoError(t, err)
	require.Equal(t, consts.DefaultRouteName, route["name"])

	domain, err := client.GetResource(context.Background(), "domains", consts.DefaultDomainName)
	require.NoError(t, err)
	require.Equal(t, consts.DefaultTLSCertificateName, domain["certIdentifier"])

	tls, err := client.GetResource(context.Background(), "tls-certificates", consts.DefaultTLSCertificateName)
	require.NoError(t, err)
	require.Equal(t, consts.DefaultTLSCertificateName, tls["name"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestServiceMigratesLegacySecretStateToDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	client := k8sclient.NewMemoryClient()
	require.NoError(t, client.UpsertSecret(context.Background(), consts.DefaultSecretName, map[string]string{
		"adminUsername":    "admin",
		"adminPassword":    "secret",
		"adminDisplayName": "Admin",
	}))

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS console_system_state").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT initialized, admin_username, admin_display_name, admin_password_hash, configs_json").
		WithArgs(consts.DefaultAdminStateKey).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO console_system_state").
		WithArgs(
			consts.DefaultAdminStateKey,
			true,
			"admin",
			"Admin",
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	svc := New(
		client,
		grafanaclient.New(grafanaclient.Config{}),
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres"}, db),
	)

	require.True(t, svc.IsSystemInitialized(context.Background()))
	user, token, err := svc.Login(context.Background(), "admin", "secret")
	require.NoError(t, err)
	require.Equal(t, "admin", user.Username)

	validated, err := svc.ValidateSessionToken(context.Background(), token)
	require.NoError(t, err)
	require.Equal(t, "admin", validated.Username)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNativeDashboardDBTimeFromMillisUsesUTC(t *testing.T) {
	value := time.Date(2026, time.April, 16, 1, 8, 44, 0, time.FixedZone("CST", 8*60*60)).UnixMilli()

	parsed := nativeDashboardDBTimeFromMillis(value)

	require.Equal(t, time.UTC, parsed.Location())
	require.Equal(t, "2026-04-15T17:08:44Z", parsed.Format(time.RFC3339))
}

func TestServiceSetAndGetAIGatewayConfig(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(), grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	current, err := svc.GetAIGatewayConfig(context.Background())
	require.NoError(t, err)
	require.Contains(t, current, "gzip:")
	require.Contains(t, current, "downstream:")
	require.NotContains(t, current, "ConfigMap")

	updated, err := svc.SetAIGatewayConfig(context.Background(), `
tracing:
  enable: true
  sampling: 100
  timeout: 500
  zipkin:
    service: zipkin.default.svc.cluster.local
    port: "9411"
gzip:
  enable: true
  minContentLength: 1024
  contentType:
    - text/html
    - application/json
  disableOnEtagHeader: true
  memoryLevel: 5
  windowBits: 12
  chunkSize: 4096
  compressionLevel: BEST_COMPRESSION
  compressionStrategy: DEFAULT_STRATEGY
downstream:
  idleTimeout: 180
  maxRequestHeadersKb: 60
  connectionBufferLimits: 32768
  http2:
    maxConcurrentStreams: 100
    initialStreamWindowSize: 65535
    initialConnectionWindowSize: 1048576
  routeTimeout: 0
upstream:
  idleTimeout: 10
  connectionBufferLimits: 10485760
addXRealIpHeader: false
disableXEnvoyHeaders: false
`)
	require.NoError(t, err)
	require.Contains(t, updated, "zipkin:")
	require.NotContains(t, updated, "ConfigMap")
}

func TestServiceGetAIGatewayConfigUsesDefaultWhenKeyMissing(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	require.NoError(t, client.UpsertConfigMap(context.Background(), consts.DefaultHigressConfigMapName, map[string]string{
		"mesh":         "defaultConfig: {}\n",
		"meshNetworks": "networks: {}\n",
		"mcpServer":    "enable: true\n",
	}))
	svc := New(client, grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	current, err := svc.GetAIGatewayConfig(context.Background())

	require.NoError(t, err)
	require.Contains(t, current, "gzip:")
	require.Contains(t, current, "upstream:")
	require.Contains(t, current, "connectionBufferLimits: 10485760")
}

func TestServiceSetAIGatewayConfigPreservesOtherConfigMapKeys(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	require.NoError(t, client.UpsertConfigMap(context.Background(), consts.DefaultHigressConfigMapName, map[string]string{
		consts.DefaultHigressConfigDataKey: "gzip:\n  enable: true\n",
		"mesh":                             "defaultConfig: {}\n",
		"meshNetworks":                     "networks: {}\n",
		"mcpServer":                        "enable: true\n",
	}))
	svc := New(client, grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	updated, err := svc.SetAIGatewayConfig(context.Background(), `
tracing:
  enable: false
  sampling: 100
  timeout: 500
gzip:
  enable: true
  minContentLength: 1024
  contentType:
    - text/html
    - application/json
  disableOnEtagHeader: true
  memoryLevel: 5
  windowBits: 12
  chunkSize: 4096
  compressionLevel: BEST_COMPRESSION
  compressionStrategy: DEFAULT_STRATEGY
downstream:
  idleTimeout: 180
  maxRequestHeadersKb: 60
  connectionBufferLimits: 32768
  http2:
    maxConcurrentStreams: 100
    initialStreamWindowSize: 65535
    initialConnectionWindowSize: 1048576
  routeTimeout: 0
upstream:
  idleTimeout: 10
  connectionBufferLimits: 10485760
mcpServer:
  enable: true
  sse_path_suffix: /sse
`)
	require.NoError(t, err)
	require.Contains(t, updated, "mcpServer:")

	stored, err := client.ReadConfigMap(context.Background(), consts.DefaultHigressConfigMapName)
	require.NoError(t, err)
	require.Equal(t, "defaultConfig: {}\n", stored["mesh"])
	require.Equal(t, "networks: {}\n", stored["meshNetworks"])
	require.Equal(t, "enable: true\n", stored["mcpServer"])
	require.Contains(t, stored[consts.DefaultHigressConfigDataKey], "mcpServer:")
}

func TestServiceSetAIGatewayConfigRejectsInvalidConfig(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(), grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	_, err := svc.SetAIGatewayConfig(context.Background(), `
tracing:
  enable: true
  sampling: 101
  timeout: 500
  skywalking:
    service: skywalking.default.svc.cluster.local
    port: "11800"
gzip:
  enable: true
  minContentLength: 1024
  contentType:
    - text/html
  disableOnEtagHeader: true
  memoryLevel: 5
  windowBits: 12
  chunkSize: 4096
  compressionLevel: BEST_COMPRESSION
  compressionStrategy: DEFAULT_STRATEGY
`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "tracing.sampling")
}

func TestServiceSetAIGatewayConfigRejectsInvalidHTTP2Window(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(), grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))

	_, err := svc.SetAIGatewayConfig(context.Background(), `
downstream:
  http2:
    initialConnectionWindowSize: 1
`)

	require.Error(t, err)
	require.Contains(t, err.Error(), "downstream.http2.initialConnectionWindowSize")
}

func TestNativeDashboardBuildsRows(t *testing.T) {
	server := newNativeDashboardPrometheusServer()
	defer server.Close()
	t.Setenv("AIGATEWAY_CONSOLE_PROMETHEUS_BASE_URL", server.URL)

	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "routes", "default", map[string]any{})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "domains", "default", map[string]any{})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "wasm-plugins", "default", map[string]any{})
	require.NoError(t, err)

	svc := New(
		client,
		grafanaclient.New(grafanaclient.Config{Enabled: true, BaseURL: "http://grafana.local/grafana"}),
		portaldbclient.New(portaldbclient.Config{}),
	)

	data, err := svc.NativeDashboard(context.Background(), "MAIN", time.Now().Add(-24*time.Hour).UnixMilli(), time.Now().UnixMilli(), "", "")
	require.NoError(t, err)
	require.Len(t, data.Rows, 5)
	require.Equal(t, "Platform", data.Rows[0].Title)
	require.Equal(t, "Gateway Request", data.Rows[1].Title)
	require.Equal(t, "Upstream Health", data.Rows[2].Title)
	require.Equal(t, "Exceptions", data.Rows[3].Title)
	require.Equal(t, "Resource Scale", data.Rows[4].Title)
	require.Equal(t, "Gateway Pod Count", data.Rows[0].Panels[3].Title)
	require.Equal(t, "Downstream Request Count", data.Rows[1].Panels[0].Title)
	require.Equal(t, "Routes", data.Rows[4].Panels[0].Title)
}

func TestNativeDashboardBuildsAIRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectNativeDashboardAIQueries(mock, 10, 8, 345, 2500000)

	server := newNativeDashboardPrometheusServer()
	defer server.Close()
	t.Setenv("AIGATEWAY_CONSOLE_PROMETHEUS_BASE_URL", server.URL)

	svc := New(
		k8sclient.NewMemoryClient(),
		grafanaclient.New(grafanaclient.Config{Enabled: true, BaseURL: "http://grafana.local/grafana"}),
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres"}, db),
	)

	from := time.Now().Add(-time.Hour).UnixMilli()
	to := time.Now().UnixMilli()
	data, err := svc.NativeDashboard(context.Background(), "AI", from, to, "", "")
	require.NoError(t, err)
	require.Len(t, data.Rows, 4)
	require.Equal(t, "AI Overview", data.Rows[0].Title)
	require.Equal(t, "Token Runtime", data.Rows[1].Title)
	require.Equal(t, "AI Request", data.Rows[2].Title)
	require.Equal(t, "AI Exceptions", data.Rows[3].Title)
	require.Equal(t, "Total Tokens", data.Rows[0].Panels[2].Title)
	require.NotNil(t, data.Rows[0].Panels[2].Stat)
	require.NotNil(t, data.Rows[0].Panels[2].Stat.Value)
	require.Equal(t, 345.0, *data.Rows[0].Panels[2].Stat.Value)
	require.Equal(t, "Failed Requests", data.Rows[3].Panels[0].Title)
	require.Equal(t, "Slow Requests", data.Rows[3].Panels[1].Title)
	require.Equal(t, "Error Code TopN", data.Rows[3].Panels[2].Title)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNativeDashboardRequestCountFallsBackToPrometheus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectNativeDashboardAIQueries(mock, 0, 0, 0, 0)

	server := newNativeDashboardPrometheusServer()
	defer server.Close()
	t.Setenv("AIGATEWAY_CONSOLE_PROMETHEUS_BASE_URL", server.URL)

	svc := New(
		k8sclient.NewMemoryClient(),
		grafanaclient.New(grafanaclient.Config{Enabled: true, BaseURL: "http://grafana.local/grafana"}),
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres"}, db),
	)

	from := time.Now().Add(-time.Hour).UnixMilli()
	to := time.Now().UnixMilli()
	data, err := svc.NativeDashboard(context.Background(), "AI", from, to, "", "")
	require.NoError(t, err)
	require.NotNil(t, data.Rows[0].Panels[0].Stat)
	require.NotNil(t, data.Rows[0].Panels[0].Stat.Value)
	require.Equal(t, 42.0, *data.Rows[0].Panels[0].Stat.Value)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestQueryNativeDashboardExceptionRouteTopTableUsesPostgresDurationExpr(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := New(
		k8sclient.NewMemoryClient(),
		grafanaclient.New(grafanaclient.Config{}),
		portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "pgx-rebind"}, db),
	)

	mock.ExpectQuery(`EXTRACT\(EPOCH FROM \(finished_at - started_at\)\) \* 1000`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{"route", "latency_ms"}).
			AddRow("/v1/chat/completions", 6200))

	table, err := svc.queryNativeDashboardExceptionRouteTopTable(context.Background(), time.Now().Add(-time.Hour).UnixMilli(), time.Now().UnixMilli(), "slow", 10)
	require.NoError(t, err)
	require.Len(t, table.Rows, 1)
	require.Equal(t, "/v1/chat/completions", table.Rows[0]["route"])
	require.Equal(t, 6200.0, table.Rows[0]["latencyMs"])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNativeDashboardSlowRequestSQLHelpers(t *testing.T) {
	postgresExpr := nativeDashboardServiceDurationMillisExpr("postgres")
	require.Contains(t, postgresExpr, "EXTRACT(EPOCH FROM (finished_at - started_at)) * 1000")
	require.NotContains(t, postgresExpr, "TIMESTAMPDIFF")
	require.Contains(t, nativeDashboardSlowRequestFilterClause("postgres"), postgresExpr+" > 5000")

	pgxRebindExpr := nativeDashboardServiceDurationMillisExpr("pgx-rebind")
	require.Equal(t, postgresExpr, pgxRebindExpr)
}

func expectNativeDashboardAIQueries(mock sqlmock.Sqlmock, requestCount, successCount, totalTokens, costMicroYuan int64) {
	mock.ExpectQuery("SELECT\\s+COALESCE\\(SUM\\(CASE WHEN request_count > 0 THEN request_count ELSE 1 END\\), 0\\) AS request_count").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"request_count", "success_count", "total_tokens", "cost_micro_yuan",
		}).AddRow(requestCount, successCount, totalTokens, costMicroYuan))
	mock.ExpectQuery("COALESCE\\(NULLIF\\(TRIM\\(request_path\\), ''\\), NULLIF\\(TRIM\\(route_name\\), ''\\), '-'\\) AS dimension_value").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"bucket_label", "dimension_value", "weighted_request",
		}).AddRow("2026-04-15 10:00:00", "/v1/chat/completions", 12))
	mock.ExpectQuery("WHERE occurred_at >= \\? AND occurred_at < \\? AND \\(http_status < 200 OR http_status >= 300 OR request_status <> 'success'\\)").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"occurred_at", "request_id", "trace_id", "consumer_name", "request_path", "route_name", "model_id",
			"request_status", "http_status", "error_code", "error_message", "total_tokens", "cost_micro_yuan", "service_duration_ms",
		}).AddRow(time.Now(), "req-failed", "trace-failed", "alice", "/v1/chat/completions", "chat-route", "qwen", "failed", 500, "upstream_error", "boom", 123, 4500, 0))
	mock.ExpectQuery("started_at IS NOT NULL AND finished_at IS NOT NULL AND CAST\\(FLOOR\\(GREATEST\\(EXTRACT\\(EPOCH FROM \\(finished_at - started_at\\)\\) \\* 1000, 0\\)\\) AS BIGINT\\) > 5000").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"occurred_at", "request_id", "trace_id", "consumer_name", "request_path", "route_name", "model_id",
			"request_status", "http_status", "error_code", "error_message", "total_tokens", "cost_micro_yuan", "service_duration_ms",
		}).AddRow(time.Now(), "req-slow", "trace-slow", "bob", "/v1/chat/completions", "chat-route", "qwen", "success", 200, "", "", 456, 7800, 6200))
	mock.ExpectQuery("SELECT COALESCE\\(NULLIF\\(TRIM\\(error_code\\), ''\\), 'unknown'\\) AS error_code").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"error_code", "request_count",
		}).AddRow("upstream_error", 3))
}

func newNativeDashboardPrometheusServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		if strings.Contains(r.URL.Path, "/api/v1/query_range") {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.Contains(query, "by (cluster_name)"):
				_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"cluster_name":"outbound_443__svc-a.default.svc.cluster.local"},"values":[[1713168000,"2"],[1713168060,"3"]]},{"metric":{"cluster_name":"outbound_443__svc-b.default.svc.cluster.local"},"values":[[1713168000,"1"],[1713168060,"1.5"]]}]}}`))
			default:
				_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"matrix","result":[{"metric":{},"values":[[1713168000,"1"],[1713168060,"2"]]}]}}`))
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.TrimSpace(query) == "envoy_server_live":
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{"app":"aigateway-gateway"},"value":[1713168000,"1"]}]}}`))
		case strings.Contains(query, "container_cpu_usage_seconds_total"):
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"23.5"]}]}}`))
		case strings.Contains(query, "container_memory_working_set_bytes"):
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"1048576"]}]}}`))
		case strings.Contains(query, "envoy_cluster_upstream_cx_active"):
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"7"]}]}}`))
		case strings.Contains(query, "route_upstream_model_consumer_metric_total_token"):
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"987"]}]}}`))
		case strings.Contains(query, "route_upstream_model_consumer_metric_request_count"):
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"42"]}]}}`))
		case strings.Contains(query, `response_code_class="2xx"`):
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"0.75"]}]}}`))
		default:
			_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1713168000,"1"]}]}}`))
		}
	}))
}

func TestResolveNativeDashboardUsageBucketExprPostgresCompatible(t *testing.T) {
	expr := resolveNativeDashboardUsageBucketExpr(
		"pgx-rebind",
		time.Now().Add(-2*time.Hour).UnixMilli(),
		time.Now().UnixMilli(),
	)

	require.Contains(t, expr, "TO_TIMESTAMP(FLOOR(EXTRACT(EPOCH FROM occurred_at) / 300) * 300)")
}

func TestResolveNativeDashboardPrometheusRateRange(t *testing.T) {
	now := time.Now()

	shortRange := resolveNativeDashboardPrometheusRateRange(
		now.Add(-5*time.Minute).UnixMilli(),
		now.UnixMilli(),
	)
	longRange := resolveNativeDashboardPrometheusRateRange(
		now.Add(-7*24*time.Hour).UnixMilli(),
		now.UnixMilli(),
	)

	require.Equal(t, "300s", shortRange)
	require.Equal(t, "14400s", longRange)
}

func TestNativeDashboardSeriesHasNonZeroPoints(t *testing.T) {
	require.False(t, nativeDashboardSeriesHasNonZeroPoints(nil))
	require.False(t, nativeDashboardSeriesHasNonZeroPoints([]response.NativeDashboardSeries{{
		Name: "zero",
		Points: []response.NativeDashboardPoint{
			{Time: 1, Value: 0},
			{Time: 2, Value: 0},
		},
	}}))
	require.True(t, nativeDashboardSeriesHasNonZeroPoints([]response.NativeDashboardSeries{{
		Name: "non-zero",
		Points: []response.NativeDashboardPoint{
			{Time: 1, Value: 0},
			{Time: 2, Value: 0.25},
		},
	}}))
}

func TestIsNativeDashboardInfraService(t *testing.T) {
	require.True(t, isNativeDashboardInfraService("prometheus_stats"))
	require.True(t, isNativeDashboardInfraService("redis-stack-server.aigateway-system.svc.cluster.local"))
	require.True(t, isNativeDashboardInfraService("xds-grpc"))
	require.False(t, isNativeDashboardInfraService("llm-doubao.internal.dns"))
	require.False(t, isNativeDashboardInfraService("svc-a.default.svc.cluster.local"))
}

func TestBootstrapDefaultResourcesLockedDoesNotOverwriteExistingResources(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "tls-certificates", consts.DefaultTLSCertificateName, map[string]any{
		"name": "default",
		"cert": "existing-cert",
		"key":  "existing-key",
	})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "domains", consts.DefaultDomainName, map[string]any{
		"name":           consts.DefaultDomainName,
		"certIdentifier": consts.DefaultTLSCertificateName,
		"enableHttps":    "force",
	})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "routes", consts.DefaultRouteName, map[string]any{
		"name":    consts.DefaultRouteName,
		"domains": []string{consts.DefaultDomainName},
		"path":    map[string]any{"matchType": "EQUAL", "matchValue": "/"},
		"services": []map[string]any{{
			"name": "existing.default.svc.cluster.local",
			"port": 8080,
		}},
	})
	require.NoError(t, err)

	svc := New(client, grafanaclient.New(grafanaclient.Config{}), portaldbclient.New(portaldbclient.Config{}))
	svc.adminUser = &response.User{
		Username:    "admin",
		DisplayName: "Admin",
	}

	require.NoError(t, svc.bootstrapDefaultResourcesLocked(context.Background(), "secret"))

	tls, err := client.GetResource(context.Background(), "tls-certificates", consts.DefaultTLSCertificateName)
	require.NoError(t, err)
	require.Equal(t, "existing-cert", tls["cert"])

	domain, err := client.GetResource(context.Background(), "domains", consts.DefaultDomainName)
	require.NoError(t, err)
	require.Equal(t, "force", domain["enableHttps"])

	route, err := client.GetResource(context.Background(), "routes", consts.DefaultRouteName)
	require.NoError(t, err)
	require.Equal(t, "existing.default.svc.cluster.local", route["services"].([]any)[0].(map[string]any)["name"])
}
