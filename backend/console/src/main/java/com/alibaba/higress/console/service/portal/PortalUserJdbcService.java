package com.alibaba.higress.console.service.portal;

import java.security.SecureRandom;
import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.UUID;
import java.util.stream.Collectors;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.RandomStringUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.PortalPasswordResetResult;
import com.alibaba.higress.console.model.portal.PortalUserRecord;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.model.consumer.Consumer;
import com.alibaba.higress.sdk.exception.ValidationException;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalUserJdbcService {

    public static final String BUILTIN_ADMIN_CONSUMER = "administrator";

    private static final String DEFAULT_USER_STATUS = "active";
    private static final String STATUS_PENDING = "pending";
    private static final String SOURCE_SYSTEM = "system";
    private static final String SOURCE_MIGRATION = "migration-keyauth";
    private static final String USER_LEVEL_NORMAL = "normal";
    private static final String USER_LEVEL_PLUS = "plus";
    private static final String USER_LEVEL_PRO = "pro";
    private static final String USER_LEVEL_ULTRA = "ultra";
    private static final List<String> SUPPORTED_USER_LEVELS =
        Arrays.asList(USER_LEVEL_NORMAL, USER_LEVEL_PLUS, USER_LEVEL_PRO, USER_LEVEL_ULTRA);
    private static final int RESET_PASSWORD_LENGTH = 16;
    private static final char[] UPPERCASE_CHARS = "ABCDEFGHJKLMNPQRSTUVWXYZ".toCharArray();
    private static final char[] LOWERCASE_CHARS = "abcdefghijkmnopqrstuvwxyz".toCharArray();
    private static final char[] DIGIT_CHARS = "23456789".toCharArray();
    private static final char[] SPECIAL_CHARS = "!@#$%^&*".toCharArray();
    private static final char[] ALL_PASSWORD_CHARS = (
        new String(UPPERCASE_CHARS) + new String(LOWERCASE_CHARS) + new String(DIGIT_CHARS)
            + new String(SPECIAL_CHARS)).toCharArray();

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    private final BCryptPasswordEncoder passwordEncoder = new BCryptPasswordEncoder();
    private final SecureRandom secureRandom = new SecureRandom();

    @PostConstruct
    public void init() {
        ensurePortalUserSchema();
        ensureAccountMembershipTable();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public boolean isBuiltinAdministrator(String consumerName) {
        return StringUtils.equalsIgnoreCase(StringUtils.trimToEmpty(consumerName), BUILTIN_ADMIN_CONSUMER);
    }

    public Map<String, PortalUserRecord> listByConsumerNames(List<String> consumerNames) {
        if (!enabled() || consumerNames == null || consumerNames.isEmpty()) {
            return Collections.emptyMap();
        }
        List<String> names = consumerNames.stream().filter(StringUtils::isNotBlank).distinct().collect(Collectors.toList());
        if (names.isEmpty()) {
            return Collections.emptyMap();
        }

        String placeholders = names.stream().map(i -> "?").collect(Collectors.joining(","));
        String sql = "SELECT u.consumer_name, u.display_name, u.email, m.department_id, m.parent_consumer_name, "
            + "u.user_level, u.status, u.source, u.last_login_at, u.is_deleted "
            + "FROM portal_user u LEFT JOIN org_account_membership m ON u.consumer_name = m.consumer_name "
            + "WHERE u.consumer_name IN (" + placeholders + ") AND COALESCE(u.is_deleted, 0) = 0";

        Map<String, PortalUserRecord> result = new HashMap<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            for (int i = 0; i < names.size(); i++) {
                statement.setString(i + 1, names.get(i));
            }
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    PortalUserRecord record = mapRecord(rs);
                    result.put(record.getConsumerName(), record);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to load portal users from MySQL.", ex);
        }
        return result;
    }

    public List<PortalUserRecord> listAllUsers() {
        if (!enabled()) {
            return Collections.emptyList();
        }
        String sql = "SELECT u.consumer_name, u.display_name, u.email, m.department_id, m.parent_consumer_name, "
            + "u.user_level, u.status, u.source, u.last_login_at, u.is_deleted "
            + "FROM portal_user u LEFT JOIN org_account_membership m ON u.consumer_name = m.consumer_name "
            + "WHERE COALESCE(u.is_deleted, 0) = 0 "
            + "ORDER BY u.consumer_name ASC";
        List<PortalUserRecord> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                result.add(mapRecord(rs));
            }
        } catch (SQLException ex) {
            log.warn("Failed to load all portal users from MySQL.", ex);
        }
        return result;
    }

    public List<String> listDistinctDepartments() {
        return Collections.emptyList();
    }

    public PortalUserRecord queryByConsumerName(String consumerName) {
        return queryByConsumerNameInternal(consumerName, false);
    }

    private PortalUserRecord queryByConsumerNameAny(String consumerName) {
        return queryByConsumerNameInternal(consumerName, true);
    }

    private PortalUserRecord queryByConsumerNameInternal(String consumerName, boolean includeDeleted) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return null;
        }
        String sql = "SELECT u.consumer_name, u.display_name, u.email, m.department_id, m.parent_consumer_name, "
            + "u.user_level, u.status, u.source, u.last_login_at, u.is_deleted "
            + "FROM portal_user u LEFT JOIN org_account_membership m ON u.consumer_name = m.consumer_name "
            + "WHERE u.consumer_name = ?"
            + (includeDeleted ? "" : " AND COALESCE(u.is_deleted, 0) = 0");
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapRecord(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query portal user {} from MySQL.", consumerName, ex);
        }
        return null;
    }

    public PortalUserRecord upsertFromConsumer(Consumer consumer, String defaultSource) {
        if (!enabled() || consumer == null || StringUtils.isBlank(consumer.getName())) {
            return null;
        }

        String consumerName = consumer.getName();
        PortalUserRecord existed = queryByConsumerNameAny(consumerName);

        String displayName = StringUtils.defaultIfBlank(
            StringUtils.firstNonBlank(consumer.getPortalDisplayName(), existed == null ? null : existed.getDisplayName()),
            consumerName);
        String email = StringUtils.defaultString(
            StringUtils.trimToNull(StringUtils.firstNonBlank(consumer.getPortalEmail(),
                existed == null ? null : existed.getEmail())));
        String userLevel = normalizeUserLevel(StringUtils.firstNonBlank(consumer.getPortalUserLevel(),
            existed == null ? null : existed.getUserLevel(), USER_LEVEL_NORMAL));
        String status = StringUtils.firstNonBlank(consumer.getPortalStatus(),
            existed == null ? null : existed.getStatus(), DEFAULT_USER_STATUS);
        String source = StringUtils.firstNonBlank(consumer.getPortalUserSource(),
            existed == null ? null : existed.getSource(), defaultSource, "console");

        String password = StringUtils.trimToNull(consumer.getPortalPassword());
        String tempPassword = null;
        if (existed == null && password == null) {
            tempPassword = RandomStringUtils.randomAlphanumeric(12);
            password = tempPassword;
        }

        try (Connection connection = openConnection()) {
            if (existed == null) {
                String insertSql = "INSERT INTO portal_user "
                    + "(consumer_name, display_name, email, user_level, password_hash, status, source) "
                    + "VALUES (?, ?, ?, ?, ?, ?, ?)";
                try (PreparedStatement statement = connection.prepareStatement(insertSql)) {
                    statement.setString(1, consumerName);
                    statement.setString(2, displayName);
                    statement.setString(3, email);
                    statement.setString(4, userLevel);
                    statement.setString(5, passwordEncoder.encode(password));
                    statement.setString(6, status);
                    statement.setString(7, source);
                    statement.executeUpdate();
                }
            } else {
                String updateSql;
                if (password == null) {
                    updateSql = "UPDATE portal_user SET display_name=?, email=?, user_level=?, status=?, source=?, "
                        + "is_deleted=0, deleted_at=NULL WHERE consumer_name=?";
                } else {
                    updateSql = "UPDATE portal_user SET display_name=?, email=?, user_level=?, status=?, source=?, "
                        + "password_hash=?, is_deleted=0, deleted_at=NULL WHERE consumer_name=?";
                }
                try (PreparedStatement statement = connection.prepareStatement(updateSql)) {
                    int idx = 1;
                    statement.setString(idx++, displayName);
                    statement.setString(idx++, email);
                    statement.setString(idx++, userLevel);
                    statement.setString(idx++, status);
                    statement.setString(idx++, source);
                    if (password != null) {
                        statement.setString(idx++, passwordEncoder.encode(password));
                    }
                    statement.setString(idx, consumerName);
                    statement.executeUpdate();
                }
            }
            ensureMembershipRow(connection, consumerName);
        } catch (SQLException ex) {
            log.warn("Failed to upsert portal user {}.", consumerName, ex);
            return null;
        }

        PortalUserRecord updated = queryByConsumerName(consumerName);
        if (updated != null) {
            updated.setTempPassword(tempPassword);
        }
        return updated;
    }

    public void updateStatus(String consumerName, String status) {
        if (!enabled() || StringUtils.isBlank(consumerName) || StringUtils.isBlank(status)) {
            return;
        }
        String sql = "UPDATE portal_user SET status = ? WHERE consumer_name = ? AND COALESCE(is_deleted, 0) = 0";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, status);
            statement.setString(2, consumerName);
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to update portal user status for {}.", consumerName, ex);
        }
    }

    public void logicalDelete(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return;
        }
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                try (PreparedStatement statement = connection.prepareStatement(
                    "UPDATE portal_user SET status='disabled', is_deleted=1, deleted_at=CURRENT_TIMESTAMP "
                        + "WHERE consumer_name = ? AND COALESCE(is_deleted, 0) = 0")) {
                    statement.setString(1, consumerName);
                    int affected = statement.executeUpdate();
                    if (affected <= 0) {
                        throw new ValidationException("Consumer not found: " + consumerName);
                    }
                }
                try (PreparedStatement statement = connection.prepareStatement(
                    "UPDATE portal_api_key SET status='disabled', deleted_at=COALESCE(deleted_at, CURRENT_TIMESTAMP) "
                        + "WHERE consumer_name = ?")) {
                    statement.setString(1, consumerName);
                    statement.executeUpdate();
                }
                try (PreparedStatement statement = connection.prepareStatement(
                    "DELETE FROM portal_session WHERE consumer_name = ?")) {
                    statement.setString(1, consumerName);
                    statement.executeUpdate();
                }
                try (PreparedStatement statement = connection.prepareStatement(
                    "UPDATE org_account_membership SET parent_consumer_name = NULL WHERE parent_consumer_name = ?")) {
                    statement.setString(1, consumerName);
                    statement.executeUpdate();
                }
                connection.commit();
            } catch (Exception ex) {
                connection.rollback();
                throw ex;
            } finally {
                connection.setAutoCommit(true);
            }
        } catch (ValidationException ex) {
            throw ex;
        } catch (Exception ex) {
            log.warn("Failed to logically delete portal user {}.", consumerName, ex);
            throw new IllegalStateException("Failed to delete consumer.");
        }
    }

    public void disableAllApiKeys(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return;
        }
        String sql = "UPDATE portal_api_key SET status='disabled' WHERE consumer_name = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to disable portal api keys for {}.", consumerName, ex);
        }
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    private PortalUserRecord mapRecord(ResultSet rs) throws SQLException {
        Timestamp lastLogin = rs.getTimestamp("last_login_at");
        LocalDateTime lastLoginAt = null;
        if (lastLogin != null) {
            lastLoginAt = ConsoleDateTimeUtil.toLocalDateTime(lastLogin);
        }
        return PortalUserRecord.builder().consumerName(rs.getString("consumer_name"))
            .displayName(rs.getString("display_name")).email(rs.getString("email"))
            .departmentId(rs.getString("department_id"))
            .parentConsumerName(rs.getString("parent_consumer_name"))
            .userLevel(normalizeUserLevel(rs.getString("user_level")))
            .status(rs.getString("status"))
            .source(rs.getString("source")).deleted(rs.getBoolean("is_deleted")).lastLoginAt(lastLoginAt).build();
    }

    private String normalizeUserLevel(String userLevel) {
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(userLevel));
        if (SUPPORTED_USER_LEVELS.contains(normalized)) {
            return normalized;
        }
        return USER_LEVEL_NORMAL;
    }

    private void ensurePortalUserSchema() {
        if (!enabled()) {
            return;
        }
        String createTableSql = "CREATE TABLE IF NOT EXISTS portal_user ("
            + "id BIGINT AUTO_INCREMENT PRIMARY KEY,"
            + "consumer_name VARCHAR(128) NOT NULL UNIQUE,"
            + "display_name VARCHAR(128) NOT NULL,"
            + "email VARCHAR(255) NOT NULL DEFAULT '',"
            + "password_hash VARCHAR(255) NOT NULL,"
            + "status VARCHAR(16) NOT NULL DEFAULT 'active',"
            + "source VARCHAR(16) NOT NULL DEFAULT 'portal',"
            + "last_login_at DATETIME NULL,"
            + "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
            + "user_level VARCHAR(16) NOT NULL DEFAULT 'normal'"
            + ")";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(createTableSql)) {
            statement.executeUpdate();
            ensurePortalUserColumn(connection, "display_name",
                "ALTER TABLE portal_user ADD COLUMN display_name VARCHAR(128) NOT NULL DEFAULT ''");
            ensurePortalUserColumn(connection, "email",
                "ALTER TABLE portal_user ADD COLUMN email VARCHAR(255) NOT NULL DEFAULT ''");
            ensurePortalUserColumn(connection, "password_hash",
                "ALTER TABLE portal_user ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT ''");
            ensurePortalUserColumn(connection, "status",
                "ALTER TABLE portal_user ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT 'active'");
            ensurePortalUserColumn(connection, "source",
                "ALTER TABLE portal_user ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'portal'");
            ensurePortalUserColumn(connection, "last_login_at",
                "ALTER TABLE portal_user ADD COLUMN last_login_at DATETIME NULL");
            ensurePortalUserColumn(connection, "created_at",
                "ALTER TABLE portal_user ADD COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP");
            ensurePortalUserColumn(connection, "updated_at",
                "ALTER TABLE portal_user ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP "
                    + "ON UPDATE CURRENT_TIMESTAMP");
            ensurePortalUserColumn(connection, "user_level",
                "ALTER TABLE portal_user ADD COLUMN user_level VARCHAR(16) NOT NULL DEFAULT 'normal'");
            ensurePortalUserColumn(connection, "is_deleted",
                "ALTER TABLE portal_user ADD COLUMN is_deleted TINYINT(1) NOT NULL DEFAULT 0");
            ensurePortalUserColumn(connection, "deleted_at",
                "ALTER TABLE portal_user ADD COLUMN deleted_at DATETIME NULL");
        } catch (SQLException ex) {
            log.warn("Failed to ensure portal_user schema.", ex);
            return;
        }
    }

    private void ensurePortalUserColumn(Connection connection, String columnName, String alterSql) throws SQLException {
        String existsSql = "SELECT COUNT(1) AS cnt FROM information_schema.COLUMNS "
            + "WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'portal_user' AND COLUMN_NAME = ?";
        try (PreparedStatement statement = connection.prepareStatement(existsSql)) {
            statement.setString(1, columnName);
            try (ResultSet resultSet = statement.executeQuery()) {
                if (resultSet.next() && resultSet.getInt("cnt") > 0) {
                    return;
                }
            }
        }
        try (PreparedStatement statement = connection.prepareStatement(alterSql)) {
            statement.executeUpdate();
        }
    }

    public List<String> listActiveRawKeys(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return Collections.emptyList();
        }
        String sql = "SELECT raw_key FROM portal_api_key WHERE consumer_name=? AND status='active' "
            + "AND deleted_at IS NULL ORDER BY id ASC";
        List<String> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    String raw = rs.getString("raw_key");
                    if (StringUtils.isNotBlank(raw)) {
                        result.add(raw);
                    }
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list active raw keys for {}.", consumerName, ex);
        }
        return result;
    }

    public Map<String, List<String>> listActiveRawKeysByConsumerNames(List<String> consumerNames) {
        if (!enabled() || consumerNames == null || consumerNames.isEmpty()) {
            return Collections.emptyMap();
        }
        List<String> names = consumerNames.stream().filter(StringUtils::isNotBlank).distinct().collect(Collectors.toList());
        if (names.isEmpty()) {
            return Collections.emptyMap();
        }

        String placeholders = names.stream().map(i -> "?").collect(Collectors.joining(","));
        String sql =
            "SELECT consumer_name, raw_key FROM portal_api_key WHERE status='active' AND deleted_at IS NULL "
                + "AND consumer_name IN (" + placeholders + ") ORDER BY id ASC";
        Map<String, List<String>> result = new HashMap<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            for (int i = 0; i < names.size(); i++) {
                statement.setString(i + 1, names.get(i));
            }
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    String consumerName = rs.getString("consumer_name");
                    String raw = rs.getString("raw_key");
                    if (StringUtils.isBlank(consumerName) || StringUtils.isBlank(raw)) {
                        continue;
                    }
                    result.computeIfAbsent(consumerName, k -> new ArrayList<>()).add(raw);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list active raw keys in batch.", ex);
        }
        return result;
    }

    public PortalUserRecord ensureMigrationUser(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return null;
        }
        PortalUserRecord existed = queryByConsumerNameAny(consumerName);
        if (existed != null) {
            return existed;
        }
        String sql = "INSERT INTO portal_user (consumer_name, display_name, email, user_level, password_hash, status, source) "
            + "VALUES (?, ?, ?, ?, ?, ?, ?)";
        String randomPassword = UUID.randomUUID().toString().replace("-", "");
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            statement.setString(2, consumerName);
            statement.setString(3, "");
            statement.setString(4, USER_LEVEL_NORMAL);
            statement.setString(5, passwordEncoder.encode(randomPassword));
            statement.setString(6, STATUS_PENDING);
            statement.setString(7, SOURCE_MIGRATION);
            statement.executeUpdate();
            ensureMembershipRow(connection, consumerName);
        } catch (SQLException ex) {
            if (!isDuplicateKey(ex)) {
                log.warn("Failed to ensure migration portal user {}.", consumerName, ex);
                return null;
            }
        }
        return queryByConsumerName(consumerName);
    }

    public PortalUserRecord ensureBuiltinAdministrator() {
        if (!enabled()) {
            return null;
        }
        PortalUserRecord existed = queryByConsumerNameAny(BUILTIN_ADMIN_CONSUMER);
        if (existed == null) {
            String insertSql = "INSERT INTO portal_user "
                + "(consumer_name, display_name, email, user_level, password_hash, status, source) "
                + "VALUES (?, ?, ?, ?, ?, ?, ?)";
            String randomPassword = UUID.randomUUID().toString().replace("-", "");
            try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(
                insertSql)) {
                statement.setString(1, BUILTIN_ADMIN_CONSUMER);
                statement.setString(2, BUILTIN_ADMIN_CONSUMER);
                statement.setString(3, "");
                statement.setString(4, USER_LEVEL_NORMAL);
                statement.setString(5, passwordEncoder.encode(randomPassword));
                statement.setString(6, STATUS_PENDING);
                statement.setString(7, SOURCE_SYSTEM);
                statement.executeUpdate();
                ensureMembershipRow(connection, BUILTIN_ADMIN_CONSUMER);
            } catch (SQLException ex) {
                if (!isDuplicateKey(ex)) {
                    log.warn("Failed to ensure built-in administrator user.", ex);
                    return null;
                }
            }
            return queryByConsumerName(BUILTIN_ADMIN_CONSUMER);
        }

        String normalizedStatus = StringUtils.lowerCase(StringUtils.trimToEmpty(existed.getStatus()));
        String normalizedSource = StringUtils.lowerCase(StringUtils.trimToEmpty(existed.getSource()));
        if (STATUS_PENDING.equals(normalizedStatus) && SOURCE_SYSTEM.equals(normalizedSource)) {
            return existed;
        }

        String updateSql = "UPDATE portal_user SET status=?, source=? WHERE consumer_name=?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(updateSql)) {
            statement.setString(1, STATUS_PENDING);
            statement.setString(2, SOURCE_SYSTEM);
            statement.setString(3, BUILTIN_ADMIN_CONSUMER);
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to enforce built-in administrator user status/source.", ex);
            return null;
        }
        return queryByConsumerName(BUILTIN_ADMIN_CONSUMER);
    }

    public void upsertMigratedApiKey(String consumerName, String rawKey) {
        if (!enabled() || StringUtils.isBlank(consumerName) || StringUtils.isBlank(rawKey)) {
            return;
        }
        String keyHash = sha256Hex(rawKey.trim());
        String querySql = "SELECT key_id FROM portal_api_key WHERE consumer_name = ? AND key_hash = ? LIMIT 1";
        try (Connection connection = openConnection()) {
            try (PreparedStatement query = connection.prepareStatement(querySql)) {
                query.setString(1, consumerName);
                query.setString(2, keyHash);
                try (ResultSet rs = query.executeQuery()) {
                    if (rs.next()) {
                        return;
                    }
                }
            }

            String insertSql = "INSERT INTO portal_api_key (key_id, consumer_name, name, key_masked, key_hash, raw_key, status) "
                + "VALUES (?, ?, ?, ?, ?, ?, 'active')";
            String keyId = "MIG" + UUID.randomUUID().toString().replace("-", "").substring(0, 29);
            String name = "Migrated Key";
            try (PreparedStatement insert = connection.prepareStatement(insertSql)) {
                insert.setString(1, keyId);
                insert.setString(2, consumerName);
                insert.setString(3, name);
                insert.setString(4, maskKey(rawKey.trim()));
                insert.setString(5, keyHash);
                insert.setString(6, rawKey.trim());
                insert.executeUpdate();
            }
        } catch (SQLException ex) {
            if (!isDuplicateKey(ex)) {
                log.warn("Failed to upsert migrated api key for {}.", consumerName, ex);
            }
        }
    }

    public PortalPasswordResetResult resetPassword(String consumerName) {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
        if (StringUtils.isBlank(consumerName)) {
            throw new ValidationException("consumerName cannot be blank.");
        }
        PortalUserRecord record = queryByConsumerName(consumerName);
        if (record == null) {
            throw new ValidationException("Consumer not found: " + consumerName);
        }

        String tempPassword = generateTempPassword();
        String sql = "UPDATE portal_user SET password_hash = ? WHERE consumer_name = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, passwordEncoder.encode(tempPassword));
            statement.setString(2, consumerName);
            int affectedRows = statement.executeUpdate();
            if (affectedRows <= 0) {
                throw new ValidationException("Consumer not found: " + consumerName);
            }
        } catch (SQLException ex) {
            log.warn("Failed to reset portal password for {}.", consumerName, ex);
            throw new IllegalStateException("Failed to reset portal password.");
        }

        return PortalPasswordResetResult.builder().consumerName(consumerName).tempPassword(tempPassword)
            .updatedAt(ConsoleDateTimeUtil.now()).build();
    }

    private String generateTempPassword() {
        char[] result = new char[RESET_PASSWORD_LENGTH];
        int index = 0;
        result[index++] = pickRandomChar(UPPERCASE_CHARS);
        result[index++] = pickRandomChar(LOWERCASE_CHARS);
        result[index++] = pickRandomChar(DIGIT_CHARS);
        result[index++] = pickRandomChar(SPECIAL_CHARS);
        while (index < result.length) {
            result[index++] = pickRandomChar(ALL_PASSWORD_CHARS);
        }
        shuffleChars(result);
        return new String(result);
    }

    private char pickRandomChar(char[] source) {
        return source[secureRandom.nextInt(source.length)];
    }

    private void shuffleChars(char[] data) {
        for (int i = data.length - 1; i > 0; i--) {
            int j = secureRandom.nextInt(i + 1);
            char tmp = data[i];
            data[i] = data[j];
            data[j] = tmp;
        }
    }

    private boolean isDuplicateKey(SQLException ex) {
        String sqlState = ex.getSQLState();
        if ("23000".equals(sqlState) || "23505".equals(sqlState)) {
            return true;
        }
        String message = StringUtils.defaultString(ex.getMessage()).toLowerCase(Locale.ROOT);
        return message.contains("duplicate") || message.contains("unique");
    }

    private String maskKey(String raw) {
        if (raw == null || raw.length() <= 8) {
            return "****";
        }
        return raw.substring(0, 4) + "****" + raw.substring(raw.length() - 4);
    }

    private String sha256Hex(String value) {
        try {
            MessageDigest digest = MessageDigest.getInstance("SHA-256");
            byte[] bytes = digest.digest(value.getBytes(java.nio.charset.StandardCharsets.UTF_8));
            StringBuilder builder = new StringBuilder(bytes.length * 2);
            for (byte b : bytes) {
                builder.append(String.format("%02x", b));
            }
            return builder.toString();
        } catch (NoSuchAlgorithmException ex) {
            throw new IllegalStateException("SHA-256 algorithm is unavailable.", ex);
        }
    }

    private void ensureAccountMembershipTable() {
        if (!enabled()) {
            return;
        }
        String sql = "CREATE TABLE IF NOT EXISTS org_account_membership ("
            + " id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
            + " consumer_name VARCHAR(128) NOT NULL UNIQUE,"
            + " department_id VARCHAR(64) NULL,"
            + " parent_consumer_name VARCHAR(128) NULL,"
            + " created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + " updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
            + " INDEX idx_org_account_department (department_id),"
            + " INDEX idx_org_account_parent (parent_consumer_name)"
            + ")";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to ensure org_account_membership table.", ex);
        }
    }

    private void ensureMembershipRow(Connection connection, String consumerName) throws SQLException {
        if (connection == null || StringUtils.isBlank(consumerName)) {
            return;
        }
        String sql = "INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name) "
            + "VALUES (?, NULL, NULL) "
            + "ON DUPLICATE KEY UPDATE consumer_name = VALUES(consumer_name), "
            + "updated_at = CURRENT_TIMESTAMP";
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            statement.executeUpdate();
        }
    }
}
