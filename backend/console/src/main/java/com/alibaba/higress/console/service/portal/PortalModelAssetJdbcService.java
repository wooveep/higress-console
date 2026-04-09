package com.alibaba.higress.console.service.portal;

import java.math.BigDecimal;
import java.math.RoundingMode;
import java.sql.Connection;
import java.sql.DriverManager;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.sql.Timestamp;
import java.time.LocalDateTime;
import java.time.format.DateTimeFormatter;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Comparator;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.Objects;
import java.util.Set;

import javax.annotation.PostConstruct;
import javax.annotation.Resource;

import org.apache.commons.lang3.StringUtils;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;

import com.alibaba.higress.console.model.portal.ModelAssetBindingRecord;
import com.alibaba.higress.console.model.portal.ModelAssetCapabilityOptions;
import com.alibaba.higress.console.model.portal.ModelAssetCapabilities;
import com.alibaba.higress.console.model.portal.ModelAssetOptionsRecord;
import com.alibaba.higress.console.model.portal.ModelAssetRecord;
import com.alibaba.higress.console.model.portal.ModelBindingLimits;
import com.alibaba.higress.console.model.portal.ModelBindingPricing;
import com.alibaba.higress.console.model.portal.ModelBindingPriceVersionRecord;
import com.alibaba.higress.console.model.portal.ProviderModelCatalogRecord;
import com.alibaba.higress.console.model.portal.ProviderModelOptionRecord;
import com.alibaba.higress.console.util.ConsoleDateTimeUtil;
import com.alibaba.higress.sdk.exception.BusinessException;
import com.alibaba.higress.sdk.exception.ValidationException;
import com.alibaba.higress.sdk.model.CommonPageQuery;
import com.alibaba.higress.sdk.model.PaginatedResult;
import com.alibaba.higress.sdk.model.ai.LlmProvider;
import com.alibaba.higress.sdk.service.ai.LlmProviderService;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;

import lombok.extern.slf4j.Slf4j;

@Slf4j
@Service
public class PortalModelAssetJdbcService {

    private static final String PORTAL_MODEL_META_KEY = "portalModelMeta";
    private static final String STATUS_DRAFT = "draft";
    private static final String STATUS_PUBLISHED = "published";
    private static final String STATUS_UNPUBLISHED = "unpublished";
    private static final String BILLING_STATUS_ACTIVE = "active";
    private static final String BILLING_STATUS_INACTIVE = "inactive";
    private static final String BILLING_STATUS_DISABLED = "disabled";
    private static final String CURRENCY_CNY = "CNY";
    private static final String DEFAULT_ENDPOINT = "-";
    private static final String DEFAULT_PROTOCOL = "openai/v1";
    private static final long MICRO_YUAN_PER_RMB = 1_000_000L;
    private static final Set<String> PRESET_TAGS = Collections.unmodifiableSet(new LinkedHashSet<>(java.util.Arrays.asList(
        "旗舰",
        "高性价比",
        "推理",
        "长上下文",
        "视觉",
        "多模态",
        "代码",
        "Embedding",
        "图像生成",
        "语音",
        "函数调用",
        "结构化输出")));
    private static final List<String> PRESET_MODALITIES = Collections.unmodifiableList(Arrays.asList(
        "text",
        "image",
        "audio",
        "video",
        "embedding"));
    private static final Set<String> PRESET_MODALITY_SET = Collections.unmodifiableSet(new LinkedHashSet<>(PRESET_MODALITIES));
    private static final List<String> PRESET_FEATURES = Collections.unmodifiableList(Arrays.asList(
        "reasoning",
        "vision",
        "function_calling",
        "structured_output",
        "long_context",
        "code",
        "multimodal"));
    private static final Set<String> PRESET_FEATURE_SET = Collections.unmodifiableSet(new LinkedHashSet<>(PRESET_FEATURES));
    private static final List<String> PRESET_REQUEST_KINDS = Collections.unmodifiableList(Arrays.asList(
        "chat_completions",
        "responses",
        "embeddings",
        "images",
        "audio"));
    private static final Set<String> PRESET_REQUEST_KIND_SET = Collections.unmodifiableSet(
        new LinkedHashSet<>(PRESET_REQUEST_KINDS));
    private static final Map<String, List<ProviderModelOptionRecord>> PROVIDER_MODEL_CATALOG = createProviderModelCatalog();
    private static final TypeReference<List<String>> STRING_LIST_TYPE = new TypeReference<List<String>>() {
    };
    private static final TypeReference<ModelBindingPricing> PRICING_TYPE = new TypeReference<ModelBindingPricing>() {
    };

    @Value("${higress.portal.db.url:}")
    private String dbUrl;

    @Value("${higress.portal.db.username:}")
    private String dbUsername;

    @Value("${higress.portal.db.password:}")
    private String dbPassword;

    private ObjectMapper objectMapper;
    private LlmProviderService llmProviderService;

    @Resource
    public void setObjectMapper(ObjectMapper objectMapper) {
        this.objectMapper = objectMapper;
    }

    @Resource
    public void setLlmProviderService(LlmProviderService llmProviderService) {
        this.llmProviderService = llmProviderService;
    }

    @PostConstruct
    public void init() {
        ensureModelAssetTables();
        backfillLegacyProviders();
    }

    public boolean enabled() {
        return StringUtils.isNotBlank(dbUrl);
    }

    public PaginatedResult<ModelAssetRecord> listAssets(CommonPageQuery query) {
        ensureEnabled();
        List<ModelAssetRecord> items = listAssetsInternal();
        return PaginatedResult.createFromFullList(items, query);
    }

    public ModelAssetRecord queryAsset(String assetId) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        return requireAsset(normalizedAssetId);
    }

    public ModelAssetOptionsRecord queryOptions() {
        ensureEnabled();
        return ModelAssetOptionsRecord.builder()
            .capabilities(ModelAssetCapabilityOptions.builder()
                .modalities(PRESET_MODALITIES)
                .features(PRESET_FEATURES)
                .requestKinds(PRESET_REQUEST_KINDS)
                .build())
            .providerModels(listProviderModelCatalogs())
            .build();
    }

    public ModelAssetRecord createAsset(ModelAssetRecord request) {
        ensureEnabled();
        AssetMutation normalized = normalizeAssetRequest(request);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "INSERT INTO portal_model_asset "
                    + "(asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json) "
                    + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)")) {
            statement.setString(1, normalized.assetId);
            statement.setString(2, normalized.canonicalName);
            statement.setString(3, normalized.displayName);
            statement.setString(4, normalized.intro);
            statement.setString(5, writeJson(normalized.tags));
            statement.setString(6, writeJson(normalized.modalities));
            statement.setString(7, writeJson(normalized.features));
            statement.setString(8, writeJson(normalized.requestKinds));
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to create model asset.", ex);
        }
        return requireAsset(normalized.assetId);
    }

    public ModelAssetRecord updateAsset(String assetId, ModelAssetRecord request) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        if (!assetExists(normalizedAssetId)) {
            throw new ValidationException("Model asset not found: " + normalizedAssetId);
        }
        AssetMutation normalized = normalizeAssetRequest(request);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "UPDATE portal_model_asset "
                    + "SET canonical_name = ?, display_name = ?, intro = ?, tags_json = ?, modalities_json = ?, features_json = ?, request_kinds_json = ? "
                    + "WHERE asset_id = ?")) {
            statement.setString(1, normalized.canonicalName);
            statement.setString(2, normalized.displayName);
            statement.setString(3, normalized.intro);
            statement.setString(4, writeJson(normalized.tags));
            statement.setString(5, writeJson(normalized.modalities));
            statement.setString(6, writeJson(normalized.features));
            statement.setString(7, writeJson(normalized.requestKinds));
            statement.setString(8, normalizedAssetId);
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to update model asset.", ex);
        }
        return requireAsset(normalizedAssetId);
    }

    public ModelAssetBindingRecord createBinding(String assetId, ModelAssetBindingRecord request) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        if (!assetExists(normalizedAssetId)) {
            throw new ValidationException("Model asset not found: " + normalizedAssetId);
        }
        BindingMutation normalized = normalizeBindingRequest(request);
        ensureProviderExists(normalized.providerName);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "INSERT INTO portal_model_binding "
                    + "(binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, rpm, tpm, context_window, status) "
                    + "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")) {
            statement.setString(1, normalized.bindingId);
            statement.setString(2, normalizedAssetId);
            statement.setString(3, normalized.modelId);
            statement.setString(4, normalized.providerName);
            statement.setString(5, normalized.targetModel);
            statement.setString(6, normalized.protocol);
            statement.setString(7, normalized.endpoint);
            statement.setString(8, writeJson(normalized.pricing));
            statement.setLong(9, normalized.rpm);
            statement.setLong(10, normalized.tpm);
            statement.setLong(11, normalized.contextWindow);
            statement.setString(12, STATUS_DRAFT);
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to create model binding.", ex);
        }
        return requireBinding(normalizedAssetId, normalized.bindingId);
    }

    public ModelAssetBindingRecord updateBinding(String assetId, String bindingId, ModelAssetBindingRecord request) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        String normalizedBindingId = requireNonBlank(bindingId, "bindingId cannot be blank.");
        BindingRow existing = requireBindingRow(normalizedAssetId, normalizedBindingId);
        BindingMutation normalized = normalizeBindingRequest(request);
        ensureProviderExists(normalized.providerName);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "UPDATE portal_model_binding "
                    + "SET model_id = ?, provider_name = ?, target_model = ?, protocol = ?, endpoint = ?, pricing_json = ?, rpm = ?, tpm = ?, context_window = ? "
                    + "WHERE asset_id = ? AND binding_id = ?")) {
            statement.setString(1, normalized.modelId);
            statement.setString(2, normalized.providerName);
            statement.setString(3, normalized.targetModel);
            statement.setString(4, normalized.protocol);
            statement.setString(5, normalized.endpoint);
            statement.setString(6, writeJson(normalized.pricing));
            statement.setLong(7, normalized.rpm);
            statement.setLong(8, normalized.tpm);
            statement.setLong(9, normalized.contextWindow);
            statement.setString(10, normalizedAssetId);
            statement.setString(11, normalizedBindingId);
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to update model binding.", ex);
        }
        // Keep current binding status until an explicit publish/unpublish action happens.
        if (StringUtils.equals(existing.status, STATUS_PUBLISHED)) {
            log.info("Model binding {} updated while published; waiting for explicit publish to rotate Portal snapshot.",
                normalizedBindingId);
        }
        return requireBinding(normalizedAssetId, normalizedBindingId);
    }

    public ModelAssetBindingRecord publishBinding(String assetId, String bindingId) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        String normalizedBindingId = requireNonBlank(bindingId, "bindingId cannot be blank.");
        ModelAssetRecord asset = requireAsset(normalizedAssetId);
        BindingRow binding = requireBindingRow(normalizedAssetId, normalizedBindingId);
        LlmProvider provider = requireProvider(binding.providerName);
        BindingMutation normalized = toMutation(binding);
        validateBindingMutation(normalized);

        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                PublishedBindingMeta meta = buildPublishedBindingMeta(asset, normalized, provider);
                upsertCatalog(connection, meta);
                upsertPriceVersion(connection, meta);
                try (PreparedStatement statement = connection.prepareStatement(
                    "UPDATE portal_model_binding SET status = ?, published_at = ?, unpublished_at = NULL WHERE asset_id = ? AND binding_id = ?")) {
                    statement.setString(1, STATUS_PUBLISHED);
                    statement.setTimestamp(2, ConsoleDateTimeUtil.nowTimestamp());
                    statement.setString(3, normalizedAssetId);
                    statement.setString(4, normalizedBindingId);
                    statement.executeUpdate();
                }
                connection.commit();
            } catch (Exception ex) {
                connection.rollback();
                throw ex;
            } finally {
                connection.setAutoCommit(true);
            }
        } catch (Exception ex) {
            throw new BusinessException("Failed to publish model binding.", ex);
        }
        return requireBinding(normalizedAssetId, normalizedBindingId);
    }

    public ModelAssetBindingRecord unpublishBinding(String assetId, String bindingId) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        String normalizedBindingId = requireNonBlank(bindingId, "bindingId cannot be blank.");
        BindingRow binding = requireBindingRow(normalizedAssetId, normalizedBindingId);
        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                disablePublishedModel(connection, binding.modelId);
                try (PreparedStatement statement = connection.prepareStatement(
                    "UPDATE portal_model_binding SET status = ?, unpublished_at = ? WHERE asset_id = ? AND binding_id = ?")) {
                    statement.setString(1, STATUS_UNPUBLISHED);
                    statement.setTimestamp(2, ConsoleDateTimeUtil.nowTimestamp());
                    statement.setString(3, normalizedAssetId);
                    statement.setString(4, normalizedBindingId);
                    statement.executeUpdate();
                }
                connection.commit();
            } catch (Exception ex) {
                connection.rollback();
                throw ex;
            } finally {
                connection.setAutoCommit(true);
            }
        } catch (Exception ex) {
            throw new BusinessException("Failed to unpublish model binding.", ex);
        }
        return requireBinding(normalizedAssetId, normalizedBindingId);
    }

    public boolean hasBindingsForProvider(String providerName) {
        if (!enabled()) {
            return false;
        }
        String normalizedProviderName = StringUtils.trimToNull(providerName);
        if (normalizedProviderName == null) {
            return false;
        }
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "SELECT 1 FROM portal_model_binding WHERE provider_name = ? LIMIT 1")) {
            statement.setString(1, normalizedProviderName);
            try (ResultSet rs = statement.executeQuery()) {
                return rs.next();
            }
        } catch (SQLException ex) {
            log.warn("Failed to check model bindings for provider {}.", normalizedProviderName, ex);
            return false;
        }
    }

    public List<ModelBindingPriceVersionRecord> listPriceVersions(String assetId, String bindingId) {
        ensureEnabled();
        BindingRow binding = requireBindingRow(requireNonBlank(assetId, "assetId cannot be blank."),
            requireNonBlank(bindingId, "bindingId cannot be blank."));
        List<ModelBindingPriceVersionRecord> items = new ArrayList<>();
        String sql = "SELECT id, model_id, currency, input_price_per_1k_micro_yuan, output_price_per_1k_micro_yuan, "
            + "input_request_price_micro_yuan, cache_creation_input_token_price_per_1k_micro_yuan, "
            + "cache_creation_input_token_price_above_1hr_per_1k_micro_yuan, cache_read_input_token_price_per_1k_micro_yuan, "
            + "input_token_price_above_200k_per_1k_micro_yuan, output_token_price_above_200k_per_1k_micro_yuan, "
            + "cache_creation_input_token_price_above_200k_per_1k_micro_yuan, cache_read_input_token_price_above_200k_per_1k_micro_yuan, "
            + "output_image_price_micro_yuan, output_image_token_price_per_1k_micro_yuan, input_image_price_micro_yuan, "
            + "input_image_token_price_per_1k_micro_yuan, supports_prompt_caching, effective_from, effective_to, status, "
            + "created_at, updated_at FROM billing_model_price_version WHERE model_id = ? ORDER BY id DESC";
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, binding.modelId);
            try (ResultSet rs = statement.executeQuery()) {
                while (rs.next()) {
                    items.add(toPriceVersionRecord(rs));
                }
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to query model price versions.", ex);
        }
        return items;
    }

    public ModelAssetBindingRecord restorePriceVersion(String assetId, String bindingId, Long versionId) {
        ensureEnabled();
        String normalizedAssetId = requireNonBlank(assetId, "assetId cannot be blank.");
        String normalizedBindingId = requireNonBlank(bindingId, "bindingId cannot be blank.");
        if (versionId == null || versionId <= 0) {
            throw new ValidationException("versionId must be positive.");
        }
        BindingRow binding = requireBindingRow(normalizedAssetId, normalizedBindingId);
        ModelBindingPricing pricing = requirePriceVersion(binding.modelId, versionId);
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "UPDATE portal_model_binding SET pricing_json = ? WHERE asset_id = ? AND binding_id = ?")) {
            statement.setString(1, writeJson(pricing));
            statement.setString(2, normalizedAssetId);
            statement.setString(3, normalizedBindingId);
            statement.executeUpdate();
        } catch (SQLException ex) {
            throw new BusinessException("Failed to restore model price version.", ex);
        }
        return requireBinding(normalizedAssetId, normalizedBindingId);
    }

    private List<ModelAssetRecord> listAssetsInternal() {
        Map<String, ModelAssetRecord> assetMap = new LinkedHashMap<>();
        try (Connection connection = openConnection();
            PreparedStatement assetStatement = connection.prepareStatement(
                "SELECT asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at "
                    + "FROM portal_model_asset ORDER BY canonical_name ASC, asset_id ASC");
            ResultSet assetRows = assetStatement.executeQuery()) {
            while (assetRows.next()) {
                String assetId = assetRows.getString("asset_id");
                assetMap.put(assetId, ModelAssetRecord.builder()
                    .assetId(assetId)
                    .canonicalName(StringUtils.trimToEmpty(assetRows.getString("canonical_name")))
                    .displayName(StringUtils.trimToEmpty(assetRows.getString("display_name")))
                    .intro(StringUtils.trimToEmpty(assetRows.getString("intro")))
                    .tags(readStringList(assetRows.getString("tags_json")))
                    .capabilities(ModelAssetCapabilities.builder()
                        .modalities(readStringList(assetRows.getString("modalities_json")))
                        .features(readStringList(assetRows.getString("features_json")))
                        .requestKinds(readStringList(assetRows.getString("request_kinds_json")))
                        .build())
                    .createdAt(formatTimestamp(assetRows.getTimestamp("created_at")))
                    .updatedAt(formatTimestamp(assetRows.getTimestamp("updated_at")))
                    .bindings(new ArrayList<>())
                    .build());
            }
            if (assetMap.isEmpty()) {
                return Collections.emptyList();
            }

            try (PreparedStatement bindingStatement = connection.prepareStatement(
                "SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, rpm, tpm, context_window, status, "
                    + "published_at, unpublished_at, created_at, updated_at "
                    + "FROM portal_model_binding ORDER BY asset_id ASC, created_at ASC, binding_id ASC");
                ResultSet bindingRows = bindingStatement.executeQuery()) {
                while (bindingRows.next()) {
                    String assetId = bindingRows.getString("asset_id");
                    ModelAssetRecord asset = assetMap.get(assetId);
                    if (asset == null) {
                        continue;
                    }
                    asset.getBindings().add(toBindingRecord(bindingRows));
                }
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to query model assets.", ex);
        }

        List<ModelAssetRecord> result = new ArrayList<>(assetMap.values());
        result.sort(Comparator.comparing(ModelAssetRecord::getCanonicalName).thenComparing(ModelAssetRecord::getAssetId));
        return result;
    }

    private ModelAssetRecord requireAsset(String assetId) {
        return listAssetsInternal().stream()
            .filter(item -> StringUtils.equals(item.getAssetId(), assetId))
            .findFirst()
            .orElseThrow(() -> new ValidationException("Model asset not found: " + assetId));
    }

    private ModelAssetBindingRecord requireBinding(String assetId, String bindingId) {
        return requireAsset(assetId).getBindings().stream()
            .filter(item -> StringUtils.equals(item.getBindingId(), bindingId))
            .findFirst()
            .orElseThrow(() -> new ValidationException("Model binding not found: " + bindingId));
    }

    private BindingRow requireBindingRow(String assetId, String bindingId) {
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(
                "SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, rpm, tpm, context_window, status, "
                    + "published_at, unpublished_at "
                    + "FROM portal_model_binding WHERE asset_id = ? AND binding_id = ? LIMIT 1")) {
            statement.setString(1, assetId);
            statement.setString(2, bindingId);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    return BindingRow.builder()
                        .bindingId(StringUtils.trimToEmpty(rs.getString("binding_id")))
                        .assetId(StringUtils.trimToEmpty(rs.getString("asset_id")))
                        .modelId(StringUtils.trimToEmpty(rs.getString("model_id")))
                        .providerName(StringUtils.trimToEmpty(rs.getString("provider_name")))
                        .targetModel(StringUtils.trimToEmpty(rs.getString("target_model")))
                        .protocol(StringUtils.trimToEmpty(rs.getString("protocol")))
                        .endpoint(StringUtils.trimToEmpty(rs.getString("endpoint")))
                        .pricing(readPricing(rs.getString("pricing_json")))
                        .limits(ModelBindingLimits.builder()
                            .rpm(readNullableLong(rs, "rpm"))
                            .tpm(readNullableLong(rs, "tpm"))
                            .contextWindow(readNullableLong(rs, "context_window"))
                            .build())
                        .status(StringUtils.trimToEmpty(rs.getString("status")))
                        .publishedAt(formatTimestamp(rs.getTimestamp("published_at")))
                        .unpublishedAt(formatTimestamp(rs.getTimestamp("unpublished_at")))
                        .build();
                }
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to query model binding.", ex);
        }
        throw new ValidationException("Model binding not found: " + bindingId);
    }

    private AssetMutation normalizeAssetRequest(ModelAssetRecord request) {
        if (request == null) {
            throw new ValidationException("model asset request cannot be null.");
        }
        String canonicalName = requireNonBlank(request.getCanonicalName(), "canonicalName cannot be blank.");
        String displayName = requireNonBlank(request.getDisplayName(), "displayName cannot be blank.");
        String assetId = StringUtils.trimToNull(request.getAssetId());
        if (assetId == null) {
            assetId = toIdentifier(canonicalName);
        }
        if (assetId == null) {
            throw new ValidationException("assetId cannot be blank.");
        }
        ModelAssetCapabilities capabilities = request.getCapabilities() == null
            ? ModelAssetCapabilities.builder().build()
            : request.getCapabilities();
        List<String> normalizedTags = normalizeStringList(request.getTags());
        for (String tag : normalizedTags) {
            if (!PRESET_TAGS.contains(tag)) {
                throw new ValidationException("tags must use the predefined model asset tag catalog.");
            }
        }
        List<String> normalizedModalities = normalizeStringList(capabilities.getModalities());
        validatePresetValues(normalizedModalities, PRESET_MODALITY_SET, "capabilities.modalities");
        List<String> normalizedFeatures = normalizeStringList(capabilities.getFeatures());
        validatePresetValues(normalizedFeatures, PRESET_FEATURE_SET, "capabilities.features");
        List<String> normalizedRequestKinds = normalizeStringList(capabilities.getRequestKinds());
        validatePresetValues(normalizedRequestKinds, PRESET_REQUEST_KIND_SET, "capabilities.requestKinds");
        return new AssetMutation(
            assetId,
            canonicalName,
            displayName,
            StringUtils.defaultString(StringUtils.trimToNull(request.getIntro())),
            normalizedTags,
            normalizedModalities,
            normalizedFeatures,
            normalizedRequestKinds);
    }

    private BindingMutation normalizeBindingRequest(ModelAssetBindingRecord request) {
        if (request == null) {
            throw new ValidationException("model binding request cannot be null.");
        }
        String bindingId = StringUtils.trimToNull(request.getBindingId());
        if (bindingId == null) {
            bindingId = toIdentifier(StringUtils.defaultIfBlank(request.getModelId(), request.getTargetModel()));
        }
        BindingMutation normalized = new BindingMutation(
            requireNonBlank(bindingId, "bindingId cannot be blank."),
            requireNonBlank(request.getModelId(), "modelId cannot be blank."),
            requireNonBlank(request.getProviderName(), "providerName cannot be blank."),
            requireNonBlank(request.getTargetModel(), "targetModel cannot be blank."),
            StringUtils.defaultIfBlank(StringUtils.trimToNull(request.getProtocol()), DEFAULT_PROTOCOL),
            StringUtils.defaultIfBlank(StringUtils.trimToNull(request.getEndpoint()), DEFAULT_ENDPOINT),
            normalizePricing(request.getPricing()),
            normalizeLimits(request.getLimits()));
        validateBindingMutation(normalized);
        return normalized;
    }

    private BindingMutation toMutation(BindingRow row) {
        return new BindingMutation(
            row.bindingId,
            row.modelId,
            row.providerName,
            row.targetModel,
            StringUtils.defaultIfBlank(StringUtils.trimToNull(row.protocol), DEFAULT_PROTOCOL),
            StringUtils.defaultIfBlank(StringUtils.trimToNull(row.endpoint), DEFAULT_ENDPOINT),
            normalizePricing(row.pricing),
            normalizeLimits(row.limits));
    }

    private void validatePresetValues(List<String> values, Set<String> allowedValues, String fieldName) {
        for (String value : values) {
            if (!allowedValues.contains(value)) {
                throw new ValidationException(fieldName + " must use the predefined options.");
            }
        }
    }

    private void validateBindingMutation(BindingMutation mutation) {
        requireNonBlank(mutation.modelId, "modelId cannot be blank.");
        requireNonBlank(mutation.providerName, "providerName cannot be blank.");
        requireNonBlank(mutation.targetModel, "targetModel cannot be blank.");
        if (!StringUtils.equalsIgnoreCase(StringUtils.defaultIfBlank(mutation.pricing.getCurrency(), CURRENCY_CNY),
            CURRENCY_CNY)) {
            throw new ValidationException("pricing.currency must be CNY.");
        }
        requirePricing(mutation.pricing.getInputCostPerToken(), "pricing.inputCostPerToken");
        requirePricing(mutation.pricing.getOutputCostPerToken(), "pricing.outputCostPerToken");
    }

    private ModelBindingPricing normalizePricing(ModelBindingPricing pricing) {
        if (pricing == null) {
            throw new ValidationException("pricing cannot be null.");
        }
        ModelBindingPricing normalized = ModelBindingPricing.builder()
            .currency(CURRENCY_CNY)
            .inputCostPerToken(requireNonNegative(pricing.getInputCostPerToken(), "pricing.inputCostPerToken"))
            .outputCostPerToken(requireNonNegative(pricing.getOutputCostPerToken(), "pricing.outputCostPerToken"))
            .inputCostPerRequest(optionalNonNegative(pricing.getInputCostPerRequest(), "pricing.inputCostPerRequest"))
            .cacheCreationInputTokenCost(optionalNonNegative(pricing.getCacheCreationInputTokenCost(), "pricing.cacheCreationInputTokenCost"))
            .cacheCreationInputTokenCostAbove1hr(optionalNonNegative(pricing.getCacheCreationInputTokenCostAbove1hr(),
                "pricing.cacheCreationInputTokenCostAbove1hr"))
            .cacheReadInputTokenCost(optionalNonNegative(pricing.getCacheReadInputTokenCost(), "pricing.cacheReadInputTokenCost"))
            .inputCostPerTokenAbove200kTokens(optionalNonNegative(pricing.getInputCostPerTokenAbove200kTokens(),
                "pricing.inputCostPerTokenAbove200kTokens"))
            .outputCostPerTokenAbove200kTokens(optionalNonNegative(pricing.getOutputCostPerTokenAbove200kTokens(),
                "pricing.outputCostPerTokenAbove200kTokens"))
            .cacheCreationInputTokenCostAbove200kTokens(optionalNonNegative(pricing.getCacheCreationInputTokenCostAbove200kTokens(),
                "pricing.cacheCreationInputTokenCostAbove200kTokens"))
            .cacheReadInputTokenCostAbove200kTokens(optionalNonNegative(pricing.getCacheReadInputTokenCostAbove200kTokens(),
                "pricing.cacheReadInputTokenCostAbove200kTokens"))
            .outputCostPerImage(optionalNonNegative(pricing.getOutputCostPerImage(), "pricing.outputCostPerImage"))
            .outputCostPerImageToken(optionalNonNegative(pricing.getOutputCostPerImageToken(), "pricing.outputCostPerImageToken"))
            .inputCostPerImage(optionalNonNegative(pricing.getInputCostPerImage(), "pricing.inputCostPerImage"))
            .inputCostPerImageToken(optionalNonNegative(pricing.getInputCostPerImageToken(), "pricing.inputCostPerImageToken"))
            .supportsPromptCaching(Boolean.TRUE.equals(pricing.getSupportsPromptCaching()))
            .build();
        return normalized;
    }

    private ModelBindingLimits normalizeLimits(ModelBindingLimits limits) {
        if (limits == null) {
            return ModelBindingLimits.builder().build();
        }
        return ModelBindingLimits.builder()
            .rpm(optionalNonNegativeLong(limits.getRpm(), "limits.rpm"))
            .tpm(optionalNonNegativeLong(limits.getTpm(), "limits.tpm"))
            .contextWindow(optionalNonNegativeLong(limits.getContextWindow(), "limits.contextWindow"))
            .build();
    }

    private PublishedBindingMeta buildPublishedBindingMeta(ModelAssetRecord asset, BindingMutation binding, LlmProvider provider) {
        String vendor = StringUtils.defaultIfBlank(StringUtils.trimToNull(provider.getName()),
            StringUtils.defaultIfBlank(StringUtils.trimToNull(provider.getType()), "unknown"));
        String endpoint = resolveEndpoint(provider.getRawConfigs(), binding.endpoint);
        String protocol = StringUtils.defaultIfBlank(StringUtils.trimToNull(binding.protocol),
            StringUtils.defaultIfBlank(StringUtils.trimToNull(provider.getProtocol()), DEFAULT_PROTOCOL));
        String capability = buildCapabilitySummary(asset);
        String summary = StringUtils.defaultIfBlank(StringUtils.trimToNull(asset.getIntro()), capability);
        ModelBindingPricing pricing = binding.pricing;
        return PublishedBindingMeta.builder()
            .modelId(binding.modelId)
            .name(StringUtils.defaultIfBlank(StringUtils.trimToNull(asset.getDisplayName()), binding.modelId))
            .vendor(vendor)
            .capability(capability)
            .endpoint(endpoint)
            .sdk(protocol)
            .summary(summary)
            .currency(CURRENCY_CNY)
            .inputPricePer1KMicroYuan(toPer1KMicroYuan(pricing.getInputCostPerToken()))
            .outputPricePer1KMicroYuan(toPer1KMicroYuan(pricing.getOutputCostPerToken()))
            .inputRequestPriceMicroYuan(toMicroYuan(defaultDouble(pricing.getInputCostPerRequest())))
            .cacheCreationInputTokenPricePer1KMicroYuan(toPer1KMicroYuan(defaultDouble(pricing.getCacheCreationInputTokenCost())))
            .cacheCreationInputTokenPriceAbove1hrPer1KMicroYuan(
                toPer1KMicroYuan(defaultDouble(pricing.getCacheCreationInputTokenCostAbove1hr())))
            .cacheReadInputTokenPricePer1KMicroYuan(toPer1KMicroYuan(defaultDouble(pricing.getCacheReadInputTokenCost())))
            .inputTokenPriceAbove200kPer1KMicroYuan(toPer1KMicroYuan(defaultDouble(pricing.getInputCostPerTokenAbove200kTokens())))
            .outputTokenPriceAbove200kPer1KMicroYuan(toPer1KMicroYuan(defaultDouble(pricing.getOutputCostPerTokenAbove200kTokens())))
            .cacheCreationInputTokenPriceAbove200kPer1KMicroYuan(
                toPer1KMicroYuan(defaultDouble(pricing.getCacheCreationInputTokenCostAbove200kTokens())))
            .cacheReadInputTokenPriceAbove200kPer1KMicroYuan(
                toPer1KMicroYuan(defaultDouble(pricing.getCacheReadInputTokenCostAbove200kTokens())))
            .outputImagePriceMicroYuan(toMicroYuan(defaultDouble(pricing.getOutputCostPerImage())))
            .outputImageTokenPricePer1KMicroYuan(toPer1KMicroYuan(defaultDouble(pricing.getOutputCostPerImageToken())))
            .inputImagePriceMicroYuan(toMicroYuan(defaultDouble(pricing.getInputCostPerImage())))
            .inputImageTokenPricePer1KMicroYuan(toPer1KMicroYuan(defaultDouble(pricing.getInputCostPerImageToken())))
            .supportsPromptCaching(Boolean.TRUE.equals(pricing.getSupportsPromptCaching()))
            .build();
    }

    private void ensureModelAssetTables() {
        if (!enabled()) {
            return;
        }
        String[] ddls = new String[] {
            "CREATE TABLE IF NOT EXISTS portal_model_asset ("
                + "id BIGINT AUTO_INCREMENT PRIMARY KEY,"
                + "asset_id VARCHAR(128) NOT NULL UNIQUE,"
                + "canonical_name VARCHAR(128) NOT NULL UNIQUE,"
                + "display_name VARCHAR(128) NOT NULL,"
                + "intro TEXT NOT NULL,"
                + "tags_json TEXT NULL,"
                + "modalities_json TEXT NULL,"
                + "features_json TEXT NULL,"
                + "request_kinds_json TEXT NULL,"
                + "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                + "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
                + "INDEX idx_model_asset_display_name (display_name)"
                + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
            "CREATE TABLE IF NOT EXISTS portal_model_binding ("
                + "id BIGINT AUTO_INCREMENT PRIMARY KEY,"
                + "binding_id VARCHAR(128) NOT NULL UNIQUE,"
                + "asset_id VARCHAR(128) NOT NULL,"
                + "model_id VARCHAR(128) NOT NULL UNIQUE,"
                + "provider_name VARCHAR(128) NOT NULL,"
                + "target_model VARCHAR(128) NOT NULL,"
                + "protocol VARCHAR(128) NOT NULL DEFAULT 'openai/v1',"
                + "endpoint VARCHAR(255) NOT NULL DEFAULT '-',"
                + "pricing_json TEXT NOT NULL,"
                + "rpm BIGINT NULL,"
                + "tpm BIGINT NULL,"
                + "context_window BIGINT NULL,"
                + "status VARCHAR(16) NOT NULL DEFAULT 'draft',"
                + "published_at DATETIME NULL,"
                + "unpublished_at DATETIME NULL,"
                + "created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,"
                + "updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,"
                + "UNIQUE KEY uk_model_binding_target (asset_id, provider_name, target_model),"
                + "INDEX idx_model_binding_asset (asset_id),"
                + "INDEX idx_model_binding_status (status),"
                + "INDEX idx_model_binding_provider (provider_name)"
                + ") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
        };
        try (Connection connection = openConnection()) {
            for (String ddl : ddls) {
                try (PreparedStatement statement = connection.prepareStatement(ddl)) {
                    statement.execute();
                }
            }
            try (PreparedStatement statement = connection.prepareStatement(
                "ALTER TABLE portal_model_asset ADD COLUMN request_kinds_json TEXT NULL AFTER features_json")) {
                statement.execute();
            } catch (SQLException ex) {
                if (!StringUtils.containsIgnoreCase(ex.getMessage(), "duplicate column")) {
                    throw ex;
                }
            }
        } catch (SQLException ex) {
            log.warn("Failed to ensure model asset tables.", ex);
        }
    }

    private void backfillLegacyProviders() {
        if (!enabled() || llmProviderService == null) {
            return;
        }
        try {
            PaginatedResult<LlmProvider> paginatedResult = llmProviderService.list(null);
            List<LlmProvider> providers = paginatedResult == null ? Collections.emptyList() : paginatedResult.getData();
            if (providers == null || providers.isEmpty()) {
                return;
            }
            for (LlmProvider provider : providers) {
                if (!supportsLegacyPortalMeta(provider)) {
                    continue;
                }
                backfillLegacyProvider(provider);
            }
        } catch (Exception ex) {
            log.warn("Failed to backfill legacy provider-backed models into explicit model assets.", ex);
        }
    }

    private boolean supportsLegacyPortalMeta(LlmProvider provider) {
        if (provider == null || provider.getRawConfigs() == null) {
            return false;
        }
        Object meta = provider.getRawConfigs().get(PORTAL_MODEL_META_KEY);
        if (!(meta instanceof Map)) {
            return false;
        }
        @SuppressWarnings("unchecked")
        Map<String, Object> portalModelMeta = (Map<String, Object>)meta;
        return portalModelMeta.get("pricing") instanceof Map;
    }

    private void backfillLegacyProvider(LlmProvider provider) {
        String providerName = StringUtils.trimToNull(provider.getName());
        if (providerName == null || hasBindingsForProvider(providerName)) {
            return;
        }
        @SuppressWarnings("unchecked")
        Map<String, Object> portalModelMeta = (Map<String, Object>)provider.getRawConfigs().get(PORTAL_MODEL_META_KEY);
        AssetMutation asset = new AssetMutation(
            toIdentifier(providerName),
            providerName,
            providerName,
            StringUtils.defaultString(asString(portalModelMeta.get("intro"))),
            normalizeStringList(readStringListValue(portalModelMeta.get("tags"))),
            normalizeStringList(readStringListValue(readNestedValue(portalModelMeta, "capabilities", "modalities"))),
            normalizeStringList(readStringListValue(readNestedValue(portalModelMeta, "capabilities", "features"))),
            normalizeStringList(readStringListValue(readNestedValue(portalModelMeta, "capabilities", "requestKinds"))));
        BindingMutation binding = new BindingMutation(
            toIdentifier(providerName + "-binding"),
            providerName,
            providerName,
            providerName,
            StringUtils.defaultIfBlank(StringUtils.trimToNull(provider.getProtocol()), DEFAULT_PROTOCOL),
            resolveEndpoint(provider.getRawConfigs(), DEFAULT_ENDPOINT),
            normalizePricing(fromLegacyPricing(readNestedMapValue(portalModelMeta, "pricing"))),
            normalizeLimits(fromLegacyLimits(readNestedMapValue(portalModelMeta, "limits"))));

        try (Connection connection = openConnection()) {
            connection.setAutoCommit(false);
            try {
                if (!assetExists(connection, asset.assetId)) {
                    try (PreparedStatement statement = connection.prepareStatement(
                        "INSERT INTO portal_model_asset "
                            + "(asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json) "
                            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?)")) {
                        statement.setString(1, asset.assetId);
                        statement.setString(2, asset.canonicalName);
                        statement.setString(3, asset.displayName);
                        statement.setString(4, asset.intro);
                        statement.setString(5, writeJson(asset.tags));
                        statement.setString(6, writeJson(asset.modalities));
                        statement.setString(7, writeJson(asset.features));
                        statement.setString(8, writeJson(asset.requestKinds));
                        statement.executeUpdate();
                    }
                }
                if (!bindingExists(connection, binding.bindingId)) {
                    try (PreparedStatement statement = connection.prepareStatement(
                        "INSERT INTO portal_model_binding "
                            + "(binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, rpm, tpm, context_window, status, published_at) "
                            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")) {
                        statement.setString(1, binding.bindingId);
                        statement.setString(2, asset.assetId);
                        statement.setString(3, binding.modelId);
                        statement.setString(4, binding.providerName);
                        statement.setString(5, binding.targetModel);
                        statement.setString(6, binding.protocol);
                        statement.setString(7, binding.endpoint);
                        statement.setString(8, writeJson(binding.pricing));
                        statement.setLong(9, binding.rpm);
                        statement.setLong(10, binding.tpm);
                        statement.setLong(11, binding.contextWindow);
                        statement.setString(12, STATUS_PUBLISHED);
                        statement.setTimestamp(13, ConsoleDateTimeUtil.nowTimestamp());
                        statement.executeUpdate();
                    }
                }
                PublishedBindingMeta meta = buildPublishedBindingMeta(requireAsset(asset.assetId), binding, provider);
                upsertCatalog(connection, meta);
                upsertPriceVersion(connection, meta);
                connection.commit();
            } catch (Exception ex) {
                connection.rollback();
                throw ex;
            } finally {
                connection.setAutoCommit(true);
            }
        } catch (Exception ex) {
            log.warn("Failed to backfill legacy provider {} into explicit model assets.", providerName, ex);
        }
    }

    private boolean assetExists(String assetId) {
        try (Connection connection = openConnection()) {
            return assetExists(connection, assetId);
        } catch (SQLException ex) {
            throw new BusinessException("Failed to check model asset existence.", ex);
        }
    }

    private boolean assetExists(Connection connection, String assetId) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(
            "SELECT 1 FROM portal_model_asset WHERE asset_id = ? LIMIT 1")) {
            statement.setString(1, assetId);
            try (ResultSet rs = statement.executeQuery()) {
                return rs.next();
            }
        }
    }

    private boolean bindingExists(Connection connection, String bindingId) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement(
            "SELECT 1 FROM portal_model_binding WHERE binding_id = ? LIMIT 1")) {
            statement.setString(1, bindingId);
            try (ResultSet rs = statement.executeQuery()) {
                return rs.next();
            }
        }
    }

    private void ensureProviderExists(String providerName) {
        requireProvider(providerName);
    }

    private List<ProviderModelCatalogRecord> listProviderModelCatalogs() {
        if (llmProviderService == null) {
            return Collections.emptyList();
        }
        PaginatedResult<LlmProvider> providers = llmProviderService.list(null);
        if (providers == null || providers.getData() == null) {
            return Collections.emptyList();
        }
        List<ProviderModelCatalogRecord> result = new ArrayList<>();
        for (LlmProvider provider : providers.getData()) {
            String providerName = StringUtils.trimToNull(provider.getName());
            if (providerName == null) {
                continue;
            }
            List<ProviderModelOptionRecord> models = lookupProviderModelCatalog(providerName);
            if (models.isEmpty()) {
                continue;
            }
            result.add(ProviderModelCatalogRecord.builder()
                .providerName(providerName)
                .models(models)
                .build());
        }
        result.sort(Comparator.comparing(ProviderModelCatalogRecord::getProviderName, String.CASE_INSENSITIVE_ORDER));
        return result;
    }

    private List<ProviderModelOptionRecord> lookupProviderModelCatalog(String providerName) {
        String normalizedProviderName = normalizeProviderCatalogKey(providerName);
        if (normalizedProviderName == null) {
            return Collections.emptyList();
        }
        for (Map.Entry<String, List<ProviderModelOptionRecord>> entry : PROVIDER_MODEL_CATALOG.entrySet()) {
            String catalogKey = entry.getKey();
            if (StringUtils.equals(normalizedProviderName, catalogKey)
                || StringUtils.contains(normalizedProviderName, catalogKey)
                || StringUtils.contains(catalogKey, normalizedProviderName)) {
                return entry.getValue();
            }
        }
        return Collections.emptyList();
    }

    private String normalizeProviderCatalogKey(String providerName) {
        String normalized = StringUtils.trimToNull(providerName);
        if (normalized == null) {
            return null;
        }
        return normalized.toLowerCase(Locale.ROOT).replace('_', '-').replace(' ', '-');
    }

    private static Map<String, List<ProviderModelOptionRecord>> createProviderModelCatalog() {
        Map<String, List<ProviderModelOptionRecord>> result = new LinkedHashMap<>();
        result.put("openai", buildProviderModels(
            "gpt-3",
            "gpt-35-turbo",
            "gpt-4",
            "gpt-4o",
            "gpt-4o-mini"));
        result.put("qwen", buildProviderModels(
            "qwen-max",
            "qwen-plus",
            "qwen-turbo",
            "qwen-long"));
        result.put("moonshot", buildProviderModels(
            "moonshot-v1-8k",
            "moonshot-v1-32k",
            "moonshot-v1-128k"));
        result.put("azure", buildProviderModels(
            "gpt-3",
            "gpt-35-turbo",
            "gpt-4",
            "gpt-4o",
            "gpt-4o-mini"));
        result.put("claude", buildProviderModels(
            "claude-opus-4-1",
            "claude-opus-4-0",
            "claude-sonnet-4-0",
            "claude-3-7-sonnet-latest",
            "claude-3-5-haiku-latest"));
        result.put("baichuan", buildProviderModels(
            "Baichuan4-Turbo",
            "Baichuan4-Air",
            "Baichuan4",
            "Baichuan3-Turbo",
            "Baichuan3-Turbo-128k",
            "Baichuan2-Turbo"));
        result.put("yi", buildProviderModels(
            "yi-lightning",
            "yi-large",
            "yi-medium",
            "yi-medium-200k",
            "yi-spark",
            "yi-large-rag",
            "yi-large-fc",
            "yi-large-turbo"));
        result.put("zhipuai", buildProviderModels(
            "GLM-4-Plus",
            "GLM-4-0520",
            "GLM-4-Long",
            "GLM-4-AirX",
            "GLM-4-Air",
            "GLM-4-FlashX",
            "GLM-4-Flash",
            "GLM-4-AllTools",
            "GLM-4"));
        result.put("baidu", buildProviderModels(
            "ERNIE-4.0-8K",
            "ERNIE-4.0-8K-Latest",
            "ERNIE-4.0-Turbo-8K",
            "ERNIE-3.5-8K",
            "ERNIE-3.5-128K"));
        result.put("hunyuan", buildProviderModels(
            "hunyuan-turbo-latest",
            "hunyuan-turbo",
            "hunyuan-large",
            "hunyuan-pro",
            "hunyuan-standard-256K",
            "hunyuan-standard",
            "hunyuan-lite"));
        result.put("stepfun", buildProviderModels(
            "step-1-8k",
            "step-1-32k",
            "step-1-128k",
            "step-1-256k",
            "step-2-16k",
            "step-1-flash"));
        result.put("spark", buildProviderModels(
            "lite",
            "generalv3",
            "pro-128k",
            "generalv3.5",
            "max-32k",
            "4.0Ultra"));
        result.put("doubao", buildProviderModels(
            "doubao-pro-32k",
            "doubao-pro-128k",
            "doubao-lite-32k"));
        result.put("minimax", buildProviderModels(
            "abab6.5s",
            "abab6.5g",
            "abab6.5t",
            "abab5.5s"));
        result.put("gemini", buildProviderModels(
            "gemini-1.5-flash",
            "gemini-1.5-pro"));
        result.put("openrouter", buildProviderModels(
            "anthropic/claude-sonnet-4",
            "google/gemini-2.5-flash",
            "google/gemini-2.0-flash-001",
            "deepseek/deepseek-chat-v3.1",
            "deepseek/deepseek-chat-v3-0324",
            "google/gemini-2.5-pro",
            "qwen/qwen3-coder",
            "anthropic/claude-3.7-sonnet",
            "x-ai/grok-code-fast-1",
            "x-ai/grok-4",
            "deepseek/deepseek-r1-0528:free",
            "google/gemini-2.5-flash-lite",
            "openai/gpt-5"));
        result.put("grok", buildProviderModels(
            "grok-code-fast-1",
            "grok-4-0709",
            "grok-3",
            "grok-3-mini",
            "grok-2-image-1212"));
        return Collections.unmodifiableMap(result);
    }

    private static List<ProviderModelOptionRecord> buildProviderModels(String... modelNames) {
        List<ProviderModelOptionRecord> result = new ArrayList<>();
        for (String modelName : modelNames) {
            result.add(ProviderModelOptionRecord.builder()
                .modelId(modelName)
                .targetModel(modelName)
                .label(modelName)
                .build());
        }
        return Collections.unmodifiableList(result);
    }

    private LlmProvider requireProvider(String providerName) {
        LlmProvider provider = llmProviderService == null ? null : llmProviderService.query(providerName);
        if (provider == null) {
            throw new ValidationException("Provider not found: " + providerName);
        }
        return provider;
    }

    private String buildCapabilitySummary(ModelAssetRecord asset) {
        List<String> parts = new ArrayList<>();
        if (asset.getCapabilities() != null) {
            parts.addAll(normalizeStringList(asset.getCapabilities().getModalities()));
            parts.addAll(normalizeStringList(asset.getCapabilities().getFeatures()));
        }
        String combined = StringUtils.join(parts, " / ");
        if (StringUtils.isNotBlank(combined)) {
            return combined;
        }
        if (StringUtils.isNotBlank(asset.getIntro())) {
            return asset.getIntro();
        }
        return asset.getDisplayName();
    }

    private String resolveEndpoint(Map<String, Object> rawConfigs, String fallback) {
        if (rawConfigs == null || rawConfigs.isEmpty()) {
            return fallback;
        }
        String[] candidateKeys = new String[] {
            "openaiCustomUrl",
            "azureServiceUrl",
            "qwenDomain",
            "zhipuDomain",
            "ollamaServerHost",
        };
        for (String key : candidateKeys) {
            String value = StringUtils.trimToNull(asString(rawConfigs.get(key)));
            if (value != null) {
                return value;
            }
        }
        return fallback;
    }

    private void upsertCatalog(Connection connection, PublishedBindingMeta meta) throws SQLException {
        String sql = "INSERT INTO billing_model_catalog "
            + "(model_id, name, vendor, capability, endpoint, sdk, summary, status) "
            + "VALUES (?, ?, ?, ?, ?, ?, ?, ?) "
            + "ON DUPLICATE KEY UPDATE "
            + "name = VALUES(name), "
            + "vendor = VALUES(vendor), "
            + "capability = VALUES(capability), "
            + "endpoint = VALUES(endpoint), "
            + "sdk = VALUES(sdk), "
            + "summary = VALUES(summary), "
            + "status = VALUES(status)";
        try (PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setString(1, meta.modelId);
            statement.setString(2, meta.name);
            statement.setString(3, meta.vendor);
            statement.setString(4, meta.capability);
            statement.setString(5, meta.endpoint);
            statement.setString(6, meta.sdk);
            statement.setString(7, meta.summary);
            statement.setString(8, BILLING_STATUS_ACTIVE);
            statement.executeUpdate();
        }
    }

    private void upsertPriceVersion(Connection connection, PublishedBindingMeta meta) throws SQLException {
        String selectSql = "SELECT id, currency, input_price_per_1k_micro_yuan, output_price_per_1k_micro_yuan, "
            + "input_request_price_micro_yuan, cache_creation_input_token_price_per_1k_micro_yuan, "
            + "cache_creation_input_token_price_above_1hr_per_1k_micro_yuan, cache_read_input_token_price_per_1k_micro_yuan, "
            + "input_token_price_above_200k_per_1k_micro_yuan, output_token_price_above_200k_per_1k_micro_yuan, "
            + "cache_creation_input_token_price_above_200k_per_1k_micro_yuan, cache_read_input_token_price_above_200k_per_1k_micro_yuan, "
            + "output_image_price_micro_yuan, output_image_token_price_per_1k_micro_yuan, "
            + "input_image_price_micro_yuan, input_image_token_price_per_1k_micro_yuan, supports_prompt_caching "
            + "FROM billing_model_price_version WHERE model_id = ? AND effective_to IS NULL "
            + "ORDER BY id DESC LIMIT 1";

        PriceVersionState current = null;
        try (PreparedStatement statement = connection.prepareStatement(selectSql)) {
            statement.setString(1, meta.modelId);
            try (ResultSet rs = statement.executeQuery()) {
                if (rs.next()) {
                    current = new PriceVersionState(
                        rs.getLong("id"),
                        rs.getString("currency"),
                        rs.getLong("input_price_per_1k_micro_yuan"),
                        rs.getLong("output_price_per_1k_micro_yuan"),
                        rs.getLong("input_request_price_micro_yuan"),
                        rs.getLong("cache_creation_input_token_price_per_1k_micro_yuan"),
                        rs.getLong("cache_creation_input_token_price_above_1hr_per_1k_micro_yuan"),
                        rs.getLong("cache_read_input_token_price_per_1k_micro_yuan"),
                        rs.getLong("input_token_price_above_200k_per_1k_micro_yuan"),
                        rs.getLong("output_token_price_above_200k_per_1k_micro_yuan"),
                        rs.getLong("cache_creation_input_token_price_above_200k_per_1k_micro_yuan"),
                        rs.getLong("cache_read_input_token_price_above_200k_per_1k_micro_yuan"),
                        rs.getLong("output_image_price_micro_yuan"),
                        rs.getLong("output_image_token_price_per_1k_micro_yuan"),
                        rs.getLong("input_image_price_micro_yuan"),
                        rs.getLong("input_image_token_price_per_1k_micro_yuan"),
                        rs.getInt("supports_prompt_caching") > 0);
                }
            }
        }

        if (current != null && meta.matches(current)) {
            try (PreparedStatement statement = connection.prepareStatement(
                "UPDATE billing_model_price_version SET status = ?, effective_to = NULL WHERE id = ?")) {
                statement.setString(1, BILLING_STATUS_ACTIVE);
                statement.setLong(2, current.id);
                statement.executeUpdate();
            }
            return;
        }

        LocalDateTime now = ConsoleDateTimeUtil.now();
        try (PreparedStatement deactivate = connection.prepareStatement(
            "UPDATE billing_model_price_version SET status = ?, effective_to = ? WHERE model_id = ? AND effective_to IS NULL");
            PreparedStatement insert = connection.prepareStatement(
                "INSERT INTO billing_model_price_version "
                    + "(model_id, currency, input_price_per_1k_micro_yuan, output_price_per_1k_micro_yuan, "
                    + "input_request_price_micro_yuan, cache_creation_input_token_price_per_1k_micro_yuan, "
                    + "cache_creation_input_token_price_above_1hr_per_1k_micro_yuan, cache_read_input_token_price_per_1k_micro_yuan, "
                    + "input_token_price_above_200k_per_1k_micro_yuan, output_token_price_above_200k_per_1k_micro_yuan, "
                    + "cache_creation_input_token_price_above_200k_per_1k_micro_yuan, cache_read_input_token_price_above_200k_per_1k_micro_yuan, "
                    + "output_image_price_micro_yuan, output_image_token_price_per_1k_micro_yuan, "
                    + "input_image_price_micro_yuan, input_image_token_price_per_1k_micro_yuan, supports_prompt_caching, "
                    + "effective_from, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")) {
            deactivate.setString(1, BILLING_STATUS_INACTIVE);
            deactivate.setTimestamp(2, ConsoleDateTimeUtil.toTimestamp(now));
            deactivate.setString(3, meta.modelId);
            deactivate.executeUpdate();

            insert.setString(1, meta.modelId);
            insert.setString(2, meta.currency);
            insert.setLong(3, meta.inputPricePer1KMicroYuan);
            insert.setLong(4, meta.outputPricePer1KMicroYuan);
            insert.setLong(5, meta.inputRequestPriceMicroYuan);
            insert.setLong(6, meta.cacheCreationInputTokenPricePer1KMicroYuan);
            insert.setLong(7, meta.cacheCreationInputTokenPriceAbove1hrPer1KMicroYuan);
            insert.setLong(8, meta.cacheReadInputTokenPricePer1KMicroYuan);
            insert.setLong(9, meta.inputTokenPriceAbove200kPer1KMicroYuan);
            insert.setLong(10, meta.outputTokenPriceAbove200kPer1KMicroYuan);
            insert.setLong(11, meta.cacheCreationInputTokenPriceAbove200kPer1KMicroYuan);
            insert.setLong(12, meta.cacheReadInputTokenPriceAbove200kPer1KMicroYuan);
            insert.setLong(13, meta.outputImagePriceMicroYuan);
            insert.setLong(14, meta.outputImageTokenPricePer1KMicroYuan);
            insert.setLong(15, meta.inputImagePriceMicroYuan);
            insert.setLong(16, meta.inputImageTokenPricePer1KMicroYuan);
            insert.setBoolean(17, meta.supportsPromptCaching);
            insert.setTimestamp(18, ConsoleDateTimeUtil.toTimestamp(now));
            insert.setString(19, BILLING_STATUS_ACTIVE);
            insert.executeUpdate();
        }
    }

    private void disablePublishedModel(Connection connection, String modelId) throws SQLException {
        try (PreparedStatement catalogStmt = connection.prepareStatement(
            "UPDATE billing_model_catalog SET status = ? WHERE model_id = ?");
            PreparedStatement versionStmt = connection.prepareStatement(
                "UPDATE billing_model_price_version SET status = ?, effective_to = ? WHERE model_id = ? AND effective_to IS NULL")) {
            catalogStmt.setString(1, BILLING_STATUS_DISABLED);
            catalogStmt.setString(2, modelId);
            catalogStmt.executeUpdate();

            versionStmt.setString(1, BILLING_STATUS_INACTIVE);
            versionStmt.setTimestamp(2, ConsoleDateTimeUtil.nowTimestamp());
            versionStmt.setString(3, modelId);
            versionStmt.executeUpdate();
        }
    }

    private ModelAssetBindingRecord toBindingRecord(ResultSet rs) throws SQLException {
        return ModelAssetBindingRecord.builder()
            .bindingId(StringUtils.trimToEmpty(rs.getString("binding_id")))
            .assetId(StringUtils.trimToEmpty(rs.getString("asset_id")))
            .modelId(StringUtils.trimToEmpty(rs.getString("model_id")))
            .providerName(StringUtils.trimToEmpty(rs.getString("provider_name")))
            .targetModel(StringUtils.trimToEmpty(rs.getString("target_model")))
            .protocol(StringUtils.trimToEmpty(rs.getString("protocol")))
            .endpoint(StringUtils.trimToEmpty(rs.getString("endpoint")))
            .status(StringUtils.trimToEmpty(rs.getString("status")))
            .publishedAt(formatTimestamp(rs.getTimestamp("published_at")))
            .unpublishedAt(formatTimestamp(rs.getTimestamp("unpublished_at")))
            .createdAt(formatTimestamp(rs.getTimestamp("created_at")))
            .updatedAt(formatTimestamp(rs.getTimestamp("updated_at")))
            .pricing(readPricing(rs.getString("pricing_json")))
            .limits(ModelBindingLimits.builder()
                .rpm(readNullableLong(rs, "rpm"))
                .tpm(readNullableLong(rs, "tpm"))
                .contextWindow(readNullableLong(rs, "context_window"))
                .build())
            .build();
    }

    private ModelBindingPriceVersionRecord toPriceVersionRecord(ResultSet rs) throws SQLException {
        Timestamp effectiveTo = rs.getTimestamp("effective_to");
        String status = StringUtils.trimToEmpty(rs.getString("status"));
        return ModelBindingPriceVersionRecord.builder()
            .versionId(rs.getLong("id"))
            .modelId(StringUtils.trimToEmpty(rs.getString("model_id")))
            .currency(StringUtils.defaultIfBlank(StringUtils.trimToNull(rs.getString("currency")), CURRENCY_CNY))
            .status(status)
            .active(effectiveTo == null && StringUtils.equalsIgnoreCase(status, BILLING_STATUS_ACTIVE))
            .effectiveFrom(formatTimestamp(rs.getTimestamp("effective_from")))
            .effectiveTo(formatTimestamp(effectiveTo))
            .createdAt(formatTimestamp(rs.getTimestamp("created_at")))
            .updatedAt(formatTimestamp(rs.getTimestamp("updated_at")))
            .pricing(ModelBindingPricing.builder()
                .currency(StringUtils.defaultIfBlank(StringUtils.trimToNull(rs.getString("currency")), CURRENCY_CNY))
                .inputCostPerToken(readPerTokenPrice(rs, "input_price_per_1k_micro_yuan"))
                .outputCostPerToken(readPerTokenPrice(rs, "output_price_per_1k_micro_yuan"))
                .inputCostPerRequest(readRmbPrice(rs, "input_request_price_micro_yuan"))
                .cacheCreationInputTokenCost(readPerTokenPrice(rs, "cache_creation_input_token_price_per_1k_micro_yuan"))
                .cacheCreationInputTokenCostAbove1hr(
                    readPerTokenPrice(rs, "cache_creation_input_token_price_above_1hr_per_1k_micro_yuan"))
                .cacheReadInputTokenCost(readPerTokenPrice(rs, "cache_read_input_token_price_per_1k_micro_yuan"))
                .inputCostPerTokenAbove200kTokens(readPerTokenPrice(rs, "input_token_price_above_200k_per_1k_micro_yuan"))
                .outputCostPerTokenAbove200kTokens(readPerTokenPrice(rs, "output_token_price_above_200k_per_1k_micro_yuan"))
                .cacheCreationInputTokenCostAbove200kTokens(
                    readPerTokenPrice(rs, "cache_creation_input_token_price_above_200k_per_1k_micro_yuan"))
                .cacheReadInputTokenCostAbove200kTokens(
                    readPerTokenPrice(rs, "cache_read_input_token_price_above_200k_per_1k_micro_yuan"))
                .outputCostPerImage(readRmbPrice(rs, "output_image_price_micro_yuan"))
                .outputCostPerImageToken(readPerTokenPrice(rs, "output_image_token_price_per_1k_micro_yuan"))
                .inputCostPerImage(readRmbPrice(rs, "input_image_price_micro_yuan"))
                .inputCostPerImageToken(readPerTokenPrice(rs, "input_image_token_price_per_1k_micro_yuan"))
                .supportsPromptCaching(rs.getInt("supports_prompt_caching") > 0)
                .build())
            .build();
    }

    private ModelBindingPricing requirePriceVersion(String modelId, Long versionId) {
        String sql = "SELECT model_id, currency, input_price_per_1k_micro_yuan, output_price_per_1k_micro_yuan, "
            + "input_request_price_micro_yuan, cache_creation_input_token_price_per_1k_micro_yuan, "
            + "cache_creation_input_token_price_above_1hr_per_1k_micro_yuan, cache_read_input_token_price_per_1k_micro_yuan, "
            + "input_token_price_above_200k_per_1k_micro_yuan, output_token_price_above_200k_per_1k_micro_yuan, "
            + "cache_creation_input_token_price_above_200k_per_1k_micro_yuan, cache_read_input_token_price_above_200k_per_1k_micro_yuan, "
            + "output_image_price_micro_yuan, output_image_token_price_per_1k_micro_yuan, input_image_price_micro_yuan, "
            + "input_image_token_price_per_1k_micro_yuan, supports_prompt_caching FROM billing_model_price_version "
            + "WHERE id = ? AND model_id = ? LIMIT 1";
        try (Connection connection = openConnection();
            PreparedStatement statement = connection.prepareStatement(sql)) {
            statement.setLong(1, versionId);
            statement.setString(2, modelId);
            try (ResultSet rs = statement.executeQuery()) {
                if (!rs.next()) {
                    throw new ValidationException("Model price version not found: " + versionId);
                }
                return ModelBindingPricing.builder()
                    .currency(StringUtils.defaultIfBlank(StringUtils.trimToNull(rs.getString("currency")), CURRENCY_CNY))
                    .inputCostPerToken(readPerTokenPrice(rs, "input_price_per_1k_micro_yuan"))
                    .outputCostPerToken(readPerTokenPrice(rs, "output_price_per_1k_micro_yuan"))
                    .inputCostPerRequest(readRmbPrice(rs, "input_request_price_micro_yuan"))
                    .cacheCreationInputTokenCost(readPerTokenPrice(rs, "cache_creation_input_token_price_per_1k_micro_yuan"))
                    .cacheCreationInputTokenCostAbove1hr(
                        readPerTokenPrice(rs, "cache_creation_input_token_price_above_1hr_per_1k_micro_yuan"))
                    .cacheReadInputTokenCost(readPerTokenPrice(rs, "cache_read_input_token_price_per_1k_micro_yuan"))
                    .inputCostPerTokenAbove200kTokens(readPerTokenPrice(rs, "input_token_price_above_200k_per_1k_micro_yuan"))
                    .outputCostPerTokenAbove200kTokens(readPerTokenPrice(rs, "output_token_price_above_200k_per_1k_micro_yuan"))
                    .cacheCreationInputTokenCostAbove200kTokens(
                        readPerTokenPrice(rs, "cache_creation_input_token_price_above_200k_per_1k_micro_yuan"))
                    .cacheReadInputTokenCostAbove200kTokens(
                        readPerTokenPrice(rs, "cache_read_input_token_price_above_200k_per_1k_micro_yuan"))
                    .outputCostPerImage(readRmbPrice(rs, "output_image_price_micro_yuan"))
                    .outputCostPerImageToken(readPerTokenPrice(rs, "output_image_token_price_per_1k_micro_yuan"))
                    .inputCostPerImage(readRmbPrice(rs, "input_image_price_micro_yuan"))
                    .inputCostPerImageToken(readPerTokenPrice(rs, "input_image_token_price_per_1k_micro_yuan"))
                    .supportsPromptCaching(rs.getInt("supports_prompt_caching") > 0)
                    .build();
            }
        } catch (SQLException ex) {
            throw new BusinessException("Failed to load model price version.", ex);
        }
    }

    private List<String> readStringList(String rawJson) {
        if (StringUtils.isBlank(rawJson)) {
            return Collections.emptyList();
        }
        try {
            List<String> values = objectMapper.readValue(rawJson, STRING_LIST_TYPE);
            return normalizeStringList(values);
        } catch (Exception ex) {
            log.warn("Failed to parse string list json: {}", rawJson, ex);
            return Collections.emptyList();
        }
    }

    private ModelBindingPricing readPricing(String rawJson) {
        if (StringUtils.isBlank(rawJson)) {
            return ModelBindingPricing.builder().currency(CURRENCY_CNY).build();
        }
        try {
            ModelBindingPricing pricing = objectMapper.readValue(rawJson, PRICING_TYPE);
            return pricing == null ? ModelBindingPricing.builder().currency(CURRENCY_CNY).build() : pricing;
        } catch (Exception ex) {
            log.warn("Failed to parse pricing json: {}", rawJson, ex);
            return ModelBindingPricing.builder().currency(CURRENCY_CNY).build();
        }
    }

    private String writeJson(Object value) {
        try {
            return objectMapper.writeValueAsString(value);
        } catch (JsonProcessingException ex) {
            throw new BusinessException("Failed to serialize model asset payload.", ex);
        }
    }

    private String formatTimestamp(Timestamp timestamp) {
        if (timestamp == null) {
            return null;
        }
        return timestamp.toLocalDateTime().format(DateTimeFormatter.ISO_LOCAL_DATE_TIME);
    }

    private List<String> normalizeStringList(List<String> values) {
        if (values == null || values.isEmpty()) {
            return Collections.emptyList();
        }
        Set<String> normalized = new LinkedHashSet<>();
        for (String value : values) {
            String text = StringUtils.trimToNull(value);
            if (text != null) {
                normalized.add(text);
            }
        }
        return new ArrayList<>(normalized);
    }

    private Double requireNonNegative(Double value, String field) {
        if (value == null) {
            throw new ValidationException(field + " cannot be null.");
        }
        if (value < 0) {
            throw new ValidationException(field + " cannot be negative.");
        }
        return value;
    }

    private Double optionalNonNegative(Double value, String field) {
        if (value == null) {
            return null;
        }
        if (value < 0) {
            throw new ValidationException(field + " cannot be negative.");
        }
        return value;
    }

    private Long optionalNonNegativeLong(Long value, String field) {
        if (value == null) {
            return null;
        }
        if (value < 0) {
            throw new ValidationException(field + " cannot be negative.");
        }
        return value;
    }

    private void requirePricing(Double value, String field) {
        if (value == null) {
            throw new ValidationException(field + " is required.");
        }
        if (value < 0) {
            throw new ValidationException(field + " cannot be negative.");
        }
    }

    private long toMicroYuan(double amount) {
        return BigDecimal.valueOf(amount).multiply(BigDecimal.valueOf(MICRO_YUAN_PER_RMB))
            .setScale(0, RoundingMode.HALF_UP).longValue();
    }

    private long toPer1KMicroYuan(double perTokenAmount) {
        return toMicroYuan(perTokenAmount * 1000D);
    }

    private Double readRmbPrice(ResultSet rs, String column) throws SQLException {
        long raw = rs.getLong(column);
        if (rs.wasNull()) {
            return null;
        }
        return raw / (double)MICRO_YUAN_PER_RMB;
    }

    private Double readPerTokenPrice(ResultSet rs, String column) throws SQLException {
        Double value = readRmbPrice(rs, column);
        return value == null ? null : value / 1000D;
    }

    private double defaultDouble(Double value) {
        return value == null ? 0D : value;
    }

    private String toIdentifier(String raw) {
        String value = StringUtils.trimToNull(raw);
        if (value == null) {
            return null;
        }
        value = value.toLowerCase(Locale.ROOT).replaceAll("[^a-z0-9._-]+", "-");
        value = value.replaceAll("^-+", "").replaceAll("-+$", "");
        return StringUtils.trimToNull(value);
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

    private Long readNullableLong(ResultSet rs, String column) throws SQLException {
        long value = rs.getLong(column);
        return rs.wasNull() ? null : value;
    }

    private String asString(Object value) {
        return value instanceof String ? (String)value : null;
    }

    private Object readNestedValue(Map<String, Object> source, String parent, String field) {
        if (source == null) {
            return null;
        }
        Object parentValue = source.get(parent);
        if (!(parentValue instanceof Map)) {
            return null;
        }
        @SuppressWarnings("unchecked")
        Map<String, Object> map = (Map<String, Object>)parentValue;
        return map.get(field);
    }

    private Map<String, Object> readNestedMapValue(Map<String, Object> source, String field) {
        if (source == null) {
            return Collections.emptyMap();
        }
        Object value = source.get(field);
        if (!(value instanceof Map)) {
            return Collections.emptyMap();
        }
        @SuppressWarnings("unchecked")
        Map<String, Object> result = (Map<String, Object>)value;
        return result;
    }

    private List<String> readStringListValue(Object value) {
        if (!(value instanceof List)) {
            return Collections.emptyList();
        }
        @SuppressWarnings("unchecked")
        List<Object> objects = (List<Object>)value;
        List<String> result = new ArrayList<>();
        for (Object item : objects) {
            String text = asString(item);
            if (StringUtils.isNotBlank(text)) {
                result.add(StringUtils.trim(text));
            }
        }
        return result;
    }

    private ModelBindingPricing fromLegacyPricing(Map<String, Object> pricing) {
        return ModelBindingPricing.builder()
            .currency(CURRENCY_CNY)
            .inputCostPerToken(readLegacyPerToken(pricing, "input_cost_per_token", "inputPer1K"))
            .outputCostPerToken(readLegacyPerToken(pricing, "output_cost_per_token", "outputPer1K"))
            .inputCostPerRequest(readLegacyNumber(pricing, "input_cost_per_request"))
            .cacheCreationInputTokenCost(readLegacyNumber(pricing, "cache_creation_input_token_cost"))
            .cacheCreationInputTokenCostAbove1hr(readLegacyNumber(pricing, "cache_creation_input_token_cost_above_1hr"))
            .cacheReadInputTokenCost(readLegacyNumber(pricing, "cache_read_input_token_cost"))
            .inputCostPerTokenAbove200kTokens(readLegacyNumber(pricing, "input_cost_per_token_above_200k_tokens"))
            .outputCostPerTokenAbove200kTokens(readLegacyNumber(pricing, "output_cost_per_token_above_200k_tokens"))
            .cacheCreationInputTokenCostAbove200kTokens(readLegacyNumber(pricing, "cache_creation_input_token_cost_above_200k_tokens"))
            .cacheReadInputTokenCostAbove200kTokens(readLegacyNumber(pricing, "cache_read_input_token_cost_above_200k_tokens"))
            .outputCostPerImage(readLegacyNumber(pricing, "output_cost_per_image"))
            .outputCostPerImageToken(readLegacyNumber(pricing, "output_cost_per_image_token"))
            .inputCostPerImage(readLegacyNumber(pricing, "input_cost_per_image"))
            .inputCostPerImageToken(readLegacyNumber(pricing, "input_cost_per_image_token"))
            .supportsPromptCaching(Boolean.TRUE.equals(pricing.get("supports_prompt_caching")))
            .build();
    }

    private ModelBindingLimits fromLegacyLimits(Map<String, Object> limits) {
        return ModelBindingLimits.builder()
            .rpm(toLong(limits.get("rpm")))
            .tpm(toLong(limits.get("tpm")))
            .contextWindow(toLong(limits.get("contextWindow")))
            .build();
    }

    private Double readLegacyPerToken(Map<String, Object> pricing, String fieldName, String legacyFieldName) {
        Double current = readLegacyNumber(pricing, fieldName);
        if (current != null) {
            return current;
        }
        Double legacy = readLegacyNumber(pricing, legacyFieldName);
        return legacy == null ? null : legacy / 1000D;
    }

    private Double readLegacyNumber(Map<String, Object> source, String fieldName) {
        Object value = source.get(fieldName);
        if (value instanceof Number) {
            double parsed = ((Number)value).doubleValue();
            return parsed < 0 ? null : parsed;
        }
        if (value instanceof String) {
            String text = StringUtils.trimToNull((String)value);
            if (text == null) {
                return null;
            }
            try {
                double parsed = Double.parseDouble(text);
                return parsed < 0 ? null : parsed;
            } catch (NumberFormatException ex) {
                return null;
            }
        }
        return null;
    }

    private Long toLong(Object value) {
        if (value instanceof Number) {
            long parsed = ((Number)value).longValue();
            return parsed < 0 ? null : parsed;
        }
        if (value instanceof String) {
            String text = StringUtils.trimToNull((String)value);
            if (text == null) {
                return null;
            }
            try {
                long parsed = Long.parseLong(text);
                return parsed < 0 ? null : parsed;
            } catch (NumberFormatException ex) {
                return null;
            }
        }
        return null;
    }

    private static final class AssetMutation {
        private final String assetId;
        private final String canonicalName;
        private final String displayName;
        private final String intro;
        private final List<String> tags;
        private final List<String> modalities;
        private final List<String> features;
        private final List<String> requestKinds;

        private AssetMutation(String assetId, String canonicalName, String displayName, String intro, List<String> tags,
            List<String> modalities, List<String> features, List<String> requestKinds) {
            this.assetId = assetId;
            this.canonicalName = canonicalName;
            this.displayName = displayName;
            this.intro = intro;
            this.tags = tags;
            this.modalities = modalities;
            this.features = features;
            this.requestKinds = requestKinds;
        }
    }

    private static final class BindingMutation {
        private final String bindingId;
        private final String modelId;
        private final String providerName;
        private final String targetModel;
        private final String protocol;
        private final String endpoint;
        private final ModelBindingPricing pricing;
        private final long rpm;
        private final long tpm;
        private final long contextWindow;

        private BindingMutation(String bindingId, String modelId, String providerName, String targetModel, String protocol,
            String endpoint, ModelBindingPricing pricing, ModelBindingLimits limits) {
            this.bindingId = bindingId;
            this.modelId = modelId;
            this.providerName = providerName;
            this.targetModel = targetModel;
            this.protocol = protocol;
            this.endpoint = endpoint;
            this.pricing = pricing;
            this.rpm = limits == null || limits.getRpm() == null ? 0L : limits.getRpm();
            this.tpm = limits == null || limits.getTpm() == null ? 0L : limits.getTpm();
            this.contextWindow = limits == null || limits.getContextWindow() == null ? 0L : limits.getContextWindow();
        }
    }

    @lombok.Builder
    private static final class BindingRow {
        private String bindingId;
        private String assetId;
        private String modelId;
        private String providerName;
        private String targetModel;
        private String protocol;
        private String endpoint;
        private String status;
        private String publishedAt;
        private String unpublishedAt;
        private ModelBindingPricing pricing;
        private ModelBindingLimits limits;
    }

    @lombok.Builder
    private static final class PublishedBindingMeta {
        private String modelId;
        private String name;
        private String vendor;
        private String capability;
        private String endpoint;
        private String sdk;
        private String summary;
        private String currency;
        private long inputPricePer1KMicroYuan;
        private long outputPricePer1KMicroYuan;
        private long inputRequestPriceMicroYuan;
        private long cacheCreationInputTokenPricePer1KMicroYuan;
        private long cacheCreationInputTokenPriceAbove1hrPer1KMicroYuan;
        private long cacheReadInputTokenPricePer1KMicroYuan;
        private long inputTokenPriceAbove200kPer1KMicroYuan;
        private long outputTokenPriceAbove200kPer1KMicroYuan;
        private long cacheCreationInputTokenPriceAbove200kPer1KMicroYuan;
        private long cacheReadInputTokenPriceAbove200kPer1KMicroYuan;
        private long outputImagePriceMicroYuan;
        private long outputImageTokenPricePer1KMicroYuan;
        private long inputImagePriceMicroYuan;
        private long inputImageTokenPricePer1KMicroYuan;
        private boolean supportsPromptCaching;

        private boolean matches(PriceVersionState current) {
            return current != null
                && StringUtils.equalsIgnoreCase(currency, current.currency)
                && inputPricePer1KMicroYuan == current.inputPricePer1KMicroYuan
                && outputPricePer1KMicroYuan == current.outputPricePer1KMicroYuan
                && inputRequestPriceMicroYuan == current.inputRequestPriceMicroYuan
                && cacheCreationInputTokenPricePer1KMicroYuan == current.cacheCreationInputTokenPricePer1KMicroYuan
                && cacheCreationInputTokenPriceAbove1hrPer1KMicroYuan == current.cacheCreationInputTokenPriceAbove1hrPer1KMicroYuan
                && cacheReadInputTokenPricePer1KMicroYuan == current.cacheReadInputTokenPricePer1KMicroYuan
                && inputTokenPriceAbove200kPer1KMicroYuan == current.inputTokenPriceAbove200kPer1KMicroYuan
                && outputTokenPriceAbove200kPer1KMicroYuan == current.outputTokenPriceAbove200kPer1KMicroYuan
                && cacheCreationInputTokenPriceAbove200kPer1KMicroYuan == current.cacheCreationInputTokenPriceAbove200kPer1KMicroYuan
                && cacheReadInputTokenPriceAbove200kPer1KMicroYuan == current.cacheReadInputTokenPriceAbove200kPer1KMicroYuan
                && outputImagePriceMicroYuan == current.outputImagePriceMicroYuan
                && outputImageTokenPricePer1KMicroYuan == current.outputImageTokenPricePer1KMicroYuan
                && inputImagePriceMicroYuan == current.inputImagePriceMicroYuan
                && inputImageTokenPricePer1KMicroYuan == current.inputImageTokenPricePer1KMicroYuan
                && supportsPromptCaching == current.supportsPromptCaching;
        }
    }

    @lombok.AllArgsConstructor
    private static final class PriceVersionState {
        private final long id;
        private final String currency;
        private final long inputPricePer1KMicroYuan;
        private final long outputPricePer1KMicroYuan;
        private final long inputRequestPriceMicroYuan;
        private final long cacheCreationInputTokenPricePer1KMicroYuan;
        private final long cacheCreationInputTokenPriceAbove1hrPer1KMicroYuan;
        private final long cacheReadInputTokenPricePer1KMicroYuan;
        private final long inputTokenPriceAbove200kPer1KMicroYuan;
        private final long outputTokenPriceAbove200kPer1KMicroYuan;
        private final long cacheCreationInputTokenPriceAbove200kPer1KMicroYuan;
        private final long cacheReadInputTokenPriceAbove200kPer1KMicroYuan;
        private final long outputImagePriceMicroYuan;
        private final long outputImageTokenPricePer1KMicroYuan;
        private final long inputImagePriceMicroYuan;
        private final long inputImageTokenPricePer1KMicroYuan;
        private final boolean supportsPromptCaching;
    }
}
