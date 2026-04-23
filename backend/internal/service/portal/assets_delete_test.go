package portal

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

func TestGetModelAssetOptionsIncludesPublishedBindings(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT asset_id, binding_id, model_id, provider_name, target_model
		FROM portal_model_binding
		WHERE status = 'published'
		ORDER BY provider_name, model_id, binding_id`)).
		WillReturnRows(sqlmock.NewRows([]string{
			"asset_id", "binding_id", "model_id", "provider_name", "target_model",
		}).AddRow("asset-gpt-4o", "binding-gpt-4o", "gpt-4o", "openai-main", "gpt-4o"))

	client := k8sclient.NewMemoryClient()
	_, err = client.UpsertResource(context.Background(), "ai-providers", "openai-main", map[string]any{
		"name":   "openai-main",
		"models": []map[string]any{{"modelId": "gpt-4o", "targetModel": "gpt-4o", "label": "GPT-4o"}},
	})
	require.NoError(t, err)

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db), client)
	options, err := svc.GetModelAssetOptions(context.Background())
	require.NoError(t, err)
	require.Len(t, options.ProviderModels, 1)
	require.Len(t, options.PublishedBindings, 1)
	require.Equal(t, "openai-main", options.PublishedBindings[0].ProviderName)
	require.Len(t, options.PublishedBindings[0].Bindings, 1)
	require.Equal(t, "binding-gpt-4o", options.PublishedBindings[0].Bindings[0].BindingID)
	require.Equal(t, "gpt-4o", options.PublishedBindings[0].Bindings[0].DisplayLabel)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateModelBindingGeneratesBindingIDWhenMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT model_type, input_modalities_json, output_modalities_json, feature_flags_json, modalities_json, features_json, request_kinds_json
		FROM portal_model_asset
		WHERE asset_id = ?`)).
		WithArgs("asset-gpt-4o").
		WillReturnRows(sqlmock.NewRows([]string{
			"model_type", "input_modalities_json", "output_modalities_json", "feature_flags_json", "modalities_json", "features_json", "request_kinds_json",
		}).AddRow("text", `["text"]`, `["text"]`, `[]`, `["text"]`, `[]`, `["chat_completions","responses"]`))
	mock.ExpectExec(regexp.QuoteMeta(`
		INSERT INTO portal_model_binding (
			binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, pricing_json, limits_json, rpm, tpm, context_window, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft')`)).
		WithArgs(
			sqlmock.AnyArg(),
			"asset-gpt-4o",
			"gpt-4o",
			"openai-main",
			"gpt-4o",
			"openai/v1",
			"https://api.openai.com/v1",
			`{"currency":"CNY"}`,
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
		WithArgs("asset-gpt-4o", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
			"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
		}).AddRow(
			"generated-binding-id",
			"asset-gpt-4o",
			"gpt-4o",
			"openai-main",
			"gpt-4o",
			"openai/v1",
			"https://api.openai.com/v1",
			"draft",
			nil,
			nil,
			`{"currency":"CNY"}`,
			nil,
			nil,
			nil,
			nil,
			time.Now().UTC(),
			time.Now().UTC(),
		))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	item, err := svc.CreateModelBinding(context.Background(), "asset-gpt-4o", ModelAssetBinding{
		ModelID:      "gpt-4o",
		ProviderName: "openai-main",
		TargetModel:  "gpt-4o",
		Protocol:     "openai/v1",
		Endpoint:     "https://api.openai.com/v1",
		Pricing:      &ModelBindingPricing{Currency: "CNY"},
	})
	require.NoError(t, err)
	require.Equal(t, "generated-binding-id", item.BindingID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelAssetRejectsWhenBindingsExist(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM portal_model_binding
		WHERE asset_id = ?`)).
		WithArgs("asset-has-bindings").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(2))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	err = svc.DeleteModelAsset(context.Background(), "asset-has-bindings")
	require.ErrorContains(t, err, "需先删除绑定后再删除资产")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelAssetDeletesEmptyAsset(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM portal_model_binding
		WHERE asset_id = ?`)).
		WithArgs("asset-empty").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(0))
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM portal_model_asset
		WHERE asset_id = ?`)).
		WithArgs("asset-empty").
		WillReturnResult(sqlmock.NewResult(0, 1))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	require.NoError(t, svc.DeleteModelAsset(context.Background(), "asset-empty"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelBindingRejectsPublishedBinding(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnRows(bindingRows().
			AddRow("binding-1", "asset-1", "gpt-4o", "openai-main", "gpt-4o", "openai/v1", "https://api.openai.com/v1", "published", nil, nil, "", nil, nil, nil, nil, time.Now().UTC(), time.Now().UTC()))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	err = svc.DeleteModelBinding(context.Background(), "asset-1", "binding-1")
	require.ErrorContains(t, err, "需先下架后再删除")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelBindingRejectsBindingWithGrants(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnRows(bindingRows().
			AddRow("binding-1", "asset-1", "gpt-4o", "openai-main", "gpt-4o", "openai/v1", "https://api.openai.com/v1", "draft", nil, nil, "", nil, nil, nil, nil, time.Now().UTC(), time.Now().UTC()))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(1))

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	err = svc.DeleteModelBinding(context.Background(), "asset-1", "binding-1")
	require.ErrorContains(t, err, "需先清理授权后再删除")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelBindingDeletesBindingWithPriceHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnRows(bindingRows().
			AddRow("binding-1", "asset-1", "gpt-4o", "openai-main", "gpt-4o", "openai/v1", "https://api.openai.com/v1", "draft", nil, nil, "", nil, nil, nil, nil, time.Now().UTC(), time.Now().UTC()))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM portal_model_binding_price_version
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	require.NoError(t, svc.DeleteModelBinding(context.Background(), "asset-1", "binding-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelBindingRejectsAIRouteReference(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnRows(bindingRows().
			AddRow("binding-1", "asset-1", "gpt-4o", "openai-main", "gpt-4o", "openai/v1", "https://api.openai.com/v1", "draft", nil, nil, "", nil, nil, nil, nil, time.Now().UTC(), time.Now().UTC()))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(0))
	k8s := k8sclient.NewMemoryClient()
	_, err = k8s.UpsertResource(context.Background(), "ai-routes", "route-chat", map[string]any{
		"name": "route-chat",
		"upstreams": []map[string]any{
			{
				"provider":     "openai-main",
				"weight":       100,
				"modelMapping": map[string]any{"request-model": "gpt-4o"},
			},
		},
	})
	require.NoError(t, err)

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db), k8s)
	err = svc.DeleteModelBinding(context.Background(), "asset-1", "binding-1")
	require.ErrorContains(t, err, "仍被 AI 路由引用：route-chat")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteModelBindingDeletesUnreferencedDraftBinding(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	expectSchema(mock)
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT binding_id, asset_id, model_id, provider_name, target_model, protocol, endpoint, status,
			published_at, unpublished_at, pricing_json, limits_json, rpm, tpm, context_window, created_at, updated_at
		FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnRows(bindingRows().
			AddRow("binding-1", "asset-1", "gpt-4o", "openai-main", "gpt-4o", "openai/v1", "https://api.openai.com/v1", "draft", nil, nil, "", nil, nil, nil, nil, time.Now().UTC(), time.Now().UTC()))
	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT COUNT(1)
		FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnRows(sqlmock.NewRows([]string{"COUNT(1)"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM asset_grant
		WHERE asset_type = ? AND asset_id = ?`)).
		WithArgs("model_binding", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM portal_model_binding_price_version
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`
		DELETE FROM portal_model_binding
		WHERE asset_id = ? AND binding_id = ?`)).
		WithArgs("asset-1", "binding-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := New(portaldbclient.NewFromDB(portaldbclient.Config{Enabled: true, Driver: "postgres", AutoMigrate: true}, db))
	require.NoError(t, svc.DeleteModelBinding(context.Background(), "asset-1", "binding-1"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func bindingRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"binding_id", "asset_id", "model_id", "provider_name", "target_model", "protocol", "endpoint", "status",
		"published_at", "unpublished_at", "pricing_json", "limits_json", "rpm", "tpm", "context_window", "created_at", "updated_at",
	})
}
