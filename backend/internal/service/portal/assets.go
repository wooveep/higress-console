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

	"github.com/google/uuid"

	"github.com/wooveep/aigateway-console/backend/internal/model/do"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

var (
	defaultModelTypes        = []string{"text", "multimodal", "image_generation", "video_generation", "speech_recognition", "speech_synthesis", "embedding"}
	defaultInputModalities   = []string{"text", "image", "video", "audio"}
	defaultOutputModalities  = []string{"text", "image", "video", "audio", "embedding"}
	defaultModelFeatures     = []string{"reasoning", "vision", "function_calling", "structured_output", "web_search", "prefix_completion", "prompt_cache", "batch_inference", "fine_tuning", "long_context", "model_experience"}
	defaultModelModalities   = []string{"text", "image", "audio", "video", "embedding"}
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

type PublishedBindingOption struct {
	AssetID      string `json:"assetId"`
	BindingID    string `json:"bindingId"`
	ModelID      string `json:"modelId"`
	TargetModel  string `json:"targetModel"`
	DisplayLabel string `json:"displayLabel"`
}

type PublishedBindingCatalog struct {
	ProviderName string                   `json:"providerName"`
	Bindings     []PublishedBindingOption `json:"bindings"`
}

type ModelAssetOptions struct {
	Capabilities struct {
		ModelTypes       []string `json:"modelTypes"`
		InputModalities  []string `json:"inputModalities"`
		OutputModalities []string `json:"outputModalities"`
		FeatureFlags     []string `json:"featureFlags"`
		Modalities       []string `json:"modalities"`
		Features         []string `json:"features"`
		RequestKinds     []string `json:"requestKinds"`
	} `json:"capabilities"`
	ProviderModels    []ProviderModelCatalog    `json:"providerModels"`
	PublishedBindings []PublishedBindingCatalog `json:"publishedBindings"`
}

type ModelAssetCapabilities struct {
	InputModalities  []string `json:"inputModalities,omitempty"`
	OutputModalities []string `json:"outputModalities,omitempty"`
	FeatureFlags     []string `json:"featureFlags,omitempty"`
	Modalities       []string `json:"modalities,omitempty"`
	Features         []string `json:"features,omitempty"`
	RequestKinds     []string `json:"requestKinds,omitempty"`
}

type ModelBindingPricing struct {
	Currency                                                   string   `json:"currency,omitempty"`
	InputCostPerMillionTokens                                  *float64 `json:"inputCostPerMillionTokens,omitempty"`
	OutputCostPerMillionTokens                                 *float64 `json:"outputCostPerMillionTokens,omitempty"`
	PricePerImage                                              *float64 `json:"pricePerImage,omitempty"`
	PricePerSecond                                             *float64 `json:"pricePerSecond,omitempty"`
	PricePerSecond720p                                         *float64 `json:"pricePerSecond720p,omitempty"`
	PricePerSecond1080p                                        *float64 `json:"pricePerSecond1080p,omitempty"`
	PricePer10kChars                                           *float64 `json:"pricePer10kChars,omitempty"`
	InputCostPerRequest                                        *float64 `json:"inputCostPerRequest,omitempty"`
	CacheCreationInputTokenCostPerMillionTokens                *float64 `json:"cacheCreationInputTokenCostPerMillionTokens,omitempty"`
	CacheCreationInputTokenCostAbove1hrPerMillionTokens        *float64 `json:"cacheCreationInputTokenCostAbove1hrPerMillionTokens,omitempty"`
	CacheReadInputTokenCostPerMillionTokens                    *float64 `json:"cacheReadInputTokenCostPerMillionTokens,omitempty"`
	InputCostPerMillionTokensAbove200kTokens                   *float64 `json:"inputCostPerMillionTokensAbove200kTokens,omitempty"`
	OutputCostPerMillionTokensAbove200kTokens                  *float64 `json:"outputCostPerMillionTokensAbove200kTokens,omitempty"`
	CacheCreationInputTokenCostPerMillionTokensAbove200kTokens *float64 `json:"cacheCreationInputTokenCostPerMillionTokensAbove200kTokens,omitempty"`
	CacheReadInputTokenCostPerMillionTokensAbove200kTokens     *float64 `json:"cacheReadInputTokenCostPerMillionTokensAbove200kTokens,omitempty"`
	OutputCostPerImage                                         *float64 `json:"outputCostPerImage,omitempty"`
	OutputImageTokenCostPerMillionTokens                       *float64 `json:"outputImageTokenCostPerMillionTokens,omitempty"`
	InputCostPerImage                                          *float64 `json:"inputCostPerImage,omitempty"`
	InputImageTokenCostPerMillionTokens                        *float64 `json:"inputImageTokenCostPerMillionTokens,omitempty"`
	SupportsPromptCaching                                      bool     `json:"supportsPromptCaching,omitempty"`
}

type modelBindingPricingAlias struct {
	Currency                                                   string   `json:"currency,omitempty"`
	InputCostPerMillionTokens                                  *float64 `json:"inputCostPerMillionTokens,omitempty"`
	OutputCostPerMillionTokens                                 *float64 `json:"outputCostPerMillionTokens,omitempty"`
	PricePerImage                                              *float64 `json:"pricePerImage,omitempty"`
	PricePerSecond                                             *float64 `json:"pricePerSecond,omitempty"`
	PricePerSecond720p                                         *float64 `json:"pricePerSecond720p,omitempty"`
	PricePerSecond1080p                                        *float64 `json:"pricePerSecond1080p,omitempty"`
	PricePer10kChars                                           *float64 `json:"pricePer10kChars,omitempty"`
	InputCostPerRequest                                        *float64 `json:"inputCostPerRequest,omitempty"`
	CacheCreationInputTokenCostPerMillionTokens                *float64 `json:"cacheCreationInputTokenCostPerMillionTokens,omitempty"`
	CacheCreationInputTokenCostAbove1hrPerMillionTokens        *float64 `json:"cacheCreationInputTokenCostAbove1hrPerMillionTokens,omitempty"`
	CacheReadInputTokenCostPerMillionTokens                    *float64 `json:"cacheReadInputTokenCostPerMillionTokens,omitempty"`
	InputCostPerMillionTokensAbove200kTokens                   *float64 `json:"inputCostPerMillionTokensAbove200kTokens,omitempty"`
	OutputCostPerMillionTokensAbove200kTokens                  *float64 `json:"outputCostPerMillionTokensAbove200kTokens,omitempty"`
	CacheCreationInputTokenCostPerMillionTokensAbove200kTokens *float64 `json:"cacheCreationInputTokenCostPerMillionTokensAbove200kTokens,omitempty"`
	CacheReadInputTokenCostPerMillionTokensAbove200kTokens     *float64 `json:"cacheReadInputTokenCostPerMillionTokensAbove200kTokens,omitempty"`
	OutputCostPerImage                                         *float64 `json:"outputCostPerImage,omitempty"`
	OutputImageTokenCostPerMillionTokens                       *float64 `json:"outputImageTokenCostPerMillionTokens,omitempty"`
	InputCostPerImage                                          *float64 `json:"inputCostPerImage,omitempty"`
	InputImageTokenCostPerMillionTokens                        *float64 `json:"inputImageTokenCostPerMillionTokens,omitempty"`
	SupportsPromptCaching                                      bool     `json:"supportsPromptCaching,omitempty"`

	InputCostPerToken                          *float64 `json:"inputCostPerToken,omitempty"`
	OutputCostPerToken                         *float64 `json:"outputCostPerToken,omitempty"`
	CacheCreationInputTokenCost                *float64 `json:"cacheCreationInputTokenCost,omitempty"`
	CacheCreationInputTokenCostAbove1hr        *float64 `json:"cacheCreationInputTokenCostAbove1hr,omitempty"`
	CacheReadInputTokenCost                    *float64 `json:"cacheReadInputTokenCost,omitempty"`
	InputCostPerTokenAbove200kTokens           *float64 `json:"inputCostPerTokenAbove200kTokens,omitempty"`
	OutputCostPerTokenAbove200kTokens          *float64 `json:"outputCostPerTokenAbove200kTokens,omitempty"`
	CacheCreationInputTokenCostAbove200kTokens *float64 `json:"cacheCreationInputTokenCostAbove200kTokens,omitempty"`
	CacheReadInputTokenCostAbove200kTokens     *float64 `json:"cacheReadInputTokenCostAbove200kTokens,omitempty"`
	OutputCostPerImageToken                    *float64 `json:"outputCostPerImageToken,omitempty"`
	InputCostPerImageToken                     *float64 `json:"inputCostPerImageToken,omitempty"`
}

func (p *ModelBindingPricing) UnmarshalJSON(data []byte) error {
	var alias modelBindingPricingAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	p.Currency = alias.Currency
	p.InputCostPerMillionTokens = firstNonNilFloat(alias.InputCostPerMillionTokens, alias.InputCostPerToken)
	p.OutputCostPerMillionTokens = firstNonNilFloat(alias.OutputCostPerMillionTokens, alias.OutputCostPerToken)
	p.PricePerImage = alias.PricePerImage
	p.PricePerSecond = alias.PricePerSecond
	p.PricePerSecond720p = alias.PricePerSecond720p
	p.PricePerSecond1080p = alias.PricePerSecond1080p
	p.PricePer10kChars = alias.PricePer10kChars
	p.InputCostPerRequest = alias.InputCostPerRequest
	p.CacheCreationInputTokenCostPerMillionTokens = firstNonNilFloat(alias.CacheCreationInputTokenCostPerMillionTokens, alias.CacheCreationInputTokenCost)
	p.CacheCreationInputTokenCostAbove1hrPerMillionTokens = firstNonNilFloat(alias.CacheCreationInputTokenCostAbove1hrPerMillionTokens, alias.CacheCreationInputTokenCostAbove1hr)
	p.CacheReadInputTokenCostPerMillionTokens = firstNonNilFloat(alias.CacheReadInputTokenCostPerMillionTokens, alias.CacheReadInputTokenCost)
	p.InputCostPerMillionTokensAbove200kTokens = firstNonNilFloat(alias.InputCostPerMillionTokensAbove200kTokens, alias.InputCostPerTokenAbove200kTokens)
	p.OutputCostPerMillionTokensAbove200kTokens = firstNonNilFloat(alias.OutputCostPerMillionTokensAbove200kTokens, alias.OutputCostPerTokenAbove200kTokens)
	p.CacheCreationInputTokenCostPerMillionTokensAbove200kTokens = firstNonNilFloat(alias.CacheCreationInputTokenCostPerMillionTokensAbove200kTokens, alias.CacheCreationInputTokenCostAbove200kTokens)
	p.CacheReadInputTokenCostPerMillionTokensAbove200kTokens = firstNonNilFloat(alias.CacheReadInputTokenCostPerMillionTokensAbove200kTokens, alias.CacheReadInputTokenCostAbove200kTokens)
	p.OutputCostPerImage = alias.OutputCostPerImage
	p.OutputImageTokenCostPerMillionTokens = firstNonNilFloat(alias.OutputImageTokenCostPerMillionTokens, alias.OutputCostPerImageToken)
	p.InputCostPerImage = alias.InputCostPerImage
	p.InputImageTokenCostPerMillionTokens = firstNonNilFloat(alias.InputImageTokenCostPerMillionTokens, alias.InputCostPerImageToken)
	p.SupportsPromptCaching = alias.SupportsPromptCaching
	return nil
}

func firstNonNilFloat(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

type ModelBindingLimits struct {
	MaxInputTokens                 *int `json:"maxInputTokens,omitempty"`
	MaxOutputTokens                *int `json:"maxOutputTokens,omitempty"`
	ContextWindowTokens            *int `json:"contextWindowTokens,omitempty"`
	MaxReasoningTokens             *int `json:"maxReasoningTokens,omitempty"`
	MaxInputTokensInReasoningMode  *int `json:"maxInputTokensInReasoningMode,omitempty"`
	MaxOutputTokensInReasoningMode *int `json:"maxOutputTokensInReasoningMode,omitempty"`
	RPM                            *int `json:"rpm,omitempty"`
	TPM                            *int `json:"tpm,omitempty"`
	ContextWindow                  *int `json:"contextWindow,omitempty"`
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
	ModelType     string                  `json:"modelType,omitempty"`
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
	items, err := newPortalStore(db, s.client.Driver()).listAssetGrants(ctx, strings.TrimSpace(assetType), strings.TrimSpace(assetID))
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
	if err := newPortalStore(db, s.client.Driver()).replaceAssetGrants(ctx, normalizedType, normalizedID, normalizedGrants); err != nil {
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
		SELECT asset_id, canonical_name, display_name, intro, model_type, tags_json, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at
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
		SELECT asset_id, canonical_name, display_name, intro, model_type, tags_json, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at
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
		ProviderModels:    []ProviderModelCatalog{},
		PublishedBindings: []PublishedBindingCatalog{},
	}
	options.Capabilities.ModelTypes = append([]string{}, defaultModelTypes...)
	options.Capabilities.InputModalities = append([]string{}, defaultInputModalities...)
	options.Capabilities.OutputModalities = append([]string{}, defaultOutputModalities...)
	options.Capabilities.FeatureFlags = append([]string{}, defaultModelFeatures...)
	options.Capabilities.Modalities = append([]string{}, defaultModelModalities...)
	options.Capabilities.Features = append([]string{}, defaultModelFeatures...)
	options.Capabilities.RequestKinds = append([]string{}, defaultModelRequestKinds...)

	db, err := s.db(ctx)
	if err == nil {
		options.PublishedBindings, _ = s.listPublishedBindingOptions(ctx, db)
	}
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
			asset_id, canonical_name, display_name, intro, model_type, tags_json, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		normalized.AssetID,
		normalized.CanonicalName,
		normalized.DisplayName,
		normalized.Intro,
		normalized.ModelType,
		mustJSONString(normalized.Tags),
		mustJSONString(normalized.Capabilities.InputModalities),
		mustJSONString(normalized.Capabilities.OutputModalities),
		mustJSONString(normalized.Capabilities.FeatureFlags),
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
		SET canonical_name = ?, display_name = ?, intro = ?, model_type = ?, tags_json = ?, input_modalities_json = ?, output_modalities_json = ?, feature_flags_json = ?, modalities_json = ?, features_json = ?, request_kinds_json = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ?`,
		normalized.CanonicalName,
		normalized.DisplayName,
		normalized.Intro,
		normalized.ModelType,
		mustJSONString(normalized.Tags),
		mustJSONString(normalized.Capabilities.InputModalities),
		mustJSONString(normalized.Capabilities.OutputModalities),
		mustJSONString(normalized.Capabilities.FeatureFlags),
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
	modelType, err := s.getModelAssetType(ctx, db, assetID)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO portal_model_binding (
			binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, limits_json, rpm, tpm, context_window, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft')`,
		normalized.BindingID,
		assetID,
		normalized.ModelID,
		normalized.ProviderName,
		normalized.TargetModel,
		firstNonEmpty(normalized.Protocol, "openai/v1"),
		firstNonEmpty(normalized.Endpoint, "-"),
		mustJSONString(canonicalizeModelBindingPricingForType(normalized.Pricing, modelType)),
		mustJSONString(canonicalizeModelBindingLimits(normalized.Limits)),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.RPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.TPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return preferredContextWindowValue(l) }),
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
	modelType, err := s.getModelAssetType(ctx, db, assetID)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_binding
		SET model_id = ?, provider_name = ?, target_model = ?, protocol = ?, endpoint = ?, pricing_json = ?, limits_json = ?, rpm = ?, tpm = ?, context_window = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`,
		normalized.ModelID,
		normalized.ProviderName,
		normalized.TargetModel,
		firstNonEmpty(normalized.Protocol, "openai/v1"),
		firstNonEmpty(normalized.Endpoint, "-"),
		mustJSONString(canonicalizeModelBindingPricingForType(normalized.Pricing, modelType)),
		mustJSONString(canonicalizeModelBindingLimits(normalized.Limits)),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.RPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return l.TPM }),
		intOrNil(normalized.Limits, func(l ModelBindingLimits) *int { return preferredContextWindowValue(l) }),
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
	modelType, err := s.getModelAssetType(ctx, db, assetID)
	if err != nil {
		return nil, err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	now := time.Now()
	if _, err := tx.ExecContext(ctx, `
		UPDATE portal_model_binding_price_version
		SET active = FALSE, effective_to = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ? AND active = ?`,
		now, strings.TrimSpace(assetID), strings.TrimSpace(bindingID), true); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO portal_model_binding_price_version (
			asset_id, binding_id, status, active, effective_from, pricing_json
		) VALUES (?, ?, 'active', ?, ?, ?)`,
		strings.TrimSpace(assetID), strings.TrimSpace(bindingID), true, now, mustJSONString(canonicalizeModelBindingPricingForType(binding.Pricing, modelType))); err != nil {
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
		SET active = FALSE, effective_to = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ? AND active = ?`,
		now, strings.TrimSpace(assetID), strings.TrimSpace(bindingID), true); err != nil {
		return nil, err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-unpublish"); err != nil {
		return nil, err
	}
	return s.getBinding(ctx, assetID, bindingID)
}

func (s *Service) DeleteModelAsset(ctx context.Context, assetID string) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	assetID = strings.TrimSpace(assetID)
	if assetID == "" {
		return errors.New("assetId is required")
	}
	var bindingCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM portal_model_binding
		WHERE asset_id = ?`, assetID).Scan(&bindingCount); err != nil {
		return err
	}
	if bindingCount > 0 {
		return fmt.Errorf("模型资产 %s 仍有 %d 个绑定，需先删除绑定后再删除资产", assetID, bindingCount)
	}
	result, err := db.ExecContext(ctx, `
		DELETE FROM portal_model_asset
		WHERE asset_id = ?`, assetID)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("model asset not found")
	}
	if err := s.hook.AfterWrite(ctx, "model-asset-delete"); err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteModelBinding(ctx context.Context, assetID, bindingID string) error {
	db, err := s.db(ctx)
	if err != nil {
		return err
	}
	assetID = strings.TrimSpace(assetID)
	bindingID = strings.TrimSpace(bindingID)
	if assetID == "" || bindingID == "" {
		return errors.New("assetId and bindingId are required")
	}
	binding, err := s.getBinding(ctx, assetID, bindingID)
	if err != nil {
		return err
	}
	if binding == nil {
		return errors.New("model binding not found")
	}
	if strings.EqualFold(strings.TrimSpace(binding.Status), "published") {
		return fmt.Errorf("绑定 %s 当前已发布，需先下架后再删除", bindingID)
	}

	var grantCount int
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(1)
		FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`,
		"model_binding", bindingID,
	).Scan(&grantCount); err != nil {
		return err
	}
	if grantCount > 0 {
		return fmt.Errorf("绑定 %s 仍有 %d 条授权记录，需先清理授权后再删除", bindingID, grantCount)
	}

	referencedRoutes, err := s.findAIRouteReferencesForBinding(ctx, binding.ProviderName, binding.TargetModel)
	if err != nil {
		return err
	}
	if len(referencedRoutes) > 0 {
		return fmt.Errorf("绑定 %s 仍被 AI 路由引用：%s", bindingID, strings.Join(referencedRoutes, "、"))
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`,
		"model_binding", bindingID,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM portal_model_binding_price_version
		WHERE asset_id = ? AND binding_id = ?`,
		assetID, bindingID,
	); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`,
		assetID, bindingID,
	)
	if err != nil {
		return err
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return errors.New("model binding not found")
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if err := s.hook.AfterWrite(ctx, "model-binding-delete"); err != nil {
		return err
	}
	return nil
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
	canonicalPricingJSON := pricingJSON.String
	if strings.TrimSpace(pricingJSON.String) != "" {
		modelType, typeErr := s.getModelAssetType(ctx, db, assetID)
		if typeErr != nil {
			return nil, typeErr
		}
		pricing := &ModelBindingPricing{}
		if err := json.Unmarshal([]byte(pricingJSON.String), pricing); err != nil {
			return nil, err
		}
		canonicalPricingJSON = mustJSONString(canonicalizeModelBindingPricingForType(pricing, modelType))
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE portal_model_binding
		SET pricing_json = ?, status = 'draft', updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`,
		canonicalPricingJSON, strings.TrimSpace(assetID), strings.TrimSpace(bindingID)); err != nil {
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
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
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
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
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
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
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

func (s *Service) listPublishedBindingOptions(ctx context.Context, db *sql.DB) ([]PublishedBindingCatalog, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT asset_id, binding_id, model_id, provider_name, target_model
		FROM portal_model_binding
		WHERE status = 'published'
		ORDER BY provider_name, model_id, binding_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byProvider := make(map[string][]PublishedBindingOption)
	for rows.Next() {
		var item PublishedBindingOption
		var providerName string
		if err := rows.Scan(&item.AssetID, &item.BindingID, &item.ModelID, &providerName, &item.TargetModel); err != nil {
			return nil, err
		}
		providerName = strings.TrimSpace(providerName)
		if providerName == "" {
			continue
		}
		item.DisplayLabel = item.ModelID
		if item.TargetModel != "" && item.TargetModel != item.ModelID {
			item.DisplayLabel = fmt.Sprintf("%s / %s", item.ModelID, item.TargetModel)
		}
		byProvider[providerName] = append(byProvider[providerName], item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	providers := make([]string, 0, len(byProvider))
	for providerName := range byProvider {
		providers = append(providers, providerName)
	}
	sort.Strings(providers)

	result := make([]PublishedBindingCatalog, 0, len(providers))
	for _, providerName := range providers {
		bindings := byProvider[providerName]
		sort.Slice(bindings, func(i, j int) bool {
			if bindings[i].ModelID == bindings[j].ModelID {
				return bindings[i].BindingID < bindings[j].BindingID
			}
			return bindings[i].ModelID < bindings[j].ModelID
		})
		result = append(result, PublishedBindingCatalog{
			ProviderName: providerName,
			Bindings:     bindings,
		})
	}
	return result, nil
}

func (s *Service) findAIRouteReferencesForBinding(ctx context.Context, providerName, targetModel string) ([]string, error) {
	if s.k8sClient == nil {
		return nil, nil
	}
	providerName = strings.TrimSpace(providerName)
	targetModel = strings.TrimSpace(targetModel)
	if providerName == "" || targetModel == "" {
		return nil, nil
	}
	routes, err := s.k8sClient.ListResources(ctx, "ai-routes")
	if err != nil {
		return nil, nil
	}
	referenced := make([]string, 0)
	for _, route := range routes {
		if aiRouteReferencesBinding(route, providerName, targetModel) {
			name := strings.TrimSpace(fmt.Sprint(route["name"]))
			if name != "" {
				referenced = append(referenced, name)
			}
		}
	}
	sort.Strings(referenced)
	return referenced, nil
}

func aiRouteReferencesBinding(route map[string]any, providerName, targetModel string) bool {
	for _, upstream := range toMapSlice(route["upstreams"]) {
		if aiRouteUpstreamReferencesBinding(upstream, providerName, targetModel) {
			return true
		}
	}
	fallback := mapValue(route["fallbackConfig"])
	for _, upstream := range toMapSlice(fallback["upstreams"]) {
		if aiRouteUpstreamReferencesBinding(upstream, providerName, targetModel) {
			return true
		}
	}
	return false
}

func aiRouteUpstreamReferencesBinding(upstream map[string]any, providerName, targetModel string) bool {
	if strings.TrimSpace(fmt.Sprint(upstream["provider"])) != providerName {
		return false
	}
	for _, mappedTarget := range mapValue(upstream["modelMapping"]) {
		if strings.TrimSpace(fmt.Sprint(mappedTarget)) == targetModel {
			return true
		}
	}
	return false
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

func (s *Service) getModelAssetType(ctx context.Context, db *sql.DB, assetID string) (string, error) {
	row := db.QueryRowContext(ctx, `
		SELECT model_type, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		FROM portal_model_asset
		WHERE asset_id = ?`, strings.TrimSpace(assetID))
	var (
		modelType            sql.NullString
		inputModalitiesJSON  sql.NullString
		outputModalitiesJSON sql.NullString
		featureFlagsJSON     sql.NullString
		modalitiesJSON       sql.NullString
		featuresJSON         sql.NullString
		requestKindsJSON     sql.NullString
	)
	if err := row.Scan(&modelType, &inputModalitiesJSON, &outputModalitiesJSON, &featureFlagsJSON, &modalitiesJSON, &featuresJSON, &requestKindsJSON); err != nil {
		return "", err
	}
	normalizedType := normalizeModelType(modelType.String)
	if normalizedType != "" {
		return normalizedType, nil
	}
	inferredType := inferModelTypeFromCapabilities(normalizeCapabilities("", &ModelAssetCapabilities{
		InputModalities:  parseJSONStringList(inputModalitiesJSON.String),
		OutputModalities: parseJSONStringList(outputModalitiesJSON.String),
		FeatureFlags:     parseJSONStringList(featureFlagsJSON.String),
		Modalities:       parseJSONStringList(modalitiesJSON.String),
		Features:         parseJSONStringList(featuresJSON.String),
		RequestKinds:     parseJSONStringList(requestKindsJSON.String),
	}))
	if inferredType == "" {
		return "", errors.New("modelType is required")
	}
	return inferredType, nil
}

func scanModelAsset(scanner interface{ Scan(...any) error }) (ModelAsset, error) {
	var (
		item                 ModelAsset
		intro                sql.NullString
		modelType            sql.NullString
		tagsJSON             sql.NullString
		inputModalitiesJSON  sql.NullString
		outputModalitiesJSON sql.NullString
		featureFlagsJSON     sql.NullString
		modalitiesJSON       sql.NullString
		featuresJSON         sql.NullString
		requestKindsJSON     sql.NullString
		createdAt            sql.NullTime
		updatedAt            sql.NullTime
	)
	if err := scanner.Scan(&item.AssetID, &item.CanonicalName, &item.DisplayName, &intro, &modelType, &tagsJSON, &inputModalitiesJSON, &outputModalitiesJSON, &featureFlagsJSON, &modalitiesJSON, &featuresJSON, &requestKindsJSON, &createdAt, &updatedAt); err != nil {
		return ModelAsset{}, err
	}
	item.Intro = intro.String
	item.ModelType = normalizeModelType(modelType.String)
	item.Tags = parseJSONStringList(tagsJSON.String)
	item.Capabilities = normalizeCapabilities(item.ModelType, &ModelAssetCapabilities{
		InputModalities:  parseJSONStringList(inputModalitiesJSON.String),
		OutputModalities: parseJSONStringList(outputModalitiesJSON.String),
		FeatureFlags:     parseJSONStringList(featureFlagsJSON.String),
		Modalities:       parseJSONStringList(modalitiesJSON.String),
		Features:         parseJSONStringList(featuresJSON.String),
		RequestKinds:     parseJSONStringList(requestKindsJSON.String),
	})
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
		limitsJSON    sql.NullString
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
		&publishedAt, &unpublishedAt, &pricingJSON, &limitsJSON, &rpm, &tpm, &contextWindow, &createdAt, &updatedAt,
	); err != nil {
		return ModelAssetBinding{}, err
	}
	item.Endpoint = endpoint.String
	if pricingJSON.String != "" {
		item.Pricing = &ModelBindingPricing{}
		_ = json.Unmarshal([]byte(pricingJSON.String), item.Pricing)
		item.Pricing = canonicalizeModelBindingPricing(item.Pricing)
	}
	if limitsJSON.String != "" {
		item.Limits = &ModelBindingLimits{}
		_ = json.Unmarshal([]byte(limitsJSON.String), item.Limits)
		item.Limits = canonicalizeModelBindingLimits(item.Limits)
	}
	if item.Limits == nil && (rpm.Valid || tpm.Valid || contextWindow.Valid) {
		item.Limits = &ModelBindingLimits{}
	}
	if item.Limits != nil {
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
			if item.Limits.ContextWindowTokens == nil {
				item.Limits.ContextWindowTokens = &value
			}
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
		item.Pricing = canonicalizeModelBindingPricing(item.Pricing)
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
	asset.ModelType = normalizeModelType(asset.ModelType)
	if asset.AssetID == "" {
		asset.AssetID = uuid.NewString()
	}
	if asset.CanonicalName == "" || asset.DisplayName == "" {
		return ModelAsset{}, errors.New("canonicalName and displayName are required")
	}
	if asset.ModelType == "" {
		asset.ModelType = inferModelTypeFromCapabilities(asset.Capabilities)
		if asset.ModelType == "" {
			return ModelAsset{}, errors.New("modelType is required")
		}
	}
	if asset.Capabilities == nil {
		asset.Capabilities = &ModelAssetCapabilities{}
	}
	asset.Tags = normalizeStringList(asset.Tags)
	asset.Capabilities = normalizeCapabilities(asset.ModelType, asset.Capabilities)
	return asset, nil
}

func normalizeBinding(binding ModelAssetBinding) (ModelAssetBinding, error) {
	binding.BindingID = strings.TrimSpace(binding.BindingID)
	binding.ModelID = strings.TrimSpace(binding.ModelID)
	binding.ProviderName = strings.TrimSpace(binding.ProviderName)
	binding.TargetModel = strings.TrimSpace(binding.TargetModel)
	binding.Protocol = strings.TrimSpace(binding.Protocol)
	binding.Endpoint = strings.TrimSpace(binding.Endpoint)
	if binding.BindingID == "" {
		binding.BindingID = uuid.NewString()
	}
	if binding.ModelID == "" || binding.ProviderName == "" || binding.TargetModel == "" {
		return ModelAssetBinding{}, errors.New("modelId, providerName and targetModel are required")
	}
	if binding.Protocol == "" {
		binding.Protocol = "openai/v1"
	}
	binding.Pricing = canonicalizeModelBindingPricing(binding.Pricing)
	binding.Limits = canonicalizeModelBindingLimits(binding.Limits)
	return binding, nil
}

func canonicalizeModelBindingPricing(pricing *ModelBindingPricing) *ModelBindingPricing {
	if pricing == nil {
		return nil
	}
	canonical := *pricing
	canonical.Currency = firstNonEmpty(canonical.Currency, "CNY")
	return &canonical
}

func canonicalizeModelBindingPricingForType(pricing *ModelBindingPricing, modelType string) *ModelBindingPricing {
	if pricing == nil {
		return nil
	}
	canonical := canonicalizeModelBindingPricing(pricing)
	if canonical == nil {
		return nil
	}
	modelType = normalizeModelType(modelType)
	filtered := &ModelBindingPricing{Currency: canonical.Currency}
	switch modelType {
	case "text", "multimodal":
		filtered.InputCostPerMillionTokens = canonical.InputCostPerMillionTokens
		filtered.OutputCostPerMillionTokens = canonical.OutputCostPerMillionTokens
		filtered.CacheCreationInputTokenCostPerMillionTokens = canonical.CacheCreationInputTokenCostPerMillionTokens
		filtered.CacheReadInputTokenCostPerMillionTokens = canonical.CacheReadInputTokenCostPerMillionTokens
		filtered.SupportsPromptCaching = canonical.SupportsPromptCaching
	case "embedding":
		filtered.InputCostPerMillionTokens = canonical.InputCostPerMillionTokens
	case "image_generation":
		filtered.PricePerImage = firstNonNilFloat(canonical.PricePerImage, canonical.OutputCostPerImage)
	case "video_generation":
		filtered.PricePerSecond720p = canonical.PricePerSecond720p
		filtered.PricePerSecond1080p = canonical.PricePerSecond1080p
	case "speech_recognition":
		filtered.PricePerSecond = canonical.PricePerSecond
	case "speech_synthesis":
		filtered.PricePer10kChars = canonical.PricePer10kChars
	default:
		return canonical
	}
	return filtered
}

func canonicalizeModelBindingLimits(limits *ModelBindingLimits) *ModelBindingLimits {
	if limits == nil {
		return nil
	}
	canonical := *limits
	if canonical.ContextWindowTokens == nil {
		canonical.ContextWindowTokens = canonical.ContextWindow
	}
	if canonical.ContextWindow == nil {
		canonical.ContextWindow = canonical.ContextWindowTokens
	}
	return &canonical
}

func normalizeCapabilities(modelType string, capabilities *ModelAssetCapabilities) *ModelAssetCapabilities {
	if capabilities == nil {
		capabilities = &ModelAssetCapabilities{}
	}
	normalized := &ModelAssetCapabilities{
		InputModalities:  normalizeStringList(capabilities.InputModalities),
		OutputModalities: normalizeStringList(capabilities.OutputModalities),
		FeatureFlags:     normalizeStringList(capabilities.FeatureFlags),
		Modalities:       normalizeStringList(capabilities.Modalities),
		Features:         normalizeStringList(capabilities.Features),
		RequestKinds:     normalizeStringList(capabilities.RequestKinds),
	}
	if len(normalized.InputModalities) == 0 && len(normalized.Modalities) > 0 {
		normalized.InputModalities = append([]string{}, normalized.Modalities...)
	}
	if len(normalized.OutputModalities) == 0 && len(normalized.Modalities) > 0 {
		normalized.OutputModalities = append([]string{}, normalized.Modalities...)
	}
	if len(normalized.FeatureFlags) == 0 && len(normalized.Features) > 0 {
		normalized.FeatureFlags = append([]string{}, normalized.Features...)
	}
	if len(normalized.Modalities) == 0 {
		normalized.Modalities = normalizeStringList(append(append([]string{}, normalized.InputModalities...), normalized.OutputModalities...))
	}
	if len(normalized.Features) == 0 {
		normalized.Features = append([]string{}, normalized.FeatureFlags...)
	}
	if len(normalized.RequestKinds) == 0 {
		normalized.RequestKinds = defaultRequestKindsForModelType(modelType)
	}
	return normalized
}

func normalizeModelType(modelType string) string {
	normalized := strings.TrimSpace(strings.ToLower(modelType))
	switch normalized {
	case "text", "multimodal", "image_generation", "video_generation", "speech_recognition", "speech_synthesis", "embedding":
		return normalized
	default:
		return ""
	}
}

func inferModelTypeFromCapabilities(capabilities *ModelAssetCapabilities) string {
	if capabilities == nil {
		return ""
	}
	inputs := normalizeStringList(append([]string{}, capabilities.InputModalities...))
	outputs := normalizeStringList(append([]string{}, capabilities.OutputModalities...))
	modalities := normalizeStringList(append(append([]string{}, inputs...), outputs...))
	if len(modalities) == 0 {
		modalities = normalizeStringList(capabilities.Modalities)
	}
	features := normalizeStringList(append([]string{}, capabilities.FeatureFlags...))
	if len(features) == 0 {
		features = normalizeStringList(capabilities.Features)
	}
	requestKinds := normalizeStringList(capabilities.RequestKinds)

	hasInput := func(target string) bool { return containsString(inputs, target) || containsString(modalities, target) }
	hasOutput := func(target string) bool { return containsString(outputs, target) }
	hasFeature := func(target string) bool { return containsString(features, target) }
	hasRequestKind := func(target string) bool { return containsString(requestKinds, target) }

	switch {
	case hasOutput("embedding") || hasRequestKind("embeddings"):
		return "embedding"
	case hasOutput("image") && !hasOutput("text") && !hasOutput("audio") && !hasOutput("video"):
		return "image_generation"
	case hasOutput("video"):
		return "video_generation"
	case hasOutput("audio"):
		return "speech_synthesis"
	case hasInput("audio") && !hasOutput("audio") && !hasOutput("text") && !hasOutput("image") && !hasOutput("video"):
		return "speech_recognition"
	case hasInput("image") || hasInput("video") || hasOutput("image") || hasOutput("video") || hasFeature("vision"):
		return "multimodal"
	case hasInput("text") || hasOutput("text") || len(requestKinds) > 0 || len(features) > 0:
		return "text"
	default:
		return ""
	}
}

func defaultRequestKindsForModelType(modelType string) []string {
	switch normalizeModelType(modelType) {
	case "embedding":
		return []string{"embeddings"}
	case "image_generation":
		return []string{"images"}
	case "video_generation":
		return []string{"video"}
	case "speech_recognition", "speech_synthesis":
		return []string{"audio"}
	default:
		return []string{"chat_completions", "responses"}
	}
}

func containsString(values []string, target string) bool {
	target = strings.TrimSpace(strings.ToLower(target))
	for _, value := range values {
		if strings.TrimSpace(strings.ToLower(value)) == target {
			return true
		}
	}
	return false
}

func preferredContextWindowValue(limits ModelBindingLimits) *int {
	if limits.ContextWindowTokens != nil {
		return limits.ContextWindowTokens
	}
	return limits.ContextWindow
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

func mapValue(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case map[string]string:
		result := make(map[string]any, len(typed))
		for key, item := range typed {
			result[key] = item
		}
		return result
	default:
		return map[string]any{}
	}
}

func toMapSlice(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		result := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			record, ok := item.(map[string]any)
			if ok {
				result = append(result, record)
			}
		}
		return result
	default:
		return nil
	}
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
