package gateway

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

type stubPortal struct {
	accounts    []portalsvc.OrgAccountRecord
	departments []*portalsvc.OrgDepartmentNode
}

func (s stubPortal) ListAccounts(ctx context.Context) ([]portalsvc.OrgAccountRecord, error) {
	return append([]portalsvc.OrgAccountRecord{}, s.accounts...), nil
}

func (s stubPortal) ListDepartmentTree(ctx context.Context) ([]*portalsvc.OrgDepartmentNode, error) {
	return append([]*portalsvc.OrgDepartmentNode{}, s.departments...), nil
}

func TestProtectedResourceDeleteIsRejected(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	err := svc.Delete(context.Background(), "routes", "default")
	require.Error(t, err)
	require.Contains(t, err.Error(), "protected")
}

func TestInternalResourceWriteIsRejected(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	_, err := svc.Save(context.Background(), "proxy-servers", map[string]any{
		"name": "mesh.internal",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "internal resource")
}

func TestPluginInstanceRoundTrip(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "wasm-plugins", "demo-plugin", map[string]any{
		"name": "demo-plugin",
	})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "routes", "demo-route", map[string]any{
		"name": "demo-route",
		"path": map[string]any{"matchType": "PRE", "matchValue": "/demo"},
		"services": []map[string]any{{
			"name": "upstream.demo",
		}},
	})
	require.NoError(t, err)
	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})

	_, err = svc.SavePluginInstance(context.Background(), "route", "demo-route", "demo-plugin", map[string]any{
		"config": map[string]any{"consumers": []string{"demo"}},
	})
	require.NoError(t, err)

	item, err := svc.GetPluginInstance(context.Background(), "route", "demo-route", "demo-plugin")
	require.NoError(t, err)
	require.Equal(t, "demo-plugin", item["name"])

	list, err := svc.ListPluginInstances(context.Background(), "route", "demo-route")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestPluginInstanceDeleteAndServiceScope(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "services", "svc-a", map[string]any{
		"name":      "svc-a",
		"namespace": "default",
		"port":      8080,
	})
	require.NoError(t, err)
	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})

	_, err = svc.SavePluginInstance(context.Background(), "service", "svc-a", "ai-statistics", map[string]any{
		"config": map[string]any{"enabled": true},
	})
	require.NoError(t, err)

	list, err := svc.ListPluginInstances(context.Background(), "service", "svc-a")
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, svc.DeletePluginInstance(context.Background(), "service", "svc-a", "ai-statistics"))
	list, err = svc.ListPluginInstances(context.Background(), "service", "svc-a")
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestAIRoutePluginInstanceRoundTrip(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-routes", "doubao", map[string]any{
		"name": "doubao",
		"pathPredicate": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/",
		},
		"upstreams": []map[string]any{{
			"provider": "doubao",
			"model":    "doubao-pro",
			"weight":   100,
		}},
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})

	_, err = svc.SavePluginInstance(context.Background(), "route", "ai-route-doubao.internal", "ai-data-masking", map[string]any{
		"enabled":           true,
		"rawConfigurations": "",
	})
	require.NoError(t, err)

	item, err := client.GetResource(context.Background(), "route-plugin-instances:ai-route-doubao.internal", "ai-data-masking")
	require.NoError(t, err)
	require.Equal(t, "ai-data-masking", item["name"])
	require.Equal(t, true, item["enabled"])

	list, err := svc.ListPluginInstances(context.Background(), "route", "ai-route-doubao.internal")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "ai-data-masking", list[0]["name"])
	require.Equal(t, true, list[0]["enabled"])

	plugin, err := client.GetResource(context.Background(), "wasmplugin.extensions.higress.io", "ai-data-masking.internal")
	require.NoError(t, err)
	spec, _ := plugin["spec"].(map[string]any)
	rules := toMapSlice(spec["matchRules"])
	require.Len(t, rules, 2)

	require.NoError(t, svc.DeletePluginInstance(context.Background(), "route", "ai-route-doubao.internal", "ai-data-masking"))
	plugin, err = client.GetResource(context.Background(), "wasmplugin.extensions.higress.io", "ai-data-masking.internal")
	require.NoError(t, err)
	spec, _ = plugin["spec"].(map[string]any)
	rules = toMapSlice(spec["matchRules"])
	require.Empty(t, rules)
}

func TestRoutePluginInstancesIncludeBuiltinRuntimeBindings(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "wasmplugin.extensions.higress.io", "ai-statistics-1.0.0", map[string]any{
		"metadata": map[string]any{
			"name": "ai-statistics-1.0.0",
			"labels": map[string]any{
				"higress.io/wasm-plugin-name": "ai-statistics",
			},
		},
		"spec": map[string]any{
			"matchRules": []map[string]any{
				{
					"config":        map[string]any{"attributes": []any{"model"}},
					"configDisable": false,
					"ingress":       []string{"ai-route-demo.internal-internal"},
				},
			},
		},
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})
	list, err := svc.ListPluginInstances(context.Background(), "route", "ai-route-demo.internal", "ai-route-demo.internal-internal")
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "ai-statistics", list[0]["pluginName"])
	require.Equal(t, true, list[0]["enabled"])
	require.Equal(t, "ai-route-demo.internal-internal", list[0]["runtimeTarget"])

	item, err := svc.GetPluginInstance(context.Background(), "route", "ai-route-demo.internal", "ai-statistics", "ai-route-demo.internal-internal")
	require.NoError(t, err)
	require.Equal(t, "builtin-rule", item["runtimeSource"])
}

func TestGlobalPluginInstanceFallsBackToBuiltinRuntimeBindings(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "wasmplugin.extensions.higress.io", "ai-statistics-1.0.0", map[string]any{
		"metadata": map[string]any{
			"name": "ai-statistics-1.0.0",
			"labels": map[string]any{
				"higress.io/wasm-plugin-name": "ai-statistics",
			},
		},
		"spec": map[string]any{
			"matchRules": []map[string]any{
				{
					"config":        map[string]any{"use_default_response_attributes": true},
					"configDisable": false,
					"ingress":       []string{"ai-route-demo.internal"},
				},
			},
		},
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})

	item, err := svc.GetPluginInstance(context.Background(), "global", "", "ai-statistics")
	require.NoError(t, err)
	require.Equal(t, true, item["enabled"])
	require.Equal(t, "builtin-rule-global", item["runtimeSource"])
	require.Equal(t, []string{"ai-route-demo.internal"}, item["runtimeTargets"])
}

func TestWasmPluginReadmeFallsBackToDescription(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "wasm-plugins", "demo-plugin", map[string]any{
		"name":        "demo-plugin",
		"description": "A demo plugin.",
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})
	readme, err := svc.GetWasmPluginReadme(context.Background(), "demo-plugin")
	require.NoError(t, err)
	require.Contains(t, readme, "A demo plugin.")
}

func TestRouteValidationAndIngressClassNormalization(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(k8sclient.Config{IngressClass: "aigateway"}))

	item, err := svc.Save(context.Background(), "routes", map[string]any{
		"name": "demo-route",
		"path": map[string]any{
			"matchType":  "pre",
			"matchValue": "/demo",
		},
		"methods": []any{"get", "post"},
		"services": []any{
			map[string]any{"name": "backend.default", "port": 8080, "weight": 100},
		},
	})
	require.NoError(t, err)
	item, err = svc.Get(context.Background(), "routes", "demo-route")
	require.NoError(t, err)
	require.Equal(t, "aigateway", item["ingressClass"])
	require.Equal(t, "PRE", item["path"].(map[string]any)["matchType"])
}

func TestBuiltinWasmPluginFallbackExposesConfig(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	config, err := svc.GetWasmPluginConfig(context.Background(), "ai-security-guard")
	require.NoError(t, err)
	require.NotNil(t, config["schema"])

	readme, err := svc.GetWasmPluginReadme(context.Background(), "ai-security-guard")
	require.NoError(t, err)
	require.Contains(t, readme, "AI Content Security")
}

func TestBuiltinWasmPluginSnapshotOverridesLegacyMetadata(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	config, err := svc.GetWasmPluginConfig(context.Background(), "ai-statistics")
	require.NoError(t, err)
	schema := mapValueTest(config["schema"])
	openAPIV3Schema := mapValueTest(schema["openAPIV3Schema"])
	properties := mapValueTest(openAPIV3Schema["properties"])
	valueLengthLimit := mapValueTest(properties["value_length_limit"])
	require.Equal(t, 32000, toInt(valueLengthLimit["default"]))

	readme, err := svc.GetWasmPluginReadme(context.Background(), "ai-statistics")
	require.NoError(t, err)
	require.Contains(t, readme, "32000")
	require.Contains(t, readme, "Detailed Usage Normalization")
}

func TestBuiltinWasmPluginFallbackIncludesAIQuota(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	items, err := svc.List(context.Background(), "wasm-plugins")
	require.NoError(t, err)

	var aiQuota map[string]any
	for _, item := range items {
		if fmt.Sprint(item["name"]) == "ai-quota" {
			aiQuota = item
			break
		}
	}

	require.NotNil(t, aiQuota)
	require.Equal(t, true, aiQuota["builtIn"])
	require.Equal(t, "ai", aiQuota["category"])
	require.Equal(t, "AI Quota", aiQuota["title"])
}

func TestMCPServerRouteMetadataIsExplicit(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(k8sclient.Config{IngressClass: "aigateway"}))

	item, err := svc.Save(context.Background(), "mcp-servers", map[string]any{
		"name": "knowledge-base",
		"type": "OPEN_API",
	})
	require.NoError(t, err)

	metadata, _ := item["routeMetadata"].(map[string]any)
	require.Equal(t, "knowledge-base", metadata["mcpServerName"])
	require.Equal(t, "mcp-server-knowledge-base.internal", metadata["routeName"])
	require.Equal(t, "aigateway", metadata["ingressClass"])
}

func TestMCPServerValidationRejectsInvalidDatabasePayload(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(k8sclient.Config{IngressClass: "aigateway"}))

	_, err := svc.Save(context.Background(), "mcp-servers", map[string]any{
		"name":   "db-tools",
		"type":   "DATABASE",
		"dbType": "mysql",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "dsn is required")
}

func TestMCPServerValidationRejectsInvalidOpenAPIYaml(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(k8sclient.Config{IngressClass: "aigateway"}))

	_, err := svc.Save(context.Background(), "mcp-servers", map[string]any{
		"name":              "api-tools",
		"type":              "OPEN_API",
		"rawConfigurations": "tools: [",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid rawConfigurations yaml")
}

func TestMCPServerValidationRejectsInvalidDirectRouteTransport(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient(k8sclient.Config{IngressClass: "aigateway"}))

	_, err := svc.Save(context.Background(), "mcp-servers", map[string]any{
		"name": "route-tools",
		"type": "DIRECT_ROUTE",
		"directRouteConfig": map[string]any{
			"path":          "/events",
			"transportType": "grpc",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported directRoute transportType")
}

func TestAIProviderValidationRejectsAzureWithoutScheme(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	_, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name": "azure-demo",
		"type": "azure",
		"rawConfigs": map[string]any{
			"azureServiceUrl": "azure.openai.com/openai/deployments/demo",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "must have a scheme")
}

func TestAIProviderValidationNormalizesClaudeDefaults(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	item, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name": "claude-demo",
		"type": "claude",
		"rawConfigs": map[string]any{
			"claudeCodeMode": "true",
		},
	})
	require.NoError(t, err)
	rawConfigs, _ := item["rawConfigs"].(map[string]any)
	require.Equal(t, "2023-06-01", rawConfigs["claudeVersion"])
	require.Equal(t, true, rawConfigs["claudeCodeMode"])
}

func TestAIProviderValidationRejectsInvalidVertexAuthKey(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	_, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name": "vertex-demo",
		"type": "vertex",
		"rawConfigs": map[string]any{
			"vertexRegion":    "Asia-East1",
			"vertexProjectId": "demo-project",
			"vertexAuthKey":   "{\"client_email\":\"demo\"}",
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "private_key_id")
}

func TestAIProviderValidationAllowsVertexExpressModeWithTokens(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	item, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name":   "vertex-express",
		"type":   "vertex",
		"tokens": []any{"api-key"},
		"rawConfigs": map[string]any{
			"vertexRegion":     "Asia-East1",
			"providerBasePath": "/v1beta1",
		},
	})
	require.NoError(t, err)
	rawConfigs := mapValueTest(item["rawConfigs"])
	require.Equal(t, "asia-east1", rawConfigs["vertexRegion"])
	require.Equal(t, "/v1beta1", rawConfigs["providerBasePath"])
	require.Empty(t, rawConfigs["vertexAuthServiceName"])
}

func TestAIProviderValidationNormalizesAdvancedRawConfigs(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	item, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name": "bedrock-cache",
		"type": "bedrock",
		"rawConfigs": map[string]any{
			"awsRegion":                        "us-west-2",
			"awsAccessKey":                     "demo-ak",
			"awsSecretKey":                     "demo-sk",
			"providerDomain":                   "proxy.example.com",
			"providerBasePath":                 "/bedrock",
			"promoteThinkingOnEmpty":           "false",
			"hiclawMode":                       "true",
			"promptCacheRetention":             "in-memory",
			"bedrockPromptCachePointPositions": map[string]any{"system_prompt": true, "last-user-message": "true"},
		},
	})
	require.NoError(t, err)
	rawConfigs := mapValueTest(item["rawConfigs"])
	require.Equal(t, "proxy.example.com", rawConfigs["providerDomain"])
	require.Equal(t, "/bedrock", rawConfigs["providerBasePath"])
	require.Equal(t, true, rawConfigs["hiclawMode"])
	require.Equal(t, true, rawConfigs["promoteThinkingOnEmpty"])
	require.Equal(t, "in_memory", rawConfigs["promptCacheRetention"])
	positions := mapValueTest(rawConfigs["bedrockPromptCachePointPositions"])
	require.Equal(t, true, positions["systemPrompt"])
	require.Equal(t, true, positions["lastUserMessage"])
}

func TestAIProviderValidationNormalizesQwenDefaults(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	item, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name": "qwen-demo",
		"type": "qwen",
		"rawConfigs": map[string]any{
			"qwenFileIds": []any{"file-a", "file-b"},
		},
	})
	require.NoError(t, err)
	rawConfigs, _ := item["rawConfigs"].(map[string]any)
	require.Equal(t, false, rawConfigs["qwenEnableSearch"])
	require.Equal(t, true, rawConfigs["qwenEnableCompatible"])
	require.ElementsMatch(t, []string{"file-a", "file-b"}, rawConfigs["qwenFileIds"])
}

func TestAIProviderValidationNormalizesVolcengineConfigs(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	item, err := svc.Save(context.Background(), "ai-providers", map[string]any{
		"name":   "volcengine-demo",
		"type":   "doubao",
		"tokens": []any{"api-key"},
		"rawConfigs": map[string]any{
			"volcengineBaseUrl":          "https://ark.cn-beijing.volces.com/api/v3",
			"volcengineClientRequestId":  " request-1 ",
			"volcengineEnableEncryption": "true",
			"volcengineEnableTrace":      true,
			"retryOnFailure": map[string]any{
				"enabled":      "true",
				"maxRetries":   "2",
				"retryTimeout": "30000",
				"retryOnStatus": []any{
					"4.*",
					"5.*",
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "volcengine", item["type"])
	rawConfigs := mapValueTest(item["rawConfigs"])
	require.Equal(t, "volcengine", rawConfigs["type"])
	require.Equal(t, "ark.cn-beijing.volces.com", rawConfigs["providerDomain"])
	require.Equal(t, "/api/v3", rawConfigs["providerBasePath"])
	require.Equal(t, "request-1", rawConfigs["volcengineClientRequestId"])
	require.Equal(t, true, rawConfigs["volcengineEnableEncryption"])
	require.Equal(t, true, rawConfigs["volcengineEnableTrace"])
	retryOnFailure := mapValueTest(rawConfigs["retryOnFailure"])
	require.Equal(t, true, retryOnFailure["enabled"])
	require.EqualValues(t, 2, retryOnFailure["maxRetries"])
	require.EqualValues(t, 30000, retryOnFailure["retryTimeout"])
	require.ElementsMatch(t, []string{"4.*", "5.*"}, normalizeStringSlice(retryOnFailure["retryOnStatus"]))
	require.Empty(t, rawConfigs["volcengineBaseUrl"])
}

func TestAIRouteValidationRejectsUnknownProvider(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	_, err := svc.Save(context.Background(), "ai-routes", map[string]any{
		"name": "chat-demo",
		"pathPredicate": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/v1/chat/completions",
		},
		"upstreams": []any{
			map[string]any{"provider": "missing-provider", "weight": 100},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown provider: missing-provider")
}

func TestAIRouteValidationNormalizesFrontendPayload(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-providers", "openai-demo", map[string]any{
		"name": "openai-demo",
		"type": "openai",
	})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "ai-providers", "backup-demo", map[string]any{
		"name": "backup-demo",
		"type": "openai",
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})
	item, err := svc.Save(context.Background(), "ai-routes", map[string]any{
		"name": "chat-demo",
		"pathPredicate": map[string]any{
			"matchType":  "pre",
			"matchValue": "/v1/chat/completions",
		},
		"headerPredicates": []any{
			map[string]any{"key": "x-tenant", "matchType": "equal", "matchValue": "team-a"},
		},
		"urlParamPredicates": []any{
			map[string]any{"key": "model", "matchType": "pre", "matchValue": "gpt-4"},
		},
		"modelPredicates": []any{
			map[string]any{"matchType": "pre", "matchValue": "gpt-4"},
		},
		"upstreams": []any{
			map[string]any{"provider": "openai-demo"},
		},
		"authConfig": map[string]any{
			"enabled":               true,
			"allowedConsumerLevels": []any{"normal"},
		},
		"fallbackConfig": map[string]any{
			"enabled":       true,
			"responseCodes": []any{"5XX", "4xx", "5xx"},
			"upstreams": []any{
				map[string]any{"provider": "backup-demo"},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "PRE", item["pathPredicate"].(map[string]any)["matchType"])
	require.Equal(t, 100, toInt(toMapSlice(item["upstreams"])[0]["weight"]))
	require.Equal(t, "EQUAL", toMapSlice(item["headerPredicates"])[0]["matchType"])
	require.Equal(t, "PRE", toMapSlice(item["modelPredicates"])[0]["matchType"])

	fallbackConfig, _ := item["fallbackConfig"].(map[string]any)
	require.Equal(t, "RAND", fallbackConfig["fallbackStrategy"])
	require.Equal(t, []string{"4xx", "5xx"}, normalizeStringSlice(fallbackConfig["responseCodes"]))
}

func TestAIRouteValidationAcceptsLegacyFallbackStrategyAlias(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-providers", "openai-demo", map[string]any{
		"name": "openai-demo",
		"type": "openai",
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})
	item, err := svc.Save(context.Background(), "ai-routes", map[string]any{
		"name": "chat-demo",
		"pathPredicate": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/v1/chat/completions",
		},
		"upstreams": []any{
			map[string]any{"provider": "openai-demo", "weight": 100},
		},
		"authConfig": map[string]any{
			"enabled":               true,
			"allowedConsumerLevels": []any{"normal"},
		},
		"fallbackConfig": map[string]any{
			"enabled":       true,
			"strategy":      "SEQ",
			"responseCodes": []any{"5xx"},
			"upstreams": []any{
				map[string]any{"provider": "openai-demo", "weight": 100},
			},
		},
	})
	require.NoError(t, err)
	fallbackConfig, _ := item["fallbackConfig"].(map[string]any)
	require.Equal(t, "SEQ", fallbackConfig["fallbackStrategy"])
}

func TestAIRouteValidationRejectsEnabledFallbackWithoutResponseCodes(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-providers", "openai-demo", map[string]any{
		"name": "openai-demo",
		"type": "openai",
	})
	require.NoError(t, err)

	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "normal-user", UserLevel: "normal", Status: "active"},
		},
	})
	_, err = svc.Save(context.Background(), "ai-routes", map[string]any{
		"name": "chat-demo",
		"pathPredicate": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/v1/chat/completions",
		},
		"upstreams": []any{
			map[string]any{"provider": "openai-demo", "weight": 100},
		},
		"authConfig": map[string]any{
			"enabled":               true,
			"allowedConsumerLevels": []any{"normal"},
		},
		"fallbackConfig": map[string]any{
			"enabled": true,
			"upstreams": []any{
				map[string]any{"provider": "openai-demo", "weight": 100},
			},
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "response codes cannot be empty")
}

func mapValueTest(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return typed
}

func TestRouteSaveExpandsAllowedDepartmentsAndLevels(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	svc := New(client, stubPortal{
		accounts: []portalsvc.OrgAccountRecord{
			{ConsumerName: "dept-admin", DepartmentID: "dept-parent", Status: "active", UserLevel: "normal"},
			{ConsumerName: "dept-child-user", DepartmentID: "dept-child", Status: "active", UserLevel: "plus"},
			{ConsumerName: "pro-user", DepartmentID: "dept-other", Status: "active", UserLevel: "pro"},
			{ConsumerName: "disabled-user", DepartmentID: "dept-child", Status: "disabled", UserLevel: "pro"},
		},
		departments: []*portalsvc.OrgDepartmentNode{
			{
				DepartmentID: "dept-parent",
				Name:         "Parent",
				Children: []*portalsvc.OrgDepartmentNode{
					{DepartmentID: "dept-child", Name: "Child"},
				},
			},
		},
	})

	item, err := svc.Save(context.Background(), "routes", map[string]any{
		"name": "team-route",
		"path": map[string]any{"matchType": "PRE", "matchValue": "/team"},
		"services": []any{
			map[string]any{"name": "svc-a", "weight": 100},
		},
		"authConfig": map[string]any{
			"enabled":               true,
			"allowedDepartments":    []any{"dept-parent"},
			"allowedConsumerLevels": []any{"pro"},
		},
	})
	require.NoError(t, err)

	authConfig := mapValueTest(item["authConfig"])
	require.Equal(t, []string{"dept-parent"}, normalizeStringSlice(authConfig["allowedDepartments"]))
	require.Equal(t, []string{"pro"}, normalizeStringSlice(authConfig["allowedConsumerLevels"]))
	require.ElementsMatch(t, []string{"dept-admin", "dept-child-user", "pro-user"}, normalizeStringSlice(authConfig["allowedConsumers"]))
}

func TestAIRouteRequiresEnabledAuthConfig(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-providers", "openai-demo", map[string]any{
		"name": "openai-demo",
		"type": "openai",
	})
	require.NoError(t, err)

	svc := New(client)
	_, err = svc.Save(context.Background(), "ai-routes", map[string]any{
		"name": "chat-demo",
		"pathPredicate": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/v1/chat/completions",
		},
		"upstreams": []any{
			map[string]any{"provider": "openai-demo", "weight": 100},
		},
		"authConfig": map[string]any{
			"enabled": false,
		},
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "authConfig.enabled must be true")
}
