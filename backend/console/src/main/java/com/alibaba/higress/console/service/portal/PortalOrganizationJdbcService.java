package com.alibaba.higress.console.service.portal;

import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.UUID;
import java.util.stream.Collectors;

import javax.annotation.PostConstruct;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.security.crypto.bcrypt.BCryptPasswordEncoder;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.OrgAccountRecord;
import com.alibaba.higress.console.model.portal.OrgDepartmentNode;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.exception.ValidationException;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Data;
import lombok.NoArgsConstructor;
import lombok.extern.slf4j.Slf4j;

/**
 * @author Codex
 */
@Slf4j
@Service
public class PortalOrganizationJdbcService {

    public static final String ROOT_DEPARTMENT_ID = "root";

    private static final String STATUS_ACTIVE = "active";
    private static final String STATUS_SYSTEM = "system";
    private static final String SOURCE_CONSOLE = "console";
    private static final String USER_LEVEL_NORMAL = "normal";

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    private final BCryptPasswordEncoder passwordEncoder = new BCryptPasswordEncoder();

    @PostConstruct
    public void init() {
        ensureOrganizationSchema();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public List<OrgDepartmentNode> listDepartmentTree() {
        if (!enabled()) {
            return Collections.emptyList();
        }
        Map<String, DepartmentRow> departmentMap = loadDepartmentRows();
        if (departmentMap.isEmpty()) {
            return Collections.emptyList();
        }
        Map<String, Integer> memberCountMap = loadDepartmentMemberCounts();
        Map<String, String> displayNameMap = loadPortalUserDisplayNames();
        Map<String, OrgDepartmentNode> nodeMap = new LinkedHashMap<>();
        for (DepartmentRow row : departmentMap.values()) {
            if (StringUtils.equals(row.getDepartmentId(), ROOT_DEPARTMENT_ID)) {
                continue;
            }
            nodeMap.put(row.getDepartmentId(), OrgDepartmentNode.builder()
                .departmentId(row.getDepartmentId())
                .name(row.getName())
                .parentDepartmentId(row.getParentDepartmentId())
                .adminConsumerName(row.getAdminConsumerName())
                .adminDisplayName(displayNameMap.get(row.getAdminConsumerName()))
                .level(row.getLevel())
                .memberCount(memberCountMap.getOrDefault(row.getDepartmentId(), 0))
                .children(new ArrayList<>())
                .build());
        }

        List<OrgDepartmentNode> rootChildren = new ArrayList<>();
        for (OrgDepartmentNode node : nodeMap.values()) {
            String parentDepartmentId = StringUtils.defaultIfBlank(node.getParentDepartmentId(), ROOT_DEPARTMENT_ID);
            if (StringUtils.equals(parentDepartmentId, ROOT_DEPARTMENT_ID)) {
                rootChildren.add(node);
                continue;
            }
            OrgDepartmentNode parent = nodeMap.get(parentDepartmentId);
            if (parent == null) {
                rootChildren.add(node);
                continue;
            }
            parent.getChildren().add(node);
        }
        sortDepartmentNodes(rootChildren);
        return rootChildren;
    }

    public List<OrgAccountRecord> listAccounts() {
        if (!enabled()) {
            return Collections.emptyList();
        }
        Map<String, DepartmentRow> departmentMap = loadDepartmentRows();
        Set<String> adminConsumers = departmentMap.values().stream()
            .map(DepartmentRow::getAdminConsumerName)
            .filter(StringUtils::isNotBlank)
            .collect(Collectors.toSet());
        String sql = "SELECT u.consumer_name, u.display_name, u.email, u.status, u.user_level, u.source, u.last_login_at, "
            + "m.department_id, m.parent_consumer_name "
            + "FROM portal_user u "
            + "LEFT JOIN org_account_membership m ON u.consumer_name = m.consumer_name "
            + "WHERE COALESCE(u.is_deleted, 0) = 0 "
            + "ORDER BY u.consumer_name ASC";
        List<OrgAccountRecord> result = new ArrayList<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                String consumerName = rs.getString("consumer_name");
                if (StringUtils.isBlank(consumerName)
                    || StringUtils.equalsIgnoreCase(consumerName, PortalUserJdbcService.BUILTIN_ADMIN_CONSUMER)) {
                    continue;
                }
                Timestamp lastLogin = rs.getTimestamp("last_login_at");
                LocalDateTime lastLoginAt = lastLogin == null ? null : ConsoleDateTimeUtil.toLocalDateTime(lastLogin);
                String departmentId = StringUtils.trimToNull(rs.getString("department_id"));
                result.add(OrgAccountRecord.builder()
                    .consumerName(consumerName)
                    .displayName(rs.getString("display_name"))
                    .email(rs.getString("email"))
                    .status(rs.getString("status"))
                    .userLevel(rs.getString("user_level"))
                    .source(rs.getString("source"))
                    .departmentId(departmentId)
                    .departmentName(resolveDepartmentName(departmentMap, departmentId))
                    .departmentPath(resolveDepartmentPath(departmentMap, departmentId))
                    .parentConsumerName(StringUtils.trimToNull(rs.getString("parent_consumer_name")))
                    .isDepartmentAdmin(adminConsumers.contains(consumerName))
                    .lastLoginAt(lastLoginAt)
                    .build());
            }
        } catch (SQLException ex) {
            log.warn("Failed to list organization accounts from MySQL.", ex);
        }
        return result;
    }

    public OrgDepartmentNode createDepartment(String name, String parentDepartmentId) {
        return createDepartment(name, parentDepartmentId, null, null);
    }

    public OrgDepartmentNode createDepartment(String name, String parentDepartmentId, DepartmentAdminMutation admin,
        String defaultPassword) {
        ensureEnabled();
        String normalizedName = StringUtils.trimToNull(name);
        if (normalizedName == null) {
            throw new ValidationException("Department name cannot be blank.");
        }
        DepartmentAdminMutation normalizedAdmin = requireDepartmentAdmin(admin, defaultPassword);

        DepartmentRow parent = requireDepartment(resolveParentDepartmentId(parentDepartmentId));
        String departmentId = UUID.randomUUID().toString().replace("-", "");
        int sortOrder = nextSiblingSortOrder(parent.getDepartmentId());
        String path = buildDepartmentPath(parent.getPath(), departmentId);
        int level = parent.getLevel() + 1;

        String sql = "INSERT INTO org_department "
            + "(department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status) "
            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)";
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                try (PreparedStatement statement = connection.prepareStatement(sql)) {
                    statement.setString(1, departmentId);
                    statement.setString(2, normalizedName);
                    statement.setString(3, parent.getDepartmentId());
                    statement.setString(4, normalizedAdmin.getConsumerName());
                    statement.setString(5, path);
                    statement.setInt(6, level);
                    statement.setInt(7, sortOrder);
                    statement.setString(8, STATUS_ACTIVE);
                    statement.executeUpdate();
                }
                upsertPortalUser(connection, normalizedAdmin, defaultPassword);
                upsertMembership(connection, normalizedAdmin.getConsumerName(), departmentId, null);
                connection.commit();
            } catch (Exception ex) {
                connection.rollback();
                throw ex;
            }
        } catch (ValidationException ex) {
            throw ex;
        } catch (SQLException ex) {
            log.warn("Failed to create org department. name={}, parent={}", normalizedName, parent.getDepartmentId(), ex);
            throw new IllegalStateException("Failed to create department.");
        }
        return queryDepartmentNode(departmentId);
    }

    public OrgDepartmentNode updateDepartment(String departmentId, String name) {
        return updateDepartment(departmentId, name, null);
    }

    public OrgDepartmentNode updateDepartment(String departmentId, String name, String adminConsumerName) {
        ensureEnabled();
        String normalizedDepartmentId = requireNonBlank(departmentId, "departmentId cannot be blank.");
        if (StringUtils.equals(normalizedDepartmentId, ROOT_DEPARTMENT_ID)) {
            throw new ValidationException("Root department cannot be renamed.");
        }
        String normalizedName = StringUtils.trimToNull(name);
        if (normalizedName == null && StringUtils.isBlank(adminConsumerName)) {
            throw new ValidationException("Department update cannot be empty.");
        }
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                if (normalizedName != null) {
                    String sql = "UPDATE org_department SET name=? WHERE department_id=?";
                    try (PreparedStatement statement = connection.prepareStatement(sql)) {
                        statement.setString(1, normalizedName);
                        statement.setString(2, normalizedDepartmentId);
                        int affectedRows = statement.executeUpdate();
                        if (affectedRows <= 0) {
                            throw new ValidationException("Department not found: " + normalizedDepartmentId);
                        }
                    }
                } else {
                    requireDepartment(normalizedDepartmentId);
                }
                if (StringUtils.isNotBlank(adminConsumerName)) {
                    updateDepartmentAdministrator(connection, normalizedDepartmentId, adminConsumerName);
                }
                connection.commit();
            } catch (Exception ex) {
                connection.rollback();
                throw ex;
            }
        } catch (ValidationException ex) {
            throw ex;
        } catch (SQLException ex) {
            log.warn("Failed to rename org department {}.", normalizedDepartmentId, ex);
            throw new IllegalStateException("Failed to update department.");
        }
        return queryDepartmentNode(normalizedDepartmentId);
    }

    public OrgDepartmentNode moveDepartment(String departmentId, String parentDepartmentId) {
        ensureEnabled();
        String normalizedDepartmentId = requireNonBlank(departmentId, "departmentId cannot be blank.");
        if (StringUtils.equals(normalizedDepartmentId, ROOT_DEPARTMENT_ID)) {
            throw new ValidationException("Root department cannot be moved.");
        }
        DepartmentRow current = requireDepartment(normalizedDepartmentId);
        DepartmentRow targetParent = requireDepartment(resolveParentDepartmentId(parentDepartmentId));
        if (StringUtils.equals(current.getDepartmentId(), targetParent.getDepartmentId())) {
            throw new ValidationException("Department cannot be moved under itself.");
        }
        if (isDescendantDepartment(current.getDepartmentId(), targetParent.getDepartmentId())) {
            throw new ValidationException("Department cannot be moved under its descendant.");
        }

        int nextSortOrder = nextSiblingSortOrder(targetParent.getDepartmentId());
        String newPathPrefix = buildDepartmentPath(targetParent.getPath(), current.getDepartmentId());
        int newLevelBase = targetParent.getLevel() + 1;

        Map<String, DepartmentRow> subtree = loadDepartmentSubtree(current.getDepartmentId());
        List<DepartmentRow> ordered = subtree.values().stream()
            .sorted(Comparator.comparingInt(DepartmentRow::getLevel))
            .collect(Collectors.toList());

        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                try (PreparedStatement updateCurrent = connection.prepareStatement(
                    "UPDATE org_department SET parent_department_id=?, path=?, level=?, sort_order=? WHERE department_id=?")) {
                    updateCurrent.setString(1, targetParent.getDepartmentId());
                    updateCurrent.setString(2, newPathPrefix);
                    updateCurrent.setInt(3, newLevelBase);
                    updateCurrent.setInt(4, nextSortOrder);
                    updateCurrent.setString(5, current.getDepartmentId());
                    updateCurrent.executeUpdate();
                }
                try (PreparedStatement updateChild = connection.prepareStatement(
                    "UPDATE org_department SET path=?, level=? WHERE department_id=?")) {
                    for (DepartmentRow row : ordered) {
                        if (StringUtils.equals(row.getDepartmentId(), current.getDepartmentId())) {
                            continue;
                        }
                        String suffix = StringUtils.removeStart(row.getPath(), current.getPath());
                        String normalizedSuffix = StringUtils.removeStart(StringUtils.defaultString(suffix), "/");
                        String newPath = StringUtils.isBlank(normalizedSuffix) ? newPathPrefix : newPathPrefix + "/"
                            + normalizedSuffix;
                        int newLevel = newLevelBase + (row.getLevel() - current.getLevel());
                        updateChild.setString(1, newPath);
                        updateChild.setInt(2, newLevel);
                        updateChild.setString(3, row.getDepartmentId());
                        updateChild.addBatch();
                    }
                    updateChild.executeBatch();
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
            log.warn("Failed to move department {} under {}.", normalizedDepartmentId, targetParent.getDepartmentId(), ex);
            throw new IllegalStateException("Failed to move department.");
        }
        return queryDepartmentNode(normalizedDepartmentId);
    }

    public void deleteDepartment(String departmentId) {
        ensureEnabled();
        String normalizedDepartmentId = requireNonBlank(departmentId, "departmentId cannot be blank.");
        if (StringUtils.equals(normalizedDepartmentId, ROOT_DEPARTMENT_ID)) {
            throw new ValidationException("Root department cannot be deleted.");
        }
        if (hasChildDepartments(normalizedDepartmentId)) {
            throw new ValidationException("Department has child departments.");
        }
        if (hasDepartmentMembers(normalizedDepartmentId)) {
            throw new ValidationException("Department still has assigned accounts.");
        }
        if (hasDepartmentGrants(normalizedDepartmentId)) {
            throw new ValidationException("Department still has asset grants.");
        }
        String sql = "DELETE FROM org_department WHERE department_id=?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, normalizedDepartmentId);
            int affectedRows = statement.executeUpdate();
            if (affectedRows <= 0) {
                throw new ValidationException("Department not found: " + normalizedDepartmentId);
            }
        } catch (ValidationException ex) {
            throw ex;
        } catch (SQLException ex) {
            log.warn("Failed to delete org department {}.", normalizedDepartmentId, ex);
            throw new IllegalStateException("Failed to delete department.");
        }
    }

    public OrgAccountRecord assignAccount(String consumerName, String departmentId, String parentConsumerName) {
        ensureEnabled();
        String normalizedConsumerName = requireNonBlank(consumerName, "consumerName cannot be blank.");
        if (StringUtils.equalsIgnoreCase(normalizedConsumerName, PortalUserJdbcService.BUILTIN_ADMIN_CONSUMER)) {
            throw new ValidationException("Built-in administrator cannot be assigned.");
        }
        ensurePortalUserExists(normalizedConsumerName);

        DepartmentRow managedDepartment = findDepartmentManagedByConsumer(normalizedConsumerName);
        String normalizedDepartmentId = normalizeDepartmentAssignment(departmentId);
        if (managedDepartment != null) {
            if (normalizedDepartmentId != null
                && !StringUtils.equals(normalizedDepartmentId, managedDepartment.getDepartmentId())) {
                throw new ValidationException("Department administrator must stay in its managed department.");
            }
            normalizedDepartmentId = managedDepartment.getDepartmentId();
        } else if (normalizedDepartmentId != null) {
            requireDepartment(normalizedDepartmentId);
        }

        String normalizedParentConsumer = StringUtils.trimToNull(parentConsumerName);
        if (managedDepartment != null) {
            normalizedParentConsumer = null;
        } else if (normalizedParentConsumer == null && normalizedDepartmentId != null) {
            DepartmentRow departmentRow = requireDepartment(normalizedDepartmentId);
            normalizedParentConsumer = StringUtils.trimToNull(departmentRow.getAdminConsumerName());
            if (StringUtils.equalsIgnoreCase(normalizedParentConsumer, normalizedConsumerName)) {
                normalizedParentConsumer = null;
            }
        }
        if (normalizedParentConsumer != null) {
            if (StringUtils.equalsIgnoreCase(normalizedParentConsumer, PortalUserJdbcService.BUILTIN_ADMIN_CONSUMER)) {
                throw new ValidationException("Built-in administrator cannot be used as parent account.");
            }
            if (StringUtils.equalsIgnoreCase(normalizedParentConsumer, normalizedConsumerName)) {
                throw new ValidationException("Parent account cannot be itself.");
            }
            ensurePortalUserExists(normalizedParentConsumer);
            validateParentLoop(normalizedConsumerName, normalizedParentConsumer);
        }

        try (Connection connection = openConnection()) {
            upsertMembership(connection, normalizedConsumerName, normalizedDepartmentId, normalizedParentConsumer);
        } catch (SQLException ex) {
            log.warn("Failed to assign org account {}. department={}, parent={}", normalizedConsumerName,
                normalizedDepartmentId, normalizedParentConsumer, ex);
            throw new IllegalStateException("Failed to update account assignment.");
        }
        return queryAccount(normalizedConsumerName);
    }

    public OrgAccountRecord queryAccount(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return null;
        }
        return listAccounts().stream()
            .filter(item -> StringUtils.equals(item.getConsumerName(), consumerName))
            .findFirst()
            .orElse(null);
    }

    public Set<String> listDepartmentIdsInSubtree(String departmentId) {
        if (!enabled()) {
            return Collections.emptySet();
        }
        String normalizedDepartmentId = StringUtils.trimToNull(departmentId);
        if (normalizedDepartmentId == null) {
            return Collections.emptySet();
        }
        DepartmentRow root = requireDepartment(normalizedDepartmentId);
        Map<String, DepartmentRow> rows = loadDepartmentRows();
        return rows.values().stream()
            .filter(row -> StringUtils.equals(row.getDepartmentId(), normalizedDepartmentId)
                || StringUtils.startsWith(row.getPath(), root.getPath() + "/"))
            .map(DepartmentRow::getDepartmentId)
            .filter(StringUtils::isNotBlank)
            .collect(Collectors.toCollection(LinkedHashSet::new));
    }

    public boolean isDepartmentAdministrator(String consumerName) {
        if (!enabled() || StringUtils.isBlank(consumerName)) {
            return false;
        }
        return exists("SELECT 1 FROM org_department WHERE admin_consumer_name = ? LIMIT 1", consumerName);
    }

    private void ensureOrganizationSchema() {
        if (!enabled()) {
            return;
        }
        List<String> statements = new ArrayList<>();
        statements.add("CREATE TABLE IF NOT EXISTS org_department ("
            + " id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
            + " department_id VARCHAR(64) NOT NULL UNIQUE,"
            + " name VARCHAR(128) NOT NULL,"
            + " parent_department_id VARCHAR(64) NULL,"
            + " admin_consumer_name VARCHAR(128) NULL,"
            + " path VARCHAR(512) NOT NULL,"
            + " level INT NOT NULL DEFAULT 0,"
            + " sort_order INT NOT NULL DEFAULT 0,"
            + " status VARCHAR(32) NOT NULL DEFAULT 'active',"
            + " created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + " updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
            + " INDEX idx_org_department_parent (parent_department_id),"
            + " INDEX idx_org_department_status (status),"
            + " UNIQUE KEY uk_org_department_admin_consumer (admin_consumer_name)"
            + ")");
        statements.add("CREATE TABLE IF NOT EXISTS org_account_membership ("
            + " id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,"
            + " consumer_name VARCHAR(128) NOT NULL UNIQUE,"
            + " department_id VARCHAR(64) NULL,"
            + " parent_consumer_name VARCHAR(128) NULL,"
            + " created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + " updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
            + " INDEX idx_org_account_department (department_id),"
            + " INDEX idx_org_account_parent (parent_consumer_name)"
            + ")");
        try (Connection connection = openConnection()) {
            for (String sql : statements) {
                try (PreparedStatement statement = connection.prepareStatement(sql)) {
                    statement.executeUpdate();
                }
            }
            ensureDepartmentAdminColumn(connection);
            ensureRootDepartment(connection);
            ensureMembershipRows(connection);
        } catch (SQLException ex) {
            log.warn("Failed to ensure organization schema in portal database.", ex);
        }
    }

    private void ensureRootDepartment(Connection connection) throws SQLException {
        String sql = "INSERT INTO org_department "
            + "(department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status) "
            + "VALUES (?, ?, NULL, NULL, ?, 0, 0, 'system') "
            + "ON DUPLICATE KEY UPDATE name = VALUES(name), path = VALUES(path), "
            + "level = VALUES(level), sort_order = VALUES(sort_order), status = VALUES(status)";
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, ROOT_DEPARTMENT_ID);
            statement.setString(2, "ROOT");
            statement.setString(3, ROOT_DEPARTMENT_ID);
            statement.executeUpdate();
        }
    }

    private void ensureDepartmentAdminColumn(Connection connection) {
        List<String> statements = new ArrayList<>();
        statements.add("ALTER TABLE org_department ADD COLUMN admin_consumer_name VARCHAR(128) NULL AFTER parent_department_id");
        statements.add("ALTER TABLE org_department ADD UNIQUE KEY uk_org_department_admin_consumer (admin_consumer_name)");
        for (String sql : statements) {
            try (PreparedStatement statement = connection.prepareStatement(sql)) {
                statement.executeUpdate();
            } catch (SQLException ex) {
                log.debug("Skip organization schema adjustment. sql={}", sql, ex);
            }
        }
    }

    private void ensureMembershipRows(Connection connection) throws SQLException {
        String sql = "INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name) "
            + "SELECT consumer_name, NULL, NULL FROM portal_user "
            + "ON DUPLICATE KEY UPDATE consumer_name = VALUES(consumer_name), updated_at = CURRENT_TIMESTAMP";
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.executeUpdate();
        }
    }

    private Map<String, DepartmentRow> loadDepartmentRows() {
        if (!enabled()) {
            return Collections.emptyMap();
        }
        String sql = "SELECT department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status "
            + "FROM org_department ORDER BY level ASC, sort_order ASC, name ASC";
        Map<String, DepartmentRow> result = new LinkedHashMap<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                DepartmentRow row = DepartmentRow.builder()
                    .departmentId(rs.getString("department_id"))
                    .name(rs.getString("name"))
                    .parentDepartmentId(rs.getString("parent_department_id"))
                    .adminConsumerName(StringUtils.trimToNull(rs.getString("admin_consumer_name")))
                    .path(rs.getString("path"))
                    .level(rs.getInt("level"))
                    .sortOrder(rs.getInt("sort_order"))
                    .status(rs.getString("status"))
                    .build();
                result.put(row.getDepartmentId(), row);
            }
        } catch (SQLException ex) {
            log.warn("Failed to load org departments from MySQL.", ex);
        }
        return result;
    }

    private Map<String, Integer> loadDepartmentMemberCounts() {
        if (!enabled()) {
            return Collections.emptyMap();
        }
        String sql = "SELECT m.department_id, COUNT(1) AS cnt FROM org_account_membership m "
            + "JOIN portal_user u ON u.consumer_name = m.consumer_name "
            + "WHERE m.department_id IS NOT NULL AND COALESCE(u.is_deleted, 0) = 0 "
            + "GROUP BY m.department_id";
        Map<String, Integer> result = new HashMap<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                result.put(rs.getString("department_id"), rs.getInt("cnt"));
            }
        } catch (SQLException ex) {
            log.warn("Failed to load department member counts.", ex);
        }
        return result;
    }

    private Map<String, DepartmentRow> loadDepartmentSubtree(String departmentId) {
        DepartmentRow root = requireDepartment(departmentId);
        String sql = "SELECT department_id, name, parent_department_id, admin_consumer_name, path, level, sort_order, status "
            + "FROM org_department WHERE path = ? OR path LIKE ? ORDER BY level ASC, sort_order ASC, name ASC";
        Map<String, DepartmentRow> result = new LinkedHashMap<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, root.getPath());
            statement.setString(2, root.getPath() + "/%");
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    DepartmentRow row = DepartmentRow.builder()
                        .departmentId(rs.getString("department_id"))
                        .name(rs.getString("name"))
                        .parentDepartmentId(rs.getString("parent_department_id"))
                        .adminConsumerName(StringUtils.trimToNull(rs.getString("admin_consumer_name")))
                        .path(rs.getString("path"))
                        .level(rs.getInt("level"))
                        .sortOrder(rs.getInt("sort_order"))
                        .status(rs.getString("status"))
                        .build();
                    result.put(row.getDepartmentId(), row);
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to load department subtree for {}.", departmentId, ex);
        }
        return result;
    }

    private DepartmentRow requireDepartment(String departmentId) {
        String normalizedDepartmentId = requireNonBlank(departmentId, "departmentId cannot be blank.");
        DepartmentRow row = loadDepartmentRows().get(normalizedDepartmentId);
        if (row == null) {
            throw new ValidationException("Department not found: " + normalizedDepartmentId);
        }
        return row;
    }

    private OrgDepartmentNode queryDepartmentNode(String departmentId) {
        DepartmentRow row = requireDepartment(departmentId);
        if (StringUtils.equals(row.getDepartmentId(), ROOT_DEPARTMENT_ID)) {
            return null;
        }
        int memberCount = loadDepartmentMemberCounts().getOrDefault(row.getDepartmentId(), 0);
        Map<String, String> displayNameMap = loadPortalUserDisplayNames();
        return OrgDepartmentNode.builder()
            .departmentId(row.getDepartmentId())
            .name(row.getName())
            .parentDepartmentId(row.getParentDepartmentId())
            .adminConsumerName(row.getAdminConsumerName())
            .adminDisplayName(displayNameMap.get(row.getAdminConsumerName()))
            .level(row.getLevel())
            .memberCount(memberCount)
            .children(new ArrayList<>())
            .build();
    }

    private int nextSiblingSortOrder(String parentDepartmentId) {
        String sql = "SELECT COALESCE(MAX(sort_order), 0) + 1 AS next_sort_order "
            + "FROM org_department WHERE parent_department_id = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, parentDepartmentId);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return rs.getInt("next_sort_order");
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to calculate next department sort order. parent={}", parentDepartmentId, ex);
        }
        return 1;
    }

    private boolean hasChildDepartments(String departmentId) {
        return exists("SELECT 1 FROM org_department WHERE parent_department_id = ? LIMIT 1", departmentId);
    }

    private boolean hasDepartmentMembers(String departmentId) {
        return exists("SELECT 1 FROM org_account_membership m JOIN portal_user u ON u.consumer_name = m.consumer_name "
            + "WHERE m.department_id = ? AND COALESCE(u.is_deleted, 0) = 0 LIMIT 1", departmentId);
    }

    private boolean hasDepartmentGrants(String departmentId) {
        return exists("SELECT 1 FROM asset_grant WHERE subject_type = 'department' AND subject_id = ? LIMIT 1",
            departmentId);
    }

    private Map<String, String> loadPortalUserDisplayNames() {
        String sql = "SELECT consumer_name, display_name FROM portal_user WHERE COALESCE(is_deleted, 0) = 0";
        Map<String, String> result = new HashMap<>();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                result.put(StringUtils.trimToEmpty(rs.getString("consumer_name")),
                    StringUtils.trimToNull(rs.getString("display_name")));
            }
        } catch (SQLException ex) {
            log.warn("Failed to load portal user display names.", ex);
        }
        return result;
    }

    private boolean exists(String sql, String parameter) {
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, parameter);
            try (ResultSet rs = statement.executeQuery()) {
                return rs.next();
            }
        } catch (SQLException ex) {
            log.warn("Failed to execute existence query. sql={}", sql, ex);
            return false;
        }
    }

    private void ensurePortalUserExists(String consumerName) {
        if (!exists("SELECT 1 FROM portal_user WHERE consumer_name = ? AND COALESCE(is_deleted, 0) = 0 LIMIT 1",
            consumerName)) {
            throw new ValidationException("Consumer not found: " + consumerName);
        }
    }

    private void validateParentLoop(String consumerName, String parentConsumerName) {
        String current = parentConsumerName;
        Set<String> visited = new LinkedHashSet<>();
        while (current != null) {
            if (!visited.add(current)) {
                throw new ValidationException("Detected parent account loop.");
            }
            if (StringUtils.equalsIgnoreCase(current, consumerName)) {
                throw new ValidationException("Parent account loop is not allowed.");
            }
            current = loadParentConsumerName(current);
        }
    }

    private String loadParentConsumerName(String consumerName) {
        String sql = "SELECT parent_consumer_name FROM org_account_membership WHERE consumer_name = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return StringUtils.trimToNull(rs.getString("parent_consumer_name"));
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to load parent consumer for {}.", consumerName, ex);
        }
        return null;
    }

    private boolean isDescendantDepartment(String departmentId, String targetDepartmentId) {
        Map<String, DepartmentRow> subtree = loadDepartmentSubtree(departmentId);
        return subtree.containsKey(targetDepartmentId);
    }

    private DepartmentRow findDepartmentManagedByConsumer(String consumerName) {
        if (StringUtils.isBlank(consumerName)) {
            return null;
        }
        return loadDepartmentRows().values().stream()
            .filter(row -> StringUtils.equalsIgnoreCase(row.getAdminConsumerName(), consumerName))
            .findFirst()
            .orElse(null);
    }

    private String resolveParentDepartmentId(String parentDepartmentId) {
        String normalized = StringUtils.trimToNull(parentDepartmentId);
        return normalized == null ? ROOT_DEPARTMENT_ID : normalized;
    }

    private String normalizeDepartmentAssignment(String departmentId) {
        String normalized = StringUtils.trimToNull(departmentId);
        if (normalized == null || StringUtils.equals(normalized, ROOT_DEPARTMENT_ID)) {
            return null;
        }
        return normalized;
    }

    private String buildDepartmentPath(String parentPath, String departmentId) {
        String normalizedParentPath = StringUtils.trimToEmpty(parentPath);
        if (StringUtils.isBlank(normalizedParentPath)) {
            return departmentId;
        }
        return normalizedParentPath + "/" + departmentId;
    }

    private String resolveDepartmentName(Map<String, DepartmentRow> departmentMap, String departmentId) {
        DepartmentRow row = departmentMap.get(StringUtils.defaultString(departmentId));
        if (row == null || StringUtils.equals(row.getDepartmentId(), ROOT_DEPARTMENT_ID)) {
            return null;
        }
        return row.getName();
    }

    private String resolveDepartmentPath(Map<String, DepartmentRow> departmentMap, String departmentId) {
        DepartmentRow current = departmentMap.get(StringUtils.defaultString(departmentId));
        if (current == null || StringUtils.equals(current.getDepartmentId(), ROOT_DEPARTMENT_ID)) {
            return null;
        }
        List<String> names = new ArrayList<>();
        Set<String> visited = new LinkedHashSet<>();
        while (current != null && !StringUtils.equals(current.getDepartmentId(), ROOT_DEPARTMENT_ID)) {
            if (!visited.add(current.getDepartmentId())) {
                break;
            }
            names.add(current.getName());
            current = departmentMap.get(StringUtils.defaultString(current.getParentDepartmentId()));
        }
        Collections.reverse(names);
        return String.join(" / ", names);
    }

    private void sortDepartmentNodes(List<OrgDepartmentNode> nodes) {
        if (nodes == null) {
            return;
        }
        nodes.sort(Comparator.comparing(node -> StringUtils.defaultString(node.getName())));
        for (OrgDepartmentNode node : nodes) {
            sortDepartmentNodes(node.getChildren());
        }
    }

    private String requireNonBlank(String value, String message) {
        String normalized = StringUtils.trimToNull(value);
        if (normalized == null) {
            throw new ValidationException(message);
        }
        return normalized;
    }

    private void ensureEnabled() {
        if (!enabled()) {
            throw new IllegalStateException("Portal database is unavailable.");
        }
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    private DepartmentAdminMutation requireDepartmentAdmin(DepartmentAdminMutation admin, String defaultPassword) {
        if (admin == null || StringUtils.isBlank(admin.getConsumerName())) {
            throw new ValidationException("Department administrator is required.");
        }
        DepartmentAdminMutation normalized = DepartmentAdminMutation.builder()
            .consumerName(requireNonBlank(admin.getConsumerName(), "Administrator consumerName cannot be blank."))
            .displayName(StringUtils.trimToNull(admin.getDisplayName()))
            .email(StringUtils.trimToNull(admin.getEmail()))
            .userLevel(StringUtils.defaultIfBlank(StringUtils.trimToNull(admin.getUserLevel()), USER_LEVEL_NORMAL))
            .password(StringUtils.trimToNull(admin.getPassword()))
            .build();
        if (StringUtils.isBlank(normalized.getPassword()) && StringUtils.isBlank(defaultPassword)) {
            throw new ValidationException("Default organization password is not configured.");
        }
        return normalized;
    }

    private void upsertPortalUser(Connection connection, DepartmentAdminMutation admin, String defaultPassword)
        throws SQLException {
        String consumerName = admin.getConsumerName();
        String password = StringUtils.firstNonBlank(StringUtils.trimToNull(admin.getPassword()),
            StringUtils.trimToNull(defaultPassword));
        String displayName = StringUtils.defaultIfBlank(admin.getDisplayName(), consumerName);
        String email = StringUtils.defaultString(admin.getEmail());
        String userLevel = StringUtils.defaultIfBlank(StringUtils.lowerCase(admin.getUserLevel()), USER_LEVEL_NORMAL);

        try (PreparedStatement check = connection
            .prepareStatement("SELECT consumer_name FROM portal_user WHERE consumer_name = ? LIMIT 1")) {
            check.setString(1, consumerName);
            try (ResultSet rs = check.executeQuery()) {
                if (rs.next()) {
                    String updateSql = StringUtils.isBlank(password)
                        ? "UPDATE portal_user SET display_name=?, email=?, user_level=?, source=?, is_deleted=0, deleted_at=NULL WHERE consumer_name=?"
                        : "UPDATE portal_user SET display_name=?, email=?, user_level=?, source=?, password_hash=?, is_deleted=0, deleted_at=NULL WHERE consumer_name=?";
                    try (PreparedStatement update = connection.prepareStatement(updateSql)) {
                        int idx = 1;
                        update.setString(idx++, displayName);
                        update.setString(idx++, email);
                        update.setString(idx++, userLevel);
                        update.setString(idx++, SOURCE_CONSOLE);
                        if (StringUtils.isNotBlank(password)) {
                            update.setString(idx++, passwordEncoder.encode(password));
                        }
                        update.setString(idx, consumerName);
                        update.executeUpdate();
                    }
                    return;
                }
            }
        }

        try (PreparedStatement insert = connection.prepareStatement(
            "INSERT INTO portal_user (consumer_name, display_name, email, user_level, password_hash, status, source) "
                + "VALUES (?, ?, ?, ?, ?, ?, ?)")) {
            insert.setString(1, consumerName);
            insert.setString(2, displayName);
            insert.setString(3, email);
            insert.setString(4, userLevel);
            insert.setString(5, passwordEncoder.encode(password));
            insert.setString(6, STATUS_ACTIVE);
            insert.setString(7, SOURCE_CONSOLE);
            insert.executeUpdate();
        }
    }

    private void upsertMembership(Connection connection, String consumerName, String departmentId, String parentConsumerName)
        throws SQLException {
        String sql = "INSERT INTO org_account_membership (consumer_name, department_id, parent_consumer_name) "
            + "VALUES (?, ?, ?) "
            + "ON DUPLICATE KEY UPDATE department_id = VALUES(department_id), "
            + "parent_consumer_name = VALUES(parent_consumer_name), "
            + "updated_at = CURRENT_TIMESTAMP";
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, consumerName);
            statement.setString(2, departmentId);
            statement.setString(3, parentConsumerName);
            statement.executeUpdate();
        }
    }

    private void updateDepartmentAdministrator(Connection connection, String departmentId, String adminConsumerName)
        throws SQLException {
        String normalizedAdminConsumer = requireNonBlank(adminConsumerName, "adminConsumerName cannot be blank.");
        ensurePortalUserExists(normalizedAdminConsumer);
        DepartmentRow currentAdminDepartment = findDepartmentManagedByConsumer(normalizedAdminConsumer);
        if (currentAdminDepartment != null
            && !StringUtils.equals(currentAdminDepartment.getDepartmentId(), departmentId)) {
            throw new ValidationException("A department administrator can only manage one department.");
        }
        try (PreparedStatement statement = connection.prepareStatement(
            "UPDATE org_department SET admin_consumer_name=? WHERE department_id=?")) {
            statement.setString(1, normalizedAdminConsumer);
            statement.setString(2, departmentId);
            int affected = statement.executeUpdate();
            if (affected <= 0) {
                throw new ValidationException("Department not found: " + departmentId);
            }
        }
        upsertMembership(connection, normalizedAdminConsumer, departmentId, null);
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    private static class DepartmentRow {

        private String departmentId;
        private String name;
        private String parentDepartmentId;
        private String adminConsumerName;
        private String path;
        private Integer level;
        private Integer sortOrder;
        private String status;
    }

    @Data
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class DepartmentAdminMutation {

        private String consumerName;
        private String displayName;
        private String email;
        private String userLevel;
        private String password;
    }
}
