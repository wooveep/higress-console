package k8s

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIngressToRouteParsesHigressAnnotations(t *testing.T) {
	item := map[string]any{
		"metadata": map[string]any{
			"name":            "demo-route",
			"resourceVersion": "12",
			"annotations": map[string]any{
				higressAnnotationDestination:              "50% svc-a.default:8080 v1\n50% svc-b.default:8081",
				higressAnnotationRewriteEnabled:           "true",
				higressAnnotationRewritePath:              "/landing",
				higressAnnotationAuthConsumerDepartments:  "dept-a,dept-b",
				higressAnnotationAuthConsumerLevels:       "normal,pro",
				higressAnnotationIgnorePathCase:           "true",
				higressAnnotationMatchMethod:              "GET,POST",
				"higress.io/exact-match-header-x-user-id": "1001",
				"higress.io/prefix-match-query-model":     "doubao",
				higressAnnotationProxyNextEnabled:         "true",
				higressAnnotationProxyNextUpstream:        "error,timeout",
				higressAnnotationProxyNextTries:           "2",
				higressAnnotationProxyNextTimeout:         "5000",
			},
			"labels": map[string]any{
				higressLabelResourceDefiner:           higressLabelResourceDefinerValue,
				higressLabelDomainPrefix + "ai.local": higressAnnotationTrueValue,
			},
		},
		"spec": map[string]any{
			"ingressClassName": "aigateway",
			"rules": []any{
				map[string]any{
					"host": "ai.local",
					"http": map[string]any{
						"paths": []any{
							map[string]any{
								"path":     "/demo",
								"pathType": "Prefix",
							},
						},
					},
				},
			},
		},
	}

	route := ingressToRoute(item, "aigateway")

	require.Equal(t, "demo-route", route["name"])
	require.Equal(t, "12", route["version"])
	require.Equal(t, "aigateway", route["ingressClass"])
	require.Equal(t, []string{"ai.local"}, route["domains"])
	require.Equal(t, "PRE", route["path"].(map[string]any)["matchType"])
	require.Equal(t, false, route["path"].(map[string]any)["caseSensitive"])
	require.Equal(t, []string{"GET", "POST"}, route["methods"])
	require.Equal(t, "/landing", route["rewrite"].(map[string]any)["path"])
	require.Equal(t, []string{"dept-a", "dept-b"}, route["authConfig"].(map[string]any)["allowedDepartments"])
	require.Equal(t, []string{"normal", "pro"}, route["authConfig"].(map[string]any)["allowedConsumerLevels"])
	require.Len(t, route["headers"], 1)
	require.Equal(t, "x-user-id", route["headers"].([]map[string]any)[0]["key"])
	require.Len(t, route["urlParams"], 1)
	require.Equal(t, "model", route["urlParams"].([]map[string]any)[0]["key"])
	require.Equal(t, 2, route["proxyNextUpstream"].(map[string]any)["attempts"])
	require.Len(t, route["services"], 2)
}

func TestConfigMapToAIRouteReadsLegacyDataField(t *testing.T) {
	item := map[string]any{
		"metadata": map[string]any{
			"resourceVersion": "7",
		},
		"data": map[string]any{
			higressDataField: `{"name":"doubao","domains":["ai.local"],"pathPredicate":{"matchType":"PRE","matchValue":"/doubao"},"upstreams":[{"provider":"doubao","weight":100}],"fallbackConfig":{"enabled":false}}`,
		},
	}

	route, err := configMapToAIRoute(item)
	require.NoError(t, err)
	require.Equal(t, "doubao", route["name"])
	require.Equal(t, "7", route["version"])
	require.Equal(t, "PRE", route["pathPredicate"].(map[string]any)["matchType"])
}

func TestBuildAIRouteIngressPayloadMapsFrontendContract(t *testing.T) {
	data := map[string]any{
		"domains":      []any{"api.example.com"},
		"ingressClass": "aigateway",
		"pathPredicate": map[string]any{
			"matchType":     "PRE",
			"matchValue":    "/v1/chat/completions",
			"caseSensitive": true,
		},
		"headerPredicates": []any{
			map[string]any{"key": "x-tenant", "matchType": "EQUAL", "matchValue": "team-a"},
		},
		"urlParamPredicates": []any{
			map[string]any{"key": "model", "matchType": "PRE", "matchValue": "gpt-4"},
		},
		"modelPredicates": []any{
			map[string]any{"matchType": "PRE", "matchValue": "gpt-4"},
		},
		"methods": []any{"GET", "POST"},
		"proxyNextUpstream": map[string]any{
			"enabled":    true,
			"conditions": []any{"error", "timeout"},
			"attempts":   3,
			"timeout":    5,
		},
		"authConfig": map[string]any{
			"enabled":            true,
			"allowedConsumers":   []any{"consumer-a"},
			"allowedDepartments": []any{"dept-a"},
		},
	}
	services := []map[string]any{
		{"name": "llm-openai.internal.dns", "port": 443, "weight": 100},
	}

	publicPayload := buildAIRouteIngressPayload("chat-demo", aiRouteIngressName("chat-demo"), data, services, false)
	publicAnnotations := routeAnnotations(publicPayload)

	require.Equal(t, []string{"api.example.com"}, publicPayload["domains"])
	require.Equal(t, "/v1/chat/completions", mapValue(publicPayload["path"])["matchValue"])
	require.Len(t, toMapSlice(publicPayload["headers"]), 1)
	require.Equal(t, "team-a", publicAnnotations["higress.io/exact-match-header-x-tenant"])
	_, hasModelHeader := publicAnnotations["higress.io/prefix-match-header-x-higress-llm-model"]
	require.False(t, hasModelHeader)
	require.Equal(t, "gpt-4", publicAnnotations["higress.io/prefix-match-query-model"])
	require.Equal(t, "GET POST", publicAnnotations[higressAnnotationMatchMethod])
	require.Equal(t, "dept-a", publicAnnotations[higressAnnotationAuthConsumerDepartments])
	require.Equal(t, "5", publicAnnotations[higressAnnotationProxyNextTimeout])

	internalPayload := buildAIRouteIngressPayload("chat-demo", aiRouteInternalIngressName("chat-demo"), data, services, true)
	internalAnnotations := routeAnnotations(internalPayload)
	require.Equal(t, []string{}, internalPayload["domains"])
	require.Equal(t, higressAIRouteInternalPathPrefix+"chat-demo", mapValue(internalPayload["path"])["matchValue"])
	require.Equal(t, "3", internalAnnotations[higressAnnotationProxyNextTries])
	_, exists := internalAnnotations[higressAnnotationProxyNextTimeout]
	require.False(t, exists)
}

func TestBuildAIRouteFallbackIngressPayloadAddsFallbackHeader(t *testing.T) {
	data := map[string]any{
		"pathPredicate": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/v1/chat/completions",
		},
		"headerPredicates": []any{
			map[string]any{"key": "x-tenant", "matchType": "EQUAL", "matchValue": "team-a"},
		},
	}
	services := []map[string]any{
		{"name": "llm-openai.internal.dns", "port": 443, "weight": 100},
	}

	payload := buildAIRouteFallbackIngressPayload(
		"chat-demo",
		aiRouteFallbackIngressName("chat-demo"),
		aiRouteIngressName("chat-demo"),
		data,
		services,
		false,
	)
	annotations := routeAnnotations(payload)

	require.Equal(t, "/v1/chat/completions", mapValue(payload["path"])["matchValue"])
	require.Equal(t, "team-a", annotations["higress.io/exact-match-header-x-tenant"])
	require.Equal(t, aiRouteIngressName("chat-demo"), annotations["higress.io/exact-match-header-x-higress-fallback-from"])
}

func TestSyncMirroredBuiltinIngressRuleSpecCopiesSourceRule(t *testing.T) {
	spec := map[string]any{
		"matchRules": []any{
			map[string]any{
				"ingress":       []any{"ai-route-chat-demo.internal"},
				"config":        map[string]any{"provider": "doubao", "chargeType": "amount"},
				"configDisable": false,
			},
			map[string]any{
				"ingress":       []any{"ai-route-chat-demo.internal-internal"},
				"config":        map[string]any{"provider": "stale"},
				"configDisable": true,
			},
			map[string]any{
				"service":       []any{"llm-doubao.internal.dns"},
				"config":        map[string]any{"keep": "me"},
				"configDisable": false,
			},
		},
	}

	syncMirroredBuiltinIngressRuleSpec(spec, "ai-route-chat-demo.internal", "ai-route-chat-demo.internal-internal")

	matchRules := toMapSlice(spec["matchRules"])
	require.Len(t, matchRules, 3)

	internalRuleFound := false
	for _, rule := range matchRules {
		if wasmRuleMatchesTargets(rule, map[string][]string{"ingress": {"ai-route-chat-demo.internal-internal"}}) {
			internalRuleFound = true
			require.Equal(t, map[string]any{"provider": "doubao", "chargeType": "amount"}, mapValue(rule["config"]))
			require.Equal(t, false, boolValue(rule["configDisable"]))
		}
	}
	require.True(t, internalRuleFound)
}

func TestSyncMirroredBuiltinIngressRuleSpecRemovesTargetWhenSourceMissing(t *testing.T) {
	spec := map[string]any{
		"matchRules": []any{
			map[string]any{
				"ingress":       []any{"ai-route-chat-demo.internal-internal"},
				"config":        map[string]any{"provider": "stale"},
				"configDisable": true,
			},
			map[string]any{
				"service":       []any{"llm-doubao.internal.dns"},
				"config":        map[string]any{"keep": "me"},
				"configDisable": false,
			},
		},
	}

	syncMirroredBuiltinIngressRuleSpec(spec, "ai-route-chat-demo.internal", "ai-route-chat-demo.internal-internal")

	matchRules := toMapSlice(spec["matchRules"])
	require.Len(t, matchRules, 1)
	require.True(t, wasmRuleMatchesTargets(matchRules[0], map[string][]string{"service": {"llm-doubao.internal.dns"}}))
}

func TestBuildAIQuotaRuleConfigUsesAmountBillingDefaults(t *testing.T) {
	config := buildAIQuotaRuleConfig("qwen", "redis-server", "aigateway-redis")
	redis := mapValue(config["redis"])

	require.Equal(t, higressAIQuotaQuotaUnitAmount, config["quota_unit"])
	require.Equal(t, higressAIQuotaBalanceKeyPrefix, config["balance_key_prefix"])
	require.Equal(t, higressAIQuotaPriceKeyPrefix, config["price_key_prefix"])
	require.Equal(t, higressAIQuotaUsageEventStream, config["usage_event_stream"])
	require.Equal(t, higressAIQuotaAdminConsumer, config["admin_consumer"])
	require.Equal(t, "/v1/ai/quotas/routes/qwen/consumers", config["admin_path"])
	require.Equal(t, "redis-server", redis["service_name"])
	require.Equal(t, 6379, redis["service_port"])
	require.Equal(t, higressAIQuotaRedisTimeoutMillis, redis["timeout"])
	require.Equal(t, 0, redis["database"])
	require.Equal(t, "aigateway-redis", redis["password"])
}

func TestBuildAIQuotaRuleConfigFallsBackForBlankRouteAndPassword(t *testing.T) {
	config := buildAIQuotaRuleConfig("", "", "")
	redis := mapValue(config["redis"])

	require.Equal(t, "/v1/ai/quotas/routes/default/consumers", config["admin_path"])
	require.Equal(t, higressAIQuotaRedisServiceDefault, redis["service_name"])
	_, hasPassword := redis["password"]
	require.False(t, hasPassword)
}

func TestWasmRuleLessPrefersRouteSpecificServiceRule(t *testing.T) {
	matchRules := []map[string]any{
		{
			"service": []any{"llm-bailian.internal.dns"},
			"config":  map[string]any{"activeProviderId": "bailian"},
		},
		{
			"ingress": []any{"ai-route-qwen.internal"},
			"service": []any{"llm-bailian.internal.dns"},
			"config":  map[string]any{"provider": map[string]any{"id": "bailian"}},
		},
		{
			"domain": []any{"api.ai.local"},
			"config": map[string]any{"fallback": true},
		},
	}

	sort.Slice(matchRules, func(i, j int) bool {
		return wasmRuleLess(matchRules[i], matchRules[j])
	})

	require.True(t, wasmRuleMatchesTargets(matchRules[0], map[string][]string{
		"ingress": {"ai-route-qwen.internal"},
		"service": {"llm-bailian.internal.dns"},
	}))
	require.True(t, wasmRuleMatchesTargets(matchRules[1], map[string][]string{
		"service": {"llm-bailian.internal.dns"},
	}))
	require.True(t, wasmRuleMatchesTargets(matchRules[2], map[string][]string{
		"domain": {"api.ai.local"},
	}))
}

func TestWasmPluginToProvidersExposesTokensProtocolAndModels(t *testing.T) {
	item := map[string]any{
		"spec": map[string]any{
			"defaultConfig": map[string]any{
				"providers": []any{
					map[string]any{
						"id":           "doubao",
						"type":         "doubao",
						"protocol":     "openai",
						"doubaoDomain": "proxy.example.com",
						"apiTokens": []any{
							"token-a",
							"token-b",
						},
						"portalModelMeta": map[string]any{
							"intro": "demo",
						},
					},
				},
			},
		},
	}

	providers := wasmPluginToProviders(item)
	require.Len(t, providers, 1)
	require.Equal(t, "doubao", providers[0]["name"])
	require.Equal(t, "volcengine", providers[0]["type"])
	require.Equal(t, "openai/v1", providers[0]["protocol"])
	require.Equal(t, []string{"token-a", "token-b"}, providers[0]["tokens"])
	require.NotEmpty(t, providers[0]["models"])
	rawConfigs := mapValue(providers[0]["rawConfigs"])
	require.Equal(t, "volcengine", rawConfigs["type"])
	require.Equal(t, "proxy.example.com", rawConfigs["providerDomain"])
	require.Empty(t, rawConfigs["doubaoDomain"])
}

func TestBuiltinWasmPluginManifestUsesInternalRuntimeName(t *testing.T) {
	manifest, ok := builtinWasmPluginManifest(higressWasmPluginNameAIProxy, "aigateway-system")
	require.True(t, ok)
	require.Equal(t, "extensions.higress.io/v1alpha1", manifest["apiVersion"])
	require.Equal(t, "WasmPlugin", manifest["kind"])

	metadata := mapValue(manifest["metadata"])
	require.Equal(t, "ai-proxy.internal", metadata["name"])
	require.Equal(t, "aigateway-system", metadata["namespace"])

	labels := mapValue(metadata["labels"])
	require.Equal(t, higressWasmPluginNameAIProxy, labels[higressLabelWasmPluginName])
	require.Equal(t, higressAnnotationTrueValue, labels[higressLabelInternal])
	require.Equal(t, "2.0.0", labels[higressLabelWasmPluginVersion])

	spec := mapValue(manifest["spec"])
	require.Equal(t, higressWasmPluginPhaseUnspecified, spec["phase"])
	require.EqualValues(t, higressWasmPluginPriorityAIProxy, spec["priority"])
	require.Equal(t, "http://aigateway-plugin-server.aigateway-system.svc.cluster.local/plugins/ai-proxy/2.0.0/plugin.wasm", spec["url"])
	require.Equal(t, true, spec["defaultConfigDisable"])
	require.Empty(t, toMapSlice(spec["matchRules"]))
}

func TestBuiltinWasmPluginManifestSupportsAIDataMasking(t *testing.T) {
	manifest, ok := builtinWasmPluginManifest(higressWasmPluginNameAIDataMasking, "aigateway-system")
	require.True(t, ok)

	metadata := mapValue(manifest["metadata"])
	require.Equal(t, "ai-data-masking.internal", metadata["name"])

	spec := mapValue(manifest["spec"])
	require.Equal(t, higressWasmPluginPhaseAuthN, spec["phase"])
	require.EqualValues(t, higressWasmPluginPriorityAIDataMasking, spec["priority"])
	require.Equal(t, "http://aigateway-plugin-server.aigateway-system.svc.cluster.local/plugins/ai-data-masking/2.0.0/plugin.wasm", spec["url"])
}

func TestMemoryClientSyncAIDataMaskingRuntimeUsesBundledDictionaryFallback(t *testing.T) {
	client := NewMemoryClient()
	_, err := client.UpsertResource(context.Background(), "ai-routes", "doubao", map[string]any{
		"name": "doubao",
		"fallbackConfig": map[string]any{
			"enabled": false,
		},
	})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "ai-sensitive-projections", "default", map[string]any{
		"name": "default",
		"detectRules": []map[string]any{{
			"pattern":   "敏感词验证词",
			"matchType": "contains",
			"priority":  100,
			"enabled":   true,
		}},
		"replaceRules": []map[string]any{{
			"pattern":      "%{IP}",
			"replaceType":  "replace",
			"replaceValue": "***.***.***.***",
			"restore":      true,
			"enabled":      true,
		}},
		"systemConfig": map[string]any{
			"systemDenyEnabled": true,
			"dictionaryText":    "这段内容不应该下发到运行时",
		},
		"runtimeConfig": map[string]any{
			"denyOpenai":      true,
			"denyJsonpath":    []any{"$.messages[*].content"},
			"denyRaw":         false,
			"denyCode":        200,
			"denyMessage":     "blocked",
			"denyRawMessage":  "{\"errmsg\":\"blocked\"}",
			"denyContentType": "application/json",
			"auditSink": map[string]any{
				"serviceName": "audit",
			},
		},
	})
	require.NoError(t, err)
	_, err = client.UpsertResource(context.Background(), "route-plugin-instances:ai-route-doubao.internal", "ai-data-masking", map[string]any{
		"name":       "ai-data-masking",
		"pluginName": "ai-data-masking",
		"enabled":    true,
	})
	require.NoError(t, err)

	plugin, err := client.GetResource(context.Background(), higressWasmPluginResource, builtinWasmPluginResourceName(higressWasmPluginNameAIDataMasking))
	require.NoError(t, err)
	spec := mapValue(plugin["spec"])
	rules := toMapSlice(spec["matchRules"])
	require.Len(t, rules, 2)

	var publicRule map[string]any
	for _, rule := range rules {
		if normalizeStringSlice(rule["ingress"])[0] == "ai-route-doubao.internal" {
			publicRule = rule
			break
		}
	}
	require.NotEmpty(t, publicRule)
	config := mapValue(publicRule["config"])
	require.Equal(t, true, config["system_deny"])
	require.Equal(t, "blocked", config["deny_message"])
	require.NotContains(t, config, "system_deny_words")
	require.NotContains(t, config, "audit_sink")
	require.Len(t, toMapSlice(config["deny_rules"]), 1)
	require.Len(t, toMapSlice(config["replace_rules"]), 1)
}

func TestDefaultMcpBridgeManifestStartsEmpty(t *testing.T) {
	manifest := defaultMcpBridgeManifest("aigateway-system")
	require.Equal(t, "networking.higress.io/v1", manifest["apiVersion"])
	require.Equal(t, "McpBridge", manifest["kind"])

	metadata := mapValue(manifest["metadata"])
	require.Equal(t, higressMcpBridgeDefaultName, metadata["name"])
	require.Equal(t, "aigateway-system", metadata["namespace"])

	spec := mapValue(manifest["spec"])
	require.Empty(t, toMapSlice(spec["registries"]))
	require.Empty(t, toMapSlice(spec["proxies"]))
}

func TestProviderPayloadRoundTripPreservesFailoverAndTokens(t *testing.T) {
	resource := map[string]any{
		"type":     "openai",
		"protocol": "openai/v1",
		"tokens":   []any{"token-a"},
		"rawConfigs": map[string]any{
			"openaiCustomUrl": "https://api.openai.com/v1",
			"note":            "keep-me",
		},
		"tokenFailoverConfig": map[string]any{
			"strategy": "random",
		},
	}

	item := map[string]any{
		"spec": map[string]any{
			"defaultConfig": map[string]any{
				"providers": []any{
					providerPayloadFromResource("openai-demo", resource),
				},
			},
		},
	}

	providers := wasmPluginToProviders(item)
	require.Len(t, providers, 1)
	require.Equal(t, "openai-demo", providers[0]["name"])
	require.Equal(t, "openai/v1", providers[0]["protocol"])
	require.Equal(t, []string{"token-a"}, providers[0]["tokens"])
	require.Equal(t, "random", mapValue(providers[0]["tokenFailoverConfig"])["strategy"])
	rawConfigs := mapValue(providers[0]["rawConfigs"])
	require.Equal(t, "openai-demo", rawConfigs["id"])
	require.Equal(t, "https://api.openai.com/v1", rawConfigs["openaiCustomUrl"])
	require.Equal(t, "keep-me", rawConfigs["note"])
}

func TestProviderPayloadFromResourceNormalizesLegacyVolcengineFields(t *testing.T) {
	payload := providerPayloadFromResource("doubao", map[string]any{
		"type": "doubao",
		"rawConfigs": map[string]any{
			"doubaoDomain": "proxy.example.com",
		},
	})

	require.Equal(t, "volcengine", payload["type"])
	require.Equal(t, "proxy.example.com", payload["providerDomain"])
	require.Empty(t, payload["doubaoDomain"])
}

func TestRouteAnnotationsRenderMCPMetadataAndRuntimeFields(t *testing.T) {
	data := map[string]any{
		"name":        "mcp-server-search.internal",
		"type":        "OPEN_API",
		"description": "search tools",
		"domains":     []any{"api.example.com"},
		"path": map[string]any{
			"matchType":  "PRE",
			"matchValue": "/mcp-servers/search",
		},
		"headers": []any{
			map[string]any{"key": "x-user-id", "matchType": "EQUAL", "matchValue": "u-1"},
			map[string]any{"key": ":authority", "matchType": "PRE", "matchValue": "api."},
		},
		"urlParams": []any{
			map[string]any{"key": "model", "matchType": "REGULAR", "matchValue": "doubao.*"},
		},
		"proxyNextUpstream": map[string]any{
			"enabled":    true,
			"conditions": []any{"error", "timeout"},
			"attempts":   3,
			"timeout":    1500,
		},
		"services": []any{
			map[string]any{"name": "llm-search.internal.dns", "port": 443, "weight": 100},
		},
		"routeMetadata": map[string]any{
			"mcpServerName": "search",
		},
	}

	labels := routeLabels(data)
	annotations := routeAnnotations(data)

	require.Equal(t, higressLabelBizTypeMCPServer, labels[higressLabelBizType])
	require.Equal(t, "OPEN_API", labels[higressLabelMCPServerType])
	require.Equal(t, higressAnnotationTrueValue, annotations[higressAnnotationMCPServer])
	require.Equal(t, "api.example.com", annotations[higressAnnotationMCPMatchRuleDomains])
	require.Equal(t, "prefix", annotations[higressAnnotationMCPMatchRuleType])
	require.Equal(t, "/mcp-servers/search", annotations[higressAnnotationMCPMatchRuleValue])
	require.Equal(t, "dns", annotations[higressAnnotationMCPUpstreamType])
	require.Equal(t, "search tools", annotations[higressAnnotationResourceDescription])
	require.Equal(t, "u-1", annotations["higress.io/exact-match-header-x-user-id"])
	require.Equal(t, "api.", annotations["higress.io/prefix-match-pseudo-header-authority"])
	require.Equal(t, "doubao.*", annotations["higress.io/regex-match-query-model"])
	require.Equal(t, "error,timeout", annotations[higressAnnotationProxyNextUpstream])
	require.Equal(t, "3", annotations[higressAnnotationProxyNextTries])
	require.Equal(t, "1500", annotations[higressAnnotationProxyNextTimeout])
}

func TestDeriveProviderServiceSourceFromCustomURL(t *testing.T) {
	registry, serviceRef, err := deriveProviderServiceSource("custom", map[string]any{
		"rawConfigs": map[string]any{
			"openaiCustomUrl": "https://1.2.3.4:8443/v1/chat/completions",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "llm-custom.internal", registry["name"])
	require.Equal(t, "static", registry["type"])
	require.Equal(t, "1.2.3.4:8443", registry["domain"])
	require.Equal(t, 80, registry["port"])
	require.Equal(t, "llm-custom.internal.static", serviceRef["name"])
	require.Equal(t, 80, serviceRef["port"])
}

func TestDeriveProviderServiceSourceFromMultipleCustomURLs(t *testing.T) {
	registry, serviceRef, err := deriveProviderServiceSource("custom", map[string]any{
		"type": "openai",
		"rawConfigs": map[string]any{
			"openaiCustomUrl":       "https://1.2.3.4:8443/v1",
			"openaiExtraCustomUrls": []any{"https://5.6.7.8:8443/v1"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, "static", registry["type"])
	require.Equal(t, "1.2.3.4:8443,5.6.7.8:8443", registry["domain"])
	require.Equal(t, "llm-custom.internal.static", serviceRef["name"])
	require.Equal(t, 80, serviceRef["port"])
}

func TestDeriveProviderRuntimePlanAddsVertexExtraRegistry(t *testing.T) {
	plan, err := deriveProviderRuntimePlan("vertex-demo", map[string]any{
		"type": "vertex",
		"rawConfigs": map[string]any{
			"vertexRegion": "Asia-East1",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "llm-vertex-demo.internal", plan.primaryRegistry["name"])
	require.Equal(t, "asia-east1-aiplatform.googleapis.com", plan.primaryRegistry["domain"])
	require.Equal(t, "llm-vertex-demo.internal.dns", plan.primaryServiceRef["name"])
	require.Len(t, plan.extraRegistries, 1)
	require.Equal(t, "vertex-auth.internal", plan.extraRegistries[0]["name"])
	require.Equal(t, []string{"vertex-auth.internal"}, plan.deletableExtraRegistryNames)
}

func TestDeriveProviderRuntimePlanUsesGlobalVertexEndpointForExpressMode(t *testing.T) {
	plan, err := deriveProviderRuntimePlan("vertex-express", map[string]any{
		"type":   "vertex",
		"tokens": []any{"api-key"},
		"rawConfigs": map[string]any{
			"vertexRegion": "asia-east1",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "aiplatform.googleapis.com", plan.primaryRegistry["domain"])
	require.Equal(t, "llm-vertex-express.internal.dns", plan.primaryServiceRef["name"])
	require.Empty(t, plan.extraRegistries)
	require.Empty(t, plan.deletableExtraRegistryNames)
}

func TestDeriveProviderRuntimePlanUsesProviderDomainOverride(t *testing.T) {
	plan, err := deriveProviderRuntimePlan("claude-demo", map[string]any{
		"type": "claude",
		"rawConfigs": map[string]any{
			"providerDomain": "proxy.example.com",
		},
	})

	require.NoError(t, err)
	require.Equal(t, "proxy.example.com", plan.primaryRegistry["domain"])
	require.Equal(t, "llm-claude-demo.internal.dns", plan.primaryServiceRef["name"])
}

func TestBuildMCPDirectRouteRewriteMatchesLegacySemantics(t *testing.T) {
	rewrite, directConfig, err := buildMCPDirectRouteRewrite(map[string]any{
		"services": []any{
			map[string]any{"name": "llm-demo.internal.dns", "port": 443},
		},
		"directRouteConfig": map[string]any{
			"path":          "/demo/events",
			"transportType": "sse",
		},
	}, map[string]map[string]any{
		"llm-demo.internal": {
			"type":   "dns",
			"domain": "demo.example.com",
		},
	}, "/events")

	require.NoError(t, err)
	require.Equal(t, "/demo", rewrite["path"])
	require.Equal(t, "/", rewrite["prefix"])
	require.Equal(t, "demo.example.com", rewrite["host"])
	require.Equal(t, "/demo/events", directConfig["path"])
	require.Equal(t, "sse", directConfig["transportType"])
}

func TestBuildMCPDirectRouteRewriteRejectsInvalidSSEPath(t *testing.T) {
	_, _, err := buildMCPDirectRouteRewrite(map[string]any{
		"directRouteConfig": map[string]any{
			"path":          "/demo/events",
			"transportType": "sse",
		},
	}, nil, "/sse")

	require.Error(t, err)
	require.Contains(t, err.Error(), "must end with /sse")
}

func TestRestoreMCPDirectRouteConfigRestoresSSEPath(t *testing.T) {
	config := restoreMCPDirectRouteConfig(map[string]any{
		"rewrite": map[string]any{
			"path": "/demo",
		},
	}, map[string]string{
		higressAnnotationMCPUpstreamTransport: "sse",
	}, "/events")

	require.Equal(t, "sse", config["transportType"])
	require.Equal(t, "/demo/events", config["path"])
}

func TestRestoreMCPDirectRouteConfigUsesDefaultSSEPathSuffix(t *testing.T) {
	config := restoreMCPDirectRouteConfig(map[string]any{
		"rewrite": map[string]any{
			"path": "/demo",
		},
	}, map[string]string{
		higressAnnotationMCPUpstreamTransport: "sse",
	}, "")

	require.Equal(t, "sse", config["transportType"])
	require.Equal(t, "/demo/sse", config["path"])
}

func TestBuildMCPDatabaseRawConfigIncludesDatabaseTools(t *testing.T) {
	raw := buildMCPDatabaseRawConfig("db-demo", "mysql")
	require.Contains(t, raw, "server: db-demo")
	require.Contains(t, raw, "name: query")
	require.Contains(t, raw, "name: execute")
	require.Contains(t, raw, "database mysql")
}

func TestMCPRouteTargetNamesIncludesLegacyCleanupTarget(t *testing.T) {
	require.Equal(t, []string{"mcp-server-search.internal", "search"}, mcpRouteTargetNames("search"))
}

func TestOpenAIProviderEndpointsRequireScheme(t *testing.T) {
	_, err := openAIProviderEndpoints(map[string]any{
		"openaiCustomUrl": "api.openai.com/v1",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "scheme")
}

func TestAzureProviderEndpointRequiresScheme(t *testing.T) {
	_, ok, err := providerEndpointFromRawURL("azure.openai.com/openai/deployments/demo", true)
	require.False(t, ok)
	require.Error(t, err)
	require.Contains(t, err.Error(), "scheme")
}

func TestProviderEndpointFromDomainRejectsInvalidValue(t *testing.T) {
	_, err := providerEndpointFromDomain("bad domain/value")
	require.Error(t, err)
}

func TestValidateMCPRedisConfigRejectsMissingConfig(t *testing.T) {
	err := validateMCPRedisConfig(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Redis configuration is missing")
}

func TestValidateMCPRedisConfigRejectsBlankAddress(t *testing.T) {
	err := validateMCPRedisConfig(map[string]any{
		"username": "demo",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Redis address is not configured")
}

func TestValidateMCPRedisConfigRejectsPlaceholderAddress(t *testing.T) {
	err := validateMCPRedisConfig(map[string]any{
		"address": "your.redis.host:6379",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "still a placeholder")
}

func TestEnsureMCPServerConfigSectionAppliesLegacyDefaults(t *testing.T) {
	section, changed := ensureMCPServerConfigSection(map[string]any{})

	require.True(t, changed)
	require.Equal(t, true, section[higressMCPServerConfigSectionEnabledKey])
	require.Equal(t, "/sse", section[higressMCPSSEPathSuffixKey])
	require.Equal(t, []map[string]any{}, section[higressMCPServersKey])

	redis := mapValue(section[higressMCPRedisKey])
	require.Equal(t, "your.redis.host:6379", redis[higressMCPRedisAddressKey])
	require.Equal(t, "your_password", redis[higressMCPRedisPasswordKey])
	require.Equal(t, "your_username", redis[higressMCPRedisUsernameKey])
	require.Equal(t, 0, redis[higressMCPRedisDBKey])
}

func TestUpsertMCPMatchRulePreservesExistingUnknownFields(t *testing.T) {
	section := map[string]any{
		higressMCPMatchListKey: []map[string]any{{
			higressMCPMatchRulePathKey:   "/mcp-servers/search",
			higressMCPMatchRuleDomainKey: "old.example.com",
			"upstream_type":              "dns",
			"path_rewrite_prefix":        "/legacy",
		}},
	}

	upsertMCPMatchRule(section, map[string]any{
		higressMCPMatchRulePathKey:   "/mcp-servers/search",
		higressMCPMatchRuleDomainKey: "new.example.com",
		higressMCPMatchRuleTypeKey:   "prefix",
	})

	items := toMapSlice(section[higressMCPMatchListKey])
	require.Len(t, items, 1)
	require.Equal(t, "new.example.com", items[0][higressMCPMatchRuleDomainKey])
	require.Equal(t, "prefix", items[0][higressMCPMatchRuleTypeKey])
	require.Equal(t, "dns", items[0]["upstream_type"])
	require.Equal(t, "/legacy", items[0]["path_rewrite_prefix"])
}

func TestUpsertNamedMapItemPreservesExistingUnknownFields(t *testing.T) {
	section := map[string]any{
		higressMCPServersKey: []map[string]any{{
			higressMCPServerNameKey: "db-demo",
			"note":                  "keep-me",
			higressMCPServerConfigKey: map[string]any{
				"dsn":    "old",
				"dbType": "mysql",
				"extra":  "legacy",
			},
		}},
	}

	upsertNamedMapItem(section, higressMCPServersKey, "db-demo", map[string]any{
		higressMCPServerNameKey: "db-demo",
		higressMCPServerConfigKey: map[string]any{
			"dsn":    "new",
			"dbType": "postgresql",
		},
	})

	items := toMapSlice(section[higressMCPServersKey])
	require.Len(t, items, 1)
	require.Equal(t, "keep-me", items[0]["note"])
	config := mapValue(items[0][higressMCPServerConfigKey])
	require.Equal(t, "new", config["dsn"])
	require.Equal(t, "postgresql", config["dbType"])
	require.Empty(t, config["extra"])
}

func TestServiceSourceRegistryRoundTripPreservesCoreFieldsAndProperties(t *testing.T) {
	registry := map[string]any{
		"name":          "bravesearch",
		"type":          "dns",
		"domain":        "api.search.brave.com",
		"port":          443,
		"protocol":      "https",
		"proxyName":     "default-http-proxy",
		"sni":           "api.search.brave.com",
		"nacosGroups":   "DEFAULT_GROUP",
		"consulTag":     "blue",
		"refreshPeriod": "10s",
	}

	resource := serviceSourceFromRegistry(registry)
	require.Equal(t, "bravesearch", resource["name"])
	require.Equal(t, "dns", resource["type"])
	require.Equal(t, "api.search.brave.com", resource["domain"])
	require.Equal(t, 443, resource["port"])
	require.Equal(t, "https", resource["protocol"])
	require.Equal(t, "default-http-proxy", resource["proxyName"])
	require.Equal(t, "api.search.brave.com", resource["sni"])
	require.Equal(t, "DEFAULT_GROUP", mapValue(resource["properties"])["nacosGroups"])
	require.Equal(t, "blue", mapValue(resource["properties"])["consulTag"])

	roundTrip := serviceSourceToRegistry("bravesearch", resource)
	require.Equal(t, "bravesearch", roundTrip["name"])
	require.Equal(t, "dns", roundTrip["type"])
	require.Equal(t, "api.search.brave.com", roundTrip["domain"])
	require.Equal(t, 443, roundTrip["port"])
	require.Equal(t, "https", roundTrip["protocol"])
	require.Equal(t, "default-http-proxy", roundTrip["proxyName"])
	require.Equal(t, "api.search.brave.com", roundTrip["sni"])
	require.Equal(t, "DEFAULT_GROUP", roundTrip["nacosGroups"])
	require.Equal(t, "blue", roundTrip["consulTag"])
}
