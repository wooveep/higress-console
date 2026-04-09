package com.alibaba.higress.console.service.portal;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;
import java.util.StringJoiner;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.PortalDepartmentBillRecord;
import com.alibaba.higress.console.model.portal.PortalUsageEventRecord;
import com.alibaba.higress.console.model.portal.PortalUsageStatRecord;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalUsageStatsService {

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    @PostConstruct
    public void init() {
        if (enabled()) {
            log.info("Portal usage stats will read from Portal billing ledger.");
        }
    }

    public List<PortalUsageStatRecord> listUsage(Long fromMillis, Long toMillis) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable for portal usage stats.");
        }

        long now = System.currentTimeMillis();
        long to = toMillis != null && toMillis > 0 ? toMillis : now;
        long from = fromMillis != null && fromMillis > 0 && fromMillis < to ? fromMillis : to - 3600_000L;

        String sql = "SELECT consumer_name, model_id, "
            + "COALESCE(SUM(request_count), 0) AS request_count, "
            + "COALESCE(SUM(input_tokens), 0) AS input_tokens, "
            + "COALESCE(SUM(output_tokens), 0) AS output_tokens, "
            + "COALESCE(SUM(total_tokens), 0) AS total_tokens, "
            + "COALESCE(SUM(cache_creation_input_tokens), 0) AS cache_creation_input_tokens, "
            + "COALESCE(SUM(cache_creation_5m_input_tokens), 0) AS cache_creation_5m_input_tokens, "
            + "COALESCE(SUM(cache_creation_1h_input_tokens), 0) AS cache_creation_1h_input_tokens, "
            + "COALESCE(SUM(cache_read_input_tokens), 0) AS cache_read_input_tokens, "
            + "COALESCE(SUM(input_image_tokens), 0) AS input_image_tokens, "
            + "COALESCE(SUM(output_image_tokens), 0) AS output_image_tokens, "
            + "COALESCE(SUM(input_image_count), 0) AS input_image_count, "
            + "COALESCE(SUM(output_image_count), 0) AS output_image_count "
            + "FROM billing_usage_event "
            + "WHERE request_status = 'success' "
            + "AND usage_status = 'parsed' "
            + "AND occurred_at >= ? "
            + "AND occurred_at < ? "
            + "GROUP BY consumer_name, model_id "
            + "ORDER BY consumer_name ASC, model_id ASC";

        List<PortalUsageStatRecord> result = new ArrayList<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setTimestamp(1, new Timestamp(from));
            statement.setTimestamp(2, new Timestamp(to));
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    long cacheCreationInputTokens = rs.getLong("cache_creation_input_tokens");
                    long cacheCreation5mInputTokens = rs.getLong("cache_creation_5m_input_tokens");
                    long cacheCreation1hInputTokens = rs.getLong("cache_creation_1h_input_tokens");
                    long cacheCreationEffectiveTokens = Math.max(cacheCreationInputTokens,
                        cacheCreation5mInputTokens + cacheCreation1hInputTokens);
                    long inputTokens = rs.getLong("input_tokens");
                    long outputTokens = rs.getLong("output_tokens");
                    long cacheReadInputTokens = rs.getLong("cache_read_input_tokens");
                    long inputImageTokens = rs.getLong("input_image_tokens");
                    long outputImageTokens = rs.getLong("output_image_tokens");
                    long totalTokens = rs.getLong("total_tokens");
                    if (totalTokens <= 0) {
                        totalTokens = inputTokens + outputTokens + cacheCreationEffectiveTokens + cacheReadInputTokens
                            + inputImageTokens + outputImageTokens;
                    }
                    result.add(PortalUsageStatRecord.builder()
                        .consumerName(StringUtils.trimToEmpty(rs.getString("consumer_name")))
                        .modelName(StringUtils.trimToEmpty(rs.getString("model_id")))
                        .requestCount(rs.getLong("request_count"))
                        .inputTokens(inputTokens)
                        .outputTokens(outputTokens)
                        .totalTokens(totalTokens)
                        .cacheCreationInputTokens(cacheCreationInputTokens)
                        .cacheCreation5mInputTokens(cacheCreation5mInputTokens)
                        .cacheCreation1hInputTokens(cacheCreation1hInputTokens)
                        .cacheReadInputTokens(cacheReadInputTokens)
                        .inputImageTokens(inputImageTokens)
                        .outputImageTokens(outputImageTokens)
                        .inputImageCount(rs.getLong("input_image_count"))
                        .outputImageCount(rs.getLong("output_image_count"))
                        .build());
                }
            }
        } catch (SQLException ex) {
            throw new IllegalStateException("Failed to query Portal billing usage stats.", ex);
        }

        result.sort(Comparator.comparing(PortalUsageStatRecord::getConsumerName)
            .thenComparing(PortalUsageStatRecord::getModelName));
        return result;
    }

    public List<PortalUsageEventRecord> listUsageEvents(Long fromMillis, Long toMillis, String consumerName,
        String departmentId, Boolean includeChildren, String apiKeyId, String modelId, String routeName,
        String requestStatus, String usageStatus, Integer pageNum, Integer pageSize) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable for portal usage events.");
        }
        long now = System.currentTimeMillis();
        long to = toMillis != null && toMillis > 0 ? toMillis : now;
        long from = fromMillis != null && fromMillis > 0 && fromMillis < to ? fromMillis : to - 24L * 3600_000L;
        int normalizedPageNum = pageNum == null || pageNum <= 0 ? 1 : pageNum;
        int normalizedPageSize = pageSize == null || pageSize <= 0 ? 50 : Math.min(pageSize, 200);

        StringBuilder sql = new StringBuilder("SELECT event_id, request_id, trace_id, consumer_name, department_id, "
            + "department_path, api_key_id, model_id, price_version_id, route_name, request_kind, request_status, "
            + "usage_status, http_status, input_tokens, output_tokens, total_tokens, request_count, cost_micro_yuan, "
            + "occurred_at FROM billing_usage_event WHERE occurred_at >= ? AND occurred_at < ?");
        List<Object> args = new ArrayList<>();
        args.add(new Timestamp(from));
        args.add(new Timestamp(to));
        appendEqualsFilter(sql, args, "consumer_name", consumerName);
        appendEqualsFilter(sql, args, "api_key_id", apiKeyId);
        appendEqualsFilter(sql, args, "model_id", modelId);
        appendEqualsFilter(sql, args, "route_name", routeName);
        appendEqualsFilter(sql, args, "request_status", requestStatus);
        appendEqualsFilter(sql, args, "usage_status", usageStatus);
        appendDepartmentScope(sql, args, departmentId, includeChildren);
        sql.append(" ORDER BY occurred_at DESC, id DESC LIMIT ? OFFSET ?");
        args.add(normalizedPageSize);
        args.add((normalizedPageNum - 1) * normalizedPageSize);

        List<PortalUsageEventRecord> result = new ArrayList<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql.toString())) {
            bindArgs(statement, args);
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    result.add(PortalUsageEventRecord.builder()
                        .eventId(StringUtils.trimToEmpty(rs.getString("event_id")))
                        .requestId(StringUtils.trimToEmpty(rs.getString("request_id")))
                        .traceId(StringUtils.trimToEmpty(rs.getString("trace_id")))
                        .consumerName(StringUtils.trimToEmpty(rs.getString("consumer_name")))
                        .departmentId(StringUtils.trimToEmpty(rs.getString("department_id")))
                        .departmentPath(StringUtils.trimToEmpty(rs.getString("department_path")))
                        .apiKeyId(StringUtils.trimToEmpty(rs.getString("api_key_id")))
                        .modelId(StringUtils.trimToEmpty(rs.getString("model_id")))
                        .priceVersionId(rs.getLong("price_version_id"))
                        .routeName(StringUtils.trimToEmpty(rs.getString("route_name")))
                        .requestKind(StringUtils.trimToEmpty(rs.getString("request_kind")))
                        .requestStatus(StringUtils.trimToEmpty(rs.getString("request_status")))
                        .usageStatus(StringUtils.trimToEmpty(rs.getString("usage_status")))
                        .httpStatus(rs.getInt("http_status"))
                        .inputTokens(rs.getLong("input_tokens"))
                        .outputTokens(rs.getLong("output_tokens"))
                        .totalTokens(rs.getLong("total_tokens"))
                        .requestCount(rs.getLong("request_count"))
                        .costMicroYuan(rs.getLong("cost_micro_yuan"))
                        .occurredAt(rs.getTimestamp("occurred_at") == null ? null
                            : rs.getTimestamp("occurred_at").toLocalDateTime().toString())
                        .build());
                }
            }
        } catch (SQLException ex) {
            throw new IllegalStateException("Failed to query Portal billing usage events.", ex);
        }
        return result;
    }

    public List<PortalDepartmentBillRecord> listDepartmentBills(Long fromMillis, Long toMillis, String departmentId,
        Boolean includeChildren) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable for portal department bills.");
        }
        long now = System.currentTimeMillis();
        long to = toMillis != null && toMillis > 0 ? toMillis : now;
        long from = fromMillis != null && fromMillis > 0 && fromMillis < to ? fromMillis : to - 30L * 24 * 3600_000L;

        StringBuilder sql = new StringBuilder("SELECT d.department_id, COALESCE(o.name, '') AS department_name, "
            + "COALESCE(d.department_path, '') AS department_path, COALESCE(SUM(d.request_count), 0) AS request_count, "
            + "COALESCE(SUM(d.total_tokens), 0) AS total_tokens, COALESCE(SUM(d.cost_amount), 0) AS total_cost, "
            + "COUNT(DISTINCT d.consumer_name) AS active_consumers "
            + "FROM portal_usage_daily d LEFT JOIN org_department o ON o.department_id = d.department_id "
            + "WHERE d.billing_date >= ? AND d.billing_date <= ?");
        List<Object> args = new ArrayList<>();
        args.add(new Timestamp(from).toLocalDateTime().toLocalDate().toString());
        args.add(new Timestamp(to).toLocalDateTime().toLocalDate().toString());
        appendDepartmentScope(sql, args, departmentId, includeChildren);
        sql.append(" GROUP BY d.department_id, o.name, d.department_path ORDER BY d.department_path ASC, d.department_id ASC");

        List<PortalDepartmentBillRecord> result = new ArrayList<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql.toString())) {
            bindArgs(statement, args);
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    result.add(PortalDepartmentBillRecord.builder()
                        .departmentId(StringUtils.trimToEmpty(rs.getString("department_id")))
                        .departmentName(StringUtils.trimToEmpty(rs.getString("department_name")))
                        .departmentPath(StringUtils.trimToEmpty(rs.getString("department_path")))
                        .requestCount(rs.getLong("request_count"))
                        .totalTokens(rs.getLong("total_tokens"))
                        .totalCost(rs.getDouble("total_cost"))
                        .activeConsumers(rs.getLong("active_consumers"))
                        .build());
                }
            }
        } catch (SQLException ex) {
            throw new IllegalStateException("Failed to query Portal department bills.", ex);
        }
        return result;
    }

    private boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    private void appendEqualsFilter(StringBuilder sql, List<Object> args, String columnName, String value) {
        String normalized = StringUtils.trimToNull(value);
        if (normalized == null) {
            return;
        }
        sql.append(" AND ").append(columnName).append(" = ?");
        args.add(normalized);
    }

    private void appendDepartmentScope(StringBuilder sql, List<Object> args, String departmentId, Boolean includeChildren) {
        String normalizedDepartmentId = StringUtils.trimToNull(departmentId);
        if (normalizedDepartmentId == null) {
            return;
        }
        List<String> scopedDepartmentIds = listDepartmentIds(normalizedDepartmentId,
            includeChildren == null || includeChildren.booleanValue());
        if (scopedDepartmentIds.isEmpty()) {
            sql.append(" AND department_id = ''");
            return;
        }
        sql.append(" AND department_id IN (");
        StringJoiner joiner = new StringJoiner(", ");
        for (String ignored : scopedDepartmentIds) {
            joiner.add("?");
        }
        sql.append(joiner.toString()).append(")");
        args.addAll(scopedDepartmentIds);
    }

    private List<String> listDepartmentIds(String departmentId, boolean includeChildren) {
        String normalizedDepartmentId = StringUtils.trimToNull(departmentId);
        if (normalizedDepartmentId == null) {
            return java.util.Collections.emptyList();
        }
        if (!includeChildren) {
            return java.util.Collections.singletonList(normalizedDepartmentId);
        }
        try {
            String rootPath = null;
            try (Connection connection = openConnection();
                PreparedStatement statement = connection.prepareStatement(
                    "SELECT path FROM org_department WHERE department_id = ? LIMIT 1")) {
                statement.setString(1, normalizedDepartmentId);
                try (ResultSet rs = statement.executeQuery()) {
                    if (rs.next()) {
                        rootPath = rs.getString("path");
                    }
                }
            }
            if (StringUtils.isBlank(rootPath)) {
                return java.util.Collections.singletonList(normalizedDepartmentId);
            }
            List<String> result = new ArrayList<>();
            try (Connection connection = openConnection();
                PreparedStatement statement = connection.prepareStatement(
                    "SELECT department_id FROM org_department WHERE department_id = ? OR path = ? OR path LIKE ? ORDER BY path ASC")) {
                statement.setString(1, normalizedDepartmentId);
                statement.setString(2, rootPath);
                statement.setString(3, rootPath + "/%");
                try (ResultSet rs = statement.executeQuery()) {
                    while (rs.next()) {
                        result.add(StringUtils.trimToEmpty(rs.getString("department_id")));
                    }
                }
            }
            return result.isEmpty() ? java.util.Collections.singletonList(normalizedDepartmentId) : result;
        } catch (Exception ex) {
            log.warn("Failed to resolve department subtree for portal stats, fallback to root department only.", ex);
            return java.util.Collections.singletonList(normalizedDepartmentId);
        }
    }

    private void bindArgs(PreparedStatement statement, List<Object> args) throws SQLException {
        for (int i = 0; i < args.size(); i++) {
            statement.setObject(i + 1, args.get(i));
        }
    }
}
