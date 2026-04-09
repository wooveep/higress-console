package com.alibaba.higress.console.service.portal;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.AssetGrantRecord;
import com.alibaba.higress.sdk.model.RouteAuthConfig;
import com.alibaba.higress.sdk.exception.ValidationException;

import lombok.extern.slf4j.Slf4j;

/**
 * @author Codex
 */
@Slf4j
@Service
public class PortalAssetGrantJdbcService {

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    @PostConstruct
    public void init() {
        ensureAssetGrantTable();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public List<AssetGrantRecord> listGrants(String assetType, String assetId) {
        if (!enabled()) {
            return Collections.emptyList();
        }
        String normalizedAssetType = requireNonBlank(assetType, "assetType cannot be blank.");
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");

        String sql = "SELECT asset_type, asset_id, subject_type, subject_id "
            + "FROM asset_grant WHERE asset_type = ? AND asset_id = ? "
            + "ORDER BY subject_type ASC, subject_id ASC";
        List<AssetGrantRecord> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, normalizedAssetType);
            statement.setString(2, normalizedAssetId);
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    result.add(AssetGrantRecord.builder()
                        .assetType(rs.getString("asset_type"))
                        .assetId(rs.getString("asset_id"))
                        .subjectType(rs.getString("subject_type"))
                        .subjectId(rs.getString("subject_id"))
                        .build());
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to list asset grants. assetType={}, assetId={}", normalizedAssetType, normalizedAssetId, ex);
        }
        return result;
    }

    public List<AssetGrantRecord> replaceGrants(String assetType, String assetId, List<AssetGrantRecord> grants) {
        ensureEnabled();
        String normalizedAssetType = requireNonBlank(assetType, "assetType cannot be blank.");
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");

        Map<String, AssetGrantRecord> normalizedMap = new LinkedHashMap<>();
        if (grants != null) {
            for (AssetGrantRecord item : grants) {
                if (item == null) {
                    continue;
                }
                String subjectType = normalizeSubjectType(item.getSubjectType());
                String subjectId = normalizeSubjectId(subjectType, item.getSubjectId());
                String key = subjectType + ":" + subjectId;
                normalizedMap.put(key, AssetGrantRecord.builder()
                    .assetType(normalizedAssetType)
                    .assetId(normalizedAssetId)
                    .subjectType(subjectType)
                    .subjectId(subjectId)
                    .build());
            }
        }

        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                try (PreparedStatement delete = connection.prepareStatement(
                    "DELETE FROM asset_grant WHERE asset_type = ? AND asset_id = ?")) {
                    delete.setString(1, normalizedAssetType);
                    delete.setString(2, normalizedAssetId);
                    delete.executeUpdate();
                }

                if (!normalizedMap.isEmpty()) {
                    try (PreparedStatement insert = connection.prepareStatement(
                        "INSERT INTO asset_grant (asset_type, asset_id, subject_type, subject_id) VALUES (?, ?, ?, ?)")) {
                        for (AssetGrantRecord record : normalizedMap.values()) {
                            insert.setString(1, record.getAssetType());
                            insert.setString(2, record.getAssetId());
                            insert.setString(3, record.getSubjectType());
                            insert.setString(4, record.getSubjectId());
                            insert.addBatch();
                        }
                        insert.executeBatch();
                    }
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
            log.warn("Failed to replace asset grants. assetType={}, assetId={}", normalizedAssetType, normalizedAssetId,
                ex);
            throw new IllegalStateException("Failed to update asset grants.");
        }
        return listGrants(normalizedAssetType, normalizedAssetId);
    }

    private void ensureAssetGrantTable() {
        if (!enabled()) {
            return;
        }
        String sql = "CREATE TABLE IF NOT EXISTS asset_grant ("
            + " id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
            + " asset_type VARCHAR(64) NOT NULL,"
            + " asset_id VARCHAR(128) NOT NULL,"
            + " subject_type VARCHAR(32) NOT NULL,"
            + " subject_id VARCHAR(128) NOT NULL,"
            + " created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + " updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
            + " UNIQUE KEY uk_asset_grant_subject (asset_type, asset_id, subject_type, subject_id),"
            + " INDEX idx_asset_grant_lookup (asset_type, asset_id),"
            + " INDEX idx_asset_grant_subject (subject_type, subject_id)"
            + ")";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to ensure asset_grant table.", ex);
        }
    }

    private String normalizeSubjectType(String subjectType) {
        String normalized = StringUtils.lowerCase(requireNonBlank(subjectType, "subjectType cannot be blank."));
        if (!StringUtils.equals(normalized, "consumer")
            && !StringUtils.equals(normalized, "department")
            && !StringUtils.equals(normalized, "user_level")) {
            throw new ValidationException("subjectType must be consumer, department or user_level.");
        }
        return normalized;
    }

    private void ensureEnabled() {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
    }

    private String requireNonBlank(String value, String message) {
        String normalized = StringUtils.trimToNull(value);
        if (normalized == null) {
            throw new ValidationException(message);
        }
        return normalized;
    }

    private String normalizeSubjectId(String subjectType, String subjectId) {
        String normalizedSubjectId = requireNonBlank(subjectId, "subjectId cannot be blank.");
        if (StringUtils.equals(subjectType, "user_level")) {
            java.util.List<String> levels = RouteAuthConfig.normalizeAllowedConsumerLevels(
                java.util.Collections.singletonList(normalizedSubjectId));
            if (levels.isEmpty()) {
                throw new ValidationException("subjectId must be a supported user level.");
            }
            return levels.get(0);
        }
        return normalizedSubjectId;
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }
}
