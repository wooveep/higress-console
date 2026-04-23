package portal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type UsageStatsQuery struct {
	From *int64
	To   *int64
}

type UsageEventsQuery struct {
	From            *int64
	To              *int64
	ConsumerNames   []string
	DepartmentIDs   []string
	IncludeChildren *bool
	APIKeyIDs       []string
	ModelIDs        []string
	RouteNames      []string
	RequestStatuses []string
	UsageStatuses   []string
	PageNum         int
	PageSize        int
}

type DepartmentBillsQuery struct {
	From            *int64
	To              *int64
	DepartmentIDs   []string
	IncludeChildren *bool
}

type UsageTrendQuery struct {
	From            *int64
	To              *int64
	Bucket          string
	ConsumerName    string
	DepartmentID    string
	IncludeChildren *bool
	ModelID         string
	RouteName       string
}

type PortalStatsSelectOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type PortalUsageEventFilterOptions struct {
	Consumers       []PortalStatsSelectOption `json:"consumers"`
	Departments     []PortalStatsSelectOption `json:"departments"`
	APIKeys         []PortalStatsSelectOption `json:"apiKeys"`
	Models          []PortalStatsSelectOption `json:"models"`
	Routes          []PortalStatsSelectOption `json:"routes"`
	RequestStatuses []PortalStatsSelectOption `json:"requestStatuses"`
	UsageStatuses   []PortalStatsSelectOption `json:"usageStatuses"`
}

type PortalUsageStatRecord struct {
	ConsumerName               string `json:"consumerName"`
	ModelName                  string `json:"modelName"`
	RequestCount               int64  `json:"requestCount"`
	InputTokens                int64  `json:"inputTokens"`
	OutputTokens               int64  `json:"outputTokens"`
	TotalTokens                int64  `json:"totalTokens"`
	CacheCreationInputTokens   int64  `json:"cacheCreationInputTokens"`
	CacheCreation5mInputTokens int64  `json:"cacheCreation5mInputTokens"`
	CacheCreation1hInputTokens int64  `json:"cacheCreation1hInputTokens"`
	CacheReadInputTokens       int64  `json:"cacheReadInputTokens"`
	InputImageTokens           int64  `json:"inputImageTokens"`
	OutputImageTokens          int64  `json:"outputImageTokens"`
	InputImageCount            int64  `json:"inputImageCount"`
	OutputImageCount           int64  `json:"outputImageCount"`
}

type PortalUsageEventRecord struct {
	EventID           string `json:"eventId"`
	RequestID         string `json:"requestId"`
	TraceID           string `json:"traceId"`
	ConsumerName      string `json:"consumerName"`
	DepartmentID      string `json:"departmentId"`
	DepartmentPath    string `json:"departmentPath"`
	APIKeyID          string `json:"apiKeyId"`
	ModelID           string `json:"modelId"`
	PriceVersionID    *int64 `json:"priceVersionId,omitempty"`
	RouteName         string `json:"routeName"`
	RequestPath       string `json:"requestPath"`
	RequestKind       string `json:"requestKind"`
	RequestStatus     string `json:"requestStatus"`
	UsageStatus       string `json:"usageStatus"`
	HTTPStatus        *int   `json:"httpStatus,omitempty"`
	ErrorCode         string `json:"errorCode"`
	ErrorMessage      string `json:"errorMessage"`
	InputTokens       int64  `json:"inputTokens"`
	OutputTokens      int64  `json:"outputTokens"`
	TotalTokens       int64  `json:"totalTokens"`
	RequestCount      int64  `json:"requestCount"`
	CostMicroYuan     int64  `json:"costMicroYuan"`
	StartedAt         string `json:"startedAt,omitempty"`
	FinishedAt        string `json:"finishedAt,omitempty"`
	ServiceDurationMs int64  `json:"serviceDurationMs"`
	OccurredAt        string `json:"occurredAt,omitempty"`
}

type PortalDepartmentBillRecord struct {
	DepartmentID    string  `json:"departmentId"`
	DepartmentName  string  `json:"departmentName"`
	DepartmentPath  string  `json:"departmentPath"`
	RequestCount    int64   `json:"requestCount"`
	TotalTokens     int64   `json:"totalTokens"`
	TotalCost       float64 `json:"totalCost"`
	ActiveConsumers int64   `json:"activeConsumers"`
}

type PortalUsageTrendPoint struct {
	BucketLabel     string `json:"bucketLabel"`
	RequestCount    int64  `json:"requestCount"`
	InputTokens     int64  `json:"inputTokens"`
	OutputTokens    int64  `json:"outputTokens"`
	TotalTokens     int64  `json:"totalTokens"`
	CostMicroYuan   int64  `json:"costMicroYuan"`
	ActiveConsumers int64  `json:"activeConsumers"`
}

func (s *Service) ListUsageStats(ctx context.Context, query UsageStatsQuery) ([]PortalUsageStatRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	fromTime, toTime := normalizeStatsRange(query.From, query.To, time.Hour)
	rows, err := db.QueryContext(ctx, `
		SELECT consumer_name, model_id,
			COALESCE(SUM(request_count), 0) AS request_count,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(cache_creation_input_tokens), 0) AS cache_creation_input_tokens,
			COALESCE(SUM(cache_creation_5m_input_tokens), 0) AS cache_creation_5m_input_tokens,
			COALESCE(SUM(cache_creation_1h_input_tokens), 0) AS cache_creation_1h_input_tokens,
			COALESCE(SUM(cache_read_input_tokens), 0) AS cache_read_input_tokens,
			COALESCE(SUM(input_image_tokens), 0) AS input_image_tokens,
			COALESCE(SUM(output_image_tokens), 0) AS output_image_tokens,
			COALESCE(SUM(input_image_count), 0) AS input_image_count,
			COALESCE(SUM(output_image_count), 0) AS output_image_count
		FROM billing_usage_event
		WHERE request_status = 'success'
			AND usage_status = 'parsed'
			AND occurred_at >= ?
			AND occurred_at < ?
		GROUP BY consumer_name, model_id
		ORDER BY consumer_name ASC, model_id ASC`,
		fromTime,
		toTime,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query portal billing usage stats: %w", err)
	}
	defer rows.Close()

	items := make([]PortalUsageStatRecord, 0)
	for rows.Next() {
		var item PortalUsageStatRecord
		if err := rows.Scan(
			&item.ConsumerName,
			&item.ModelName,
			&item.RequestCount,
			&item.InputTokens,
			&item.OutputTokens,
			&item.TotalTokens,
			&item.CacheCreationInputTokens,
			&item.CacheCreation5mInputTokens,
			&item.CacheCreation1hInputTokens,
			&item.CacheReadInputTokens,
			&item.InputImageTokens,
			&item.OutputImageTokens,
			&item.InputImageCount,
			&item.OutputImageCount,
		); err != nil {
			return nil, err
		}
		if item.TotalTokens <= 0 {
			cacheCreationEffectiveTokens := maxInt64(
				item.CacheCreationInputTokens,
				item.CacheCreation5mInputTokens+item.CacheCreation1hInputTokens,
			)
			item.TotalTokens = item.InputTokens + item.OutputTokens + cacheCreationEffectiveTokens +
				item.CacheReadInputTokens + item.InputImageTokens + item.OutputImageTokens
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) ListUsageTrend(ctx context.Context, query UsageTrendQuery) ([]PortalUsageTrendPoint, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	fromTime, toTime := normalizeStatsRange(query.From, query.To, 7*24*time.Hour)
	bucketExpr, bucketName := resolveStatsBucket(query.Bucket, fromTime, toTime)

	statement := strings.Builder{}
	statement.WriteString(`SELECT `)
	statement.WriteString(bucketExpr)
	statement.WriteString(` AS bucket_label,
		COALESCE(SUM(request_count), 0) AS request_count,
		COALESCE(SUM(input_tokens), 0) AS input_tokens,
		COALESCE(SUM(output_tokens), 0) AS output_tokens,
		COALESCE(SUM(
			CASE
				WHEN total_tokens > 0 THEN total_tokens
				ELSE input_tokens + output_tokens + GREATEST(cache_creation_input_tokens, cache_creation_5m_input_tokens + cache_creation_1h_input_tokens) +
					cache_read_input_tokens + input_image_tokens + output_image_tokens
			END
		), 0) AS total_tokens,
		COALESCE(SUM(cost_micro_yuan), 0) AS cost_micro_yuan,
		COUNT(DISTINCT consumer_name) AS active_consumers
		FROM billing_usage_event
		WHERE request_status = 'success'
			AND usage_status = 'parsed'
			AND occurred_at >= ? AND occurred_at < ?`)
	args := []any{fromTime, toTime}
	appendPortalStatsEqualsFilter(&statement, &args, "consumer_name", query.ConsumerName)
	appendPortalStatsEqualsFilter(&statement, &args, "model_id", query.ModelID)
	appendPortalStatsEqualsFilter(&statement, &args, "route_name", query.RouteName)
	if err := s.appendPortalStatsDepartmentScope(ctx, db, &statement, &args, []string{query.DepartmentID}, query.IncludeChildren); err != nil {
		return nil, err
	}
	statement.WriteString(` GROUP BY bucket_label ORDER BY bucket_label ASC`)

	rows, err := db.QueryContext(ctx, statement.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query portal billing usage trend (%s): %w", bucketName, err)
	}
	defer rows.Close()

	items := make([]PortalUsageTrendPoint, 0)
	for rows.Next() {
		var item PortalUsageTrendPoint
		if err := rows.Scan(
			&item.BucketLabel,
			&item.RequestCount,
			&item.InputTokens,
			&item.OutputTokens,
			&item.TotalTokens,
			&item.CostMicroYuan,
			&item.ActiveConsumers,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) ListUsageEvents(ctx context.Context, query UsageEventsQuery) ([]PortalUsageEventRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	fromTime, toTime := normalizeStatsRange(query.From, query.To, 24*time.Hour)
	pageNum := query.PageNum
	if pageNum <= 0 {
		pageNum = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}

	statement := strings.Builder{}
	statement.WriteString(`SELECT event_id, request_id, trace_id, consumer_name, department_id,
		department_path, api_key_id, model_id, price_version_id, route_name, request_path, request_kind,
		request_status, usage_status, http_status, error_code, error_message,
		input_tokens, output_tokens, total_tokens, request_count, cost_micro_yuan,
		started_at, finished_at, occurred_at
		FROM billing_usage_event
		WHERE occurred_at >= ? AND occurred_at < ?`)
	args := []any{fromTime, toTime}
	appendPortalStatsInFilter(&statement, &args, "consumer_name", query.ConsumerNames)
	appendPortalStatsInFilter(&statement, &args, "api_key_id", query.APIKeyIDs)
	appendPortalStatsInFilter(&statement, &args, "model_id", query.ModelIDs)
	appendPortalStatsInFilter(&statement, &args, "route_name", query.RouteNames)
	appendPortalStatsInFilter(&statement, &args, "request_status", query.RequestStatuses)
	appendPortalStatsInFilter(&statement, &args, "usage_status", query.UsageStatuses)
	if err := s.appendPortalStatsDepartmentScope(ctx, db, &statement, &args, query.DepartmentIDs, query.IncludeChildren); err != nil {
		return nil, err
	}
	statement.WriteString(` ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?`)
	args = append(args, pageSize, (pageNum-1)*pageSize)

	rows, err := db.QueryContext(ctx, statement.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query portal billing usage events: %w", err)
	}
	defer rows.Close()

	items := make([]PortalUsageEventRecord, 0)
	for rows.Next() {
		var item PortalUsageEventRecord
		var apiKeyID sql.NullString
		var priceVersionID sql.NullInt64
		var httpStatus sql.NullInt64
		var startedAt sql.NullTime
		var finishedAt sql.NullTime
		var occurredAt sql.NullTime
		if err := rows.Scan(
			&item.EventID,
			&item.RequestID,
			&item.TraceID,
			&item.ConsumerName,
			&item.DepartmentID,
			&item.DepartmentPath,
			&apiKeyID,
			&item.ModelID,
			&priceVersionID,
			&item.RouteName,
			&item.RequestPath,
			&item.RequestKind,
			&item.RequestStatus,
			&item.UsageStatus,
			&httpStatus,
			&item.ErrorCode,
			&item.ErrorMessage,
			&item.InputTokens,
			&item.OutputTokens,
			&item.TotalTokens,
			&item.RequestCount,
			&item.CostMicroYuan,
			&startedAt,
			&finishedAt,
			&occurredAt,
		); err != nil {
			return nil, err
		}
		if priceVersionID.Valid {
			value := priceVersionID.Int64
			item.PriceVersionID = &value
		}
		if apiKeyID.Valid {
			item.APIKeyID = strings.TrimSpace(apiKeyID.String)
		}
		if httpStatus.Valid {
			value := int(httpStatus.Int64)
			item.HTTPStatus = &value
		}
		if startedAt.Valid {
			item.StartedAt = startedAt.Time.Format(time.RFC3339)
		}
		if finishedAt.Valid {
			item.FinishedAt = finishedAt.Time.Format(time.RFC3339)
		}
		if startedAt.Valid && finishedAt.Valid {
			duration := finishedAt.Time.Sub(startedAt.Time).Milliseconds()
			if duration > 0 {
				item.ServiceDurationMs = duration
			}
		}
		if occurredAt.Valid {
			item.OccurredAt = occurredAt.Time.Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) ListDepartmentBills(ctx context.Context, query DepartmentBillsQuery) ([]PortalDepartmentBillRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	fromTime, toTime := normalizeStatsRange(query.From, query.To, 30*24*time.Hour)
	statement := strings.Builder{}
	statement.WriteString(`SELECT e.department_id, COALESCE(o.name, '') AS department_name,
		COALESCE(MAX(NULLIF(TRIM(e.department_path), '')), '') AS department_path,
		COALESCE(SUM(CASE WHEN e.request_count > 0 THEN e.request_count ELSE 1 END), 0) AS request_count,
		COALESCE(SUM(
			CASE
				WHEN e.total_tokens > 0 THEN e.total_tokens
				ELSE e.input_tokens + e.output_tokens + GREATEST(e.cache_creation_input_tokens, e.cache_creation_5m_input_tokens + e.cache_creation_1h_input_tokens) +
					e.cache_read_input_tokens + e.input_image_tokens + e.output_image_tokens
			END
		), 0) AS total_tokens,
		COALESCE(SUM(e.cost_micro_yuan), 0) / 1000000.0 AS total_cost,
		COUNT(DISTINCT CASE WHEN TRIM(e.consumer_name) <> '' THEN e.consumer_name END) AS active_consumers
		FROM billing_usage_event e
		LEFT JOIN org_department o ON o.department_id = e.department_id
		WHERE e.request_status = 'success'
			AND e.usage_status = 'parsed'
			AND e.occurred_at >= ? AND e.occurred_at < ?`)
	args := []any{fromTime, toTime}
	if err := s.appendPortalStatsDepartmentScope(ctx, db, &statement, &args, query.DepartmentIDs, query.IncludeChildren); err != nil {
		return nil, err
	}
	statement.WriteString(` GROUP BY e.department_id, o.name ORDER BY department_path ASC, e.department_id ASC`)

	rows, err := db.QueryContext(ctx, statement.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query portal department bills: %w", err)
	}
	defer rows.Close()

	items := make([]PortalDepartmentBillRecord, 0)
	for rows.Next() {
		var item PortalDepartmentBillRecord
		if err := rows.Scan(
			&item.DepartmentID,
			&item.DepartmentName,
			&item.DepartmentPath,
			&item.RequestCount,
			&item.TotalTokens,
			&item.TotalCost,
			&item.ActiveConsumers,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) ListUsageEventFilterOptions(ctx context.Context, query UsageEventsQuery) (*PortalUsageEventFilterOptions, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	fromTime, toTime := normalizeStatsRange(query.From, query.To, 24*time.Hour)
	scopedDepartmentIDs, err := s.resolvePortalStatsScopedDepartmentIDs(ctx, db, query.DepartmentIDs, query.IncludeChildren)
	if err != nil {
		return nil, err
	}
	baseWhere := strings.Builder{}
	baseWhere.WriteString(` FROM billing_usage_event WHERE occurred_at >= ? AND occurred_at < ?`)
	baseArgs := []any{fromTime, toTime}

	buildStatement := func(column string) (string, []any) {
		statement := strings.Builder{}
		statement.WriteString(`SELECT DISTINCT `)
		statement.WriteString(column)
		statement.WriteString(` AS value`)
		statement.WriteString(baseWhere.String())
		args := append([]any{}, baseArgs...)
		appendPortalUsageEventOptionFilters(&statement, &args, query, column, scopedDepartmentIDs, "")
		statement.WriteString(` AND `)
		statement.WriteString(column)
		statement.WriteString(` IS NOT NULL AND TRIM(`)
		statement.WriteString(column)
		statement.WriteString(`) <> '' ORDER BY value ASC LIMIT 200`)
		return statement.String(), args
	}

	loadOptions := func(column string) ([]PortalStatsSelectOption, error) {
		statement, args := buildStatement(column)
		rows, err := db.QueryContext(ctx, statement, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		items := make([]PortalStatsSelectOption, 0)
		for rows.Next() {
			var value string
			if err := rows.Scan(&value); err != nil {
				return nil, err
			}
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			items = append(items, PortalStatsSelectOption{Label: trimmed, Value: trimmed})
		}
		return items, rows.Err()
	}

	departments, err := s.listUsageEventDepartmentOptions(ctx, db, fromTime, toTime, query)
	if err != nil {
		return nil, err
	}
	consumers, err := loadOptions("consumer_name")
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event consumer options: %w", err)
	}
	apiKeys, err := loadOptions("api_key_id")
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event api key options: %w", err)
	}
	models, err := loadOptions("model_id")
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event model options: %w", err)
	}
	routes, err := loadOptions("route_name")
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event route options: %w", err)
	}
	requestStatuses, err := loadOptions("request_status")
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event request status options: %w", err)
	}
	usageStatuses, err := loadOptions("usage_status")
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event usage status options: %w", err)
	}

	return &PortalUsageEventFilterOptions{
		Consumers:       consumers,
		Departments:     departments,
		APIKeys:         apiKeys,
		Models:          models,
		Routes:          routes,
		RequestStatuses: requestStatuses,
		UsageStatuses:   usageStatuses,
	}, nil
}

func normalizeStatsRange(fromMillis, toMillis *int64, fallback time.Duration) (time.Time, time.Time) {
	now := time.Now().UTC()
	toTime := now
	if toMillis != nil && *toMillis > 0 {
		toTime = time.UnixMilli(*toMillis).UTC()
	}
	fromTime := toTime.Add(-fallback)
	if fromMillis != nil && *fromMillis > 0 && *fromMillis < toTime.UnixMilli() {
		fromTime = time.UnixMilli(*fromMillis).UTC()
	}
	return fromTime, toTime
}

func appendPortalStatsEqualsFilter(statement *strings.Builder, args *[]any, columnName, value string) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return
	}
	statement.WriteString(" AND ")
	statement.WriteString(columnName)
	statement.WriteString(" = ?")
	*args = append(*args, normalized)
}

func appendPortalStatsInFilter(statement *strings.Builder, args *[]any, columnName string, values []string) {
	normalizedValues := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			normalizedValues = append(normalizedValues, trimmed)
		}
	}
	if len(normalizedValues) == 0 {
		return
	}
	statement.WriteString(" AND ")
	if len(normalizedValues) == 1 {
		statement.WriteString(columnName)
		statement.WriteString(" = ?")
		*args = append(*args, normalizedValues[0])
		return
	}
	statement.WriteString(columnName)
	statement.WriteString(" IN (")
	for index, value := range normalizedValues {
		if index > 0 {
			statement.WriteString(", ")
		}
		statement.WriteString("?")
		*args = append(*args, value)
	}
	statement.WriteString(")")
}

func (s *Service) appendPortalStatsDepartmentScope(
	ctx context.Context,
	db *sql.DB,
	statement *strings.Builder,
	args *[]any,
	departmentIDs []string,
	includeChildren *bool,
) error {
	scopedDepartmentIDs, err := s.resolvePortalStatsScopedDepartmentIDs(ctx, db, departmentIDs, includeChildren)
	if err != nil {
		return err
	}
	appendPortalStatsDepartmentIDs(statement, args, "department_id", scopedDepartmentIDs)
	return nil
}

func (s *Service) resolvePortalStatsDepartmentIDs(
	ctx context.Context,
	db *sql.DB,
	departmentID string,
	includeChildren bool,
) ([]string, error) {
	if !includeChildren {
		return []string{departmentID}, nil
	}

	var rootPath string
	if err := db.QueryRowContext(ctx,
		`SELECT path FROM org_department WHERE department_id = ? LIMIT 1`,
		departmentID,
	).Scan(&rootPath); err != nil {
		if err == sql.ErrNoRows {
			return []string{departmentID}, nil
		}
		return nil, err
	}
	if strings.TrimSpace(rootPath) == "" {
		return []string{departmentID}, nil
	}

	rows, err := db.QueryContext(ctx,
		`SELECT department_id FROM org_department
		WHERE department_id = ? OR path = ? OR path LIKE ?
		ORDER BY path ASC`,
		departmentID,
		rootPath,
		rootPath+"/%",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]string, 0)
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, err
		}
		if strings.TrimSpace(item) != "" {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return []string{departmentID}, nil
	}
	return items, rows.Err()
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

func (s *Service) listUsageEventDepartmentOptions(ctx context.Context, db *sql.DB, fromTime, toTime time.Time, query UsageEventsQuery) ([]PortalStatsSelectOption, error) {
	statement := strings.Builder{}
	statement.WriteString(`
		SELECT DISTINCT e.department_id, COALESCE(NULLIF(TRIM(o.name), ''), NULLIF(TRIM(e.department_path), ''), e.department_id) AS label
		FROM billing_usage_event e
		LEFT JOIN org_department o ON o.department_id = e.department_id
		WHERE e.occurred_at >= ? AND e.occurred_at < ?
			AND e.department_id IS NOT NULL
			AND TRIM(e.department_id) <> ''
	`)
	args := []any{fromTime, toTime}
	appendPortalUsageEventOptionFilters(&statement, &args, query, "department_id", nil, "e.")
	statement.WriteString(`
		ORDER BY label ASC, e.department_id ASC
		LIMIT 200`)
	rows, err := db.QueryContext(ctx, statement.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query usage event department options: %w", err)
	}
	defer rows.Close()

	items := make([]PortalStatsSelectOption, 0)
	for rows.Next() {
		var (
			value string
			label string
		)
		if err := rows.Scan(&value, &label); err != nil {
			return nil, err
		}
		items = append(items, PortalStatsSelectOption{
			Label: strings.TrimSpace(label),
			Value: strings.TrimSpace(value),
		})
	}
	return items, rows.Err()
}

func (s *Service) resolvePortalStatsScopedDepartmentIDs(
	ctx context.Context,
	db *sql.DB,
	departmentIDs []string,
	includeChildren *bool,
) ([]string, error) {
	normalizedDepartmentIDs := make([]string, 0, len(departmentIDs))
	for _, departmentID := range departmentIDs {
		if trimmed := strings.TrimSpace(departmentID); trimmed != "" {
			normalizedDepartmentIDs = append(normalizedDepartmentIDs, trimmed)
		}
	}
	if len(normalizedDepartmentIDs) == 0 {
		return nil, nil
	}
	shouldIncludeChildren := true
	if includeChildren != nil {
		shouldIncludeChildren = *includeChildren
	}
	scopedDepartmentIDs := make([]string, 0)
	seen := map[string]struct{}{}
	for _, departmentID := range normalizedDepartmentIDs {
		resolvedIDs, err := s.resolvePortalStatsDepartmentIDs(ctx, db, departmentID, shouldIncludeChildren)
		if err != nil {
			return nil, err
		}
		for _, item := range resolvedIDs {
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			scopedDepartmentIDs = append(scopedDepartmentIDs, item)
		}
	}
	return scopedDepartmentIDs, nil
}

func appendPortalStatsDepartmentIDs(statement *strings.Builder, args *[]any, columnName string, departmentIDs []string) {
	if len(departmentIDs) == 0 {
		return
	}
	statement.WriteString(" AND ")
	if len(departmentIDs) == 1 {
		statement.WriteString(columnName)
		statement.WriteString(" = ?")
		*args = append(*args, departmentIDs[0])
		return
	}
	statement.WriteString(columnName)
	statement.WriteString(" IN (")
	for index, item := range departmentIDs {
		if index > 0 {
			statement.WriteString(", ")
		}
		statement.WriteString("?")
		*args = append(*args, item)
	}
	statement.WriteString(")")
}

func appendPortalUsageEventOptionFilters(
	statement *strings.Builder,
	args *[]any,
	query UsageEventsQuery,
	excludedColumn string,
	scopedDepartmentIDs []string,
	columnPrefix string,
) {
	qualify := func(column string) string {
		return columnPrefix + column
	}
	if excludedColumn != "consumer_name" {
		appendPortalStatsInFilter(statement, args, qualify("consumer_name"), query.ConsumerNames)
	}
	if excludedColumn != "api_key_id" {
		appendPortalStatsInFilter(statement, args, qualify("api_key_id"), query.APIKeyIDs)
	}
	if excludedColumn != "model_id" {
		appendPortalStatsInFilter(statement, args, qualify("model_id"), query.ModelIDs)
	}
	if excludedColumn != "route_name" {
		appendPortalStatsInFilter(statement, args, qualify("route_name"), query.RouteNames)
	}
	if excludedColumn != "request_status" {
		appendPortalStatsInFilter(statement, args, qualify("request_status"), query.RequestStatuses)
	}
	if excludedColumn != "usage_status" {
		appendPortalStatsInFilter(statement, args, qualify("usage_status"), query.UsageStatuses)
	}
	if excludedColumn != "department_id" {
		appendPortalStatsDepartmentIDs(statement, args, qualify("department_id"), scopedDepartmentIDs)
	}
}

func resolveStatsBucket(bucket string, fromTime, toTime time.Time) (string, string) {
	normalized := strings.ToLower(strings.TrimSpace(bucket))
	if normalized == "" {
		if toTime.Sub(fromTime) <= 6*time.Hour {
			normalized = "5m"
		} else if toTime.Sub(fromTime) <= 48*time.Hour {
			normalized = "hour"
		} else {
			normalized = "day"
		}
	}
	switch normalized {
	case "5m":
		return "TO_CHAR(TO_TIMESTAMP(FLOOR(EXTRACT(EPOCH FROM occurred_at) / 300) * 300), 'YYYY-MM-DD HH24:MI:00')", "5m"
	case "hour":
		return "TO_CHAR(DATE_TRUNC('hour', occurred_at), 'YYYY-MM-DD HH24:00:00')", "hour"
	default:
		return "TO_CHAR(DATE_TRUNC('day', occurred_at), 'YYYY-MM-DD')", "day"
	}
}
