package portal

import (
	"context"
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestReplaceAssetGrants(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM asset_grant WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO asset_grant (asset_type, asset_id, subject_type, subject_id)
			VALUES (?, ?, ?, ?)`)).
		WithArgs("model_binding", "binding-1", "consumer", "demo").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO asset_grant (asset_type, asset_id, subject_type, subject_id)
			VALUES (?, ?, ?, ?)`)).
		WithArgs("model_binding", "binding-1", "user_level", "pro").
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT asset_type, asset_id, subject_type, subject_id
		FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?
		ORDER BY subject_type, subject_id`)).
		WithArgs("model_binding", "binding-1").
		WillReturnRows(sqlmock.NewRows([]string{"asset_type", "asset_id", "subject_type", "subject_id"}).
			AddRow("model_binding", "binding-1", "consumer", "demo").
			AddRow("model_binding", "binding-1", "user_level", "pro"))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	items, err := svc.ReplaceAssetGrants(context.Background(), "model_binding", "binding-1", []AssetGrantRecord{
		{SubjectType: "consumer", SubjectID: "demo"},
		{SubjectType: "user_level", SubjectID: "pro"},
	})
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAgentCatalogOptionsFromK8s(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "mcp-servers", "knowledge-hub", map[string]any{
		"name":        "knowledge-hub",
		"description": "Search and docs",
		"type":        "OPEN_API",
		"domains":     []any{"docs.internal"},
		"consumerAuthInfo": map[string]any{
			"enable": true,
			"type":   "key-auth",
		},
		"updatedAt": time.Now().UTC().Format(time.RFC3339),
	})
	require.NoError(t, err)

	svc := New(nil, client)
	options, err := svc.GetAgentCatalogOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, options.Servers, 1)
	require.Equal(t, "knowledge-hub", options.Servers[0].McpServerName)
	require.Equal(t, "key-auth", options.Servers[0].AuthType)
	require.NotNil(t, options.Servers[0].AuthEnabled)
	require.True(t, *options.Servers[0].AuthEnabled)
}

func TestGetModelAssetOptionsSupportsStructuredProviderModels(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-providers", "doubao", map[string]any{
		"name": "doubao",
		"models": []map[string]any{
			{"modelId": "doubao-seed-2-0-pro-260215", "targetModel": "doubao-seed-2-0-pro-260215", "label": "Doubao Pro"},
		},
	})
	require.NoError(t, err)

	svc := New(nil, client)
	options, err := svc.GetModelAssetOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, options.ProviderModels, 1)
	require.Equal(t, "doubao", options.ProviderModels[0].ProviderName)
	require.Len(t, options.ProviderModels[0].Models, 1)
	require.Equal(t, "doubao-seed-2-0-pro-260215", options.ProviderModels[0].Models[0].ModelID)
}

func TestCreateModelAssetAllowsEmptyIntroAndAutoGeneratesAssetID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_model_asset (
			asset_id, canonical_name, display_name, intro, model_type, tags_json, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)).
		WithArgs(
			sqlmock.AnyArg(),
			"doubao-seed-2-0-pro-260215",
			"doubao-seed-2-0-pro",
			"",
			"text",
			`["旗舰"]`,
			`["text"]`,
			`["text"]`,
			`["long_context","reasoning"]`,
			`["text"]`,
			`["long_context","reasoning"]`,
			`["chat_completions","responses"]`,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT asset_id, canonical_name, display_name, intro, model_type, tags_json, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json, created_at, updated_at
		FROM portal_model_asset
		WHERE asset_id = ?`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"asset_id", "canonical_name", "display_name", "intro", "model_type", "tags_json", "input_modalities_json", "output_modalities_json", "feature_flags_json", "modalities_json", "features_json", "request_kinds_json", "created_at", "updated_at",
		}).AddRow(
			"generated-asset-id",
			"doubao-seed-2-0-pro-260215",
			"doubao-seed-2-0-pro",
			"",
			"text",
			`["旗舰"]`,
			`["text"]`,
			`["text"]`,
			`["long_context","reasoning"]`,
			`["text"]`,
			`["long_context","reasoning"]`,
			`["chat_completions","responses"]`,
			time.Now().UTC(),
			time.Now().UTC(),
		))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ?
		ORDER BY binding_id`)).
		WithArgs("generated-asset-id").
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
			"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
		}))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	asset, err := svc.CreateModelAsset(context.Background(), ModelAsset{
		CanonicalName: "doubao-seed-2-0-pro-260215",
		DisplayName:   "doubao-seed-2-0-pro",
		Intro:         "",
		Tags:          []string{"旗舰"},
		Capabilities: &ModelAssetCapabilities{
			Modalities:   []string{"text"},
			Features:     []string{"reasoning", "long_context"},
			RequestKinds: []string{"responses", "chat_completions"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "generated-asset-id", asset.AssetID)
	require.Equal(t, "", asset.Intro)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateModelBindingCanonicalizesLegacyPricingJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT model_type, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		FROM portal_model_asset
		WHERE asset_id = ?`)).
		WithArgs("asset-qwen-plus").
		WillReturnRows(sqlmock.NewRows([]string{
			"model_type", "input_modalities_json", "output_modalities_json", "feature_flags_json", "modalities_json", "features_json", "request_kinds_json",
		}).AddRow("text", `["text"]`, `["text"]`, `[]`, `["text"]`, `[]`, `["chat_completions","responses"]`))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_model_binding (
			binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, limits_json, rpm, tpm, context_window, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft')`)).
		WithArgs(
			sqlmock.AnyArg(),
			"asset-qwen-plus",
			"qwen-plus",
			"aliyun",
			"qwen-plus",
			"openai/v1",
			"/v1/chat/completions",
			`{"currency":"CNY","inputCostPerMillionTokens":1}`,
			`null`,
			nil,
			nil,
			nil,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-qwen-plus", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
			"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
		}).AddRow(
			"generated-binding-id",
			"asset-qwen-plus",
			"qwen-plus",
			"aliyun",
			"qwen-plus",
			"openai/v1",
			"/v1/chat/completions",
			"draft",
			nil,
			nil,
			`{"currency":"CNY","inputCostPerMillionTokens":1}`,
			nil,
			nil,
			nil,
			nil,
			time.Now().UTC(),
			time.Now().UTC(),
		))

	var pricing ModelBindingPricing
	require.NoError(t, json.Unmarshal([]byte(`{"currency":"CNY","inputCostPerToken":1}`), &pricing))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	binding, err := svc.CreateModelBinding(context.Background(), "asset-qwen-plus", ModelAssetBinding{
		ModelID:      "qwen-plus",
		ProviderName: "aliyun",
		TargetModel:  "qwen-plus",
		Protocol:     "openai/v1",
		Endpoint:     "/v1/chat/completions",
		Pricing:      &pricing,
	})
	require.NoError(t, err)
	require.Equal(t, "generated-binding-id", binding.BindingID)
	require.Equal(t, 1.0, *binding.Pricing.InputCostPerMillionTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPublishModelBindingCanonicalizesLegacyPriceVersionJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-qwen-plus", "binding-qwen-plus").
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
			"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
		}).AddRow(
			"binding-qwen-plus",
			"asset-qwen-plus",
			"qwen-plus",
			"aliyun",
			"qwen-plus",
			"openai/v1",
			"/v1/chat/completions",
			"draft",
			nil,
			nil,
			`{"currency":"CNY","inputCostPerToken":1}`,
			nil,
			nil,
			nil,
			nil,
			time.Now().UTC(),
			time.Now().UTC(),
		))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT model_type, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		FROM portal_model_asset
		WHERE asset_id = ?`)).
		WithArgs("asset-qwen-plus").
		WillReturnRows(sqlmock.NewRows([]string{
			"model_type", "input_modalities_json", "output_modalities_json", "feature_flags_json", "modalities_json", "features_json", "request_kinds_json",
		}).AddRow("text", `["text"]`, `["text"]`, `[]`, `["text"]`, `[]`, `["chat_completions","responses"]`))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_model_binding_price_version
		SET active = FALSE, effective_to = ?, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ? AND active = ?`)).
		WithArgs(sqlmock.AnyArg(), "asset-qwen-plus", "binding-qwen-plus", true).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_model_binding_price_version (
			asset_id, binding_id, status, active, effective_from, pricing_json
		) VALUES (?, ?, 'active', ?, ?, ?)`)).
		WithArgs("asset-qwen-plus", "binding-qwen-plus", true, sqlmock.AnyArg(), `{"currency":"CNY","inputCostPerMillionTokens":1}`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_model_binding
		SET status = 'published', published_at = ?, unpublished_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs(sqlmock.AnyArg(), "asset-qwen-plus", "binding-qwen-plus").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-qwen-plus", "binding-qwen-plus").
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
			"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
		}).AddRow(
			"binding-qwen-plus",
			"asset-qwen-plus",
			"qwen-plus",
			"aliyun",
			"qwen-plus",
			"openai/v1",
			"/v1/chat/completions",
			"published",
			time.Now().UTC(),
			nil,
			`{"currency":"CNY","inputCostPerToken":1}`,
			nil,
			nil,
			nil,
			nil,
			time.Now().UTC(),
			time.Now().UTC(),
		))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	binding, err := svc.PublishModelBinding(context.Background(), "asset-qwen-plus", "binding-qwen-plus")
	require.NoError(t, err)
	require.Equal(t, "published", binding.Status)
	require.Equal(t, 1.0, *binding.Pricing.InputCostPerMillionTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRestoreBindingPriceVersionCanonicalizesLegacyPricingJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT pricing_json
		FROM portal_model_binding_price_version
		WHERE asset_id = ? AND binding_id = ? AND version_id = ?`)).
		WithArgs("asset-qwen-plus", "binding-qwen-plus", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"pricing_json"}).AddRow(`{"currency":"CNY","inputCostPerToken":1}`))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT model_type, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		FROM portal_model_asset
		WHERE asset_id = ?`)).
		WithArgs("asset-qwen-plus").
		WillReturnRows(sqlmock.NewRows([]string{
			"model_type", "input_modalities_json", "output_modalities_json", "feature_flags_json", "modalities_json", "features_json", "request_kinds_json",
		}).AddRow("text", `["text"]`, `["text"]`, `[]`, `["text"]`, `[]`, `["chat_completions","responses"]`))
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE portal_model_binding
		SET pricing_json = ?, status = 'draft', updated_at = CURRENT_TIMESTAMP
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs(`{"currency":"CNY","inputCostPerMillionTokens":1}`, "asset-qwen-plus", "binding-qwen-plus").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-qwen-plus", "binding-qwen-plus").
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
			"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
		}).AddRow(
			"binding-qwen-plus",
			"asset-qwen-plus",
			"qwen-plus",
			"aliyun",
			"qwen-plus",
			"openai/v1",
			"/v1/chat/completions",
			"draft",
			nil,
			nil,
			`{"currency":"CNY","inputCostPerMillionTokens":1}`,
			nil,
			nil,
			nil,
			nil,
			time.Now().UTC(),
			time.Now().UTC(),
		))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	binding, err := svc.RestoreBindingPriceVersion(context.Background(), "asset-qwen-plus", "binding-qwen-plus", 7)
	require.NoError(t, err)
	require.Equal(t, "draft", binding.Status)
	require.Equal(t, 1.0, *binding.Pricing.InputCostPerMillionTokens)
	require.NoError(t, mock.ExpectationsWereMet())
}
