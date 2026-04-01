package com.alibaba.higress.console.service.portal;

import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.exception.BusinessException;
import com.alibaba.higress.sdk.exception.ValidationException;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalBillingQuotaJdbcService {

    private static final String CURRENCY_CNY = "CNY";

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    @PostConstruct
    public void init() {
        ensureBillingTables();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public Map<String, Long> listConsumerBalances(List<String> consumerNames) {
        if (!enabled() || consumerNames == null || consumerNames.isEmpty()) {
            return Collections.emptyMap();
        }
        List<String> normalized = consumerNames.stream()
            .map(StringUtils::trimToEmpty)
            .filter(StringUtils::isNotBlank)
            .distinct()
            .collect(Collectors.toList());
        if (normalized.isEmpty()) {
            return Collections.emptyMap();
        }

        String placeholders = normalized.stream().map(item -> "?").collect(Collectors.joining(","));
        String sql = "SELECT consumer_name, available_micro_yuan FROM billing_wallet WHERE consumer_name IN (" + placeholders + ")";
        Map<String, Long> result = new HashMap<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            for (int i = 0; i < normalized.size(); i++) {
                statement.setString(i + 1, normalized.get(i));
            }
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    result.put(rs.getString("consumer_name"), rs.getLong("available_micro_yuan"));
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list Portal billing balances.", ex);
            throw new BusinessException("Failed to query Portal billing balances.", ex);
        }
        return result;
    }

    public long refreshConsumerBalance(String consumerName, long balanceMicroYuan, String sourceHint) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        String normalizedConsumer = StringUtils.trimToNull(consumerName);
        if (normalizedConsumer == null) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        if (balanceMicroYuan < 0) {
            throw new ValidationException("balance cannot be negative.");
        }

        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                long current = queryCurrentBalance(connection, normalizedConsumer);
                long delta = balanceMicroYuan - current;
                upsertWallet(connection, normalizedConsumer, balanceMicroYuan, true);
                insertAdjustTransaction(connection, normalizedConsumer, delta, "console_ai_quota_refresh",
                    buildSourceId(sourceHint, normalizedConsumer, balanceMicroYuan));
                connection.commit();
                return balanceMicroYuan;
            } catch (SQLException ex) {
                connection.rollback();
                throw ex;
            } finally {
                connection.setAutoCommit(true);
            }
        } catch (SQLException ex) {
            log.warn("Failed to refresh Portal billing balance for consumer {}.", normalizedConsumer, ex);
            throw new BusinessException("Failed to refresh Portal billing balance.", ex);
        }
    }

    public long deltaConsumerBalance(String consumerName, long deltaMicroYuan, String sourceHint) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        String normalizedConsumer = StringUtils.trimToNull(consumerName);
        if (normalizedConsumer == null) {
            throw new ValidationException("consumerName cannot be blank.");
        }

        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                long current = queryCurrentBalance(connection, normalizedConsumer);
                long next = current + deltaMicroYuan;
                upsertWallet(connection, normalizedConsumer, next, false);
                insertAdjustTransaction(connection, normalizedConsumer, deltaMicroYuan, "console_ai_quota_delta",
                    buildSourceId(sourceHint, normalizedConsumer, deltaMicroYuan));
                connection.commit();
                return next;
            } catch (SQLException ex) {
                connection.rollback();
                throw ex;
            } finally {
                connection.setAutoCommit(true);
            }
        } catch (SQLException ex) {
            log.warn("Failed to delta Portal billing balance for consumer {}.", normalizedConsumer, ex);
            throw new BusinessException("Failed to adjust Portal billing balance.", ex);
        }
    }

    private void ensureBillingTables() {
        if (!enabled()) {
            return;
        }
        String[] ddls = new String[] {
            "CREATE TABLE IF NOT EXISTS billing_wallet ("
                + "id BIGINT AUTO_INCREMENT PRIMARY KEY,"
                + "consumer_name VARCHAR(128) NOT NULL UNIQUE,"
                + "currency VARCHAR(8) NOT NULL DEFAULT 'CNY',"
                + "available_micro_yuan BIGINT NOT NULL DEFAULT 0,"
                + "version BIGINT NOT NULL DEFAULT 1,"
                + "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                + "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
                + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
            "CREATE TABLE IF NOT EXISTS billing_transaction ("
                + "id BIGINT AUTO_INCREMENT PRIMARY KEY,"
                + "tx_id VARCHAR(64) NOT NULL UNIQUE,"
                + "consumer_name VARCHAR(128) NOT NULL,"
                + "tx_type VARCHAR(16) NOT NULL,"
                + "amount_micro_yuan BIGINT NOT NULL,"
                + "currency VARCHAR(8) NOT NULL DEFAULT 'CNY',"
                + "source_type VARCHAR(64) NOT NULL,"
                + "source_id VARCHAR(128) NOT NULL,"
                + "request_id VARCHAR(128) NULL,"
                + "api_key_id VARCHAR(64) NULL,"
                + "model_id VARCHAR(128) NULL,"
                + "price_version_id BIGINT NULL,"
                + "input_tokens BIGINT NOT NULL DEFAULT 0,"
                + "output_tokens BIGINT NOT NULL DEFAULT 0,"
                + "occurred_at DATETIME NOT NULL,"
                + "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                + "UNIQUE KEY uk_billing_transaction_source (source_type, source_id)"
                + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
        };
        try (Connection connection = openConnection()) {
            for (String ddl : ddls) {
                try (PreparedStatement statement = connection.prepareStatement(ddl)) {
                    statement.execute();
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to ensure Portal billing tables for ai-quota sync.", ex);
        }
    }

    private long queryCurrentBalance(Connection connection, String consumerName) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(
            "SELECT available_micro_yuan FROM billing_wallet WHERE consumer_name = ? LIMIT 1")) {
            statement.setString(1, consumerName);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return rs.getLong("available_micro_yuan");
                }
            }
        }
        return 0L;
    }

    private void upsertWallet(Connection connection, String consumerName, long availableMicroYuan, boolean replace)
        throws SQLException {
        String sql;
        if (replace) {
            sql = "INSERT INTO billing_wallet (consumer_name, currency, available_micro_yuan, version) "
                + "VALUES (?, ?, ?, 1) "
                + "ON DUPLICATE KEY UPDATE available_micro_yuan = VALUES(available_micro_yuan), version = version + 1";
        } else {
            sql = "INSERT INTO billing_wallet (consumer_name, currency, available_micro_yuan, version) "
                + "VALUES (?, ?, ?, 1) "
                + "ON DUPLICATE KEY UPDATE available_micro_yuan = VALUES(available_micro_yuan), version = version + 1";
        }
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            statement.setString(2, CURRENCY_CNY);
            statement.setLong(3, availableMicroYuan);
            statement.executeUpdate();
        }
    }

    private void insertAdjustTransaction(Connection connection, String consumerName, long deltaMicroYuan, String sourceType,
        String sourceID) throws SQLException {
        if (deltaMicroYuan == 0) {
            return;
        }
        LocalDateTime now = ConsoleDateTimeUtil.now();
        try (PreparedStatement statement = connection.prepareStatement(
            "INSERT INTO billing_transaction "
                + "(tx_id, consumer_name, tx_type, amount_micro_yuan, currency, source_type, source_id, occurred_at, created_at) "
                + "VALUES (?, ?, 'adjust', ?, ?, ?, ?, ?, ?)")) {
            statement.setString(1, buildTransactionID(sourceType, sourceID));
            statement.setString(2, consumerName);
            statement.setLong(3, deltaMicroYuan);
            statement.setString(4, CURRENCY_CNY);
            statement.setString(5, sourceType);
            statement.setString(6, sourceID);
            statement.setTimestamp(7, ConsoleDateTimeUtil.toTimestamp(now));
            statement.setTimestamp(8, ConsoleDateTimeUtil.toTimestamp(now));
            statement.executeUpdate();
        }
    }

    private String buildSourceId(String sourceHint, String consumerName, long amount) {
        return String.join(":", StringUtils.defaultIfBlank(sourceHint, "ai-quota"),
            consumerName, Long.toString(amount), Long.toString(System.currentTimeMillis()));
    }

    private String buildTransactionID(String sourceType, String sourceID) {
        String digest = sha256Hex(sourceType + ":" + sourceID);
        return "a" + digest.substring(0, 32);
    }

    private Connection openConnection() throws SQLException {
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    private String sha256Hex(String value) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] hashed = digest.digest(StringUtils.trimToEmpty(value).getBytes(StandardCharsets.UTF_8));
            StringBuilder builder = new StringBuilder(hashed.length * 2);
            for (byte item : hashed) {
                builder.append(String.format("%02x", item));
            }
            return builder.toString();
        } catch (Exception ex) {
            throw new IllegalStateException("Failed to hash billing transaction id.", ex);
        }
    }
}
