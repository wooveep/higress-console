package gateway

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

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
	svc := New(client)

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
	svc := New(client)

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

func TestWasmPluginReadmeFallsBackToDescription(t *testing.T) {
	client := k8sclient.NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "wasm-plugins", "demo-plugin", map[string]any{
		"name":        "demo-plugin",
		"description": "A demo plugin.",
	})
	require.NoError(t, err)

	svc := New(client)
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

	config, err := svc.GetWasmPluginConfig(context.Background(), "cors")
	require.NoError(t, err)
	require.NotNil(t, config["schema"])

	readme, err := svc.GetWasmPluginReadme(context.Background(), "cors")
	require.NoError(t, err)
	require.Contains(t, readme, "CORS")
}

func TestBuiltinWasmPluginSnapshotOverridesLegacyMetadata(t *testing.T) {
	svc := New(k8sclient.NewMemoryClient())

	config, err := svc.GetWasmPluginConfig(context.Background(), "ai-statistics")
	require.NoError(t, err)
	schema := mapValue(config["schema"])
	openAPIV3Schema := mapValue(schema["openAPIV3Schema"])
	properties := mapValue(openAPIV3Schema["properties"])
	valueLengthLimit := mapValue(properties["value_length_limit"])
	require.Equal(t, 32000, toInt(valueLengthLimit["default"]))

	readme, err := svc.GetWasmPluginReadme(context.Background(), "ai-statistics")
	require.NoError(t, err)
	require.Contains(t, readme, "32000")
	require.Contains(t, readme, "Detailed Usage Normalization")
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
	rawConfigs := mapValue(item["rawConfigs"])
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
	rawConfigs := mapValue(item["rawConfigs"])
	require.Equal(t, "proxy.example.com", rawConfigs["providerDomain"])
	require.Equal(t, "/bedrock", rawConfigs["providerBasePath"])
	require.Equal(t, true, rawConfigs["hiclawMode"])
	require.Equal(t, true, rawConfigs["promoteThinkingOnEmpty"])
	require.Equal(t, "in_memory", rawConfigs["promptCacheRetention"])
	positions := mapValue(rawConfigs["bedrockPromptCachePointPositions"])
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

	svc := New(client)
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
	require.Equal(t, "RANDOM", fallbackConfig["fallbackStrategy"])
	require.Equal(t, []string{"4xx", "5xx"}, normalizeStringSlice(fallbackConfig["responseCodes"]))
}

func TestAIRouteValidationRejectsEnabledFallbackWithoutResponseCodes(t *testing.T) {
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

func mapValue(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return typed
}
