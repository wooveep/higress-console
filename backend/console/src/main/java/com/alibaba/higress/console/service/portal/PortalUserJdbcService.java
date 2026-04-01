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
        ensurePortalUserLevelColumn();
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
        String sql = "SELECT consumer_name, display_name, email, department, user_level, status, source, last_login_at "
            + "FROM portal_user WHERE consumer_name IN (" + placeholders + ")";

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
        String sql = "SELECT consumer_name, display_name, email, department, user_level, status, source, last_login_at "
            + "FROM portal_user ORDER BY consumer_name ASC";
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
        if (!enabled()) {
            return Collections.emptyList();
        }
        String sql = "SELECT DISTINCT department FROM portal_user WHERE department IS NOT NULL AND department <> '' ORDER BY department ASC";
        List<String> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                String department = StringUtils.trimToNull(rs.getString("department"));
                if (department != null) {
                    result.add(department);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list portal departments from MySQL.", ex);
        }
        return result;
    }

    public PortalUserRecord queryByConsumerName(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return null;
        }
        String sql = "SELECT consumer_name, display_name, email, department, user_level, status, source, last_login_at "
            + "FROM portal_user WHERE consumer_name = ?";
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
        PortalUserRecord existed = queryByConsumerName(consumerName);

        String displayName = StringUtils.firstNonBlank(consumer.getPortalDisplayName(),
            existed == null ? null : existed.getDisplayName(), consumerName);
        String email = StringUtils.firstNonBlank(consumer.getPortalEmail(), existed == null ? null : existed.getEmail(), "");
        String department = StringUtils.firstNonBlank(consumer.getDepartment(),
            existed == null ? null : existed.getDepartment(), "");
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
                    + "(consumer_name, display_name, email, department, user_level, password_hash, status, source) "
                    + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
                try (PreparedStatement statement = connection.prepareStatement(insertSql)) {
                    statement.setString(1, consumerName);
                    statement.setString(2, displayName);
                    statement.setString(3, email);
                    statement.setString(4, department);
                    statement.setString(5, userLevel);
                    statement.setString(6, passwordEncoder.encode(password));
                    statement.setString(7, status);
                    statement.setString(8, source);
                    statement.executeUpdate();
                }
            } else {
                String updateSql;
                if (password == null) {
                    updateSql = "UPDATE portal_user SET display_name=?, email=?, department=?, user_level=?, status=?, source=? "
                        + "WHERE consumer_name=?";
                } else {
                    updateSql = "UPDATE portal_user SET display_name=?, email=?, department=?, user_level=?, status=?, source=?, "
                        + "password_hash=? WHERE consumer_name=?";
                }
                try (PreparedStatement statement = connection.prepareStatement(updateSql)) {
                    int idx = 1;
                    statement.setString(idx++, displayName);
                    statement.setString(idx++, email);
                    statement.setString(idx++, department);
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
        String sql = "UPDATE portal_user SET status = ? WHERE consumer_name = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, status);
            statement.setString(2, consumerName);
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to update portal user status for {}.", consumerName, ex);
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
            .department(rs.getString("department"))
            .userLevel(normalizeUserLevel(rs.getString("user_level")))
            .status(rs.getString("status"))
            .source(rs.getString("source")).lastLoginAt(lastLoginAt).build();
    }

    private String normalizeUserLevel(String userLevel) {
        String normalized = StringUtils.lowerCase(StringUtils.trimToEmpty(userLevel));
        if (SUPPORTED_USER_LEVELS.contains(normalized)) {
            return normalized;
        }
        return USER_LEVEL_NORMAL;
    }

    private void ensurePortalUserLevelColumn() {
        if (!enabled()) {
            return;
        }
        String existsSql = "SELECT COUNT(1) AS cnt FROM information_schema.COLUMNS "
            + "WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'portal_user' AND COLUMN_NAME = 'user_level'";
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(existsSql);
            ResultSet resultSet = statement.executeQuery()) {
            boolean exists = false;
            if (resultSet.next()) {
                exists = resultSet.getInt("cnt") > 0;
            }
            if (exists) {
                return;
            }
        } catch (SQLException ex) {
            log.warn("Failed to check portal_user.user_level column existence.", ex);
            return;
        }

        String alterSql =
            "ALTER TABLE portal_user ADD COLUMN user_level VARCHAR(16) NOT NULL DEFAULT 'normal' AFTER department";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(alterSql)) {
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to add portal_user.user_level column.", ex);
        }
    }

    public List<String> listActiveRawKeys(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return Collections.emptyList();
        }
        String sql = "SELECT raw_key FROM portal_api_key WHERE consumer_name=? AND status='active' ORDER BY id ASC";
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
            "SELECT consumer_name, raw_key FROM portal_api_key WHERE status='active' AND consumer_name IN (" + placeholders
                + ") ORDER BY id ASC";
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

    public PortalUserRecord ensureMigrationUser(String consumerName, String department) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return null;
        }
        PortalUserRecord existed = queryByConsumerName(consumerName);
        if (existed != null) {
            return existed;
        }
        String sql = "INSERT INTO portal_user (consumer_name, display_name, email, department, user_level, password_hash, status, source) "
            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
        String normalizedDepartment = StringUtils.defaultString(StringUtils.trimToNull(department));
        String randomPassword = UUID.randomUUID().toString().replace("-", "");
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            statement.setString(2, consumerName);
            statement.setString(3, "");
            statement.setString(4, normalizedDepartment);
            statement.setString(5, USER_LEVEL_NORMAL);
            statement.setString(6, passwordEncoder.encode(randomPassword));
            statement.setString(7, STATUS_PENDING);
            statement.setString(8, SOURCE_MIGRATION);
            statement.executeUpdate();
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
        PortalUserRecord existed = queryByConsumerName(BUILTIN_ADMIN_CONSUMER);
        if (existed == null) {
            String insertSql = "INSERT INTO portal_user "
                + "(consumer_name, display_name, email, department, user_level, password_hash, status, source) "
                + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
            String randomPassword = UUID.randomUUID().toString().replace("-", "");
            try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(
                insertSql)) {
                statement.setString(1, BUILTIN_ADMIN_CONSUMER);
                statement.setString(2, BUILTIN_ADMIN_CONSUMER);
                statement.setString(3, "");
                statement.setString(4, "");
                statement.setString(5, USER_LEVEL_NORMAL);
                statement.setString(6, passwordEncoder.encode(randomPassword));
                statement.setString(7, STATUS_PENDING);
                statement.setString(8, SOURCE_SYSTEM);
                statement.executeUpdate();
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
}
