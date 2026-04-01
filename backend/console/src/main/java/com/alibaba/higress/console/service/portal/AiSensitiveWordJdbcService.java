package com.alibaba.higress.console.service.portal;

import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.math.BigDecimal;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Statement;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.stream.Collectors;
import java.nio.charset.StandardCharsets;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.aisensitive.AiSensitiveBlockAudit;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveBlockAuditEvent;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveDetectRule;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveReplaceRule;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveSystemConfig;
import com.alibaba.higress.console.util.AiSensitiveDateTimeUtil;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class AiSensitiveWordJdbcService {

    private static final String MATCH_TYPE_CONTAINS = "contains";
    private static final String REPLACE_TYPE_REPLACE = "replace";
    private static final long SYSTEM_CONFIG_ID = 1L;
    private static final String SYSTEM_UPDATED_BY = "system";

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    private volatile String bundledSystemDictionaryText;

    @PostConstruct
    public void init() {
        ensureTables();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public void ensureTables() {
        if (!enabled()) {
            return;
        }
        try (Connection connection = openConnection(); Statement statement = connection.createStatement()) {
            statement.execute(
                "CREATE TABLE IF NOT EXISTS ai_sensitive_detect_rule ("
                    + "id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
                    + "pattern VARCHAR(1024) NOT NULL,"
                    + "match_type VARCHAR(32) NOT NULL DEFAULT 'contains',"
                    + "description TEXT NULL,"
                    + "priority INT NOT NULL DEFAULT 0,"
                    + "is_enabled BOOLEAN NOT NULL DEFAULT TRUE,"
                    + "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                    + "updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
                    + ")");
            statement.execute(
                "CREATE TABLE IF NOT EXISTS ai_sensitive_replace_rule ("
                    + "id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
                    + "pattern VARCHAR(1024) NOT NULL,"
                    + "replace_type VARCHAR(32) NOT NULL DEFAULT 'replace',"
                    + "replace_value TEXT NULL,"
                    + "restore BOOLEAN NOT NULL DEFAULT FALSE,"
                    + "description TEXT NULL,"
                    + "priority INT NOT NULL DEFAULT 0,"
                    + "is_enabled BOOLEAN NOT NULL DEFAULT TRUE,"
                    + "created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                    + "updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
                    + ")");
            statement.execute(
                "CREATE TABLE IF NOT EXISTS ai_sensitive_block_audit ("
                    + "id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
                    + "request_id VARCHAR(128) NULL,"
                    + "route_name VARCHAR(255) NULL,"
                    + "consumer_name VARCHAR(255) NULL,"
                    + "display_name VARCHAR(255) NULL,"
                    + "blocked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                    + "blocked_by VARCHAR(64) NOT NULL DEFAULT 'sensitive_word',"
                    + "request_phase VARCHAR(32) NULL,"
                    + "blocked_reason_json TEXT NULL,"
                    + "match_type VARCHAR(32) NULL,"
                    + "matched_rule VARCHAR(1024) NULL,"
                    + "matched_excerpt TEXT NULL,"
                    + "provider_id BIGINT NOT NULL DEFAULT 0,"
                    + "cost_usd DECIMAL(18,6) NOT NULL DEFAULT 0"
                    + ")");
            statement.execute(
                "CREATE TABLE IF NOT EXISTS ai_sensitive_system_config ("
                    + "id BIGINT NOT NULL PRIMARY KEY,"
                    + "system_deny_enabled BOOLEAN NOT NULL DEFAULT FALSE,"
                    + "dictionary_text LONGTEXT NULL,"
                    + "updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
                    + "updated_by VARCHAR(255) NULL"
                    + ")");
            ensureDefaultSystemConfig(connection);
        } catch (SQLException ex) {
            log.warn("Failed to initialize AI sensitive word tables.", ex);
        }
    }

    public AiSensitiveSystemConfig getSystemConfig() {
        if (!enabled()) {
            return AiSensitiveSystemConfig.builder()
                .systemDenyEnabled(Boolean.FALSE)
                .dictionaryText(getBundledSystemDictionaryText())
                .updatedBy(SYSTEM_UPDATED_BY)
                .build();
        }
        String sql = "SELECT id, system_deny_enabled, dictionary_text, updated_at, updated_by "
            + "FROM ai_sensitive_system_config WHERE id=?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, SYSTEM_CONFIG_ID);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapSystemConfig(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query AI sensitive system config.", ex);
        }
        return AiSensitiveSystemConfig.builder()
            .systemDenyEnabled(Boolean.FALSE)
            .dictionaryText(getBundledSystemDictionaryText())
            .updatedBy(SYSTEM_UPDATED_BY)
            .build();
    }

    public AiSensitiveSystemConfig saveSystemConfig(AiSensitiveSystemConfig config, String updatedBy) {
        if (config == null) {
            return null;
        }
        AiSensitiveSystemConfig normalized = AiSensitiveSystemConfig.builder()
            .systemDenyEnabled(Boolean.TRUE.equals(config.getSystemDenyEnabled()))
            .dictionaryText(normalizeDictionaryText(config.getDictionaryText()))
            .updatedBy(StringUtils.defaultIfBlank(StringUtils.trimToNull(updatedBy), SYSTEM_UPDATED_BY))
            .build();
        if (!enabled()) {
            normalized.setUpdatedAt(AiSensitiveDateTimeUtil.formatLocalDateTime(ConsoleDateTimeUtil.now()));
            return normalized;
        }
        try (Connection connection = openConnection()) {
            if (systemConfigExists(connection)) {
                try (PreparedStatement statement = connection.prepareStatement(
                    "UPDATE ai_sensitive_system_config SET system_deny_enabled=?, dictionary_text=?, updated_by=? "
                        + "WHERE id=?")) {
                    statement.setBoolean(1, Boolean.TRUE.equals(normalized.getSystemDenyEnabled()));
                    statement.setString(2, normalized.getDictionaryText());
                    statement.setString(3, normalized.getUpdatedBy());
                    statement.setLong(4, SYSTEM_CONFIG_ID);
                    statement.executeUpdate();
                }
            } else {
                try (PreparedStatement statement = connection.prepareStatement(
                    "INSERT INTO ai_sensitive_system_config (id, system_deny_enabled, dictionary_text, updated_by) "
                        + "VALUES (?, ?, ?, ?)")) {
                    statement.setLong(1, SYSTEM_CONFIG_ID);
                    statement.setBoolean(2, Boolean.TRUE.equals(normalized.getSystemDenyEnabled()));
                    statement.setString(3, normalized.getDictionaryText());
                    statement.setString(4, normalized.getUpdatedBy());
                    statement.executeUpdate();
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to save AI sensitive system config.", ex);
            return null;
        }
        return getSystemConfig();
    }

    public List<String> parseDictionaryWords(String dictionaryText) {
        if (dictionaryText == null) {
            return Collections.emptyList();
        }
        return new ArrayList<>(java.util.Arrays.stream(dictionaryText.split("\\R"))
            .map(StringUtils::trimToEmpty)
            .filter(StringUtils::isNotBlank)
            .collect(Collectors.toCollection(LinkedHashSet::new)));
    }

    public String normalizeDictionaryText(String dictionaryText) {
        return String.join("\n", parseDictionaryWords(dictionaryText));
    }

    public String getBundledSystemDictionaryText() {
        if (bundledSystemDictionaryText != null) {
            return bundledSystemDictionaryText;
        }
        try (InputStream stream = AiSensitiveWordJdbcService.class.getClassLoader()
            .getResourceAsStream("ai-sensitive/system-dictionary.txt")) {
            if (stream == null) {
                return "";
            }
            ByteArrayOutputStream outputStream = new ByteArrayOutputStream();
            byte[] buffer = new byte[4096];
            int read;
            while ((read = stream.read(buffer)) != -1) {
                outputStream.write(buffer, 0, read);
            }
            bundledSystemDictionaryText =
                normalizeDictionaryText(new String(outputStream.toByteArray(), StandardCharsets.UTF_8));
            return bundledSystemDictionaryText;
        } catch (Exception ex) {
            log.warn("Failed to load bundled AI sensitive system dictionary.", ex);
            return "";
        }
    }

    public List<AiSensitiveDetectRule> listDetectRules() {
        if (!enabled()) {
            return Collections.emptyList();
        }
        String sql = "SELECT id, pattern, match_type, description, priority, is_enabled, created_at, updated_at "
            + "FROM ai_sensitive_detect_rule ORDER BY priority DESC, id ASC";
        List<AiSensitiveDetectRule> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                result.add(mapDetectRule(rs));
            }
        } catch (SQLException ex) {
            log.warn("Failed to list AI sensitive detect rules.", ex);
        }
        return result;
    }

    public List<AiSensitiveDetectRule> listEnabledDetectRules() {
        return listDetectRules().stream().filter(rule -> Boolean.TRUE.equals(rule.getEnabled()))
            .collect(Collectors.toList());
    }

    public AiSensitiveDetectRule saveDetectRule(AiSensitiveDetectRule rule) {
        if (!enabled() || rule == null) {
            return null;
        }
        AiSensitiveDetectRule normalized = normalizeDetectRule(rule);
        try (Connection connection = openConnection()) {
            if (normalized.getId() == null) {
                String sql = "INSERT INTO ai_sensitive_detect_rule "
                    + "(pattern, match_type, description, priority, is_enabled) VALUES (?, ?, ?, ?, ?)";
                try (PreparedStatement statement =
                    connection.prepareStatement(sql, Statement.RETURN_GENERATED_KEYS)) {
                    statement.setString(1, normalized.getPattern());
                    statement.setString(2, normalized.getMatchType());
                    statement.setString(3, normalized.getDescription());
                    statement.setInt(4, normalized.getPriority());
                    statement.setBoolean(5, Boolean.TRUE.equals(normalized.getEnabled()));
                    statement.executeUpdate();
                    try (ResultSet keys = statement.getGeneratedKeys()) {
                        if (keys.next()) {
                            normalized.setId(keys.getLong(1));
                        }
                    }
                }
            } else {
                String sql = "UPDATE ai_sensitive_detect_rule SET pattern=?, match_type=?, description=?, priority=?, "
                    + "is_enabled=? WHERE id=?";
                try (PreparedStatement statement = connection.prepareStatement(sql)) {
                    statement.setString(1, normalized.getPattern());
                    statement.setString(2, normalized.getMatchType());
                    statement.setString(3, normalized.getDescription());
                    statement.setInt(4, normalized.getPriority());
                    statement.setBoolean(5, Boolean.TRUE.equals(normalized.getEnabled()));
                    statement.setLong(6, normalized.getId());
                    statement.executeUpdate();
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to save AI sensitive detect rule {}.", normalized.getPattern(), ex);
            return null;
        }
        return getDetectRule(normalized.getId());
    }

    public AiSensitiveDetectRule getDetectRule(Long id) {
        if (!enabled() || id == null) {
            return null;
        }
        String sql = "SELECT id, pattern, match_type, description, priority, is_enabled, created_at, updated_at "
            + "FROM ai_sensitive_detect_rule WHERE id=?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, id);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapDetectRule(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query AI sensitive detect rule {}.", id, ex);
        }
        return null;
    }

    public void deleteDetectRule(Long id) {
        if (!enabled() || id == null) {
            return;
        }
        try (Connection connection = openConnection();
            PreparedStatement statement =
                connection.prepareStatement("DELETE FROM ai_sensitive_detect_rule WHERE id=?")) {
            statement.setLong(1, id);
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to delete AI sensitive detect rule {}.", id, ex);
        }
    }

    public List<AiSensitiveReplaceRule> listReplaceRules() {
        if (!enabled()) {
            return Collections.emptyList();
        }
        String sql = "SELECT id, pattern, replace_type, replace_value, restore, description, priority, is_enabled, "
            + "created_at, updated_at FROM ai_sensitive_replace_rule ORDER BY priority DESC, id ASC";
        List<AiSensitiveReplaceRule> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                result.add(mapReplaceRule(rs));
            }
        } catch (SQLException ex) {
            log.warn("Failed to list AI sensitive replace rules.", ex);
        }
        return result;
    }

    public List<AiSensitiveReplaceRule> listEnabledReplaceRules() {
        return listReplaceRules().stream().filter(rule -> Boolean.TRUE.equals(rule.getEnabled()))
            .collect(Collectors.toList());
    }

    public AiSensitiveReplaceRule saveReplaceRule(AiSensitiveReplaceRule rule) {
        if (!enabled() || rule == null) {
            return null;
        }
        AiSensitiveReplaceRule normalized = normalizeReplaceRule(rule);
        try (Connection connection = openConnection()) {
            if (normalized.getId() == null) {
                String sql = "INSERT INTO ai_sensitive_replace_rule "
                    + "(pattern, replace_type, replace_value, restore, description, priority, is_enabled) "
                    + "VALUES (?, ?, ?, ?, ?, ?, ?)";
                try (PreparedStatement statement =
                    connection.prepareStatement(sql, Statement.RETURN_GENERATED_KEYS)) {
                    statement.setString(1, normalized.getPattern());
                    statement.setString(2, normalized.getReplaceType());
                    statement.setString(3, normalized.getReplaceValue());
                    statement.setBoolean(4, Boolean.TRUE.equals(normalized.getRestore()));
                    statement.setString(5, normalized.getDescription());
                    statement.setInt(6, normalized.getPriority());
                    statement.setBoolean(7, Boolean.TRUE.equals(normalized.getEnabled()));
                    statement.executeUpdate();
                    try (ResultSet keys = statement.getGeneratedKeys()) {
                        if (keys.next()) {
                            normalized.setId(keys.getLong(1));
                        }
                    }
                }
            } else {
                String sql = "UPDATE ai_sensitive_replace_rule SET pattern=?, replace_type=?, replace_value=?, "
                    + "restore=?, description=?, priority=?, is_enabled=? WHERE id=?";
                try (PreparedStatement statement = connection.prepareStatement(sql)) {
                    statement.setString(1, normalized.getPattern());
                    statement.setString(2, normalized.getReplaceType());
                    statement.setString(3, normalized.getReplaceValue());
                    statement.setBoolean(4, Boolean.TRUE.equals(normalized.getRestore()));
                    statement.setString(5, normalized.getDescription());
                    statement.setInt(6, normalized.getPriority());
                    statement.setBoolean(7, Boolean.TRUE.equals(normalized.getEnabled()));
                    statement.setLong(8, normalized.getId());
                    statement.executeUpdate();
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to save AI sensitive replace rule {}.", normalized.getPattern(), ex);
            return null;
        }
        return getReplaceRule(normalized.getId());
    }

    public AiSensitiveReplaceRule getReplaceRule(Long id) {
        if (!enabled() || id == null) {
            return null;
        }
        String sql = "SELECT id, pattern, replace_type, replace_value, restore, description, priority, is_enabled, "
            + "created_at, updated_at FROM ai_sensitive_replace_rule WHERE id=?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, id);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapReplaceRule(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query AI sensitive replace rule {}.", id, ex);
        }
        return null;
    }

    public void deleteReplaceRule(Long id) {
        if (!enabled() || id == null) {
            return;
        }
        try (Connection connection = openConnection();
            PreparedStatement statement =
                connection.prepareStatement("DELETE FROM ai_sensitive_replace_rule WHERE id=?")) {
            statement.setLong(1, id);
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to delete AI sensitive replace rule {}.", id, ex);
        }
    }

    public boolean hasAnyRules() {
        return countDetectRules() > 0 || countReplaceRules() > 0;
    }

    public int countDetectRules() {
        return count("SELECT COUNT(*) FROM ai_sensitive_detect_rule");
    }

    public int countReplaceRules() {
        return count("SELECT COUNT(*) FROM ai_sensitive_replace_rule");
    }

    public int countAuditRecords() {
        return count("SELECT COUNT(*) FROM ai_sensitive_block_audit");
    }

    public AiSensitiveBlockAudit saveAudit(AiSensitiveBlockAuditEvent event, String displayName) {
        if (!enabled() || event == null) {
            return null;
        }
        LocalDateTime blockedAt = event.getBlockedAt() == null ? ConsoleDateTimeUtil.now() : event.getBlockedAt();
        String sql = "INSERT INTO ai_sensitive_block_audit "
            + "(request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, request_phase, "
            + "blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd) "
            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql, Statement.RETURN_GENERATED_KEYS)) {
            statement.setString(1, event.getRequestId());
            statement.setString(2, event.getRouteName());
            statement.setString(3, event.getConsumerName());
            statement.setString(4, displayName);
            statement.setTimestamp(5, ConsoleDateTimeUtil.toTimestamp(blockedAt));
            statement.setString(6, StringUtils.defaultIfBlank(event.getBlockedBy(), "sensitive_word"));
            statement.setString(7, event.getRequestPhase());
            statement.setString(8, event.getBlockedReasonJson());
            statement.setString(9, event.getMatchType());
            statement.setString(10, event.getMatchedRule());
            statement.setString(11, event.getMatchedExcerpt());
            statement.setLong(12, 0L);
            statement.setBigDecimal(13, BigDecimal.ZERO);
            statement.executeUpdate();
            try (ResultSet keys = statement.getGeneratedKeys()) {
                if (keys.next()) {
                    return getAudit(keys.getLong(1));
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to save AI sensitive block audit event.", ex);
        }
        return null;
    }

    public List<AiSensitiveBlockAudit> listAudits(String consumerName, String displayName, String routeName,
        String matchType, String startTime, String endTime, Integer limit) {
        if (!enabled()) {
            return Collections.emptyList();
        }
        StringBuilder sql = new StringBuilder(
            "SELECT id, request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, request_phase, "
                + "blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd "
                + "FROM ai_sensitive_block_audit WHERE 1=1");
        List<Object> params = new ArrayList<>();
        if (StringUtils.isNotBlank(consumerName)) {
            sql.append(" AND consumer_name = ?");
            params.add(consumerName.trim());
        }
        if (StringUtils.isNotBlank(displayName)) {
            sql.append(" AND display_name LIKE ?");
            params.add("%" + displayName.trim() + "%");
        }
        if (StringUtils.isNotBlank(routeName)) {
            sql.append(" AND route_name = ?");
            params.add(routeName.trim());
        }
        if (StringUtils.isNotBlank(matchType)) {
            sql.append(" AND match_type = ?");
            params.add(matchType.trim());
        }
        if (StringUtils.isNotBlank(startTime)) {
            sql.append(" AND blocked_at >= ?");
            params.add(AiSensitiveDateTimeUtil.parseTimestamp(startTime, "startTime"));
        }
        if (StringUtils.isNotBlank(endTime)) {
            sql.append(" AND blocked_at <= ?");
            params.add(AiSensitiveDateTimeUtil.parseTimestamp(endTime, "endTime"));
        }
        sql.append(" ORDER BY blocked_at DESC, id DESC");
        sql.append(" LIMIT ?");
        params.add(limit == null || limit <= 0 ? 200 : Math.min(limit, 1000));
        List<AiSensitiveBlockAudit> result = new ArrayList<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql.toString())) {
            for (int i = 0; i < params.size(); i++) {
                statement.setObject(i + 1, params.get(i));
            }
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    result.add(mapAudit(rs));
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list AI sensitive block audits.", ex);
        }
        return result;
    }

    public AiSensitiveBlockAudit getAudit(Long id) {
        if (!enabled() || id == null) {
            return null;
        }
        String sql = "SELECT id, request_id, route_name, consumer_name, display_name, blocked_at, blocked_by, "
            + "request_phase, blocked_reason_json, match_type, matched_rule, matched_excerpt, provider_id, cost_usd "
            + "FROM ai_sensitive_block_audit WHERE id=?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, id);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return mapAudit(rs);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to query AI sensitive block audit {}.", id, ex);
        }
        return null;
    }

    private int count(String sql) {
        if (!enabled()) {
            return 0;
        }
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            if (rs.next()) {
                return rs.getInt(1);
            }
        } catch (SQLException ex) {
            log.warn("Failed to execute count SQL: {}", sql, ex);
        }
        return 0;
    }

    private void ensureDefaultSystemConfig(Connection connection) throws SQLException {
        if (systemConfigExists(connection)) {
            return;
        }
        try (PreparedStatement statement = connection.prepareStatement(
            "INSERT INTO ai_sensitive_system_config (id, system_deny_enabled, dictionary_text, updated_by) "
                + "VALUES (?, ?, ?, ?)")) {
            statement.setLong(1, SYSTEM_CONFIG_ID);
            statement.setBoolean(2, false);
            statement.setString(3, getBundledSystemDictionaryText());
            statement.setString(4, SYSTEM_UPDATED_BY);
            statement.executeUpdate();
        }
    }

    private boolean systemConfigExists(Connection connection) throws SQLException {
        try (PreparedStatement statement =
            connection.prepareStatement("SELECT COUNT(*) FROM ai_sensitive_system_config WHERE id=?")) {
            statement.setLong(1, SYSTEM_CONFIG_ID);
            try (ResultSet rs = statement.executeQuery()) {
                return rs.next() && rs.getInt(1) > 0;
            }
        }
    }

    private AiSensitiveDetectRule normalizeDetectRule(AiSensitiveDetectRule rule) {
        return AiSensitiveDetectRule.builder()
            .id(rule.getId())
            .pattern(StringUtils.trimToEmpty(rule.getPattern()))
            .matchType(StringUtils.defaultIfBlank(StringUtils.trimToNull(rule.getMatchType()), MATCH_TYPE_CONTAINS))
            .description(StringUtils.trimToNull(rule.getDescription()))
            .priority(rule.getPriority() == null ? 0 : rule.getPriority())
            .enabled(rule.getEnabled() == null ? Boolean.TRUE : rule.getEnabled())
            .build();
    }

    private AiSensitiveReplaceRule normalizeReplaceRule(AiSensitiveReplaceRule rule) {
        return AiSensitiveReplaceRule.builder()
            .id(rule.getId())
            .pattern(StringUtils.trimToEmpty(rule.getPattern()))
            .replaceType(
                StringUtils.defaultIfBlank(StringUtils.trimToNull(rule.getReplaceType()), REPLACE_TYPE_REPLACE))
            .replaceValue(StringUtils.defaultString(rule.getReplaceValue()))
            .restore(rule.getRestore() == null ? Boolean.FALSE : rule.getRestore())
            .description(StringUtils.trimToNull(rule.getDescription()))
            .priority(rule.getPriority() == null ? 0 : rule.getPriority())
            .enabled(rule.getEnabled() == null ? Boolean.TRUE : rule.getEnabled())
            .build();
    }

    private AiSensitiveDetectRule mapDetectRule(ResultSet rs) throws SQLException {
        return AiSensitiveDetectRule.builder()
            .id(rs.getLong("id"))
            .pattern(rs.getString("pattern"))
            .matchType(rs.getString("match_type"))
            .description(rs.getString("description"))
            .priority(rs.getInt("priority"))
            .enabled(rs.getBoolean("is_enabled"))
            .createdAt(AiSensitiveDateTimeUtil.formatTimestamp(rs.getTimestamp("created_at")))
            .updatedAt(AiSensitiveDateTimeUtil.formatTimestamp(rs.getTimestamp("updated_at")))
            .build();
    }

    private AiSensitiveReplaceRule mapReplaceRule(ResultSet rs) throws SQLException {
        return AiSensitiveReplaceRule.builder()
            .id(rs.getLong("id"))
            .pattern(rs.getString("pattern"))
            .replaceType(rs.getString("replace_type"))
            .replaceValue(rs.getString("replace_value"))
            .restore(rs.getBoolean("restore"))
            .description(rs.getString("description"))
            .priority(rs.getInt("priority"))
            .enabled(rs.getBoolean("is_enabled"))
            .createdAt(AiSensitiveDateTimeUtil.formatTimestamp(rs.getTimestamp("created_at")))
            .updatedAt(AiSensitiveDateTimeUtil.formatTimestamp(rs.getTimestamp("updated_at")))
            .build();
    }

    private AiSensitiveBlockAudit mapAudit(ResultSet rs) throws SQLException {
        return AiSensitiveBlockAudit.builder()
            .id(rs.getLong("id"))
            .requestId(rs.getString("request_id"))
            .routeName(rs.getString("route_name"))
            .consumerName(rs.getString("consumer_name"))
            .displayName(rs.getString("display_name"))
            .blockedAt(AiSensitiveDateTimeUtil.formatTimestamp(rs.getTimestamp("blocked_at")))
            .blockedBy(rs.getString("blocked_by"))
            .requestPhase(rs.getString("request_phase"))
            .blockedReasonJson(rs.getString("blocked_reason_json"))
            .matchType(rs.getString("match_type"))
            .matchedRule(rs.getString("matched_rule"))
            .matchedExcerpt(rs.getString("matched_excerpt"))
            .providerId(rs.getLong("provider_id"))
            .costUsd(rs.getBigDecimal("cost_usd"))
            .build();
    }

    private AiSensitiveSystemConfig mapSystemConfig(ResultSet rs) throws SQLException {
        return AiSensitiveSystemConfig.builder()
            .systemDenyEnabled(rs.getBoolean("system_deny_enabled"))
            .dictionaryText(StringUtils.defaultString(rs.getString("dictionary_text")))
            .updatedAt(AiSensitiveDateTimeUtil.formatTimestamp(rs.getTimestamp("updated_at")))
            .updatedBy(rs.getString("updated_by"))
            .build();
    }

    private Connection openConnection() throws SQLException {
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }
}
