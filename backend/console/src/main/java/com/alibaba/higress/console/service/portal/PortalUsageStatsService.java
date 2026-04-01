package com.alibaba.higress.console.service.portal;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.util.ArrayList;
import java.util.Comparator;
import java.util.List;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

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

    private boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }
}
