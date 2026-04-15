package k8s

import (
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
		"authConfig": map[string]any{
			"enabled":          true,
			"allowedConsumers": []any{"consumer-a"},
		},
	}
	services := []map[string]any{
		{"name": "llm-openai.internal.dns", "port": 443, "weight": 100},
	}

	publicPayload := buildAIRouteIngressPayload("chat-demo", aiRouteIngressName("chat-demo"), data, services, false)
	publicAnnotations := routeAnnotations(publicPayload)

	require.Equal(t, []string{"api.example.com"}, publicPayload["domains"])
	require.Equal(t, "/v1/chat/completions", mapValue(publicPayload["path"])["matchValue"])
	require.Len(t, toMapSlice(publicPayload["headers"]), 2)
	require.Equal(t, "team-a", publicAnnotations["higress.io/exact-match-header-x-tenant"])
	require.Equal(t, "gpt-4", publicAnnotations["higress.io/prefix-match-header-x-higress-llm-model"])
	require.Equal(t, "gpt-4", publicAnnotations["higress.io/prefix-match-query-model"])
	require.Equal(t, "GET POST", publicAnnotations[higressAnnotationMatchMethod])

	internalPayload := buildAIRouteIngressPayload("chat-demo", aiRouteInternalIngressName("chat-demo"), data, services, true)
	require.Equal(t, []string{}, internalPayload["domains"])
	require.Equal(t, higressAIRouteInternalPathPrefix+"chat-demo", mapValue(internalPayload["path"])["matchValue"])
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

func TestWasmPluginToProvidersExposesTokensProtocolAndModels(t *testing.T) {
	item := map[string]any{
		"spec": map[string]any{
			"defaultConfig": map[string]any{
				"providers": []any{
					map[string]any{
						"id":       "doubao",
						"type":     "doubao",
						"protocol": "openai",
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
	require.Equal(t, "openai/v1", providers[0]["protocol"])
	require.Equal(t, []string{"token-a", "token-b"}, providers[0]["tokens"])
	require.NotEmpty(t, providers[0]["models"])
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
