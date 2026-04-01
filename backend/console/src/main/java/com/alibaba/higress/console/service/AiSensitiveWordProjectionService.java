package com.alibaba.higress.console.service;

import java.time.LocalDateTime;
import java.util.ArrayList;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.stream.Collectors;

import javax.annotation.PostConstruct;
import javax.annotation.Resource;

import org.apache.commons.collections4.CollectionUtils;
import org.apache.commons.collections4.MapUtils;
import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.aisensitive.AiSensitiveDetectRule;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveReplaceRule;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveStatus;
import com.alibaba.higress.console.model.aisensitive.AiSensitiveSystemConfig;
import com.alibaba.higress.console.service.portal.AiSensitiveWordJdbcService;
import com.alibaba.higress.console.util.AiSensitiveDateTimeUtil;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.constant.plugin.BuiltInPluginName;
import com.alibaba.higress.sdk.model.WasmPluginInstance;
import com.alibaba.higress.sdk.model.WasmPluginInstanceScope;
import com.alibaba.higress.sdk.service.WasmPluginInstanceService;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class AiSensitiveWordProjectionService {

    private static final String KEY_DENY_RULES = "deny_rules";
    private static final String KEY_REPLACE_RULES = "replace_rules";
    private static final String KEY_AUDIT_SINK = "audit_sink";
    private static final String KEY_SYSTEM_DENY = "system_deny";
    private static final String KEY_SYSTEM_DENY_WORDS = "system_deny_words";

    private AiSensitiveWordJdbcService aiSensitiveWordJdbcService;
    private WasmPluginInstanceService wasmPluginInstanceService;

    @Value("${higress.portal.ai-sensitive.projection.write-enabled:true}")
    private boolean projectionWriteEnabled;

    @Value("${higress.portal.ai-sensitive.audit.service-name:aigateway-console}")
    private String auditServiceName;

    @Value("${higress.portal.ai-sensitive.audit.namespace:default}")
    private String auditNamespace;

    @Value("${higress.portal.ai-sensitive.audit.port:8080}")
    private int auditPort;

    @Value("${higress.portal.ai-sensitive.audit.path:/v1/internal/ai/sensitive-block-events}")
    private String auditPath;

    @Value("${higress.portal.ai-sensitive.audit.timeout-ms:2000}")
    private int auditTimeoutMs;

    private volatile LocalDateTime lastReconciledAt;
    private volatile LocalDateTime lastMigratedAt;
    private volatile String lastError;
    private volatile int projectedInstanceCount;
    private volatile boolean migrationChecked;

    @Resource
    public void setAiSensitiveWordJdbcService(AiSensitiveWordJdbcService aiSensitiveWordJdbcService) {
        this.aiSensitiveWordJdbcService = aiSensitiveWordJdbcService;
    }

    @Resource
    public void setWasmPluginInstanceService(WasmPluginInstanceService wasmPluginInstanceService) {
        this.wasmPluginInstanceService = wasmPluginInstanceService;
    }

    @PostConstruct
    public void init() {
        syncNow();
    }

    @Scheduled(initialDelayString = "${higress.portal.ai-sensitive.sync.initial-delay-millis:10000}",
        fixedDelayString = "${higress.portal.ai-sensitive.sync.interval-millis:30000}")
    public void scheduledSync() {
        syncNow();
    }

    public synchronized void syncNow() {
        if (!aiSensitiveWordJdbcService.enabled() || !projectionWriteEnabled) {
            return;
        }
        try {
            if (!migrationChecked) {
                migrateLegacyRulesIfNeeded();
                migrationChecked = true;
            }
            AiSensitiveSystemConfig systemConfig = aiSensitiveWordJdbcService.getSystemConfig();
            List<WasmPluginInstance> instances =
                wasmPluginInstanceService.list(BuiltInPluginName.AI_DATA_MASKING, false);
            List<WasmPluginInstance> instancesToUpdate = new ArrayList<>();
            int activeInstanceCount = 0;
            for (WasmPluginInstance instance : instances) {
                if (instance == null || !Boolean.TRUE.equals(instance.getEnabled())) {
                    continue;
                }
                activeInstanceCount++;
                Map<String, Object> mergedConfigurations =
                    buildProjectedConfigurations(instance.getConfigurations(), systemConfig);
                if (Objects.equals(instance.getConfigurations(), mergedConfigurations)) {
                    continue;
                }
                instance.setConfigurations(mergedConfigurations);
                instance.setRawConfigurations(null);
                instancesToUpdate.add(instance);
            }
            if (CollectionUtils.isNotEmpty(instancesToUpdate)) {
                wasmPluginInstanceService.addOrUpdateAll(instancesToUpdate);
            }
            projectedInstanceCount = activeInstanceCount;
            lastReconciledAt = ConsoleDateTimeUtil.now();
            lastError = null;
        } catch (Exception ex) {
            lastError = ex.getMessage();
            log.warn("Failed to reconcile AI sensitive word rules to ai-data-masking.", ex);
        }
    }

    public AiSensitiveStatus getStatus() {
        AiSensitiveSystemConfig systemConfig = aiSensitiveWordJdbcService.getSystemConfig();
        return AiSensitiveStatus.builder()
            .dbEnabled(aiSensitiveWordJdbcService.enabled())
            .detectRuleCount(aiSensitiveWordJdbcService.countDetectRules())
            .replaceRuleCount(aiSensitiveWordJdbcService.countReplaceRules())
            .auditRecordCount(aiSensitiveWordJdbcService.countAuditRecords())
            .systemDenyEnabled(Boolean.TRUE.equals(systemConfig.getSystemDenyEnabled()))
            .systemDictionaryWordCount(aiSensitiveWordJdbcService.parseDictionaryWords(systemConfig.getDictionaryText())
                .size())
            .systemDictionaryUpdatedAt(systemConfig.getUpdatedAt())
            .projectedInstanceCount(projectedInstanceCount)
            .lastReconciledAt(AiSensitiveDateTimeUtil.formatLocalDateTime(lastReconciledAt))
            .lastMigratedAt(AiSensitiveDateTimeUtil.formatLocalDateTime(lastMigratedAt))
            .lastError(lastError)
            .build();
    }

    private Map<String, Object> buildProjectedConfigurations(
        Map<String, Object> existingConfigurations,
        AiSensitiveSystemConfig systemConfig) {
        Map<String, Object> configurations = new LinkedHashMap<>();
        if (MapUtils.isNotEmpty(existingConfigurations)) {
            configurations.putAll(existingConfigurations);
        }
        configurations.remove("deny_words");
        configurations.remove("replace_roles");
        configurations.remove(KEY_SYSTEM_DENY_WORDS);
        configurations.put(KEY_DENY_RULES, buildDetectRuleConfigs());
        configurations.put(KEY_REPLACE_RULES, buildReplaceRuleConfigs());
        configurations.put(KEY_AUDIT_SINK, buildAuditSinkConfig());
        configurations.put(KEY_SYSTEM_DENY, Boolean.TRUE.equals(systemConfig.getSystemDenyEnabled()));
        if (Boolean.TRUE.equals(systemConfig.getSystemDenyEnabled()) && useProjectedSystemDictionary(systemConfig)) {
            configurations.put(
                KEY_SYSTEM_DENY_WORDS,
                aiSensitiveWordJdbcService.parseDictionaryWords(systemConfig.getDictionaryText()));
        }
        return configurations;
    }

    private boolean useProjectedSystemDictionary(AiSensitiveSystemConfig systemConfig) {
        String bundledDictionary = aiSensitiveWordJdbcService.getBundledSystemDictionaryText();
        String normalizedDictionary = aiSensitiveWordJdbcService.normalizeDictionaryText(systemConfig.getDictionaryText());
        return !Objects.equals(bundledDictionary, normalizedDictionary);
    }

    private List<Map<String, Object>> buildDetectRuleConfigs() {
        return aiSensitiveWordJdbcService.listEnabledDetectRules().stream().map(rule -> {
            Map<String, Object> item = new LinkedHashMap<>();
            item.put("id", rule.getId());
            item.put("pattern", rule.getPattern());
            item.put("match_type", rule.getMatchType());
            item.put("description", rule.getDescription());
            item.put("priority", rule.getPriority());
            item.put("enabled", rule.getEnabled());
            return item;
        }).collect(Collectors.toList());
    }

    private List<Map<String, Object>> buildReplaceRuleConfigs() {
        return aiSensitiveWordJdbcService.listEnabledReplaceRules().stream().map(rule -> {
            Map<String, Object> item = new LinkedHashMap<>();
            item.put("id", rule.getId());
            item.put("pattern", rule.getPattern());
            item.put("replace_type", rule.getReplaceType());
            item.put("replace_value", rule.getReplaceValue());
            item.put("restore", rule.getRestore());
            item.put("description", rule.getDescription());
            item.put("priority", rule.getPriority());
            item.put("enabled", rule.getEnabled());
            return item;
        }).collect(Collectors.toList());
    }

    private Map<String, Object> buildAuditSinkConfig() {
        Map<String, Object> item = new LinkedHashMap<>();
        item.put("service_name", auditServiceName);
        item.put("namespace", auditNamespace);
        item.put("port", auditPort);
        item.put("path", auditPath);
        item.put("timeout_ms", auditTimeoutMs);
        return item;
    }

    @SuppressWarnings("unchecked")
    private void migrateLegacyRulesIfNeeded() {
        if (aiSensitiveWordJdbcService.hasAnyRules()) {
            return;
        }
        WasmPluginInstance globalInstance = wasmPluginInstanceService.query(WasmPluginInstanceScope.GLOBAL, null,
            BuiltInPluginName.AI_DATA_MASKING, false);
        WasmPluginInstance sourceInstance = globalInstance;
        if (sourceInstance == null) {
            sourceInstance = wasmPluginInstanceService.list(BuiltInPluginName.AI_DATA_MASKING, false).stream()
                .filter(Objects::nonNull)
                .filter(instance -> Boolean.TRUE.equals(instance.getEnabled()))
                .findFirst()
                .orElse(null);
        }
        if (sourceInstance == null || MapUtils.isEmpty(sourceInstance.getConfigurations())) {
            return;
        }
        boolean imported = false;
        Object denyRules = sourceInstance.getConfigurations().get(KEY_DENY_RULES);
        if (denyRules instanceof List<?>) {
            imported |= importDetectRules((List<?>) denyRules);
        } else {
            Object denyWords = sourceInstance.getConfigurations().get("deny_words");
            if (denyWords instanceof List<?>) {
                for (Object item : (List<?>) denyWords) {
                    String pattern = StringUtils.trimToNull(String.valueOf(item));
                    if (pattern == null) {
                        continue;
                    }
                    aiSensitiveWordJdbcService.saveDetectRule(AiSensitiveDetectRule.builder()
                        .pattern(pattern)
                        .matchType("contains")
                        .enabled(Boolean.TRUE)
                        .priority(0)
                        .build());
                    imported = true;
                }
            }
        }

        Object replaceRules = sourceInstance.getConfigurations().get(KEY_REPLACE_RULES);
        if (replaceRules instanceof List<?>) {
            imported |= importReplaceRules((List<?>) replaceRules);
        } else {
            Object replaceRoles = sourceInstance.getConfigurations().get("replace_roles");
            if (replaceRoles instanceof List<?>) {
                imported |= importReplaceRules((List<?>) replaceRoles);
            }
        }

        if (imported) {
            lastMigratedAt = ConsoleDateTimeUtil.now();
        }
    }

    @SuppressWarnings("unchecked")
    private boolean importDetectRules(List<?> rawRules) {
        boolean imported = false;
        for (Object item : rawRules) {
            if (!(item instanceof Map<?, ?>)) {
                continue;
            }
            Map<?, ?> ruleMap = (Map<?, ?>) item;
            String pattern = StringUtils.trimToNull(String.valueOf(ruleMap.get("pattern")));
            if (pattern == null) {
                continue;
            }
            aiSensitiveWordJdbcService.saveDetectRule(AiSensitiveDetectRule.builder()
                .pattern(pattern)
                .matchType(StringUtils.defaultIfBlank(StringUtils.trimToNull(String.valueOf(ruleMap.get("match_type"))),
                    "contains"))
                .description(toNullableString(ruleMap.get("description")))
                .priority(toInteger(ruleMap.get("priority")))
                .enabled(toBoolean(ruleMap.get("enabled"), true))
                .build());
            imported = true;
        }
        return imported;
    }

    private boolean importReplaceRules(List<?> rawRules) {
        boolean imported = false;
        for (Object item : rawRules) {
            if (!(item instanceof Map<?, ?>)) {
                continue;
            }
            Map<?, ?> ruleMap = (Map<?, ?>) item;
            String pattern = toNullableString(firstNonNull(ruleMap.get("pattern"), ruleMap.get("regex")));
            if (pattern == null) {
                continue;
            }
            aiSensitiveWordJdbcService.saveReplaceRule(AiSensitiveReplaceRule.builder()
                .pattern(pattern)
                .replaceType(StringUtils.defaultIfBlank(
                    toNullableString(firstNonNull(ruleMap.get("replace_type"), ruleMap.get("type"))), "replace"))
                .replaceValue(StringUtils.defaultString(toNullableString(firstNonNull(ruleMap.get("replace_value"),
                    ruleMap.get("value")))))
                .restore(toBoolean(ruleMap.get("restore"), false))
                .description(toNullableString(ruleMap.get("description")))
                .priority(toInteger(ruleMap.get("priority")))
                .enabled(toBoolean(ruleMap.get("enabled"), true))
                .build());
            imported = true;
        }
        return imported;
    }

    private Object firstNonNull(Object left, Object right) {
        return left != null ? left : right;
    }

    private String toNullableString(Object value) {
        return StringUtils.trimToNull(value == null ? null : String.valueOf(value));
    }

    private Integer toInteger(Object value) {
        if (value instanceof Number) {
            return ((Number) value).intValue();
        }
        String text = toNullableString(value);
        if (text == null) {
            return 0;
        }
        try {
            return Integer.parseInt(text);
        } catch (NumberFormatException ex) {
            return 0;
        }
    }

    private Boolean toBoolean(Object value, boolean defaultValue) {
        if (value instanceof Boolean) {
            return (Boolean) value;
        }
        String text = toNullableString(value);
        if (text == null) {
            return defaultValue;
        }
        return Boolean.parseBoolean(text);
    }
}
