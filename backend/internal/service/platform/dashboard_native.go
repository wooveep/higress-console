package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wooveep/aigateway-console/backend/internal/model/response"
)

type nativeDashboardUsageSummary struct {
	RequestCount    int64
	TotalTokens     int64
	CostMicroYuan   int64
	ActiveConsumers int64
}

type nativeDashboardTrendRow struct {
	BucketLabel   string
	RequestCount  float64
	TotalTokens   float64
	CostMicroYuan float64
}

type nativeDashboardRecentEvent struct {
	OccurredAt    string
	ConsumerName  string
	RouteName     string
	ModelID       string
	RequestStatus string
	TotalTokens   int64
	CostMicroYuan int64
}

type nativeDashboardAIGeneralSummary struct {
	RequestCount  int64
	SuccessCount  int64
	TotalTokens   int64
	CostMicroYuan int64
}

type nativeDashboardRequestTrendDimensionRow struct {
	BucketLabel     string
	DimensionValue  string
	WeightedRequest int64
}

type nativeDashboardExceptionRecord struct {
	OccurredAt        string
	RequestID         string
	TraceID           string
	ConsumerName      string
	RequestPath       string
	RouteName         string
	ModelID           string
	RequestStatus     string
	HTTPStatus        int
	ErrorCode         string
	ErrorMessage      string
	TotalTokens       int64
	CostMicroYuan     int64
	ServiceDurationMs int64
}

type nativeDashboardResourcePanel struct {
	Title string
	Kind  string
}

func nativeDashboardDBTimeFromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func (s *Service) buildNativeDashboardRows(ctx context.Context, dashboardType string, from, to int64) []response.NativeDashboardRow {
	switch strings.ToUpper(strings.TrimSpace(dashboardType)) {
	case "AI":
		return s.buildNativeAIDashboardRows(ctx, from, to)
	default:
		return s.buildNativeMainDashboardRows(ctx, from, to)
	}
}

func (s *Service) buildNativeMainDashboardRows(ctx context.Context, from, to int64) []response.NativeDashboardRow {
	selector := s.discoverNativeDashboardGatewaySelector(ctx)
	step := resolveNativeDashboardPrometheusStep(from, to)
	upstreamSelector := joinNativeDashboardPromSelector(
		selector,
		`cluster_name=~"outbound.*"`,
	)
	rateRange := resolveNativeDashboardPrometheusRateRange(from, to)
	windowRange := formatNativeDashboardPromRange(from, to)
	rows := make([]response.NativeDashboardRow, 0, 5)

	cpuUsage, cpuUsageErr := s.queryNativeDashboardPrometheusScalar(ctx,
		`100 * sum(irate(container_cpu_usage_seconds_total{namespace="aigateway-system", pod=~".*gateway.*"}[1m]))`,
		to,
	)
	memoryUsage, memoryUsageErr := s.queryNativeDashboardPrometheusScalar(ctx,
		`sum(max(container_memory_working_set_bytes{namespace="aigateway-system", pod=~".*gateway.*"} ) by (pod))`,
		to,
	)
	activeConnections, activeConnectionsErr := s.queryNativeDashboardPrometheusScalar(ctx,
		fmt.Sprintf(`sum(envoy_cluster_upstream_cx_active{%s, cluster_name!="xds-grpc", cluster_name!="prometheus_stats", cluster_name!="agent", cluster_name!="BlackHoleCluster", cluster_name!="sds-grpc"})`, selector),
		to,
	)
	gatewayPodCount, gatewayPodCountErr := s.queryNativeDashboardPrometheusScalar(ctx,
		fmt.Sprintf(`count(envoy_server_live{%s})`, selector),
		to,
	)
	rows = append(rows, response.NativeDashboardRow{
		Title:     "Platform",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardFloatStatPanel(1, "Current CPU Usage", "percent", 0, 0, 6, cpuUsage, cpuUsageErr),
			makeNativeDashboardFloatStatPanel(2, "Current Memory Usage", "bytes", 6, 0, 6, memoryUsage, memoryUsageErr),
			makeNativeDashboardFloatStatPanel(3, "Current Active Connections", "short", 12, 0, 6, activeConnections, activeConnectionsErr),
			makeNativeDashboardFloatStatPanel(4, "Gateway Pod Count", "short", 18, 0, 6, gatewayPodCount, gatewayPodCountErr),
		},
	})

	downstreamRequestCount, downstreamRequestCountErr := s.queryNativeDashboardPrometheusCounterIncrease(ctx,
		fmt.Sprintf(`envoy_http_downstream_rq_total{%s}`, selector),
		from,
		to,
	)
	downstreamSuccessRate, downstreamSuccessRateErr := s.queryNativeDashboardPrometheusScalar(ctx,
		fmt.Sprintf(`sum(increase(envoy_http_downstream_rq{%s, response_code_class="2xx"}[%s])) / clamp_min(sum(increase(envoy_http_downstream_rq_total{%s}[%s])), 1)`, selector, windowRange, selector, windowRange),
		to,
	)
	downstreamQPSSeries, downstreamQPSSeriesErr := s.queryNativeDashboardPrometheusSeries(ctx,
		fmt.Sprintf(`sum(rate(envoy_http_downstream_rq_total{%s}[%s]))`, selector, rateRange),
		"QPS",
		from,
		to,
		step,
	)
	downstreamLatencySeries, downstreamLatencySeriesErr := s.queryNativeDashboardPrometheusLatencySeries(ctx, selector, "downstream", from, to, step)
	rows = append(rows, response.NativeDashboardRow{
		Title:     "Gateway Request",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardFloatStatPanel(5, "Downstream Request Count", "short", 0, 0, 6, downstreamRequestCount, downstreamRequestCountErr),
			makeNativeDashboardFloatStatPanel(6, "Downstream Success Rate", "percentunit", 6, 0, 6, downstreamSuccessRate, downstreamSuccessRateErr),
			makeNativeDashboardTimeseriesPanel(7, "Downstream QPS Trend", "reqps", 12, 0, 12, downstreamQPSSeries, downstreamQPSSeriesErr),
			makeNativeDashboardTimeseriesPanel(8, "Downstream Latency", "ms", 0, 1, 24, downstreamLatencySeries, downstreamLatencySeriesErr),
		},
	})

	upstreamRequestCount, upstreamRequestCountErr := s.queryNativeDashboardPrometheusCounterIncrease(ctx,
		fmt.Sprintf(`envoy_cluster_upstream_rq_total{%s}`, upstreamSelector),
		from,
		to,
	)
	upstreamSuccess, upstreamSuccessErr := s.queryNativeDashboardPrometheusScalar(ctx,
		fmt.Sprintf(`sum(increase(envoy_cluster_upstream_rq{%s, response_code_class="2xx"}[%s])) / clamp_min(sum(increase(envoy_cluster_upstream_rq_total{%s}[%s])), 1)`, upstreamSelector, windowRange, upstreamSelector, windowRange),
		to,
	)
	upstreamQPSSeries, upstreamQPSSeriesErr := s.queryNativeDashboardPrometheusSeries(ctx,
		fmt.Sprintf(`sum(rate(envoy_cluster_upstream_rq_total{%s}[%s]))`, upstreamSelector, rateRange),
		"QPS",
		from,
		to,
		step,
	)
	upstreamLatencySeries, upstreamLatencySeriesErr := s.queryNativeDashboardPrometheusLatencySeries(ctx, upstreamSelector, "upstream", from, to, step)
	rows = append(rows, response.NativeDashboardRow{
		Title:     "Upstream Health",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardFloatStatPanel(9, "Upstream Request Count", "short", 0, 0, 6, upstreamRequestCount, upstreamRequestCountErr),
			makeNativeDashboardFloatStatPanel(10, "Upstream Attempt Success Rate", "percentunit", 6, 0, 6, upstreamSuccess, upstreamSuccessErr),
			makeNativeDashboardTimeseriesPanel(11, "Upstream QPS Trend", "reqps", 12, 0, 12, upstreamQPSSeries, upstreamQPSSeriesErr),
			makeNativeDashboardTimeseriesPanel(12, "Upstream Latency", "ms", 0, 1, 24, upstreamLatencySeries, upstreamLatencySeriesErr),
		},
	})

	downstream5xxCount, downstream5xxCountErr := s.queryNativeDashboardPrometheusCounterIncrease(ctx,
		fmt.Sprintf(`envoy_http_downstream_rq{%s, response_code_class="5xx"}`, selector),
		from,
		to,
	)
	upstreamFailureCount, upstreamFailureCountErr := s.queryNativeDashboardPrometheusScalar(ctx,
		fmt.Sprintf(`sum(increase(envoy_cluster_upstream_rq{%s, response_code_class="5xx"}[%s])) + sum(increase(envoy_cluster_upstream_rq_timeout{%s}[%s]))`, upstreamSelector, windowRange, upstreamSelector, windowRange),
		to,
	)
	failureTopTable, failureTopTableErr := s.queryNativeDashboardPrometheusTopTable(
		ctx,
		fmt.Sprintf(`topk(10, label_replace(label_replace(sum(rate(envoy_cluster_upstream_rq{%s, response_code_class=~"(4|5)xx"}[%s])) by (cluster_name), "service", "$3", "cluster_name", "outbound_([0-9]+)_(.*)_(.*)$"), "port", "$1", "cluster_name", "outbound_([0-9]+)_(.*)_(.*)$"))`, upstreamSelector, rateRange),
		"QPS",
		to,
	)
	slowTopTable, slowTopTableErr := s.queryNativeDashboardPrometheusTopTable(
		ctx,
		fmt.Sprintf(`topk(10, label_replace(label_replace(sum(rate(envoy_cluster_upstream_rq_time_sum{%s}[%s])) by (cluster_name) / clamp_min(sum(rate(envoy_cluster_upstream_rq_time_count{%s}[%s])) by (cluster_name), 1), "service", "$3", "cluster_name", "outbound_([0-9]+)_(.*)_(.*)$"), "port", "$1", "cluster_name", "outbound_([0-9]+)_(.*)_(.*)$"))`, upstreamSelector, rateRange, upstreamSelector, rateRange),
		"Latency",
		to,
	)
	if len(failureTopTable.Rows) == 0 && failureTopTableErr == nil {
		failureTopTable, failureTopTableErr = s.queryNativeDashboardExceptionRouteTopTable(ctx, from, to, "failed", 10)
	}
	if len(slowTopTable.Rows) == 0 && slowTopTableErr == nil {
		slowTopTable, slowTopTableErr = s.queryNativeDashboardExceptionRouteTopTable(ctx, from, to, "slow", 10)
	}
	rows = append(rows, response.NativeDashboardRow{
		Title:     "Exceptions",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardFloatStatPanel(13, "Downstream 5xx Count", "short", 0, 0, 6, downstream5xxCount, downstream5xxCountErr),
			makeNativeDashboardFloatStatPanel(14, "Upstream 5xx/Timeout Count", "short", 6, 0, 6, upstreamFailureCount, upstreamFailureCountErr),
			makeNativeDashboardTablePanel(15, "Failure Route TopN", 12, 0, 12, failureTopTable, failureTopTableErr),
			makeNativeDashboardTablePanel(16, "Slow Route TopN", 0, 1, 24, slowTopTable, slowTopTableErr),
		},
	})

	rows = append(rows, response.NativeDashboardRow{
		Title:     "Resource Scale",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			s.makeNativeDashboardResourceStatPanel(ctx, 17, "Routes", "routes", 0),
			s.makeNativeDashboardResourceStatPanel(ctx, 18, "Domains", "domains", 8),
			s.makeNativeDashboardResourceStatPanel(ctx, 19, "Plugins", "wasm-plugins", 16),
		},
	})

	return rows
}

func (s *Service) buildNativeAIDashboardRows(ctx context.Context, from, to int64) []response.NativeDashboardRow {
	selector := s.discoverNativeDashboardGatewaySelector(ctx)
	step := resolveNativeDashboardPrometheusStep(from, to)
	upstreamSelector := joinNativeDashboardPromSelector(
		selector,
		`cluster_name=~"outbound.*"`,
	)
	tokenRateRange := resolveNativeDashboardPrometheusRateRange(from, to)
	windowRange := formatNativeDashboardPromRange(from, to)
	rows := make([]response.NativeDashboardRow, 0, 4)

	generalSummary, generalSummaryErr := s.queryNativeDashboardAIGeneralSummary(ctx, from, to)
	upstreamSuccess, upstreamSuccessErr := s.queryNativeDashboardPrometheusScalar(ctx,
		fmt.Sprintf(`sum(increase(envoy_cluster_upstream_rq{%s, response_code_class="2xx"}[%s])) / clamp_min(sum(increase(envoy_cluster_upstream_rq_total{%s}[%s])), 1)`, upstreamSelector, windowRange, upstreamSelector, windowRange),
		to,
	)
	requestSuccessRate := 0.0
	if generalSummary.RequestCount > 0 {
		requestSuccessRate = float64(generalSummary.SuccessCount) / float64(generalSummary.RequestCount)
	}
	requestCountValue := generalSummary.RequestCount
	requestCountErr := generalSummaryErr
	if s.portalClient.DB() == nil || generalSummaryErr != nil || requestCountValue <= 0 {
		requestCountFallback, fallbackErr := s.queryNativeDashboardPrometheusCounterIncrease(ctx,
			`route_upstream_model_consumer_metric_request_count`,
			from,
			to,
		)
		if fallbackErr == nil && requestCountFallback > 0 {
			requestCountValue = int64(math.Round(requestCountFallback))
			requestCountErr = nil
		} else if requestCountErr == nil {
			requestCountErr = fallbackErr
		}
	}
	totalTokensValue := float64(generalSummary.TotalTokens)
	totalTokensErr := generalSummaryErr
	if s.portalClient.DB() == nil || generalSummaryErr != nil {
		totalTokensValue, totalTokensErr = s.queryNativeDashboardPrometheusCounterIncrease(ctx,
			`route_upstream_model_consumer_metric_total_token`,
			from,
			to,
		)
	}
	rows = append(rows, response.NativeDashboardRow{
		Title:     "AI Overview",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardStatPanel(1, "AI Request Count", "short", 0, 0, 5, requestCountValue, requestCountErr),
			makeNativeDashboardFloatStatPanel(2, "AI Request Success Rate", "percentunit", 5, 0, 5, requestSuccessRate, generalSummaryErr),
			makeNativeDashboardFloatStatPanel(3, "Total Tokens", "short", 10, 0, 5, totalTokensValue, totalTokensErr),
			makeNativeDashboardFloatStatPanel(4, "Estimated Cost", "currencyCny", 15, 0, 4, float64(generalSummary.CostMicroYuan)/1_000_000, generalSummaryErr),
			makeNativeDashboardFloatStatPanel(5, "Upstream Attempt Success Rate", "percentunit", 19, 0, 5, upstreamSuccess, upstreamSuccessErr),
		},
	})

	tokenPerSecond, tokenPerSecondErr := s.queryNativeDashboardPrometheusSeries(ctx,
		fmt.Sprintf(`sum(rate(route_upstream_model_consumer_metric_total_token[%s]))`, tokenRateRange),
		"Total Token",
		from,
		to,
		step,
	)
	cacheTokenPerSecond, cacheTokenPerSecondErr := s.queryNativeDashboardPrometheusTokenFamilySeries(
		ctx,
		from,
		to,
		step,
		map[string]string{
			"Cache Creation Token": fmt.Sprintf(`sum(rate(route_upstream_model_consumer_metric_cache_creation_input_token[%[1]s])) + sum(rate(route_upstream_model_consumer_metric_cache_creation_5m_input_token[%[1]s])) + sum(rate(route_upstream_model_consumer_metric_cache_creation_1h_input_token[%[1]s]))`, tokenRateRange),
			"Cache Read Token":     fmt.Sprintf(`sum(rate(route_upstream_model_consumer_metric_cache_read_input_token[%s]))`, tokenRateRange),
		},
	)
	imageTokenPerSecond, imageTokenPerSecondErr := s.queryNativeDashboardPrometheusTokenFamilySeries(
		ctx,
		from,
		to,
		step,
		map[string]string{
			"Input Image Token":  fmt.Sprintf(`sum(rate(route_upstream_model_consumer_metric_input_image_token[%s]))`, tokenRateRange),
			"Output Image Token": fmt.Sprintf(`sum(rate(route_upstream_model_consumer_metric_output_image_token[%s]))`, tokenRateRange),
		},
	)
	usageDetailTrend, usageDetailTrendErr := s.queryNativeDashboardUsageDetailTrend(ctx, from, to)
	usageBucketSeconds := resolveNativeDashboardUsageBucketSize(from, to).Seconds()
	if !nativeDashboardSeriesHasNonZeroPoints(tokenPerSecond) && usageDetailTrendErr == nil {
		tokenPerSecond = buildNativeDashboardUsageDetailSeries(
			usageDetailTrend,
			usageBucketSeconds,
			nativeDashboardUsageDetailSeriesSpec{
				Name: "Total Token",
				Pick: func(item nativeDashboardUsageDetailTrendRow) float64 {
					return float64(item.TotalTokens)
				},
			},
		)
		tokenPerSecondErr = nil
	}
	if !nativeDashboardSeriesHasNonZeroPoints(cacheTokenPerSecond) && usageDetailTrendErr == nil {
		cacheTokenPerSecond = buildNativeDashboardUsageDetailSeries(
			usageDetailTrend,
			usageBucketSeconds,
			nativeDashboardUsageDetailSeriesSpec{
				Name: "Cache Creation Token",
				Pick: func(item nativeDashboardUsageDetailTrendRow) float64 {
					return float64(item.CacheCreationTokens)
				},
			},
			nativeDashboardUsageDetailSeriesSpec{
				Name: "Cache Read Token",
				Pick: func(item nativeDashboardUsageDetailTrendRow) float64 {
					return float64(item.CacheReadTokens)
				},
			},
		)
		cacheTokenPerSecondErr = nil
	}
	if !nativeDashboardSeriesHasNonZeroPoints(imageTokenPerSecond) && usageDetailTrendErr == nil {
		imageTokenPerSecond = buildNativeDashboardUsageDetailSeries(
			usageDetailTrend,
			usageBucketSeconds,
			nativeDashboardUsageDetailSeriesSpec{
				Name: "Input Image Token",
				Pick: func(item nativeDashboardUsageDetailTrendRow) float64 {
					return float64(item.InputImageTokens)
				},
			},
			nativeDashboardUsageDetailSeriesSpec{
				Name: "Output Image Token",
				Pick: func(item nativeDashboardUsageDetailTrendRow) float64 {
					return float64(item.OutputImageTokens)
				},
			},
		)
		imageTokenPerSecondErr = nil
	}
	rows = append(rows, response.NativeDashboardRow{
		Title:     "Token Runtime",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardTimeseriesPanel(6, "Token Per Second", "ops", 0, 0, 8, tokenPerSecond, tokenPerSecondErr),
			makeNativeDashboardTimeseriesPanel(7, "Cache Token Per Second", "ops", 8, 0, 8, cacheTokenPerSecond, cacheTokenPerSecondErr),
			makeNativeDashboardTimeseriesPanel(8, "Image Token Per Second", "ops", 16, 0, 8, imageTokenPerSecond, imageTokenPerSecondErr),
		},
	})

	upstreamTrend, upstreamTrendErr := s.queryNativeDashboardPrometheusUpstreamServiceTrend(ctx, selector, from, to, step, 5)
	downstreamTrend, downstreamTrendErr := s.queryNativeDashboardDownstreamRequestTrend(ctx, from, to, 5)
	upstreamDurationSeries, upstreamDurationErr := s.queryNativeDashboardPrometheusLatencySeries(ctx, upstreamSelector, "upstream", from, to, step)
	downstreamDurationSeries, downstreamDurationErr := s.queryNativeDashboardPrometheusLatencySeries(ctx, selector, "downstream", from, to, step)
	rows = append(rows, response.NativeDashboardRow{
		Title:     "AI Request",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardTimeseriesPanel(9, "Downstream AI Route Request Trend", "reqps", 0, 0, 12, downstreamTrend, downstreamTrendErr),
			makeNativeDashboardTimeseriesPanel(10, "Upstream Provider/Service Request Trend", "reqps", 12, 0, 12, upstreamTrend, upstreamTrendErr),
			makeNativeDashboardTimeseriesPanel(11, "Downstream Latency", "ms", 0, 1, 12, downstreamDurationSeries, downstreamDurationErr),
			makeNativeDashboardTimeseriesPanel(12, "Upstream Latency", "ms", 12, 1, 12, upstreamDurationSeries, upstreamDurationErr),
		},
	})

	failedRequests, failedRequestsErr := s.queryNativeDashboardExceptionRecords(ctx, from, to, "failed", 10)
	slowRequests, slowRequestsErr := s.queryNativeDashboardExceptionRecords(ctx, from, to, "slow", 10)
	errorCodeTable, errorCodeTableErr := s.queryNativeDashboardAIErrorCodeTopTable(ctx, from, to, 10)
	rows = append(rows, response.NativeDashboardRow{
		Title:     "AI Exceptions",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardTablePanel(13, "Failed Requests", 0, 0, 24, buildNativeDashboardExceptionTable(failedRequests), failedRequestsErr),
			makeNativeDashboardTablePanel(14, "Slow Requests", 0, 1, 12, buildNativeDashboardExceptionTable(slowRequests), slowRequestsErr),
			makeNativeDashboardTablePanel(15, "Error Code TopN", 12, 1, 12, errorCodeTable, errorCodeTableErr),
		},
	})

	return rows
}

func (s *Service) buildNativeLegacyDashboardRows(ctx context.Context, dashboardType string, from, to int64) []response.NativeDashboardRow {
	summary, summaryErr := s.queryNativeDashboardUsageSummary(ctx, from, to)
	trend, trendErr := s.queryNativeDashboardUsageTrend(ctx, from, to)
	events, eventsErr := s.queryNativeDashboardRecentEvents(ctx, from, to, 8)

	rows := make([]response.NativeDashboardRow, 0, 2)
	rows = append(rows, response.NativeDashboardRow{
		Title:     "General",
		Collapsed: false,
		Panels: append([]response.NativeDashboardPanel{
			makeNativeDashboardStatPanel(1, "Requests", "short", 0, 0, 8, summary.RequestCount, summaryErr),
			makeNativeDashboardStatPanel(2, "Total Tokens", "short", 8, 0, 8, summary.TotalTokens, summaryErr),
			makeNativeDashboardStatPanel(3, "Active Consumers", "short", 16, 0, 8, summary.ActiveConsumers, summaryErr),
		}, s.buildNativeDashboardResourcePanels(ctx, dashboardType)...),
	})

	rows = append(rows, response.NativeDashboardRow{
		Title:     "Request",
		Collapsed: false,
		Panels: []response.NativeDashboardPanel{
			makeNativeDashboardTimeseriesPanel(7, "Requests Trend", "short", 0, 0, 12, buildNativeDashboardSeries("Requests", trend, func(item nativeDashboardTrendRow) float64 {
				return item.RequestCount
			}), trendErr),
			makeNativeDashboardTimeseriesPanel(8, "Tokens Trend", "short", 12, 0, 12, buildNativeDashboardSeries("Total Tokens", trend, func(item nativeDashboardTrendRow) float64 {
				return item.TotalTokens
			}), trendErr),
			makeNativeDashboardTablePanel(9, "Recent Usage Events", 0, 1, 24, buildNativeDashboardEventTable(events), eventsErr),
		},
	})

	return rows
}

func (s *Service) buildNativeDashboardResourcePanels(ctx context.Context, dashboardType string) []response.NativeDashboardPanel {
	var resources []nativeDashboardResourcePanel
	switch strings.ToUpper(strings.TrimSpace(dashboardType)) {
	case "AI":
		resources = []nativeDashboardResourcePanel{
			{Title: "AI Routes", Kind: "ai-routes"},
			{Title: "Providers", Kind: "ai-providers"},
			{Title: "MCP Servers", Kind: "mcp-servers"},
		}
	default:
		resources = []nativeDashboardResourcePanel{
			{Title: "Routes", Kind: "routes"},
			{Title: "Domains", Kind: "domains"},
			{Title: "Plugins", Kind: "wasm-plugins"},
		}
	}

	items := make([]response.NativeDashboardPanel, 0, len(resources))
	for index, resource := range resources {
		count, err := s.countNativeDashboardResources(ctx, resource.Kind)
		items = append(items, makeNativeDashboardStatPanel(
			4+index,
			resource.Title,
			"short",
			(index%3)*8,
			1,
			8,
			int64(count),
			err,
		))
	}
	return items
}

func (s *Service) makeNativeDashboardResourceStatPanel(ctx context.Context, id int, title, kind string, x int) response.NativeDashboardPanel {
	count, err := s.countNativeDashboardResources(ctx, kind)
	return makeNativeDashboardStatPanel(id, title, "short", x, 0, 8, int64(count), err)
}

func (s *Service) countNativeDashboardResources(ctx context.Context, kind string) (int, error) {
	if s.k8sClient == nil {
		return 0, nil
	}
	items, err := s.k8sClient.ListResources(ctx, kind)
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *Service) queryNativeDashboardUsageSummary(ctx context.Context, from, to int64) (nativeDashboardUsageSummary, error) {
	db := s.portalClient.DB()
	if db == nil {
		return nativeDashboardUsageSummary{}, nil
	}

	fromTime := nativeDashboardDBTimeFromMillis(from)
	toTime := nativeDashboardDBTimeFromMillis(to)

	var item nativeDashboardUsageSummary
	err := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(request_count), 0) AS request_count,
			COALESCE(SUM(
				CASE
					WHEN total_tokens > 0 THEN total_tokens
					ELSE input_tokens + output_tokens + cache_creation_input_tokens + cache_creation_5m_input_tokens +
						cache_creation_1h_input_tokens + cache_read_input_tokens + input_image_tokens + output_image_tokens
				END
			), 0) AS total_tokens,
			COALESCE(SUM(cost_micro_yuan), 0) AS cost_micro_yuan,
			COUNT(DISTINCT consumer_name) AS active_consumers
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?`,
		fromTime,
		toTime,
	).Scan(&item.RequestCount, &item.TotalTokens, &item.CostMicroYuan, &item.ActiveConsumers)
	if err != nil {
		return nativeDashboardUsageSummary{}, fmt.Errorf("query native dashboard usage summary: %w", err)
	}
	return item, nil
}

func (s *Service) queryNativeDashboardUsageTrend(ctx context.Context, from, to int64) ([]nativeDashboardTrendRow, error) {
	db := s.portalClient.DB()
	if db == nil {
		return []nativeDashboardTrendRow{}, nil
	}

	var (
		fromTime   = nativeDashboardDBTimeFromMillis(from)
		toTime     = nativeDashboardDBTimeFromMillis(to)
		rangeHours = toTime.Sub(fromTime).Hours()
		bucketExpr = "DATE_FORMAT(occurred_at, '%Y-%m-%d %H:00')"
	)
	if rangeHours > 72 {
		bucketExpr = "DATE_FORMAT(occurred_at, '%Y-%m-%d')"
	}

	statement := fmt.Sprintf(`
		SELECT %s AS bucket_label,
			COALESCE(SUM(request_count), 0) AS request_count,
			COALESCE(SUM(
				CASE
					WHEN total_tokens > 0 THEN total_tokens
					ELSE input_tokens + output_tokens + cache_creation_input_tokens + cache_creation_5m_input_tokens +
						cache_creation_1h_input_tokens + cache_read_input_tokens + input_image_tokens + output_image_tokens
				END
			), 0) AS total_tokens,
			COALESCE(SUM(cost_micro_yuan), 0) AS cost_micro_yuan
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?
		GROUP BY bucket_label
		ORDER BY bucket_label ASC`, bucketExpr)

	rows, err := db.QueryContext(ctx, statement, fromTime, toTime)
	if err != nil {
		return nil, fmt.Errorf("query native dashboard usage trend: %w", err)
	}
	defer rows.Close()

	items := make([]nativeDashboardTrendRow, 0)
	for rows.Next() {
		var item nativeDashboardTrendRow
		if err := rows.Scan(&item.BucketLabel, &item.RequestCount, &item.TotalTokens, &item.CostMicroYuan); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryNativeDashboardRecentEvents(ctx context.Context, from, to int64, limit int) ([]nativeDashboardRecentEvent, error) {
	db := s.portalClient.DB()
	if db == nil {
		return []nativeDashboardRecentEvent{}, nil
	}
	if limit <= 0 {
		limit = 8
	}

	rows, err := db.QueryContext(ctx, `
		SELECT occurred_at, consumer_name, route_name, model_id, request_status,
			COALESCE(
				CASE
					WHEN total_tokens > 0 THEN total_tokens
					ELSE input_tokens + output_tokens + cache_creation_input_tokens + cache_creation_5m_input_tokens +
						cache_creation_1h_input_tokens + cache_read_input_tokens + input_image_tokens + output_image_tokens
				END,
				0
			) AS total_tokens,
			COALESCE(cost_micro_yuan, 0) AS cost_micro_yuan
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?
		ORDER BY occurred_at DESC, id DESC
		LIMIT ?`,
		nativeDashboardDBTimeFromMillis(from),
		nativeDashboardDBTimeFromMillis(to),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query native dashboard recent events: %w", err)
	}
	defer rows.Close()

	items := make([]nativeDashboardRecentEvent, 0)
	for rows.Next() {
		var (
			item       nativeDashboardRecentEvent
			occurredAt sql.NullTime
		)
		if err := rows.Scan(
			&occurredAt,
			&item.ConsumerName,
			&item.RouteName,
			&item.ModelID,
			&item.RequestStatus,
			&item.TotalTokens,
			&item.CostMicroYuan,
		); err != nil {
			return nil, err
		}
		if occurredAt.Valid {
			item.OccurredAt = occurredAt.Time.Local().Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryNativeDashboardAIGeneralSummary(ctx context.Context, from, to int64) (nativeDashboardAIGeneralSummary, error) {
	db := s.portalClient.DB()
	if db == nil {
		return nativeDashboardAIGeneralSummary{}, nil
	}

	var item nativeDashboardAIGeneralSummary
	err := db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN request_count > 0 THEN request_count ELSE 1 END), 0) AS request_count,
			COALESCE(SUM(
				CASE
					WHEN http_status >= 200 AND http_status < 300 THEN
						CASE WHEN request_count > 0 THEN request_count ELSE 1 END
					ELSE 0
				END
			), 0) AS success_count,
			COALESCE(SUM(
				CASE
					WHEN total_tokens > 0 THEN total_tokens
					ELSE input_tokens + output_tokens + cache_creation_input_tokens + cache_creation_5m_input_tokens +
						cache_creation_1h_input_tokens + cache_read_input_tokens + input_image_tokens + output_image_tokens
				END
			), 0) AS total_tokens,
			COALESCE(SUM(cost_micro_yuan), 0) AS cost_micro_yuan
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?`,
		nativeDashboardDBTimeFromMillis(from),
		nativeDashboardDBTimeFromMillis(to),
	).Scan(&item.RequestCount, &item.SuccessCount, &item.TotalTokens, &item.CostMicroYuan)
	if err != nil {
		return nativeDashboardAIGeneralSummary{}, fmt.Errorf("query native dashboard AI general summary: %w", err)
	}
	return item, nil
}

func (s *Service) queryNativeDashboardDownstreamRequestTrend(ctx context.Context, from, to int64, limit int) ([]response.NativeDashboardSeries, error) {
	db := s.portalClient.DB()
	if db == nil {
		return []response.NativeDashboardSeries{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	bucketExpr := resolveNativeDashboardUsageBucketExpr(from, to)
	statement := fmt.Sprintf(`
		SELECT %s AS bucket_label,
			COALESCE(NULLIF(TRIM(request_path), ''), NULLIF(TRIM(route_name), ''), '-') AS dimension_value,
			COALESCE(SUM(CASE WHEN request_count > 0 THEN request_count ELSE 1 END), 0) AS weighted_request
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?
		GROUP BY bucket_label, dimension_value
		ORDER BY bucket_label ASC, dimension_value ASC`, bucketExpr)

	rows, err := db.QueryContext(ctx, statement, nativeDashboardDBTimeFromMillis(from), nativeDashboardDBTimeFromMillis(to))
	if err != nil {
		return nil, fmt.Errorf("query native dashboard downstream request trend: %w", err)
	}
	defer rows.Close()

	items := make([]nativeDashboardRequestTrendDimensionRow, 0)
	for rows.Next() {
		var item nativeDashboardRequestTrendDimensionRow
		if err := rows.Scan(&item.BucketLabel, &item.DimensionValue, &item.WeightedRequest); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return buildNativeDashboardDimensionSeries(items, limit, resolveNativeDashboardUsageBucketSize(from, to).Seconds()), nil
}

func (s *Service) queryNativeDashboardExceptionRecords(ctx context.Context, from, to int64, mode string, limit int) ([]nativeDashboardExceptionRecord, error) {
	db := s.portalClient.DB()
	if db == nil {
		return []nativeDashboardExceptionRecord{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	var (
		filterClause string
		orderClause  string
	)
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "slow":
		filterClause = `started_at IS NOT NULL AND finished_at IS NOT NULL AND TIMESTAMPDIFF(MICROSECOND, started_at, finished_at) > 5000000`
		orderClause = `service_duration_ms DESC, occurred_at DESC, id DESC`
	default:
		filterClause = `(http_status < 200 OR http_status >= 300 OR request_status <> 'success')`
		orderClause = `occurred_at DESC, id DESC`
	}

	query := fmt.Sprintf(`
		SELECT occurred_at, request_id, trace_id, consumer_name,
			COALESCE(NULLIF(TRIM(request_path), ''), NULLIF(TRIM(route_name), ''), '-') AS request_path,
			route_name, model_id, request_status, http_status, error_code, error_message,
			COALESCE(
				CASE
					WHEN total_tokens > 0 THEN total_tokens
					ELSE input_tokens + output_tokens + cache_creation_input_tokens + cache_creation_5m_input_tokens +
						cache_creation_1h_input_tokens + cache_read_input_tokens + input_image_tokens + output_image_tokens
				END,
				0
			) AS total_tokens,
			COALESCE(cost_micro_yuan, 0) AS cost_micro_yuan,
			CASE
				WHEN started_at IS NOT NULL AND finished_at IS NOT NULL THEN
					GREATEST(TIMESTAMPDIFF(MICROSECOND, started_at, finished_at), 0) DIV 1000
				ELSE 0
			END AS service_duration_ms
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ? AND %s
		ORDER BY %s
		LIMIT ?`, filterClause, orderClause)

	rows, err := db.QueryContext(ctx, query, nativeDashboardDBTimeFromMillis(from), nativeDashboardDBTimeFromMillis(to), limit)
	if err != nil {
		return nil, fmt.Errorf("query native dashboard %s requests: %w", mode, err)
	}
	defer rows.Close()

	items := make([]nativeDashboardExceptionRecord, 0)
	for rows.Next() {
		var (
			item       nativeDashboardExceptionRecord
			occurredAt sql.NullTime
			httpStatus sql.NullInt64
		)
		if err := rows.Scan(
			&occurredAt,
			&item.RequestID,
			&item.TraceID,
			&item.ConsumerName,
			&item.RequestPath,
			&item.RouteName,
			&item.ModelID,
			&item.RequestStatus,
			&httpStatus,
			&item.ErrorCode,
			&item.ErrorMessage,
			&item.TotalTokens,
			&item.CostMicroYuan,
			&item.ServiceDurationMs,
		); err != nil {
			return nil, err
		}
		if occurredAt.Valid {
			item.OccurredAt = occurredAt.Time.Local().Format(time.RFC3339)
		}
		if httpStatus.Valid {
			item.HTTPStatus = int(httpStatus.Int64)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryNativeDashboardAIErrorCodeTopTable(ctx context.Context, from, to int64, limit int) (response.NativeDashboardTable, error) {
	db := s.portalClient.DB()
	if db == nil {
		return response.NativeDashboardTable{Columns: []response.NativeDashboardTableColumn{}, Rows: []map[string]any{}}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	rows, err := db.QueryContext(ctx, `
		SELECT COALESCE(NULLIF(TRIM(error_code), ''), 'unknown') AS error_code,
			COALESCE(SUM(CASE WHEN request_count > 0 THEN request_count ELSE 1 END), 0) AS request_count
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?
			AND (http_status < 200 OR http_status >= 300 OR request_status <> 'success')
		GROUP BY error_code
		ORDER BY request_count DESC, error_code ASC
		LIMIT ?`,
		nativeDashboardDBTimeFromMillis(from),
		nativeDashboardDBTimeFromMillis(to),
		limit,
	)
	if err != nil {
		return response.NativeDashboardTable{}, fmt.Errorf("query native dashboard AI error top table: %w", err)
	}
	defer rows.Close()

	items := make([]map[string]any, 0, limit)
	for rows.Next() {
		var (
			errorCode    string
			requestCount int64
		)
		if err := rows.Scan(&errorCode, &requestCount); err != nil {
			return response.NativeDashboardTable{}, err
		}
		items = append(items, map[string]any{
			"errorCode":    errorCode,
			"requestCount": requestCount,
		})
	}
	if err := rows.Err(); err != nil {
		return response.NativeDashboardTable{}, err
	}

	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: "errorCode", Title: "Error Code"},
			{Key: "requestCount", Title: "Request Count"},
		},
		Rows: items,
	}, nil
}

func (s *Service) queryNativeDashboardExceptionRouteTopTable(ctx context.Context, from, to int64, mode string, limit int) (response.NativeDashboardTable, error) {
	db := s.portalClient.DB()
	if db == nil {
		return response.NativeDashboardTable{Columns: []response.NativeDashboardTableColumn{}, Rows: []map[string]any{}}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	var (
		filterClause string
		valueExpr    string
		valueKey     string
		valueTitle   string
		orderClause  string
	)
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "slow":
		filterClause = `started_at IS NOT NULL AND finished_at IS NOT NULL AND TIMESTAMPDIFF(MICROSECOND, started_at, finished_at) > 5000000`
		valueExpr = `MAX(GREATEST(TIMESTAMPDIFF(MICROSECOND, started_at, finished_at), 0) DIV 1000)`
		valueKey = "latencyMs"
		valueTitle = "Latency"
		orderClause = "latency_ms DESC, route ASC"
	default:
		filterClause = `(http_status < 200 OR http_status >= 300 OR request_status <> 'success')`
		valueExpr = `COALESCE(SUM(CASE WHEN request_count > 0 THEN request_count ELSE 1 END), 0)`
		valueKey = "requestCount"
		valueTitle = "Request Count"
		orderClause = "request_count DESC, route ASC"
	}

	statement := fmt.Sprintf(`
		SELECT
			COALESCE(NULLIF(TRIM(request_path), ''), NULLIF(TRIM(route_name), ''), '-') AS route,
			%s AS %s
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ? AND %s
		GROUP BY route
		ORDER BY %s
		LIMIT ?`, valueExpr, toSnakeCase(valueKey), filterClause, orderClause)

	rows, err := db.QueryContext(ctx, statement, nativeDashboardDBTimeFromMillis(from), nativeDashboardDBTimeFromMillis(to), limit)
	if err != nil {
		return response.NativeDashboardTable{}, fmt.Errorf("query native dashboard %s route top table: %w", mode, err)
	}
	defer rows.Close()

	items := make([]map[string]any, 0, limit)
	for rows.Next() {
		var (
			route string
			value float64
		)
		if err := rows.Scan(&route, &value); err != nil {
			return response.NativeDashboardTable{}, err
		}
		items = append(items, map[string]any{
			"route":  route,
			valueKey: value,
		})
	}
	if err := rows.Err(); err != nil {
		return response.NativeDashboardTable{}, err
	}

	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: "route", Title: "Route"},
			{Key: valueKey, Title: valueTitle},
		},
		Rows: items,
	}, nil
}

func buildNativeDashboardDimensionSeries(rows []nativeDashboardRequestTrendDimensionRow, limit int, bucketSeconds float64) []response.NativeDashboardSeries {
	if bucketSeconds <= 0 {
		bucketSeconds = 1
	}
	totalByDimension := make(map[string]int64)
	pointsByDimension := make(map[string][]response.NativeDashboardPoint)
	for _, item := range rows {
		if strings.TrimSpace(item.DimensionValue) == "" {
			continue
		}
		bucketTime, ok := parseNativeDashboardBucket(item.BucketLabel)
		if !ok {
			continue
		}
		totalByDimension[item.DimensionValue] += item.WeightedRequest
		pointsByDimension[item.DimensionValue] = append(pointsByDimension[item.DimensionValue], response.NativeDashboardPoint{
			Time:  bucketTime.UnixMilli(),
			Value: float64(item.WeightedRequest) / bucketSeconds,
		})
	}

	dimensions := make([]string, 0, len(totalByDimension))
	for dimension := range totalByDimension {
		dimensions = append(dimensions, dimension)
	}
	sort.Slice(dimensions, func(i, j int) bool {
		if totalByDimension[dimensions[i]] == totalByDimension[dimensions[j]] {
			return dimensions[i] < dimensions[j]
		}
		return totalByDimension[dimensions[i]] > totalByDimension[dimensions[j]]
	})

	items := make([]response.NativeDashboardSeries, 0, minInt(limit, len(dimensions)))
	for _, dimension := range dimensions[:minInt(limit, len(dimensions))] {
		items = append(items, response.NativeDashboardSeries{
			Name:   dimension,
			Labels: map[string]string{"dimension": dimension},
			Points: pointsByDimension[dimension],
		})
	}
	return items
}

type nativeDashboardUsageDetailTrendRow struct {
	BucketLabel         string
	RequestCount        int64
	InputTokens         int64
	OutputTokens        int64
	TotalTokens         int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	InputImageTokens    int64
	OutputImageTokens   int64
	InputImageCount     int64
	OutputImageCount    int64
}

type nativeDashboardUsageDetailSeriesSpec struct {
	Name string
	Pick func(nativeDashboardUsageDetailTrendRow) float64
}

type nativeDashboardUsageModelTrendRow struct {
	BucketLabel  string
	ModelID      string
	InputTokens  int64
	OutputTokens int64
	TotalTokens  int64
}

type nativeDashboardPromQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string                           `json:"resultType"`
		Result     []nativeDashboardPromQueryResult `json:"result"`
	} `json:"data"`
}

type nativeDashboardPromQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []any             `json:"value"`
	Values [][]any           `json:"values"`
}

func (s *Service) queryNativeDashboardUsageDetailTrend(ctx context.Context, from, to int64) ([]nativeDashboardUsageDetailTrendRow, error) {
	db := s.portalClient.DB()
	if db == nil {
		return []nativeDashboardUsageDetailTrendRow{}, nil
	}

	bucketExpr := resolveNativeDashboardUsageBucketExpr(from, to)
	statement := fmt.Sprintf(`
		SELECT %s AS bucket_label,
			COALESCE(SUM(request_count), 0) AS request_count,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(cache_creation_input_tokens), 0) +
				COALESCE(SUM(cache_creation_5m_input_tokens), 0) +
				COALESCE(SUM(cache_creation_1h_input_tokens), 0) AS cache_creation_tokens,
			COALESCE(SUM(cache_read_input_tokens), 0) AS cache_read_tokens,
			COALESCE(SUM(input_image_tokens), 0) AS input_image_tokens,
			COALESCE(SUM(output_image_tokens), 0) AS output_image_tokens,
			COALESCE(SUM(input_image_count), 0) AS input_image_count,
			COALESCE(SUM(output_image_count), 0) AS output_image_count
		FROM billing_usage_event
		WHERE request_status = 'success'
			AND usage_status = 'parsed'
			AND occurred_at >= ?
			AND occurred_at < ?
		GROUP BY bucket_label
		ORDER BY bucket_label ASC`, bucketExpr)

	rows, err := db.QueryContext(ctx, statement, nativeDashboardDBTimeFromMillis(from), nativeDashboardDBTimeFromMillis(to))
	if err != nil {
		return nil, fmt.Errorf("query native dashboard usage detail trend: %w", err)
	}
	defer rows.Close()

	items := make([]nativeDashboardUsageDetailTrendRow, 0)
	for rows.Next() {
		var item nativeDashboardUsageDetailTrendRow
		if err := rows.Scan(
			&item.BucketLabel,
			&item.RequestCount,
			&item.InputTokens,
			&item.OutputTokens,
			&item.TotalTokens,
			&item.CacheCreationTokens,
			&item.CacheReadTokens,
			&item.InputImageTokens,
			&item.OutputImageTokens,
			&item.InputImageCount,
			&item.OutputImageCount,
		); err != nil {
			return nil, err
		}
		if item.TotalTokens <= 0 {
			item.TotalTokens = item.InputTokens + item.OutputTokens + item.CacheCreationTokens + item.CacheReadTokens +
				item.InputImageTokens + item.OutputImageTokens
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) queryNativeDashboardUsageModelTrend(ctx context.Context, from, to int64, limit int) ([]nativeDashboardUsageModelTrendRow, error) {
	db := s.portalClient.DB()
	if db == nil {
		return []nativeDashboardUsageModelTrendRow{}, nil
	}
	if limit <= 0 {
		limit = 5
	}

	bucketExpr := resolveNativeDashboardUsageBucketExpr(from, to)
	statement := fmt.Sprintf(`
		SELECT %s AS bucket_label,
			model_id,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		FROM billing_usage_event
		WHERE request_status = 'success'
			AND usage_status = 'parsed'
			AND occurred_at >= ?
			AND occurred_at < ?
			AND model_id <> ''
		GROUP BY bucket_label, model_id
		ORDER BY bucket_label ASC, model_id ASC`, bucketExpr)

	rows, err := db.QueryContext(ctx, statement, nativeDashboardDBTimeFromMillis(from), nativeDashboardDBTimeFromMillis(to))
	if err != nil {
		return nil, fmt.Errorf("query native dashboard usage model trend: %w", err)
	}
	defer rows.Close()

	allRows := make([]nativeDashboardUsageModelTrendRow, 0)
	totalByModel := map[string]int64{}
	for rows.Next() {
		var item nativeDashboardUsageModelTrendRow
		if err := rows.Scan(&item.BucketLabel, &item.ModelID, &item.InputTokens, &item.OutputTokens, &item.TotalTokens); err != nil {
			return nil, err
		}
		if item.TotalTokens <= 0 {
			item.TotalTokens = item.InputTokens + item.OutputTokens
		}
		totalByModel[item.ModelID] += item.TotalTokens
		allRows = append(allRows, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	models := make([]string, 0, len(totalByModel))
	for modelID := range totalByModel {
		models = append(models, modelID)
	}
	sort.Slice(models, func(i, j int) bool {
		if totalByModel[models[i]] == totalByModel[models[j]] {
			return models[i] < models[j]
		}
		return totalByModel[models[i]] > totalByModel[models[j]]
	})
	allow := map[string]struct{}{}
	for _, modelID := range models[:minInt(limit, len(models))] {
		allow[modelID] = struct{}{}
	}

	items := make([]nativeDashboardUsageModelTrendRow, 0, len(allRows))
	for _, item := range allRows {
		if _, ok := allow[item.ModelID]; ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *Service) queryNativeDashboardUsageTable(ctx context.Context, from, to int64, dimension string, limit int) (response.NativeDashboardTable, error) {
	db := s.portalClient.DB()
	if db == nil {
		return response.NativeDashboardTable{Columns: []response.NativeDashboardTableColumn{}, Rows: []map[string]any{}}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	column := "consumer_name"
	title := "Consumer"
	switch strings.ToLower(strings.TrimSpace(dimension)) {
	case "model":
		column = "model_id"
		title = "Model"
	}

	statement := fmt.Sprintf(`
		SELECT %s AS dimension_value,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		FROM billing_usage_event
		WHERE request_status = 'success'
			AND usage_status = 'parsed'
			AND occurred_at >= ?
			AND occurred_at < ?
			AND %s <> ''
		GROUP BY dimension_value
		ORDER BY total_tokens DESC, dimension_value ASC
		LIMIT ?`, column, column)

	rows, err := db.QueryContext(ctx, statement, nativeDashboardDBTimeFromMillis(from), nativeDashboardDBTimeFromMillis(to), limit)
	if err != nil {
		return response.NativeDashboardTable{}, fmt.Errorf("query native dashboard %s usage table: %w", dimension, err)
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var (
			name         string
			inputTokens  int64
			outputTokens int64
			totalTokens  int64
		)
		if err := rows.Scan(&name, &inputTokens, &outputTokens, &totalTokens); err != nil {
			return response.NativeDashboardTable{}, err
		}
		if totalTokens <= 0 {
			totalTokens = inputTokens + outputTokens
		}
		items = append(items, map[string]any{
			strings.ToLower(title): name,
			"inputTokens":          inputTokens,
			"outputTokens":         outputTokens,
			"totalTokens":          totalTokens,
		})
	}
	if err := rows.Err(); err != nil {
		return response.NativeDashboardTable{}, err
	}

	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: strings.ToLower(title), Title: title},
			{Key: "inputTokens", Title: "Input Token"},
			{Key: "outputTokens", Title: "Output Token"},
			{Key: "totalTokens", Title: "Total Token"},
		},
		Rows: items,
	}, nil
}

func (s *Service) queryNativeDashboardProviderUsage(ctx context.Context, from, to int64, limit int) (response.NativeDashboardTable, error) {
	db := s.portalClient.DB()
	if db == nil {
		return response.NativeDashboardTable{Columns: []response.NativeDashboardTableColumn{}, Rows: []map[string]any{}}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	routeProviders := s.nativeDashboardRouteProviders(ctx)
	rows, err := db.QueryContext(ctx, `
		SELECT route_name,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens
		FROM billing_usage_event
		WHERE request_status = 'success'
			AND usage_status = 'parsed'
			AND occurred_at >= ?
			AND occurred_at < ?
			AND route_name <> ''
		GROUP BY route_name
		ORDER BY total_tokens DESC, route_name ASC`,
		nativeDashboardDBTimeFromMillis(from),
		nativeDashboardDBTimeFromMillis(to),
	)
	if err != nil {
		return response.NativeDashboardTable{}, fmt.Errorf("query native dashboard provider usage: %w", err)
	}
	defer rows.Close()

	type providerTotals struct {
		InputTokens  int64
		OutputTokens int64
		TotalTokens  int64
	}
	totalByProvider := map[string]providerTotals{}
	for rows.Next() {
		var (
			routeName    string
			inputTokens  int64
			outputTokens int64
			totalTokens  int64
		)
		if err := rows.Scan(&routeName, &inputTokens, &outputTokens, &totalTokens); err != nil {
			return response.NativeDashboardTable{}, err
		}
		if totalTokens <= 0 {
			totalTokens = inputTokens + outputTokens
		}
		provider := firstNonEmpty(routeProviders[routeName], routeName)
		current := totalByProvider[provider]
		current.InputTokens += inputTokens
		current.OutputTokens += outputTokens
		current.TotalTokens += totalTokens
		totalByProvider[provider] = current
	}
	if err := rows.Err(); err != nil {
		return response.NativeDashboardTable{}, err
	}

	providers := make([]string, 0, len(totalByProvider))
	for provider := range totalByProvider {
		providers = append(providers, provider)
	}
	sort.Slice(providers, func(i, j int) bool {
		if totalByProvider[providers[i]].TotalTokens == totalByProvider[providers[j]].TotalTokens {
			return providers[i] < providers[j]
		}
		return totalByProvider[providers[i]].TotalTokens > totalByProvider[providers[j]].TotalTokens
	})

	items := make([]map[string]any, 0, minInt(limit, len(providers)))
	for _, provider := range providers[:minInt(limit, len(providers))] {
		item := totalByProvider[provider]
		items = append(items, map[string]any{
			"provider":     provider,
			"inputTokens":  item.InputTokens,
			"outputTokens": item.OutputTokens,
			"totalTokens":  item.TotalTokens,
		})
	}

	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: "provider", Title: "Provider"},
			{Key: "inputTokens", Title: "Input Token"},
			{Key: "outputTokens", Title: "Output Token"},
			{Key: "totalTokens", Title: "Total Token"},
		},
		Rows: items,
	}, nil
}

func (s *Service) nativeDashboardRouteProviders(ctx context.Context) map[string]string {
	items := map[string]string{}
	if s.k8sClient == nil {
		return items
	}
	routes, err := s.k8sClient.ListResources(ctx, "ai-routes")
	if err != nil {
		return items
	}
	for _, route := range routes {
		name := strings.TrimSpace(fmt.Sprint(route["name"]))
		if name == "" {
			continue
		}
		upstreams, _ := route["upstreams"].([]map[string]any)
		if len(upstreams) == 0 {
			if raw, ok := route["upstreams"].([]any); ok {
				for _, item := range raw {
					if upstream, ok := item.(map[string]any); ok {
						upstreams = append(upstreams, upstream)
					}
				}
			}
		}
		for _, upstream := range upstreams {
			provider := strings.TrimSpace(fmt.Sprint(upstream["provider"]))
			if provider != "" {
				items[name] = provider
				break
			}
		}
	}
	return items
}

func (s *Service) queryNativeDashboardPrometheusScalar(ctx context.Context, query string, at int64) (float64, error) {
	results, err := s.queryNativeDashboardPrometheusInstant(ctx, query, at)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return nativeDashboardPromSampleValue(results[0].Value)
}

func (s *Service) queryNativeDashboardPrometheusSeries(ctx context.Context, query, name string, from, to int64, step time.Duration) ([]response.NativeDashboardSeries, error) {
	results, err := s.queryNativeDashboardPrometheusRange(ctx, query, from, to, step)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return []response.NativeDashboardSeries{}, nil
	}
	return []response.NativeDashboardSeries{{
		Name:   name,
		Labels: map[string]string{},
		Points: nativeDashboardPromPoints(results[0].Values),
	}}, nil
}

func (s *Service) queryNativeDashboardPrometheusLabeledSeries(ctx context.Context, query, labelKey string, from, to int64, step time.Duration) ([]response.NativeDashboardSeries, error) {
	results, err := s.queryNativeDashboardPrometheusRange(ctx, query, from, to, step)
	if err != nil {
		return nil, err
	}
	items := make([]response.NativeDashboardSeries, 0, len(results))
	for _, item := range results {
		name := firstNonEmpty(strings.TrimSpace(item.Metric[labelKey]), strings.TrimSpace(item.Metric["__name__"]), "Series")
		items = append(items, response.NativeDashboardSeries{
			Name:   name,
			Labels: item.Metric,
			Points: nativeDashboardPromPoints(item.Values),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (s *Service) queryNativeDashboardPrometheusCounterIncrease(ctx context.Context, metricExpr string, from, to int64) (float64, error) {
	query := fmt.Sprintf(`sum(increase(%s[%s]))`, metricExpr, formatNativeDashboardPromRange(from, to))
	return s.queryNativeDashboardPrometheusScalar(ctx, query, to)
}

func joinNativeDashboardPromSelector(parts ...string) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}
	return strings.Join(items, ", ")
}

func (s *Service) queryNativeDashboardPrometheusTokenFamilySeries(
	ctx context.Context,
	from, to int64,
	step time.Duration,
	queryBySeries map[string]string,
) ([]response.NativeDashboardSeries, error) {
	names := make([]string, 0, len(queryBySeries))
	for name := range queryBySeries {
		names = append(names, name)
	}
	sort.Strings(names)

	items := make([]response.NativeDashboardSeries, 0, len(names))
	for _, name := range names {
		series, err := s.queryNativeDashboardPrometheusSeries(ctx, queryBySeries[name], name, from, to, step)
		if err != nil {
			return nil, err
		}
		items = append(items, series...)
	}
	return items, nil
}

func (s *Service) queryNativeDashboardPrometheusLatencySeries(
	ctx context.Context,
	selector string,
	direction string,
	from, to int64,
	step time.Duration,
) ([]response.NativeDashboardSeries, error) {
	var metricPrefix string
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "downstream":
		metricPrefix = "envoy_http_downstream_rq_time"
	default:
		metricPrefix = "envoy_cluster_upstream_rq_time"
	}
	rateRange := resolveNativeDashboardPrometheusRateRange(from, to)

	p50, p50Err := s.queryNativeDashboardPrometheusSeries(ctx,
		fmt.Sprintf(`histogram_quantile(0.50, sum(rate(%s_bucket{%s}[%s])) by (le))`, metricPrefix, selector, rateRange),
		"P50",
		from,
		to,
		step,
	)
	p90, p90Err := s.queryNativeDashboardPrometheusSeries(ctx,
		fmt.Sprintf(`histogram_quantile(0.90, sum(rate(%s_bucket{%s}[%s])) by (le))`, metricPrefix, selector, rateRange),
		"P90",
		from,
		to,
		step,
	)
	p99, p99Err := s.queryNativeDashboardPrometheusSeries(ctx,
		fmt.Sprintf(`histogram_quantile(0.99, sum(rate(%s_bucket{%s}[%s])) by (le))`, metricPrefix, selector, rateRange),
		"P99",
		from,
		to,
		step,
	)
	return joinNativeDashboardSeriesErrors(append(append(p50, p90...), p99...), p50Err, p90Err, p99Err)
}

func (s *Service) queryNativeDashboardPrometheusUpstreamServiceTrend(
	ctx context.Context,
	selector string,
	from, to int64,
	step time.Duration,
	limit int,
) ([]response.NativeDashboardSeries, error) {
	rateRange := resolveNativeDashboardPrometheusRateRange(from, to)
	results, err := s.queryNativeDashboardPrometheusRange(
		ctx,
		fmt.Sprintf(`sum(rate(envoy_cluster_upstream_rq_total{%s}[%s])) by (cluster_name)`, selector, rateRange),
		from,
		to,
		step,
	)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}

	totalByService := make(map[string]float64)
	pointsByService := make(map[string][]response.NativeDashboardPoint)
	for _, item := range results {
		service, _ := parseNativeDashboardClusterName(item.Metric["cluster_name"])
		service = strings.TrimSpace(service)
		if service == "" || service == "-" || isNativeDashboardInfraService(service) {
			continue
		}
		points := nativeDashboardPromPoints(item.Values)
		if !nativeDashboardPointsHaveNonZeroValues(points) {
			continue
		}
		for _, point := range points {
			totalByService[service] += point.Value
		}
		pointsByService[service] = mergeNativeDashboardPoints(pointsByService[service], points)
	}

	services := make([]string, 0, len(totalByService))
	for service := range totalByService {
		services = append(services, service)
	}
	sort.Slice(services, func(i, j int) bool {
		if totalByService[services[i]] == totalByService[services[j]] {
			return services[i] < services[j]
		}
		return totalByService[services[i]] > totalByService[services[j]]
	})

	items := make([]response.NativeDashboardSeries, 0, minInt(limit, len(services)))
	for _, service := range services[:minInt(limit, len(services))] {
		items = append(items, response.NativeDashboardSeries{
			Name:   service,
			Labels: map[string]string{"service": service},
			Points: pointsByService[service],
		})
	}
	return items, nil
}

func (s *Service) queryNativeDashboardPrometheusTopTable(ctx context.Context, query, valueTitle string, at int64) (response.NativeDashboardTable, error) {
	results, err := s.queryNativeDashboardPrometheusInstant(ctx, query, at)
	if err != nil {
		return response.NativeDashboardTable{}, err
	}

	rows := make([]map[string]any, 0, len(results))
	for _, item := range results {
		value, err := nativeDashboardPromSampleValue(item.Value)
		if err != nil {
			return response.NativeDashboardTable{}, err
		}
		service, port := parseNativeDashboardClusterName(item.Metric["cluster_name"])
		rows = append(rows, map[string]any{
			"service":                   service,
			"port":                      port,
			strings.ToLower(valueTitle): value,
		})
	}

	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: "service", Title: "service"},
			{Key: "port", Title: "port"},
			{Key: strings.ToLower(valueTitle), Title: valueTitle},
		},
		Rows: rows,
	}, nil
}

func (s *Service) discoverNativeDashboardGatewaySelector(ctx context.Context) string {
	results, err := s.queryNativeDashboardPrometheusInstant(ctx, "envoy_server_live", time.Now().UnixMilli())
	if err == nil && len(results) > 0 {
		if app := strings.TrimSpace(results[0].Metric["app"]); app != "" {
			return fmt.Sprintf(`app="%s"`, escapeNativeDashboardPromLabelValue(app))
		}
		if gateway := strings.TrimSpace(results[0].Metric["aigateway"]); gateway != "" {
			return fmt.Sprintf(`aigateway="%s"`, escapeNativeDashboardPromLabelValue(gateway))
		}
	}
	return `app="aigateway-gateway"`
}

func (s *Service) queryNativeDashboardPrometheusInstant(ctx context.Context, query string, at int64) ([]nativeDashboardPromQueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("time", strconv.FormatFloat(float64(at)/1000, 'f', 3, 64))
	return s.queryNativeDashboardPrometheus(ctx, "/api/v1/query", params)
}

func (s *Service) queryNativeDashboardPrometheusRange(ctx context.Context, query string, from, to int64, step time.Duration) ([]nativeDashboardPromQueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", strconv.FormatFloat(float64(from)/1000, 'f', 3, 64))
	params.Set("end", strconv.FormatFloat(float64(to)/1000, 'f', 3, 64))
	params.Set("step", strconv.FormatFloat(step.Seconds(), 'f', 0, 64))
	return s.queryNativeDashboardPrometheus(ctx, "/api/v1/query_range", params)
}

func (s *Service) queryNativeDashboardPrometheus(ctx context.Context, path string, params url.Values) ([]nativeDashboardPromQueryResult, error) {
	baseURL := resolveNativeDashboardPrometheusBaseURL()
	if baseURL == "" {
		return nil, fmt.Errorf("prometheus is not configured")
	}

	requestURL, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus url: %w", err)
	}
	requestURL.RawQuery = params.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, err
	}
	response, err := (&http.Client{Timeout: 8 * time.Second}).Do(request)
	if err != nil {
		return nil, fmt.Errorf("query prometheus: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("query prometheus: %s", strings.TrimSpace(string(body)))
	}

	var payload nativeDashboardPromQueryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode prometheus response: %w", err)
	}
	return payload.Data.Result, nil
}

func resolveNativeDashboardPrometheusBaseURL() string {
	if raw := strings.TrimSpace(firstNonEmpty(
		os.Getenv("HIGRESS_CONSOLE_DASHBOARD_DATASOURCE_PROM_URL"),
		os.Getenv("AIGATEWAY_CONSOLE_PROMETHEUS_BASE_URL"),
	)); raw != "" {
		return strings.TrimRight(raw, "/")
	}

	service := strings.TrimSpace(os.Getenv("AIGATEWAY_CONSOLE_PROMETHEUS_SERVICE"))
	if service == "" {
		return ""
	}
	scheme := firstNonEmpty(os.Getenv("AIGATEWAY_CONSOLE_PROMETHEUS_SCHEME"), "http")
	namespace := firstNonEmpty(os.Getenv("AIGATEWAY_CONSOLE_NAMESPACE"), "aigateway-system")
	clusterDomain := firstNonEmpty(os.Getenv("AIGATEWAY_CONSOLE_CLUSTER_DOMAIN"), "cluster.local")
	port := firstNonEmpty(os.Getenv("AIGATEWAY_CONSOLE_PROMETHEUS_PORT"), "9090")
	path := firstNonEmpty(os.Getenv("AIGATEWAY_CONSOLE_PROMETHEUS_PATH"), "/prometheus")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s://%s.%s.svc.%s:%s%s", scheme, service, namespace, clusterDomain, port, path)
}

func resolveNativeDashboardPrometheusStep(from, to int64) time.Duration {
	window := time.UnixMilli(to).Sub(time.UnixMilli(from))
	if window <= 0 {
		return time.Minute
	}
	rawStep := window / 240
	switch {
	case rawStep <= time.Minute:
		return time.Minute
	case rawStep <= 5*time.Minute:
		return 5 * time.Minute
	case rawStep <= 15*time.Minute:
		return 15 * time.Minute
	case rawStep <= time.Hour:
		return time.Hour
	case rawStep <= 6*time.Hour:
		return 6 * time.Hour
	case rawStep <= 24*time.Hour:
		return 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}

func resolveNativeDashboardPrometheusRateRange(from, to int64) string {
	window := time.UnixMilli(to).Sub(time.UnixMilli(from))
	if window <= 0 {
		return "5m"
	}

	step := resolveNativeDashboardPrometheusStep(from, to)
	rateWindow := maxDuration(5*time.Minute, step*4)
	if rateWindow > window {
		rateWindow = window
	}
	if rateWindow < time.Minute {
		rateWindow = time.Minute
	}
	return fmt.Sprintf("%ds", int(math.Ceil(rateWindow.Seconds())))
}

func formatNativeDashboardPromRange(from, to int64) string {
	window := time.UnixMilli(to).Sub(time.UnixMilli(from))
	if window <= 0 {
		window = time.Minute
	}
	return fmt.Sprintf("%ds", int(math.Ceil(window.Seconds())))
}

func resolveNativeDashboardUsageBucketExpr(from, to int64) string {
	window := time.UnixMilli(to).Sub(time.UnixMilli(from))
	switch {
	case window <= 6*time.Hour:
		return "DATE_FORMAT(FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(occurred_at) / 300) * 300), '%%Y-%%m-%%d %%H:%%i:00')"
	case window <= 48*time.Hour:
		return "DATE_FORMAT(occurred_at, '%%Y-%%m-%%d %%H:00:00')"
	default:
		return "DATE_FORMAT(occurred_at, '%%Y-%%m-%%d')"
	}
}

func resolveNativeDashboardUsageBucketSize(from, to int64) time.Duration {
	window := time.UnixMilli(to).Sub(time.UnixMilli(from))
	switch {
	case window <= 6*time.Hour:
		return 5 * time.Minute
	case window <= 48*time.Hour:
		return time.Hour
	default:
		return 24 * time.Hour
	}
}

func buildNativeDashboardUsageDetailSeries(rows []nativeDashboardUsageDetailTrendRow, bucketSeconds float64, specs ...nativeDashboardUsageDetailSeriesSpec) []response.NativeDashboardSeries {
	if bucketSeconds <= 0 {
		bucketSeconds = 1
	}
	items := make([]response.NativeDashboardSeries, 0, len(specs))
	for _, spec := range specs {
		points := make([]response.NativeDashboardPoint, 0, len(rows))
		for _, item := range rows {
			if bucketTime, ok := parseNativeDashboardBucket(item.BucketLabel); ok {
				points = append(points, response.NativeDashboardPoint{
					Time:  bucketTime.UnixMilli(),
					Value: spec.Pick(item) / bucketSeconds,
				})
			}
		}
		items = append(items, response.NativeDashboardSeries{
			Name:   spec.Name,
			Labels: map[string]string{},
			Points: points,
		})
	}
	return items
}

func buildNativeDashboardModelTrendSeries(rows []nativeDashboardUsageModelTrendRow, bucketSeconds float64, pick func(nativeDashboardUsageModelTrendRow) float64) []response.NativeDashboardSeries {
	if bucketSeconds <= 0 {
		bucketSeconds = 1
	}
	pointsByModel := map[string][]response.NativeDashboardPoint{}
	for _, item := range rows {
		if bucketTime, ok := parseNativeDashboardBucket(item.BucketLabel); ok {
			pointsByModel[item.ModelID] = append(pointsByModel[item.ModelID], response.NativeDashboardPoint{
				Time:  bucketTime.UnixMilli(),
				Value: pick(item) / bucketSeconds,
			})
		}
	}

	models := make([]string, 0, len(pointsByModel))
	for modelID := range pointsByModel {
		models = append(models, modelID)
	}
	sort.Strings(models)

	items := make([]response.NativeDashboardSeries, 0, len(models))
	for _, modelID := range models {
		items = append(items, response.NativeDashboardSeries{
			Name:   modelID,
			Labels: map[string]string{"model": modelID},
			Points: pointsByModel[modelID],
		})
	}
	return items
}

func nativeDashboardPromPoints(raw [][]any) []response.NativeDashboardPoint {
	items := make([]response.NativeDashboardPoint, 0, len(raw))
	for _, item := range raw {
		if len(item) < 2 {
			continue
		}
		timestamp, err := nativeDashboardPromNumber(item[0])
		if err != nil {
			continue
		}
		value, err := nativeDashboardPromNumber(item[1])
		if err != nil {
			continue
		}
		items = append(items, response.NativeDashboardPoint{
			Time:  int64(timestamp * 1000),
			Value: value,
		})
	}
	return items
}

func nativeDashboardPromSampleValue(raw []any) (float64, error) {
	if len(raw) < 2 {
		return 0, nil
	}
	value, err := nativeDashboardPromNumber(raw[1])
	if err != nil {
		return 0, err
	}
	if !nativeDashboardFinite(value) {
		return 0, nil
	}
	return value, nil
}

func nativeDashboardPromNumber(value any) (float64, error) {
	var parsed float64
	switch typed := value.(type) {
	case float64:
		parsed = typed
	case string:
		floatValue, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, err
		}
		parsed = floatValue
	case json.Number:
		floatValue, err := typed.Float64()
		if err != nil {
			return 0, err
		}
		parsed = floatValue
	default:
		return 0, fmt.Errorf("unsupported prometheus sample value %T", value)
	}
	if !nativeDashboardFinite(parsed) {
		return 0, fmt.Errorf("unsupported prometheus sample value %v", parsed)
	}
	return parsed, nil
}

func nativeDashboardFinite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func parseNativeDashboardClusterName(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-", "-"
	}
	if strings.HasPrefix(value, "outbound|") {
		parts := strings.Split(value, "|")
		if len(parts) >= 4 {
			return parts[3], parts[1]
		}
	}
	if strings.HasPrefix(value, "outbound_") {
		parts := strings.SplitN(strings.TrimPrefix(value, "outbound_"), "_", 3)
		if len(parts) == 3 {
			return parts[2], parts[0]
		}
	}
	return value, "-"
}

func escapeNativeDashboardPromLabelValue(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}

func joinNativeDashboardSeriesErrors(series []response.NativeDashboardSeries, errs ...error) ([]response.NativeDashboardSeries, error) {
	for _, err := range errs {
		if err != nil {
			return series, err
		}
	}
	return series, nil
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxDuration(left, right time.Duration) time.Duration {
	if left > right {
		return left
	}
	return right
}

func makeNativeDashboardStatPanel(id int, title, unit string, x, y, w int, value int64, err error) response.NativeDashboardPanel {
	panel := response.NativeDashboardPanel{
		ID:    id,
		Title: title,
		Type:  "stat",
		Unit:  unit,
		GridPos: response.NativeDashboardGridPos{
			H: 5,
			W: w,
			X: x,
			Y: y,
		},
		Stat: &response.NativeDashboardStat{},
	}
	if err != nil {
		panel.Error = err.Error()
		return panel
	}
	floatValue := float64(value)
	panel.Stat.Value = &floatValue
	return panel
}

func makeNativeDashboardFloatStatPanel(id int, title, unit string, x, y, w int, value float64, err error) response.NativeDashboardPanel {
	panel := response.NativeDashboardPanel{
		ID:    id,
		Title: title,
		Type:  "stat",
		Unit:  unit,
		GridPos: response.NativeDashboardGridPos{
			H: 5,
			W: w,
			X: x,
			Y: y,
		},
		Stat: &response.NativeDashboardStat{},
	}
	if err != nil {
		panel.Error = err.Error()
		return panel
	}
	if !nativeDashboardFinite(value) {
		value = 0
	}
	panel.Stat.Value = &value
	return panel
}

func makeNativeDashboardTimeseriesPanel(id int, title, unit string, x, y, w int, series []response.NativeDashboardSeries, err error) response.NativeDashboardPanel {
	panel := response.NativeDashboardPanel{
		ID:     id,
		Title:  title,
		Type:   "timeseries",
		Unit:   unit,
		Series: series,
		GridPos: response.NativeDashboardGridPos{
			H: 9,
			W: w,
			X: x,
			Y: y,
		},
	}
	if err != nil {
		panel.Error = err.Error()
	}
	return panel
}

func makeNativeDashboardTablePanel(id int, title string, x, y, w int, table response.NativeDashboardTable, err error) response.NativeDashboardPanel {
	panel := response.NativeDashboardPanel{
		ID:    id,
		Title: title,
		Type:  "table",
		GridPos: response.NativeDashboardGridPos{
			H: 10,
			W: w,
			X: x,
			Y: y,
		},
		Table: &table,
	}
	if err != nil {
		panel.Error = err.Error()
	}
	return panel
}

func buildNativeDashboardSeries(name string, rows []nativeDashboardTrendRow, pick func(nativeDashboardTrendRow) float64) []response.NativeDashboardSeries {
	points := make([]response.NativeDashboardPoint, 0, len(rows))
	for _, item := range rows {
		if bucketTime, ok := parseNativeDashboardBucket(item.BucketLabel); ok {
			points = append(points, response.NativeDashboardPoint{
				Time:  bucketTime.UnixMilli(),
				Value: pick(item),
			})
		}
	}
	return []response.NativeDashboardSeries{{
		Name:   name,
		Labels: map[string]string{},
		Points: points,
	}}
}

func mergeNativeDashboardPoints(existing, incoming []response.NativeDashboardPoint) []response.NativeDashboardPoint {
	if len(existing) == 0 {
		return incoming
	}
	if len(incoming) == 0 {
		return existing
	}

	valueByTime := make(map[int64]float64, len(existing)+len(incoming))
	for _, point := range existing {
		valueByTime[point.Time] += point.Value
	}
	for _, point := range incoming {
		valueByTime[point.Time] += point.Value
	}

	timestamps := make([]int64, 0, len(valueByTime))
	for timestamp := range valueByTime {
		timestamps = append(timestamps, timestamp)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})

	items := make([]response.NativeDashboardPoint, 0, len(timestamps))
	for _, timestamp := range timestamps {
		items = append(items, response.NativeDashboardPoint{
			Time:  timestamp,
			Value: valueByTime[timestamp],
		})
	}
	return items
}

func nativeDashboardSeriesHasNonZeroPoints(series []response.NativeDashboardSeries) bool {
	for _, item := range series {
		if nativeDashboardPointsHaveNonZeroValues(item.Points) {
			return true
		}
	}
	return false
}

func nativeDashboardPointsHaveNonZeroValues(points []response.NativeDashboardPoint) bool {
	for _, item := range points {
		if math.Abs(item.Value) > 0 {
			return true
		}
	}
	return false
}

func isNativeDashboardInfraService(service string) bool {
	service = strings.ToLower(strings.TrimSpace(service))
	if service == "" {
		return true
	}
	for _, blocked := range []string{
		"prometheus",
		"prometheus_stats",
		"grafana",
		"loki",
		"xds-grpc",
		"sds-grpc",
		"redis",
		"mysql",
		"portal",
		"console",
		"agent",
	} {
		if strings.Contains(service, blocked) {
			return true
		}
	}
	return false
}

func toSnakeCase(value string) string {
	var builder strings.Builder
	for index, char := range value {
		if index > 0 && char >= 'A' && char <= 'Z' {
			builder.WriteByte('_')
		}
		builder.WriteRune(char)
	}
	return strings.ToLower(builder.String())
}

func parseNativeDashboardBucket(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func buildNativeDashboardEventTable(rows []nativeDashboardRecentEvent) response.NativeDashboardTable {
	items := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		items = append(items, map[string]any{
			"occurredAt":    item.OccurredAt,
			"consumerName":  item.ConsumerName,
			"routeName":     item.RouteName,
			"modelId":       item.ModelID,
			"requestStatus": item.RequestStatus,
			"totalTokens":   item.TotalTokens,
			"costMicroYuan": item.CostMicroYuan,
		})
	}
	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: "occurredAt", Title: "Occurred At"},
			{Key: "consumerName", Title: "Consumer"},
			{Key: "routeName", Title: "Route"},
			{Key: "modelId", Title: "Model"},
			{Key: "requestStatus", Title: "Request Status"},
			{Key: "totalTokens", Title: "Total Tokens"},
			{Key: "costMicroYuan", Title: "Cost (micro yuan)"},
		},
		Rows: items,
	}
}

func buildNativeDashboardExceptionTable(rows []nativeDashboardExceptionRecord) response.NativeDashboardTable {
	items := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		items = append(items, map[string]any{
			"occurredAt":        item.OccurredAt,
			"requestId":         item.RequestID,
			"traceId":           item.TraceID,
			"consumerName":      item.ConsumerName,
			"requestPath":       item.RequestPath,
			"routeName":         item.RouteName,
			"modelId":           item.ModelID,
			"httpStatus":        item.HTTPStatus,
			"requestStatus":     item.RequestStatus,
			"serviceDurationMs": item.ServiceDurationMs,
			"errorCode":         item.ErrorCode,
			"errorMessage":      item.ErrorMessage,
			"totalTokens":       item.TotalTokens,
			"costMicroYuan":     item.CostMicroYuan,
		})
	}
	return response.NativeDashboardTable{
		Columns: []response.NativeDashboardTableColumn{
			{Key: "occurredAt", Title: "Occurred At"},
			{Key: "requestId", Title: "Request ID"},
			{Key: "traceId", Title: "Trace ID"},
			{Key: "consumerName", Title: "Consumer"},
			{Key: "requestPath", Title: "Request Path"},
			{Key: "routeName", Title: "Route"},
			{Key: "modelId", Title: "Model"},
			{Key: "httpStatus", Title: "HTTP Status"},
			{Key: "requestStatus", Title: "Request Status"},
			{Key: "serviceDurationMs", Title: "Service Duration"},
			{Key: "errorCode", Title: "Error Code"},
			{Key: "errorMessage", Title: "Error Message"},
			{Key: "totalTokens", Title: "Total Tokens"},
			{Key: "costMicroYuan", Title: "Cost (micro yuan)"},
		},
		Rows: items,
	}
}
