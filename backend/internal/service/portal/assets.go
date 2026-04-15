package portal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/wooveep/aigateway-console/backend/internal/model/do"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

var (
	defaultModelModalities   = []string{"text", "image", "audio", "video", "embedding"}
	defaultModelFeatures     = []string{"reasoning", "vision", "function_calling", "structured_output", "long_context", "code", "multimodal"}
	defaultModelRequestKinds = []string{"chat_completions", "responses", "embeddings", "images", "audio"}
)

type AssetGrantRecord struct {
	AssetType   string `json:"assetType,omitempty"`
	AssetID     string `json:"assetId,omitempty"`
	SubjectType string `json:"subjectType,omitempty"`
	SubjectID   string `json:"subjectId,omitempty"`
}

type ProviderModelOption struct {
	ModelID     string `json:"modelId"`
	TargetModel string `json:"targetModel"`
	Label       string `json:"label"`
}

type ProviderModelCatalog struct {
	ProviderName string                `json:"providerName"`
	Models       []ProviderModelOption `json:"models"`
}

type ModelAssetOptions struct {
	Capabilities struct {
		Modalities   []string `json:"modalities"`
		Features     []string `json:"features"`
		RequestKinds []string `json:"requestKinds"`
	} `json:"capabilities"`
	ProviderModels []ProviderModelCatalog `json:"providerModels"`
}

type ModelAssetCapabilities struct {
	Modalities   []string `json:"modalities,omitempty"`
	Features     []string `json:"features,omitempty"`
	RequestKinds []string `json:"requestKinds,omitempty"`
}

type ModelBindingPricing struct {
	Currency                                   string   `json:"currency,omitempty"`
	InputCostPerToken                          *float64 `json:"inputCostPerToken,omitempty"`
	OutputCostPerToken                         *float64 `json:"outputCostPerToken,omitempty"`
	InputCostPerRequest                        *float64 `json:"inputCostPerRequest,omitempty"`
	CacheCreationInputTokenCost                *float64 `json:"cacheCreationInputTokenCost,omitempty"`
	CacheCreationInputTokenCostAbove1hr        *float64 `json:"cacheCreationInputTokenCostAbove1hr,omitempty"`
	CacheReadInputTokenCost                    *float64 `json:"cacheReadInputTokenCost,omitempty"`
	InputCostPerTokenAbove200kTokens           *float64 `json:"inputCostPerTokenAbove200kTokens,omitempty"`
	OutputCostPerTokenAbove200kTokens          *float64 `json:"outputCostPerTokenAbove200kTokens,omitempty"`
	CacheCreationInputTokenCostAbove200kTokens *float64 `json:"cacheCreationInputTokenCostAbove200kTokens,omitempty"`
	CacheReadInputTokenCostAbove200kTokens     *float64 `json:"cacheReadInputTokenCostAbove200kTokens,omitempty"`
	OutputCostPerImage                         *float64 `json:"outputCostPerImage,omitempty"`
	OutputCostPerImageToken                    *float64 `json:"outputCostPerImageToken,omitempty"`
	InputCostPerImage                          *float64 `json:"inputCostPerImage,omitempty"`
	InputCostPerImageToken                     *float64 `json:"inputCostPerImageToken,omitempty"`
	SupportsPromptCaching                      bool     `json:"supportsPromptCaching,omitempty"`
}

type ModelBindingLimits struct {
	RPM           *int `json:"rpm,omitempty"`
	TPM           *int `json:"tpm,omitempty"`
	ContextWindow *int `json:"contextWindow,omitempty"`
}

type ModelBindingPriceVersion struct {
	VersionID     int64                `json:"versionId"`
	ModelID       string               `json:"modelId"`
	Currency      string               `json:"currency,omitempty"`
	Status        string               `json:"status,omitempty"`
	Active        bool                 `json:"active,omitempty"`
	EffectiveFrom *time.Time           `json:"effectiveFrom,omitempty"`
	EffectiveTo   *time.Time           `json:"effectiveTo,omitempty"`
	CreatedAt     *time.Time           `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time           `json:"updatedAt,omitempty"`
	Pricing       *ModelBindingPricing `json:"pricing,omitempty"`
}

type ModelAssetBinding struct {
	BindingID     string               `json:"bindingId"`
	AssetID       string               `json:"assetId,omitempty"`
	ModelID       string               `json:"modelId,omitempty"`
	ProviderName  string               `json:"providerName,omitempty"`
	TargetModel   string               `json:"targetModel,omitempty"`
	Protocol      string               `json:"protocol,omitempty"`
	Endpoint      string               `json:"endpoint,omitempty"`
	Status        string               `json:"status,omitempty"`
	PublishedAt   *time.Time           `json:"publishedAt,omitempty"`
	UnpublishedAt *time.Time           `json:"unpublishedAt,omitempty"`
	CreatedAt     *time.Time           `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time           `json:"updatedAt,omitempty"`
	Pricing       *ModelBindingPricing `json:"pricing,omitempty"`
	Limits        *ModelBindingLimits  `json:"limits,omitempty"`
}

type ModelAsset struct {
	AssetID       string                  `json:"assetId"`
	CanonicalName string                  `json:"canonicalName,omitempty"`
	DisplayName   string                  `json:"displayName,omitempty"`
	Intro         string                  `json:"intro,omitempty"`
	CreatedAt     *time.Time              `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time              `json:"updatedAt,omitempty"`
	Tags          []string                `json:"tags,omitempty"`
	Capabilities  *ModelAssetCapabilities `json:"capabilities,omitempty"`
	Bindings      []ModelAssetBinding     `json:"bindings,omitempty"`
}

type AgentCatalogOptionServer struct {
	McpServerName string   `json:"mcpServerName"`
	Description   string   `json:"description,omitempty"`
	Type          string   `json:"type,omitempty"`
	Domains       []string `json:"domains,omitempty"`
	AuthEnabled   *bool    `json:"authEnabled,omitempty"`
	AuthType      string   `json:"authType,omitempty"`
}

type AgentCatalogOptions struct {
	Servers []AgentCatalogOptionServer `json:"servers"`
}

type AgentCatalogRecord struct {
	AgentID         string     `json:"agentId"`
	CanonicalName   string     `json:"canonicalName,omitempty"`
	DisplayName     string     `json:"displayName,omitempty"`
	Intro           string     `json:"intro,omitempty"`
	Description     string     `json:"description,omitempty"`
	IconURL         string     `json:"iconUrl,omitempty"`
	Tags            []string   `json:"tags,omitempty"`
	McpServerName   string     `json:"mcpServerName,omitempty"`
	Status          string     `json:"status,omitempty"`
	ToolCount       int        `json:"toolCount,omitempty"`
	TransportTypes  []string   `json:"transportTypes,omitempty"`
	ResourceSummary string     `json:"resourceSummary,omitempty"`
	PromptSummary   string     `json:"promptSummary,omitempty"`
	PublishedAt     *time.Time `json:"publishedAt,omitempty"`
	UnpublishedAt   *time.Time `json:"unpublishedAt,omitempty"`
	CreatedAt       *time.Time `json:"createdAt,omitempty"`
	UpdatedAt       *time.Time `json:"updatedAt,omitempty"`
}

func (s *Service) ListAssetGrants(ctx context.Context, assetType, assetID string) ([]AssetGrantRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	items, err := newPortalStore(db).listAssetGrants(ctx, strings.TrimSpace(assetType), strings.TrimSpace(assetID))
	if err != nil {
		return nil, err
	}
	records := make([]AssetGrantRecord, 0, len(items))
	for _, item := range items {
		records = append(records, AssetGrantRecord{
			AssetType:   item.AssetType,
			AssetID:     item.AssetId,
			SubjectType: item.SubjectType,
			SubjectID:   item.SubjectId,
		})
	}
	return records, nil
}

func (s *Service) ReplaceAssetGrants(ctx context.Context, assetType, assetID string, grants []AssetGrantRecord) ([]AssetGrantRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	normalizedType := strings.TrimSpace(assetType)
	normalizedID := strings.TrimSpace(assetID)
	if normalizedType == "" || normalizedID == "" {
		return nil, errors.New("assetType and assetId are required")
	}

	normalizedGrants := make([]do.AssetGrant, 0, len(grants))
	for _, item := range grants {
		record := AssetGrantRecord{
			AssetType:   normalizedType,
			AssetID:     normalizedID,
			SubjectType: strings.TrimSpace(strings.ToLower(item.SubjectType)),
			SubjectID:   strings.TrimSpace(item.SubjectID),
		}
		if !isSupportedGrantSubject(record.SubjectType) || record.SubjectID == "" {
			return nil, fmt.Errorf("invalid grant subject: %s/%s", record.SubjectType, record.SubjectID)
		}
		normalizedGrants = append(normalizedGrants, do.AssetGrant{
			AssetType:   record.AssetType,
			AssetId:     record.AssetID,
			SubjectType: record.SubjectType,
			SubjectId:   record.SubjectID,
		})
	}
	if err := newPortalStore(db).replaceAssetGrants(ctx, normalizedType, normalizedID, normalizedGrants); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "asset-grants-replace"); err != nil {
		return nil, err
	}
	return s.ListAssetGrants(ctx, normalizedType, normalizedID)
}

func (s *Service) ListModelAssets(ctx context.Context) ([]ModelAsset, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := db.QueryContext(ctx, `
		SELECT asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at
		FROM portal_model_asset
		ORDER BY asset_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]ModelAsset, 0)
	for rows.Next() {
		item, err := scanModelAsset(rows)
		if err != nil {
			return nil, err
		}
		assets = append(assets, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	bindings, err := s.listAllBindings(ctx, db)
	if err != nil {
		return nil, err
	}
	for index := range assets {
		assets[index].Bindings = append([]ModelAssetBinding{}, bindings[assets[index].AssetID]...)
	}
	return assets, nil
}

func (s *Service) GetModelAsset(ctx context.Context, assetID string) (*ModelAsset, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at
		FROM portal_model_asset
		WHERE asset_id = ?`, strings.TrimSpace(assetID))
	item, err := scanModelAsset(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	bindings, err := s.listBindingsByAsset(ctx, db, item.AssetID)
	if err != nil {
		return nil, err
	}
	item.Bindings = bindings
	return &item, nil
}

func (s *Service) GetModelAssetOptions(ctx context.Context) (*ModelAssetOptions, error) {
	options := &ModelAssetOptions{
		ProviderModels: []ProviderModelCatalog{},
	}
	options.Capabilities.Modalities = append([]string{}, defaultModelModalities...)
	options.Capabilities.Features = append([]string{}, defaultModelFeatures...)
	options.Capabilities.RequestKinds = append([]string{}, defaultModelRequestKinds...)

	if s.k8sClient == nil {
		return options, nil
	}
	providers, err := s.k8sClient.ListResources(ctx, "ai-providers")
	if err != nil {
		return options, nil
	}
	for _, provider := range providers {
		name := strings.TrimSpace(fmt.Sprint(provider["name"]))
		if name == "" {
			continue
		}
		models := make([]ProviderModelOption, 0)
		for _, record := range providerModelRecords(provider["models"]) {
			modelID := strings.TrimSpace(fmt.Sprint(record["modelId"]))
			targetModel := strings.TrimSpace(fmt.Sprint(record["targetModel"]))
			if modelID == "" {
				modelID = strings.TrimSpace(fmt.Sprint(record["name"]))
			}
			if targetModel == "" {
				targetModel = modelID
			}
			if modelID == "" {
				continue
			}
			models = append(models, ProviderModelOption{
				ModelID:     modelID,
				TargetModel: targetModel,
				Label:       firstNonEmpty(strings.TrimSpace(fmt.Sprint(record["label"])), modelID),
			})
		}
		options.ProviderModels = append(options.ProviderModels, ProviderModelCatalog{
			ProviderName: name,
			Models:       models,
		})
	}
	sort.Slice(options.ProviderModels, func(i, j int) bool {
		return options.ProviderModels[i].ProviderName < options.ProviderModels[j].ProviderName
	})
	return options, nil
}

func providerModelRecords(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		result := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(map[string]any)
			if !ok {
				continue
			}
			result = append(result, record)
		}
		return result
	default:
		return nil
	}
}

func (s *Service) CreateModelAsset(ctx context.Context, asset ModelAsset) (*ModelAsset, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeModelAsset(asset)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO portal_model_asset (
			asset_id, canonical_name, display_name, intro, tags_json, modalities_json, features_json, request_kinds_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.AssetID,
		normalized.CanonicalName,
		normalized.DisplayName,
		nullIfEmpty(normalized.Intro),
		mustJSONString(normalized.Tags),
		mustJSONString(normalized.Capabilities.Modalities),
		mustJSONString(normalized.Capabilities.Features),
		mustJSONString(normalized.Capabilities.RequestKinds),
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-asset-create"); err != nil {
		return nil, err
	}
	return s.GetModelAsset(ctx, normalized.AssetID)
}

func (s *Service) UpdateModelAsset(ctx context.Context, assetID string, asset ModelAsset) (*ModelAsset, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeModelAsset(asset)
	if err != nil {
		return nil, err
	}
	id := strings.TrimSpace(assetID)
	if id == "" {
		return nil, errors.New("assetId is required")
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_asset
		SET canonical_name = ?, display_name = ?, intro = ?, tags_json = ?, modalities_json = ?, features_json = ?, request_kinds_json = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ?`,
		normalized.CanonicalName,
		normalized.DisplayName,
		nullIfEmpty(normalized.Intro),
		mustJSONString(normalized.Tags),
		mustJSONString(normalized.Capabilities.Modalities),
		mustJSONString(normalized.Capabilities.Features),
		mustJSONString(normalized.Capabilities.RequestKinds),
		id,
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-asset-update"); err != nil {
		return nil, err
	}
	return s.GetModelAsset(ctx, id)
}

func (s *Service) CreateModelBinding(ctx context.Context, assetID string, binding ModelAssetBinding) (*ModelAssetBinding, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(assetID) == "" {
		return nil, errors.New("assetId is required")
	}
	normalized, err := normalizeBinding(binding)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO portal_model_binding (
			binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, rpm, tpm, context_window, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft')`,
		normalized.BindingID,
		assetID,
		normalized.ModelID,
		normalized.ProviderName,
		normalized.TargetModel,
		firstNonEmpty(normalized.Protocol, "openai/v1"),
		firstNonEmpty(normalized.Endpoint, "-"),
		mustJSONString(normalized.Pricing),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.RPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.TPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.ContextWindow }),
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-create"); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, assetID, normalized.BindingID)
}

func (s *Service) UpdateModelBinding(ctx context.Context, assetID, bindingID string, binding ModelAssetBinding) (*ModelAssetBinding, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeBinding(binding)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_binding
		SET model_id = ?, provider_name = ?, target_model = ?, protocol = ?, endpoint = ?, pricing_json = ?, rpm = ?, tpm = ?, context_window = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`,
		normalized.ModelID,
		normalized.ProviderName,
		normalized.TargetModel,
		firstNonEmpty(normalized.Protocol, "openai/v1"),
		firstNonEmpty(normalized.Endpoint, "-"),
		mustJSONString(normalized.Pricing),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.RPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.TPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.ContextWindow }),
		strings.TrimSpace(assetID),
		strings.TrimSpace(bindingID),
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-update"); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, assetID, bindingID)
}

func (s *Service) PublishModelBinding(ctx context.Context, assetID, bindingID string) (*ModelAssetBinding, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	binding, err := s.getBinding(ctx, assetID, bindingID)
	if err != nil || binding == nil {
		return binding, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `
		UPDATE portal_model_binding_price_version
		SET active = 0, effective_to = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ? AND active = 1`,
		now, strings.TrimSpace(assetID), strings.TrimSpace(bindingID)); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO portal_model_binding_price_version (
			asset_id, binding_id, status, active, effective_from, pricing_json
		) VALUES (?, ?, 'active', 1, ?, ?)`,
		strings.TrimSpace(assetID), strings.TrimSpace(bindingID), now, mustJSONString(binding.Pricing)); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE portal_model_binding
		SET status = 'published', published_at = ?, unpublished_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`,
		now, strings.TrimSpace(assetID), strings.TrimSpace(bindingID)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-publish"); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, assetID, bindingID)
}

func (s *Service) UnpublishModelBinding(ctx context.Context, assetID, bindingID string) (*ModelAssetBinding, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_binding
		SET status = 'unpublished', unpublished_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`,
		now, strings.TrimSpace(assetID), strings.TrimSpace(bindingID)); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_binding_price_version
		SET active = 0, effective_to = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ? AND active = 1`,
		now, strings.TrimSpace(assetID), strings.TrimSpace(bindingID)); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-unpublish"); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, assetID, bindingID)
}

func (s *Service) ListBindingPriceVersions(ctx context.Context, assetID, bindingID string) ([]ModelBindingPriceVersion, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT version_id, status, active, effective_from, effective_to, pricing_json, created_at, updated_at
		FROM portal_model_binding_price_version
		WHERE asset_id = ? AND binding_id = ?
		ORDER BY version_id DESC`, strings.TrimSpace(assetID), strings.TrimSpace(bindingID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := make([]ModelBindingPriceVersion, 0)
	binding, _ := s.getBinding(ctx, assetID, bindingID)
	modelID := ""
	if binding != nil {
		modelID = binding.ModelID
	}
	for rows.Next() {
		version, err := scanBindingPriceVersion(rows)
		if err != nil {
			return nil, err
		}
		version.ModelID = modelID
		if version.Pricing != nil {
			version.Currency = version.Pricing.Currency
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (s *Service) RestoreBindingPriceVersion(ctx context.Context, assetID, bindingID string, versionID int64) (*ModelAssetBinding, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT pricing_json
		FROM portal_model_binding_price_version
		WHERE asset_id = ? AND binding_id = ? AND version_id = ?`,
		strings.TrimSpace(assetID), strings.TrimSpace(bindingID), versionID)
	var pricingJSON sql.NullString
	if err := row.Scan(&pricingJSON); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_binding
		SET pricing_json = ?, status = 'draft', updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`,
		pricingJSON.String, strings.TrimSpace(assetID), strings.TrimSpace(bindingID)); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-price-restore"); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, assetID, bindingID)
}

func (s *Service) ListAgentCatalogs(ctx context.Context) ([]AgentCatalogRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
		SELECT agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, mcp_server_name,
			status, tool_count, transport_types_json, resource_summary, prompt_summary, published_at, unpublished_at, created_at, updated_at
		FROM portal_agent_catalog
		ORDER BY agent_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AgentCatalogRecord, 0)
	for rows.Next() {
		item, err := scanAgentCatalog(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) GetAgentCatalog(ctx context.Context, agentID string) (*AgentCatalogRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, mcp_server_name,
			status, tool_count, transport_types_json, resource_summary, prompt_summary, published_at, unpublished_at, created_at, updated_at
		FROM portal_agent_catalog
		WHERE agent_id = ?`, strings.TrimSpace(agentID))
	item, err := scanAgentCatalog(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *Service) GetAgentCatalogOptions(ctx context.Context) (*AgentCatalogOptions, error) {
	options := &AgentCatalogOptions{Servers: []AgentCatalogOptionServer{}}
	if s.k8sClient == nil {
		return options, nil
	}
	items, err := s.k8sClient.ListResources(ctx, "mcp-servers")
	if err != nil {
		return options, nil
	}
	for _, item := range items {
		record := AgentCatalogOptionServer{
			McpServerName: strings.TrimSpace(fmt.Sprint(item["name"])),
			Description:   strings.TrimSpace(fmt.Sprint(item["description"])),
			Type:          strings.TrimSpace(fmt.Sprint(item["type"])),
		}
		if rawDomains, ok := item["domains"].([]any); ok {
			record.Domains = anySliceToStringSlice(rawDomains)
		}
		if authInfo, ok := item["consumerAuthInfo"].(map[string]any); ok {
			if enabled, ok := authInfo["enable"].(bool); ok {
				record.AuthEnabled = &enabled
			}
			record.AuthType = strings.TrimSpace(fmt.Sprint(authInfo["type"]))
		}
		if record.McpServerName != "" {
			options.Servers = append(options.Servers, record)
		}
	}
	sort.Slice(options.Servers, func(i, j int) bool {
		return options.Servers[i].McpServerName < options.Servers[j].McpServerName
	})
	return options, nil
}

func (s *Service) CreateAgentCatalog(ctx context.Context, asset AgentCatalogRecord) (*AgentCatalogRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeAgentCatalog(asset)
	if err != nil {
		return nil, err
	}
	if normalized.McpServerName != "" {
		if ok, err := s.mcpServerExists(ctx, normalized.McpServerName); err != nil {
			return nil, err
		} else if !ok {
			return nil, fmt.Errorf("mcp server not found: %s", normalized.McpServerName)
		}
	}
	normalized.ToolCount, normalized.TransportTypes, normalized.ResourceSummary, normalized.PromptSummary = s.inspectMcpServer(ctx, normalized.McpServerName)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO portal_agent_catalog (
			agent_id, canonical_name, display_name, intro, description, icon_url, tags_json, mcp_server_name,
			status, tool_count, transport_types_json, resource_summary, prompt_summary
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'draft', ?, ?, ?, ?)`,
		normalized.AgentID,
		normalized.CanonicalName,
		normalized.DisplayName,
		firstNonEmpty(normalized.Intro, ""),
		firstNonEmpty(normalized.Description, ""),
		firstNonEmpty(normalized.IconURL, ""),
		mustJSONString(normalized.Tags),
		firstNonEmpty(normalized.McpServerName, ""),
		normalized.ToolCount,
		mustJSONString(normalized.TransportTypes),
		firstNonEmpty(normalized.ResourceSummary, ""),
		firstNonEmpty(normalized.PromptSummary, ""),
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "agent-catalog-create"); err != nil {
		return nil, err
	}
	return s.GetAgentCatalog(ctx, normalized.AgentID)
}

func (s *Service) UpdateAgentCatalog(ctx context.Context, agentID string, asset AgentCatalogRecord) (*AgentCatalogRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	normalized, err := normalizeAgentCatalog(asset)
	if err != nil {
		return nil, err
	}
	if normalized.McpServerName != "" {
		if ok, err := s.mcpServerExists(ctx, normalized.McpServerName); err != nil {
			return nil, err
		} else if !ok {
			return nil, fmt.Errorf("mcp server not found: %s", normalized.McpServerName)
		}
	}
	normalized.ToolCount, normalized.TransportTypes, normalized.ResourceSummary, normalized.PromptSummary = s.inspectMcpServer(ctx, normalized.McpServerName)
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_agent_catalog
		SET canonical_name = ?, display_name = ?, intro = ?, description = ?, icon_url = ?, tags_json = ?, mcp_server_name = ?,
			tool_count = ?, transport_types_json = ?, resource_summary = ?, prompt_summary = ?, updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = ?`,
		normalized.CanonicalName,
		normalized.DisplayName,
		firstNonEmpty(normalized.Intro, ""),
		firstNonEmpty(normalized.Description, ""),
		firstNonEmpty(normalized.IconURL, ""),
		mustJSONString(normalized.Tags),
		firstNonEmpty(normalized.McpServerName, ""),
		normalized.ToolCount,
		mustJSONString(normalized.TransportTypes),
		firstNonEmpty(normalized.ResourceSummary, ""),
		firstNonEmpty(normalized.PromptSummary, ""),
		strings.TrimSpace(agentID),
	); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "agent-catalog-update"); err != nil {
		return nil, err
	}
	return s.GetAgentCatalog(ctx, agentID)
}

func (s *Service) PublishAgentCatalog(ctx context.Context, agentID string) (*AgentCatalogRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	item, err := s.GetAgentCatalog(ctx, agentID)
	if err != nil || item == nil {
		return item, err
	}
	if item.McpServerName == "" {
		return nil, errors.New("published agent requires mcpServerName")
	}
	now := time.Now()
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_agent_catalog
		SET status = 'published', published_at = ?, unpublished_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = ?`, now, strings.TrimSpace(agentID)); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "agent-catalog-publish"); err != nil {
		return nil, err
	}
	return s.GetAgentCatalog(ctx, agentID)
}

func (s *Service) UnpublishAgentCatalog(ctx context.Context, agentID string) (*AgentCatalogRecord, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_agent_catalog
		SET status = 'unpublished', unpublished_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE agent_id = ?`, now, strings.TrimSpace(agentID)); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "agent-catalog-unpublish"); err != nil {
		return nil, err
	}
	return s.GetAgentCatalog(ctx, agentID)
}

func (s *Service) listBindingsByAsset(ctx context.Context, db *sql.DB, assetID string) ([]ModelAssetBinding, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ?
		ORDER BY binding_id`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ModelAssetBinding, 0)
	for rows.Next() {
		item, err := scanModelBinding(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) listAllBindings(ctx context.Context, db *sql.DB) (map[string][]ModelAssetBinding, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		ORDER BY asset_id, binding_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string][]ModelAssetBinding{}
	for rows.Next() {
		item, err := scanModelBinding(rows)
		if err != nil {
			return nil, err
		}
		result[item.AssetID] = append(result[item.AssetID], item)
	}
	return result, rows.Err()
}

func (s *Service) getBinding(ctx context.Context, assetID, bindingID string) (*ModelAssetBinding, error) {
	db, err := s.db(ctx)
	if err != nil {
		return nil, err
	}
	row := db.QueryRowContext(ctx, `
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`,
		strings.TrimSpace(assetID), strings.TrimSpace(bindingID))
	item, err := scanModelBinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *Service) mcpServerExists(ctx context.Context, name string) (bool, error) {
	if s.k8sClient == nil {
		return true, nil
	}
	_, err := s.k8sClient.GetResource(ctx, "mcp-servers", name)
	if err != nil {
		if errors.Is(err, k8sclient.ErrNotFound) {
			return false, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Service) inspectMcpServer(ctx context.Context, name string) (int, []string, string, string) {
	if s.k8sClient == nil || strings.TrimSpace(name) == "" {
		return 0, []string{}, "", ""
	}
	item, err := s.k8sClient.GetResource(ctx, "mcp-servers", name)
	if err != nil {
		return 0, []string{}, "", ""
	}

	toolCount := 0
	resourceSummary := strings.TrimSpace(fmt.Sprint(item["description"]))
	promptSummary := ""
	transportTypes := []string{}
	if raw, ok := item["transportTypes"].([]any); ok {
		transportTypes = anySliceToStringSlice(raw)
	}
	if len(transportTypes) == 0 {
		transportTypes = []string{"http", "sse"}
	}
	if tools, ok := item["tools"].([]any); ok {
		toolCount = len(tools)
		if toolCount > 0 {
			promptSummary = fmt.Sprintf("%d tools exposed", toolCount)
		}
	}
	return toolCount, transportTypes, resourceSummary, promptSummary
}

func scanModelAsset(scanner interface{ Scan(...any) error }) (ModelAsset, error) {
	var (
		item             ModelAsset
		intro            sql.NullString
		tagsJSON         sql.NullString
		modalitiesJSON   sql.NullString
		featuresJSON     sql.NullString
		requestKindsJSON sql.NullString
		createdAt        sql.NullTime
		updatedAt        sql.NullTime
	)
	if err := scanner.Scan(&item.AssetID, &item.CanonicalName, &item.DisplayName, &intro, &tagsJSON, &modalitiesJSON, &featuresJSON, &requestKindsJSON, &createdAt, &updatedAt); err != nil {
		return ModelAsset{}, err
	}
	item.Intro = intro.String
	item.Tags = parseJSONStringList(tagsJSON.String)
	item.Capabilities = &ModelAssetCapabilities{
		Modalities:   parseJSONStringList(modalitiesJSON.String),
		Features:     parseJSONStringList(featuresJSON.String),
		RequestKinds: parseJSONStringList(requestKindsJSON.String),
	}
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return item, nil
}

func scanModelBinding(scanner interface{ Scan(...any) error }) (ModelAssetBinding, error) {
	var (
		item          ModelAssetBinding
		endpoint      sql.NullString
		pricingJSON   sql.NullString
		rpm           sql.NullInt64
		tpm           sql.NullInt64
		contextWindow sql.NullInt64
		publishedAt   sql.NullTime
		unpublishedAt sql.NullTime
		createdAt     sql.NullTime
		updatedAt     sql.NullTime
	)
	if err := scanner.Scan(
		&item.BindingID, &item.AssetID, &item.ModelID, &item.ProviderName, &item.TargetModel, &item.Protocol, &endpoint, &item.Status,
		&publishedAt, &unpublishedAt, &pricingJSON, &rpm, &tpm, &contextWindow, &createdAt, &updatedAt,
	); err != nil {
		return ModelAssetBinding{}, err
	}
	item.Endpoint = endpoint.String
	if pricingJSON.String != "" {
		item.Pricing = &ModelBindingPricing{}
		_ = json.Unmarshal([]byte(pricingJSON.String), item.Pricing)
	}
	if rpm.Valid || tpm.Valid || contextWindow.Valid {
		item.Limits = &ModelBindingLimits{}
		if rpm.Valid {
			value := int(rpm.Int64)
			item.Limits.RPM = &value
		}
		if tpm.Valid {
			value := int(tpm.Int64)
			item.Limits.TPM = &value
		}
		if contextWindow.Valid {
			value := int(contextWindow.Int64)
			item.Limits.ContextWindow = &value
		}
	}
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	if unpublishedAt.Valid {
		item.UnpublishedAt = &unpublishedAt.Time
	}
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return item, nil
}

func scanBindingPriceVersion(scanner interface{ Scan(...any) error }) (ModelBindingPriceVersion, error) {
	var (
		item          ModelBindingPriceVersion
		pricingJSON   sql.NullString
		active        bool
		effectiveFrom sql.NullTime
		effectiveTo   sql.NullTime
		createdAt     sql.NullTime
		updatedAt     sql.NullTime
	)
	if err := scanner.Scan(&item.VersionID, &item.Status, &active, &effectiveFrom, &effectiveTo, &pricingJSON, &createdAt, &updatedAt); err != nil {
		return ModelBindingPriceVersion{}, err
	}
	item.Active = active
	if pricingJSON.String != "" {
		item.Pricing = &ModelBindingPricing{}
		_ = json.Unmarshal([]byte(pricingJSON.String), item.Pricing)
	}
	if effectiveFrom.Valid {
		item.EffectiveFrom = &effectiveFrom.Time
	}
	if effectiveTo.Valid {
		item.EffectiveTo = &effectiveTo.Time
	}
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return item, nil
}

func scanAgentCatalog(scanner interface{ Scan(...any) error }) (AgentCatalogRecord, error) {
	var (
		item               AgentCatalogRecord
		intro              sql.NullString
		description        sql.NullString
		iconURL            sql.NullString
		tagsJSON           sql.NullString
		mcpServerName      sql.NullString
		transportTypesJSON sql.NullString
		resourceSummary    sql.NullString
		promptSummary      sql.NullString
		publishedAt        sql.NullTime
		unpublishedAt      sql.NullTime
		createdAt          sql.NullTime
		updatedAt          sql.NullTime
	)
	if err := scanner.Scan(
		&item.AgentID, &item.CanonicalName, &item.DisplayName, &intro, &description, &iconURL, &tagsJSON, &mcpServerName,
		&item.Status, &item.ToolCount, &transportTypesJSON, &resourceSummary, &promptSummary, &publishedAt, &unpublishedAt, &createdAt, &updatedAt,
	); err != nil {
		return AgentCatalogRecord{}, err
	}
	item.Intro = intro.String
	item.Description = description.String
	item.IconURL = iconURL.String
	item.Tags = parseJSONStringList(tagsJSON.String)
	item.McpServerName = mcpServerName.String
	item.TransportTypes = parseJSONStringList(transportTypesJSON.String)
	item.ResourceSummary = resourceSummary.String
	item.PromptSummary = promptSummary.String
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	if unpublishedAt.Valid {
		item.UnpublishedAt = &unpublishedAt.Time
	}
	if createdAt.Valid {
		item.CreatedAt = &createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = &updatedAt.Time
	}
	return item, nil
}

func normalizeModelAsset(asset ModelAsset) (ModelAsset, error) {
	asset.AssetID = strings.TrimSpace(asset.AssetID)
	asset.CanonicalName = strings.TrimSpace(asset.CanonicalName)
	asset.DisplayName = strings.TrimSpace(asset.DisplayName)
	asset.Intro = strings.TrimSpace(asset.Intro)
	if asset.AssetID == "" || asset.CanonicalName == "" || asset.DisplayName == "" {
		return ModelAsset{}, errors.New("assetId, canonicalName and displayName are required")
	}
	if asset.Capabilities == nil {
		asset.Capabilities = &ModelAssetCapabilities{}
	}
	asset.Tags = normalizeStringList(asset.Tags)
	asset.Capabilities.Modalities = normalizeStringList(asset.Capabilities.Modalities)
	asset.Capabilities.Features = normalizeStringList(asset.Capabilities.Features)
	asset.Capabilities.RequestKinds = normalizeStringList(asset.Capabilities.RequestKinds)
	return asset, nil
}

func normalizeBinding(binding ModelAssetBinding) (ModelAssetBinding, error) {
	binding.BindingID = strings.TrimSpace(binding.BindingID)
	binding.ModelID = strings.TrimSpace(binding.ModelID)
	binding.ProviderName = strings.TrimSpace(binding.ProviderName)
	binding.TargetModel = strings.TrimSpace(binding.TargetModel)
	binding.Protocol = strings.TrimSpace(binding.Protocol)
	binding.Endpoint = strings.TrimSpace(binding.Endpoint)
	if binding.BindingID == "" || binding.ModelID == "" || binding.ProviderName == "" || binding.TargetModel == "" {
		return ModelAssetBinding{}, errors.New("bindingId, modelId, providerName and targetModel are required")
	}
	if binding.Protocol == "" {
		binding.Protocol = "openai/v1"
	}
	return binding, nil
}

func normalizeAgentCatalog(asset AgentCatalogRecord) (AgentCatalogRecord, error) {
	asset.AgentID = strings.TrimSpace(asset.AgentID)
	asset.CanonicalName = strings.TrimSpace(asset.CanonicalName)
	asset.DisplayName = strings.TrimSpace(asset.DisplayName)
	asset.Intro = strings.TrimSpace(asset.Intro)
	asset.Description = strings.TrimSpace(asset.Description)
	asset.IconURL = strings.TrimSpace(asset.IconURL)
	asset.McpServerName = strings.TrimSpace(asset.McpServerName)
	asset.Tags = normalizeStringList(asset.Tags)
	if asset.AgentID == "" || asset.CanonicalName == "" || asset.DisplayName == "" {
		return AgentCatalogRecord{}, errors.New("agentId, canonicalName and displayName are required")
	}
	return asset, nil
}

func normalizeStringList(items []string) []string {
	set := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, exists := set[normalized]; exists {
			continue
		}
		set[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func parseJSONStringList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return []string{}
	}
	return items
}

func anySliceToStringSlice(items []any) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		value := strings.TrimSpace(fmt.Sprint(item))
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}

func intOrNil(limits *ModelBindingLimits, pick func(ModelBindingLimits) *int) any {
	if limits == nil {
		return nil
	}
	value := pick(*limits)
	if value == nil {
		return nil
	}
	return *value
}

func mustJSONString(value any) string {
	if value == nil {
		return ""
	}
	bytes, _ := json.Marshal(value)
	return string(bytes)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func defaultJSON(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func isSupportedGrantSubject(subjectType string) bool {
	switch subjectType {
	case "consumer", "department", "user_level":
		return true
	default:
		return false
	}
}
