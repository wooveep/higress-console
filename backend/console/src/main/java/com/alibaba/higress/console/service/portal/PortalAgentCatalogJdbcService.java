package com.alibaba.higress.console.service.portal;

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
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Objects;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import javax.annotation.PostConstruct;
import javax.annotation.Resource;

import org.apache.commons.collections4.CollectionUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.AgentCatalogOptionServerRecord;
import com.alibaba.higress.console.model.portal.AgentCatalogOptionsRecord;
import com.alibaba.higress.console.model.portal.AgentCatalogRecord;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.exception.BusinessException;
import com.alibaba.higress.sdk.exception.ValidationException;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;
import com.alibaba.higress.sdk.model.mcp.ConsumerAuthInfo;
import com.alibaba.higress.sdk.model.mcp.McpServer;
import com.alibaba.higress.sdk.service.mcp.McpServerService;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalAgentCatalogJdbcService {

    private static final String STATUS_DRAFT = "draft";
    private static final String STATUS_PUBLISHED = "published";
    private static final String STATUS_UNPUBLISHED = "unpublished";
    private static final String ASSET_TYPE_AGENT_CATALOG = "agent_catalog";
    private static final List<String> DEFAULT_TRANSPORT_TYPES = Collections.unmodifiableList(Arrays.asList("http", "sse"));
    private static final TypeReference<List<String>> STRING_LIST_TYPE = new TypeReference<List<String>>() {
    };
    private static final Pattern TOOL_NAME_PATTERN = Pattern.compile("(?m)^\\s*-\\s*name\\s*:");

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    private ObjectMapper objectMapper;
    private McpServerService mcpServerService;
    private AuthorizationSubjectResolver authorizationSubjectResolver;

    @Resource
    public void setObjectMapper(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    @Resource
    public void setMcpServerService(McpServerService mcpServerService) {
        this.mcpServerService = mcpServerService;
    }

    @Resource
    public void setAuthorizationSubjectResolver(AuthorizationSubjectResolver authorizationSubjectResolver) {
        this.authorizationSubjectResolver = authorizationSubjectResolver;
    }

    @PostConstruct
    public void init() {
        ensureAgentCatalogTable();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public PaginatedResult<AgentCatalogRecord> listAssets(CommonPageQuery query) {
        ensureEnabled();
        return PaginatedResult.createFromFullList(listAssetsInternal(), query);
    }

    public AgentCatalogRecord queryAsset(String agentId) {
        ensureEnabled();
        return requireAsset(requireNonBlank(agentId, "agentId cannot be blank."));
    }

    public AgentCatalogOptionsRecord queryOptions() {
        ensureEnabled();
        List<AgentCatalogOptionServerRecord> servers = new ArrayList<>();
        try {
            PaginatedResult<McpServer> result = mcpServerService.list(null);
            if (result != null && CollectionUtils.isNotEmpty(result.getData())) {
                for (McpServer item : result.getData()) {
                    if (item == null || StringUtils.isBlank(item.getName())) {
                        continue;
                    }
                    ConsumerAuthInfo authInfo = item.getConsumerAuthInfo();
                    servers.add(AgentCatalogOptionServerRecord.builder()
                        .mcpServerName(item.getName())
                        .description(item.getDescription())
                        .type(item.getType() == null ? null : item.getType().name())
                        .domains(item.getDomains())
                        .authEnabled(authInfo == null ? null : authInfo.getEnable())
                        .authType(authInfo == null ? null : authInfo.getType())
                        .build());
                }
            }
        } catch (Exception ex) {
            throw new BusinessException("Failed to query MCP server options.", ex);
        }
        servers.sort((left, right) -> StringUtils.compareIgnoreCase(left.getMcpServerName(), right.getMcpServerName()));
        return AgentCatalogOptionsRecord.builder().servers(servers).build();
    }

    public AgentCatalogRecord createAsset(AgentCatalogRecord request) {
        ensureEnabled();
        AgentCatalogMutation normalized = normalizeMutation(null, request);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "INSERT INTO portal_agent_catalog "
                    + "(agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, mcp_server_name, "
                    + "tool_count, transport_types_json, resource_summary, prompt_summary, status) "
                    + "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")) {
            statement.setString(1, normalized.agentId);
            statement.setString(2, normalized.canonicalName);
            statement.setString(3, normalized.displayName);
            statement.setString(4, normalized.intro);
            statement.setString(5, normalized.description);
            statement.setString(6, normalized.iconUrl);
            statement.setString(7, writeJson(normalized.tags));
            statement.setString(8, normalized.mcpServerName);
            statement.setInt(9, normalized.toolCount);
            statement.setString(10, writeJson(normalized.transportTypes));
            statement.setString(11, normalized.resourceSummary);
            statement.setString(12, normalized.promptSummary);
            statement.setString(13, STATUS_DRAFT);
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to create agent catalog asset.", ex);
        }
        return requireAsset(normalized.agentId);
    }

    public AgentCatalogRecord updateAsset(String agentId, AgentCatalogRecord request) {
        ensureEnabled();
        String normalizedAgentId = requireNonBlank(agentId, "agentId cannot be blank.");
        AgentCatalogRecord existed = requireAsset(normalizedAgentId);
        AgentCatalogMutation normalized = normalizeMutation(existed, request);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "UPDATE portal_agent_catalog "
                    + "SET canonical_name = ?, display_name = ?, intro = ?, description = ?, icon_url = ?, tags_json = ?, "
                    + "mcp_server_name = ?, tool_count = ?, transport_types_json = ?, resource_summary = ?, prompt_summary = ? "
                    + "WHERE agent_id = ?")) {
            statement.setString(1, normalized.canonicalName);
            statement.setString(2, normalized.displayName);
            statement.setString(3, normalized.intro);
            statement.setString(4, normalized.description);
            statement.setString(5, normalized.iconUrl);
            statement.setString(6, writeJson(normalized.tags));
            statement.setString(7, normalized.mcpServerName);
            statement.setInt(8, normalized.toolCount);
            statement.setString(9, writeJson(normalized.transportTypes));
            statement.setString(10, normalized.resourceSummary);
            statement.setString(11, normalized.promptSummary);
            statement.setString(12, normalizedAgentId);
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to update agent catalog asset.", ex);
        }
        if (StringUtils.equalsIgnoreCase(existed.getStatus(), STATUS_PUBLISHED)) {
            syncPublishedAgentMcpAuthorization(normalizedAgentId);
        }
        return requireAsset(normalizedAgentId);
    }

    public AgentCatalogRecord publishAsset(String agentId) {
        ensureEnabled();
        AgentCatalogRecord existed = requireAsset(requireNonBlank(agentId, "agentId cannot be blank."));
        McpServer detail = requireMcpServer(existed.getMcpServerName());
        ConsumerAuthInfo authInfo = detail.getConsumerAuthInfo();
        if (authInfo == null || !Boolean.TRUE.equals(authInfo.getEnable())) {
            throw new ValidationException("The selected MCP server must enable consumer auth before publish.");
        }
        if (StringUtils.isBlank(authInfo.getType())) {
            throw new ValidationException("The selected MCP server must configure an auth type before publish.");
        }

        Timestamp now = ConsoleDateTimeUtil.nowTimestamp();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "UPDATE portal_agent_catalog SET status = ?, published_at = ?, unpublished_at = NULL WHERE agent_id = ?")) {
            statement.setString(1, STATUS_PUBLISHED);
            statement.setTimestamp(2, now);
            statement.setString(3, existed.getAgentId());
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to publish agent catalog asset.", ex);
        }
        syncPublishedAgentMcpAuthorization(existed.getAgentId());
        return requireAsset(existed.getAgentId());
    }

    public AgentCatalogRecord unpublishAsset(String agentId) {
        ensureEnabled();
        AgentCatalogRecord existed = requireAsset(requireNonBlank(agentId, "agentId cannot be blank."));
        Timestamp now = ConsoleDateTimeUtil.nowTimestamp();
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "UPDATE portal_agent_catalog SET status = ?, unpublished_at = ? WHERE agent_id = ?")) {
            statement.setString(1, STATUS_UNPUBLISHED);
            statement.setTimestamp(2, now);
            statement.setString(3, existed.getAgentId());
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to unpublish agent catalog asset.", ex);
        }
        return requireAsset(existed.getAgentId());
    }

    public void syncPublishedAgentMcpAuthorization(String agentId) {
        ensureEnabled();
        AgentCatalogRecord asset = requireAsset(requireNonBlank(agentId, "agentId cannot be blank."));
        if (!StringUtils.equalsIgnoreCase(asset.getStatus(), STATUS_PUBLISHED)) {
            return;
        }

        McpServer detail = requireMcpServer(asset.getMcpServerName());
        ConsumerAuthInfo authInfo = detail.getConsumerAuthInfo();
        if (authInfo == null || !Boolean.TRUE.equals(authInfo.getEnable()) || StringUtils.isBlank(authInfo.getType())) {
            throw new ValidationException(
                "The published agent requires the bound MCP server to keep consumer auth enabled.");
        }

        List<String> consumers = authorizationSubjectResolver.resolveConsumers(ASSET_TYPE_AGENT_CATALOG, asset.getAgentId());
        authInfo.setAllowedConsumerLevels(Collections.emptyList());
        authInfo.setAllowedConsumers(CollectionUtils.isEmpty(consumers) ? null : consumers);
        detail.setConsumerAuthInfo(authInfo);
        mcpServerService.addOrUpdateWithAuthorization(detail);
    }

    private List<AgentCatalogRecord> listAssetsInternal() {
        String sql = "SELECT agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, "
            + "mcp_server_name, tool_count, transport_types_json, resource_summary, prompt_summary, status, "
            + "published_at, unpublished_at, created_at, updated_at "
            + "FROM portal_agent_catalog ORDER BY updated_at DESC, created_at DESC";
        List<AgentCatalogRecord> result = new ArrayList<>();
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql);
            ResultSet rs = statement.executeQuery()) {
            while (rs.next()) {
                result.add(readRecord(rs));
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to query agent catalog list.", ex);
        }
        return result;
    }

    private AgentCatalogRecord requireAsset(String agentId) {
        String sql = "SELECT agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, "
            + "mcp_server_name, tool_count, transport_types_json, resource_summary, prompt_summary, status, "
            + "published_at, unpublished_at, created_at, updated_at "
            + "FROM portal_agent_catalog WHERE agent_id = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, agentId);
            try (ResultSet rs = statement.executeQuery()) {
                if (!rs.next()) {
                    throw new ValidationException("Agent catalog asset not found: " + agentId);
                }
                return readRecord(rs);
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to query agent catalog asset.", ex);
        }
    }

    private AgentCatalogRecord readRecord(ResultSet rs) throws SQLException {
        return AgentCatalogRecord.builder()
            .agentId(rs.getString("agent_id"))
            .canonicalName(rs.getString("canonical_name"))
            .displayName(rs.getString("display_name"))
            .intro(rs.getString("intro"))
            .description(rs.getString("description"))
            .iconUrl(rs.getString("icon_url"))
            .tags(readStringList(rs.getString("tags_json")))
            .mcpServerName(rs.getString("mcp_server_name"))
            .toolCount(rs.getInt("tool_count"))
            .transportTypes(readStringList(rs.getString("transport_types_json")))
            .resourceSummary(rs.getString("resource_summary"))
            .promptSummary(rs.getString("prompt_summary"))
            .status(rs.getString("status"))
            .publishedAt(ConsoleDateTimeUtil.formatTimestamp(rs.getTimestamp("published_at")))
            .unpublishedAt(ConsoleDateTimeUtil.formatTimestamp(rs.getTimestamp("unpublished_at")))
            .createdAt(ConsoleDateTimeUtil.formatTimestamp(rs.getTimestamp("created_at")))
            .updatedAt(ConsoleDateTimeUtil.formatTimestamp(rs.getTimestamp("updated_at")))
            .build();
    }

    private AgentCatalogMutation normalizeMutation(AgentCatalogRecord existed, AgentCatalogRecord request) {
        if (request == null) {
            throw new ValidationException("request cannot be null.");
        }
        String agentId = existed == null ? requireNonBlank(request.getAgentId(), "agentId cannot be blank.") : existed.getAgentId();
        String canonicalName = requireNonBlank(defaultIfBlank(request.getCanonicalName(), agentId),
            "canonicalName cannot be blank.");
        String displayName = requireNonBlank(defaultIfBlank(request.getDisplayName(), canonicalName),
            "displayName cannot be blank.");
        String mcpServerName = requireNonBlank(defaultIfBlank(request.getMcpServerName(), existed == null ? null : existed.getMcpServerName()),
            "mcpServerName cannot be blank.");

        if (existed != null && StringUtils.equalsIgnoreCase(existed.getStatus(), STATUS_PUBLISHED)
            && !StringUtils.equals(existed.getMcpServerName(), mcpServerName)) {
            throw new ValidationException("Please unpublish the agent before changing the bound MCP server.");
        }

        if (mcpServerNameInUse(mcpServerName, agentId)) {
            throw new ValidationException("The MCP server is already bound by another agent: " + mcpServerName);
        }

        McpServer detail = requireMcpServer(mcpServerName);
        AgentDerivedMetadata metadata = deriveMetadata(detail);
        return AgentCatalogMutation.builder()
            .agentId(agentId)
            .canonicalName(canonicalName)
            .displayName(displayName)
            .intro(StringUtils.trimToNull(request.getIntro()))
            .description(StringUtils.trimToNull(request.getDescription()))
            .iconUrl(StringUtils.trimToNull(request.getIconUrl()))
            .tags(normalizeTags(request.getTags()))
            .mcpServerName(mcpServerName)
            .toolCount(metadata.toolCount)
            .transportTypes(metadata.transportTypes)
            .resourceSummary(metadata.resourceSummary)
            .promptSummary(metadata.promptSummary)
            .build();
    }

    private AgentDerivedMetadata deriveMetadata(McpServer detail) {
        String rawConfigurations = detail == null ? null : detail.getRawConfigurations();
        int toolCount = countTools(rawConfigurations);
        return AgentDerivedMetadata.builder()
            .toolCount(toolCount)
            .transportTypes(DEFAULT_TRANSPORT_TYPES)
            .resourceSummary(buildSectionSummary(rawConfigurations, "resources", "resource"))
            .promptSummary(buildSectionSummary(rawConfigurations, "prompts", "prompt"))
            .build();
    }

    private int countTools(String rawConfigurations) {
        if (StringUtils.isBlank(rawConfigurations)) {
            return 0;
        }
        Matcher matcher = TOOL_NAME_PATTERN.matcher(rawConfigurations);
        int count = 0;
        while (matcher.find()) {
            count++;
        }
        return count;
    }

    private String buildSectionSummary(String rawConfigurations, String pluralSectionName, String singularLabel) {
        if (StringUtils.isBlank(rawConfigurations)) {
            return null;
        }
        String normalizedRaw = rawConfigurations.toLowerCase(Locale.ROOT);
        if (!normalizedRaw.contains(pluralSectionName + ":") && !normalizedRaw.contains(singularLabel + ":")) {
            return null;
        }
        return String.format("当前已声明 %s，首版仅做展示与接入说明，请以 MCP 原始配置为准。", pluralSectionName);
    }

    private boolean mcpServerNameInUse(String mcpServerName, String currentAgentId) {
        String sql = "SELECT agent_id FROM portal_agent_catalog WHERE mcp_server_name = ?";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, mcpServerName);
            try (ResultSet rs = statement.executeQuery()) {
                if (!rs.next()) {
                    return false;
                }
                return !StringUtils.equals(rs.getString("agent_id"), currentAgentId);
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to validate mcp server binding uniqueness.", ex);
        }
    }

    private McpServer requireMcpServer(String mcpServerName) {
        try {
            McpServer detail = mcpServerService.query(mcpServerName);
            if (detail == null) {
                throw new ValidationException("MCP server not found: " + mcpServerName);
            }
            return detail;
        } catch (ValidationException ex) {
            throw ex;
        } catch (Exception ex) {
            throw new BusinessException("Failed to query MCP server detail: " + mcpServerName, ex);
        }
    }

    private void ensureAgentCatalogTable() {
        if (!enabled()) {
            return;
        }
        String sql = "CREATE TABLE IF NOT EXISTS portal_agent_catalog ("
            + " agent_id VARCHAR(128) NOT NULL PRIMARY KEY,"
            + " canonical_name VARCHAR(255) NOT NULL,"
            + " display_name VARCHAR(255) NOT NULL,"
            + " intro TEXT NULL,"
            + " description TEXT NULL,"
            + " icon_url VARCHAR(1024) NULL,"
            + " tags_json TEXT NULL,"
            + " mcp_server_name VARCHAR(255) NOT NULL,"
            + " tool_count INT NOT NULL DEFAULT 0,"
            + " transport_types_json TEXT NULL,"
            + " resource_summary TEXT NULL,"
            + " prompt_summary TEXT NULL,"
            + " status VARCHAR(32) NOT NULL DEFAULT 'draft',"
            + " published_at DATETIME NULL,"
            + " unpublished_at DATETIME NULL,"
            + " created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
            + " updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
            + " UNIQUE KEY uk_portal_agent_catalog_mcp_server (mcp_server_name),"
            + " INDEX idx_portal_agent_catalog_status (status)"
            + ")";
        try (Connection connection = openConnection(); PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.executeUpdate();
        } catch (SQLException ex) {
            log.warn("Failed to ensure portal_agent_catalog table.", ex);
        }
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

    private String defaultIfBlank(String preferred, String fallback) {
        String normalizedPreferred = StringUtils.trimToNull(preferred);
        return normalizedPreferred != null ? normalizedPreferred : StringUtils.trimToNull(fallback);
    }

    private List<String> normalizeTags(List<String> tags) {
        if (CollectionUtils.isEmpty(tags)) {
            return Collections.emptyList();
        }
        return tags.stream()
            .map(StringUtils::trimToNull)
            .filter(Objects::nonNull)
            .distinct()
            .collect(Collectors.toCollection(ArrayList::new));
    }

    private List<String> readStringList(String rawValue) {
        if (StringUtils.isBlank(rawValue)) {
            return Collections.emptyList();
        }
        try {
            List<String> values = objectMapper.readValue(rawValue, STRING_LIST_TYPE);
            if (values == null) {
                return Collections.emptyList();
            }
            return values.stream().map(StringUtils::trimToNull).filter(Objects::nonNull)
                .collect(Collectors.toCollection(ArrayList::new));
        } catch (Exception ex) {
            log.warn("Failed to parse json list value: {}", rawValue, ex);
            return Collections.emptyList();
        }
    }

    private String writeJson(List<String> values) {
        if (CollectionUtils.isEmpty(values)) {
            return "[]";
        }
        List<String> normalized = new ArrayList<>(new LinkedHashSet<>(values.stream()
            .map(StringUtils::trimToNull)
            .filter(Objects::nonNull)
            .collect(Collectors.toList())));
        try {
            return objectMapper.writeValueAsString(normalized);
        } catch (JsonProcessingException ex) {
            throw new ValidationException("Failed to serialize list field.");
        }
    }

    private Connection openConnection() throws SQLException {
        if (StringUtils.isBlank(dbUsername)) {
            return DriverManager.getConnection(dbUrl);
        }
        return DriverManager.getConnection(dbUrl, dbUsername, dbPassword);
    }

    @lombok.Builder
    private static class AgentCatalogMutation {
        private String agentId;
        private String canonicalName;
        private String displayName;
        private String intro;
        private String description;
        private String iconUrl;
        private List<String> tags;
        private String mcpServerName;
        private int toolCount;
        private List<String> transportTypes;
        private String resourceSummary;
        private String promptSummary;
    }

    @lombok.Builder
    private static class AgentDerivedMetadata {
        private int toolCount;
        private List<String> transportTypes;
        private String resourceSummary;
        private String promptSummary;
    }
}
