package com.alibaba.higress.console.service.portal;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.aiquota.AiQuotaUserPolicy;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.exception.BusinessException;
import com.alibaba.higress.sdk.exception.ValidationException;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalUserQuotaPolicyJdbcService {

    private static final String DAILY_RESET_MODE_FIXED = "fixed";
    private static final String DEFAULT_DAILY_RESET_TIME = "00:00";
    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    @PostConstruct
    public void init() {
        ensureQuotaPolicyTable();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public AiQuotaUserPolicy getUserPolicy(String consumerName) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        String normalizedConsumer = StringUtils.trimToNull(consumerName);
        if (normalizedConsumer == null) {
            throw new ValidationException("consumerName cannot be blank.");
        }

        String sql = "SELECT consumer_name, limit_total_micro_yuan, limit_5h_micro_yuan, limit_daily_micro_yuan, "
            + "daily_reset_mode, daily_reset_time, limit_weekly_micro_yuan, limit_monthly_micro_yuan, cost_reset_at "
            + "FROM quota_policy_user WHERE consumer_name = ? LIMIT 1";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, normalizedConsumer);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapPolicy(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query Portal user quota policy for consumer {}.", normalizedConsumer, ex);
            throw new BusinessException("Failed to query Portal user quota policy.", ex);
        }
        return defaultPolicy(normalizedConsumer);
    }

    public AiQuotaUserPolicy saveUserPolicy(String consumerName, AiQuotaUserPolicyRequestData request) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        String normalizedConsumer = StringUtils.trimToNull(consumerName);
        if (normalizedConsumer == null) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        AiQuotaUserPolicy normalized = normalizeRequest(normalizedConsumer, request);
        String sql = "INSERT INTO quota_policy_user "
            + "(consumer_name, limit_total_micro_yuan, limit_5h_micro_yuan, limit_daily_micro_yuan, "
            + "daily_reset_mode, daily_reset_time, limit_weekly_micro_yuan, limit_monthly_micro_yuan, cost_reset_at, "
            + "created_at, updated_at) "
            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) "
            + "ON DUPLICATE KEY UPDATE "
            + "limit_total_micro_yuan = VALUES(limit_total_micro_yuan), "
            + "limit_5h_micro_yuan = VALUES(limit_5h_micro_yuan), "
            + "limit_daily_micro_yuan = VALUES(limit_daily_micro_yuan), "
            + "daily_reset_mode = VALUES(daily_reset_mode), "
            + "daily_reset_time = VALUES(daily_reset_time), "
            + "limit_weekly_micro_yuan = VALUES(limit_weekly_micro_yuan), "
            + "limit_monthly_micro_yuan = VALUES(limit_monthly_micro_yuan), "
            + "cost_reset_at = VALUES(cost_reset_at), "
            + "updated_at = CURRENT_TIMESTAMP";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, normalized.getConsumerName());
            statement.setLong(2, normalized.getLimitTotal());
            statement.setLong(3, normalized.getLimit5h());
            statement.setLong(4, normalized.getLimitDaily());
            statement.setString(5, normalized.getDailyResetMode());
            statement.setString(6, normalized.getDailyResetTime());
            statement.setLong(7, normalized.getLimitWeekly());
            statement.setLong(8, normalized.getLimitMonthly());
            if (StringUtils.isBlank(normalized.getCostResetAt())) {
                statement.setTimestamp(9, null);
            } else {
                statement.setTimestamp(9, parseTimestamp(normalized.getCostResetAt()));
            }
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to save Portal user quota policy for consumer {}.", normalizedConsumer, ex);
            throw new BusinessException("Failed to save Portal user quota policy.", ex);
        }
        return getUserPolicy(normalizedConsumer);
    }

    private void ensureQuotaPolicyTable() {
        if (!enabled()) {
            return;
        }
        String ddl = "CREATE TABLE IF NOT EXISTS quota_policy_user ("
            + "id BIGINT AUTO_INCREMENT PRIMARY KEY,"
            + "consumer_name VARCHAR(128) NOT NULL UNIQUE,"
            + "limit_total_micro_yuan BIGINT NOT NULL DEFAULT 0,"
            + "limit_5h_micro_yuan BIGINT NOT NULL DEFAULT 0,"
            + "limit_daily_micro_yuan BIGINT NOT NULL DEFAULT 0,"
            + "daily_reset_mode VARCHAR(16) NOT NULL DEFAULT 'fixed',"
            + "daily_reset_time VARCHAR(5) NOT NULL DEFAULT '00:00',"
            + "limit_weekly_micro_yuan BIGINT NOT NULL DEFAULT 0,"
            + "limit_monthly_micro_yuan BIGINT NOT NULL DEFAULT 0,"
            + "cost_reset_at DATETIME NULL,"
            + "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
            + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(ddl)) {
            statement.execute();
        } catch (SQLException ex) {
            log.warn("Failed to ensure Portal quota_policy_user table.", ex);
        }
    }

    private AiQuotaUserPolicy normalizeRequest(String consumerName, AiQuotaUserPolicyRequestData request) {
        if (request == null) {
            throw new ValidationException("user quota policy request cannot be null.");
        }
        long limitTotal = normalizeNonNegative(request.getLimitTotal(), "limitTotal");
        long limit5h = normalizeNonNegative(request.getLimit5h(), "limit5h");
        long limitDaily = normalizeNonNegative(request.getLimitDaily(), "limitDaily");
        long limitWeekly = normalizeNonNegative(request.getLimitWeekly(), "limitWeekly");
        long limitMonthly = normalizeNonNegative(request.getLimitMonthly(), "limitMonthly");
        String dailyResetMode = StringUtils.defaultIfBlank(StringUtils.trimToEmpty(request.getDailyResetMode()),
            DAILY_RESET_MODE_FIXED);
        if (!StringUtils.equalsIgnoreCase(dailyResetMode, DAILY_RESET_MODE_FIXED)) {
            throw new ValidationException("dailyResetMode only supports fixed.");
        }
        String dailyResetTime = StringUtils.defaultIfBlank(StringUtils.trimToEmpty(request.getDailyResetTime()),
            DEFAULT_DAILY_RESET_TIME);
        if (!dailyResetTime.matches("^([01][0-9]|2[0-3]):[0-5][0-9]$")) {
            throw new ValidationException("dailyResetTime must match HH:mm.");
        }
        String costResetAt = StringUtils.trimToNull(request.getCostResetAt());
        if (costResetAt != null) {
            costResetAt = parseTimestamp(costResetAt).toInstant().toString();
        }
        return AiQuotaUserPolicy.builder()
            .consumerName(consumerName)
            .limitTotal(limitTotal)
            .limit5h(limit5h)
            .limitDaily(limitDaily)
            .dailyResetMode(DAILY_RESET_MODE_FIXED)
            .dailyResetTime(dailyResetTime)
            .limitWeekly(limitWeekly)
            .limitMonthly(limitMonthly)
            .costResetAt(costResetAt)
            .build();
    }

    private AiQuotaUserPolicy mapPolicy(ResultSet rs) throws SQLException {
        Timestamp costResetAt = rs.getTimestamp("cost_reset_at");
        return AiQuotaUserPolicy.builder()
            .consumerName(rs.getString("consumer_name"))
            .limitTotal(rs.getLong("limit_total_micro_yuan"))
            .limit5h(rs.getLong("limit_5h_micro_yuan"))
            .limitDaily(rs.getLong("limit_daily_micro_yuan"))
            .dailyResetMode(StringUtils.defaultIfBlank(rs.getString("daily_reset_mode"), DAILY_RESET_MODE_FIXED))
            .dailyResetTime(StringUtils.defaultIfBlank(rs.getString("daily_reset_time"), DEFAULT_DAILY_RESET_TIME))
            .limitWeekly(rs.getLong("limit_weekly_micro_yuan"))
            .limitMonthly(rs.getLong("limit_monthly_micro_yuan"))
            .costResetAt(costResetAt != null ? costResetAt.toInstant().toString() : null)
            .build();
    }

    private AiQuotaUserPolicy defaultPolicy(String consumerName) {
        return AiQuotaUserPolicy.builder()
            .consumerName(consumerName)
            .limitTotal(0L)
            .limit5h(0L)
            .limitDaily(0L)
            .dailyResetMode(DAILY_RESET_MODE_FIXED)
            .dailyResetTime(DEFAULT_DAILY_RESET_TIME)
            .limitWeekly(0L)
            .limitMonthly(0L)
            .costResetAt(null)
            .build();
    }

    private long normalizeNonNegative(Long value, String field) {
        long normalized = value == null ? 0L : value.longValue();
        if (normalized < 0) {
            throw new ValidationException(field + " cannot be negative.");
        }
        return normalized;
    }

    private Timestamp parseTimestamp(String value) {
        return ConsoleDateTimeUtil.parseTimestamp(
            value,
            "costResetAt",
            "RFC3339 or yyyy-MM-dd'T'HH:mm[:ss] in UTC");
    }

    private Connection openConnection() throws SQLException {
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    public static class AiQuotaUserPolicyRequestData {
        private final Long limitTotal;
        private final Long limit5h;
        private final Long limitDaily;
        private final String dailyResetMode;
        private final String dailyResetTime;
        private final Long limitWeekly;
        private final Long limitMonthly;
        private final String costResetAt;

        public AiQuotaUserPolicyRequestData(Long limitTotal, Long limit5h, Long limitDaily, String dailyResetMode,
            String dailyResetTime, Long limitWeekly, Long limitMonthly, String costResetAt) {
            this.limitTotal = limitTotal;
            this.limit5h = limit5h;
            this.limitDaily = limitDaily;
            this.dailyResetMode = dailyResetMode;
            this.dailyResetTime = dailyResetTime;
            this.limitWeekly = limitWeekly;
            this.limitMonthly = limitMonthly;
            this.costResetAt = costResetAt;
        }

        public Long getLimitTotal() {
            return limitTotal;
        }

        public Long getLimit5h() {
            return limit5h;
        }

        public Long getLimitDaily() {
            return limitDaily;
        }

        public String getDailyResetMode() {
            return dailyResetMode;
        }

        public String getDailyResetTime() {
            return dailyResetTime;
        }

        public Long getLimitWeekly() {
            return limitWeekly;
        }

        public Long getLimitMonthly() {
            return limitMonthly;
        }

        public String getCostResetAt() {
            return costResetAt;
        }
    }
}
