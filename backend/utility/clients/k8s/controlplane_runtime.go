package k8s

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
)

const (
	higressAnnotationMCPServer              = "higress.io/mcp-server"
	higressAnnotationMCPMatchRuleDomains    = "higress.io/mcp-server-match-rule-domains"
	higressAnnotationMCPMatchRuleType       = "higress.io/mcp-server-match-rule-type"
	higressAnnotationMCPMatchRuleValue      = "higress.io/mcp-server-match-rule-value"
	higressAnnotationMCPUpstreamType        = "higress.io/mcp-server-upstream-type"
	higressAnnotationMCPUpstreamTransport   = "higress.io/mcp-server-upstream-transport-type"
	higressAnnotationMCPPathRewritePrefix   = "higress.io/mcp-server-path-rewrite-prefix"
	higressAnnotationMCPPathRewriteEnabled  = "higress.io/mcp-server-enable-path-rewrite"
	higressMCPServerTypeOpenAPI             = "OPEN_API"
	higressMCPServerTypeDatabase            = "DATABASE"
	higressMCPServerTypeDirectRoute         = "DIRECT_ROUTE"
	higressMCPServerTypeRedirectRoute       = "REDIRECT_ROUTE"
	higressMCPServerTypeKey                 = "type"
	higressWasmPluginNameAIProxy            = "ai-proxy"
	higressWasmPluginNameModelRouter        = "model-router"
	higressWasmPluginNameModelMapper        = "model-mapper"
	higressWasmPluginNameAIStatistics       = "ai-statistics"
	higressWasmPluginNameKeyAuth            = "key-auth"
	higressWasmPluginNameMCPServer          = "mcp-server"
	higressWasmPluginVersionDefault         = "1.0.0"
	higressWasmPluginPriorityModelRouter    = 900
	higressWasmPluginPriorityModelMapper    = 800
	higressWasmPluginPriorityAIProxy        = 100
	higressWasmPluginPriorityAIStatistics   = 900
	higressWasmPluginPriorityKeyAuth        = 310
	higressWasmPluginPriorityMCPServer      = 999
	higressWasmPluginPhaseUnspecified       = "UNSPECIFIED_PHASE"
	higressWasmPluginPhaseAuthN             = "AUTHN"
	higressWasmPluginPhaseStats             = "STATS"
	higressAIModelRoutingHeader             = "x-higress-llm-model"
	higressAIFallbackHeader                 = "x-higress-fallback-from"
	higressAIRouteInternalPathPrefix        = "/internal/ai-routes/"
	higressAIRouteFallbackSuffix            = ".fallback"
	higressMCPServerStaticPort              = 80
	higressStaticRegistryType               = "static"
	higressDNSRegistryType                  = "dns"
	higressTransportHTTP                    = "http"
	higressTransportHTTPS                   = "https"
	higressKeyAuthAllowKey                  = "allow"
	higressAIProxyActiveProviderKey         = "activeProviderId"
	higressModelRouterHeaderKey             = "modelToHeader"
	higressAIStatisticsDefaultAttrsKey      = "use_default_response_attributes"
	higressMCPConfigSectionKey              = "mcpServer"
	higressMCPMatchListKey                  = "match_list"
	higressMCPServersKey                    = "servers"
	higressMCPRedisKey                      = "redis"
	higressMCPRedisAddressKey               = "address"
	higressMCPRedisPasswordKey              = "password"
	higressMCPRedisUsernameKey              = "username"
	higressMCPRedisDBKey                    = "db"
	higressMCPSSEPathSuffixKey              = "sse_path_suffix"
	higressMCPSSEPathSuffixDefault          = "/sse"
	higressMCPRedisAddressPlaceholder       = "your.redis.host:6379"
	higressMCPRedisPasswordPlaceholder      = "your_password"
	higressMCPRedisUsernamePlaceholder      = "your_username"
	higressMCPServerNameKey                 = "name"
	higressMCPServerPathKey                 = "path"
	higressMCPServerConfigKey               = "config"
	higressMCPServerDBTypeKey               = "dbType"
	higressMCPServerDSNKey                  = "dsn"
	higressMCPMatchRulePathKey              = "match_rule_path"
	higressMCPMatchRuleDomainKey            = "match_rule_domain"
	higressMCPMatchRuleTypeKey              = "match_rule_type"
	higressMCPServerConfigSectionEnabledKey = "enable"
)

var providerDefaultEndpoints = map[string]providerEndpoint{
	"openai":     {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.openai.com", Port: 443},
	"moonshot":   {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.moonshot.cn", Port: 443},
	"qwen":       {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "dashscope.aliyuncs.com", Port: 443},
	"ai360":      {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.360.cn", Port: 443},
	"github":     {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "models.inference.ai.azure.com", Port: 443},
	"groq":       {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.groq.com", Port: 443},
	"baichuan":   {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.baichuan-ai.com", Port: 443},
	"yi":         {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.lingyiwanwu.com", Port: 443},
	"deepseek":   {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.deepseek.com", Port: 443},
	"zhipuai":    {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "open.bigmodel.cn", Port: 443},
	"baidu":      {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "qianfan.baidubce.com", Port: 443},
	"stepfun":    {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.stepfun.com", Port: 443},
	"minimax":    {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.minimax.chat", Port: 443},
	"gemini":     {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "generativelanguage.googleapis.com", Port: 443},
	"mistral":    {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.mistral.ai", Port: 443},
	"cohere":     {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.cohere.com", Port: 443},
	"doubao":     {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "ark.cn-beijing.volces.com", Port: 443},
	"coze":       {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.coze.cn", Port: 443},
	"openrouter": {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "openrouter.ai", Port: 443},
	"grok":       {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.x.ai", Port: 443},
	"claude":     {Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: "api.anthropic.com", Port: 443},
}

type providerEndpoint struct {
	Type        string
	Protocol    string
	Domain      string
	Port        int
	ContextPath string
}

func (c *RealClient) listMCPServerResources(ctx context.Context) ([]map[string]any, error) {
	ingresses, err := c.listObjects(ctx, "ingress", "-l", buildLabelSelector(higressLabelBizType, higressLabelBizTypeMCPServer))
	if err != nil {
		return nil, err
	}
	serversConfig, ssePathSuffix, _ := c.loadMCPServerRuntimeState(ctx)
	authRules, _ := c.loadRouteAuthRules(ctx)
	mcpRules, _ := c.loadBuiltinPluginRules(ctx, higressWasmPluginNameMCPServer)
	registries, _ := c.loadMcpBridgeRegistries(ctx)

	result := make([]map[string]any, 0, len(ingresses))
	for _, ingress := range ingresses {
		item, err := c.mcpServerFromIngress(ingress, serversConfig, ssePathSuffix, authRules, mcpRules, registries)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sortResourcesByName(result)
	return result, nil
}

func (c *RealClient) getMCPServerResource(ctx context.Context, name string) (map[string]any, error) {
	ingress, err := c.getObject(ctx, "ingress", defaultMCPRouteName(name))
	if err != nil {
		return nil, err
	}
	serversConfig, ssePathSuffix, _ := c.loadMCPServerRuntimeState(ctx)
	authRules, _ := c.loadRouteAuthRules(ctx)
	mcpRules, _ := c.loadBuiltinPluginRules(ctx, higressWasmPluginNameMCPServer)
	registries, _ := c.loadMcpBridgeRegistries(ctx)
	return c.mcpServerFromIngress(ingress, serversConfig, ssePathSuffix, authRules, mcpRules, registries)
}

func (c *RealClient) upsertMCPServerResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	serverType := strings.ToUpper(firstNonEmpty(stringValue(data["type"]), higressMCPServerTypeOpenAPI))
	routeAuthConfig := cloneMap(mapValue(data["consumerAuthInfo"]))
	routePayload := map[string]any{
		"name":         defaultMCPRouteName(name),
		"domains":      normalizeStringSlice(data["domains"]),
		"ingressClass": firstNonEmpty(stringValue(data["ingressClass"]), c.ingressClass),
		"path": map[string]any{
			"matchType":  "PRE",
			"matchValue": firstNonEmpty(mcpServerPath(name), stringValue(mapValue(data["routeMetadata"])["matchValue"])),
		},
		"routeMetadata": map[string]any{
			"mcpServerName": name,
			"routeName":     defaultMCPRouteName(name),
			"ingressClass":  firstNonEmpty(stringValue(data["ingressClass"]), c.ingressClass),
		},
	}
	if len(routeAuthConfig) > 0 {
		routePayload["authConfig"] = routeAuthConfig
	}
	if description := stringValue(data["description"]); description != "" {
		routePayload["description"] = description
	}
	if services := toMapSlice(data["services"]); len(services) > 0 {
		routePayload["services"] = services
	}
	if serverType == higressMCPServerTypeDirectRoute || serverType == higressMCPServerTypeRedirectRoute {
		registries, _ := c.loadMcpBridgeRegistries(ctx)
		_, ssePathSuffix, _ := c.loadMCPServerRuntimeState(ctx)
		rewrite, directConfig, err := buildMCPDirectRouteRewrite(data, registries, ssePathSuffix)
		if err != nil {
			return nil, err
		}
		if len(rewrite) > 0 {
			routePayload["rewrite"] = rewrite
			routePayload["pathRewritePrefix"] = firstNonEmpty(stringValue(rewrite["prefix"]), "/")
		}
		if len(directConfig) > 0 {
			routePayload["directRouteConfig"] = directConfig
			routePayload["upstreamType"] = "sse"
			if transport := stringValue(directConfig["transportType"]); transport != "" {
				routePayload["upstreamTransportType"] = transport
			}
		}
	}
	if _, err := c.upsertIngressResource(ctx, defaultMCPRouteName(name), routePayload); err != nil {
		return nil, err
	}
	if err := c.syncMCPServerRuntime(ctx, name, serverType, data); err != nil {
		return nil, err
	}
	return c.getMCPServerResource(ctx, name)
}

func (c *RealClient) deleteMCPServerResource(ctx context.Context, name string) error {
	_ = c.deleteObjectIfExists(ctx, "ingress", defaultMCPRouteName(name))
	_ = c.deleteObjectIfExists(ctx, "ingress", name)
	if err := c.updateMCPServerConfigMap(ctx, func(section map[string]any) error {
		removeNamedMapItem(section, higressMCPServersKey, name)
		removeMCPMatchRule(section, mcpServerPath(name))
		return nil
	}); err != nil {
		return err
	}
	for _, routeName := range mcpRouteTargetNames(name) {
		if err := c.removeBuiltinPluginRule(ctx, higressWasmPluginNameMCPServer, map[string][]string{"ingress": {routeName}}); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := c.syncRouteAuthRule(ctx, routeName, nil, false); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

func (c *RealClient) syncMCPServerRuntime(ctx context.Context, name, serverType string, data map[string]any) error {
	if err := c.ensureMCPServerConfigInitialized(ctx); err != nil {
		return err
	}
	var (
		openAPIConfig  map[string]any
		openAPIEnabled bool
	)
	if serverType == higressMCPServerTypeOpenAPI {
		config, enabled, err := parseMCPRawConfig(name, stringValue(data["description"]), stringValue(data["rawConfigurations"]))
		if err != nil {
			return err
		}
		if err := c.ensureMCPRedisConfigured(ctx); err != nil {
			return err
		}
		openAPIConfig = config
		openAPIEnabled = enabled
	}

	authInfo := mapValue(data["consumerAuthInfo"])
	if err := c.updateMCPServerConfigMap(ctx, func(section map[string]any) error {
		upsertMCPMatchRule(section, map[string]any{
			higressMCPMatchRulePathKey:   mcpServerPath(name),
			higressMCPMatchRuleDomainKey: firstNonEmpty(strings.Join(normalizeStringSlice(data["domains"]), ","), "*"),
			higressMCPMatchRuleTypeKey:   "prefix",
		})
		if serverType == higressMCPServerTypeDatabase {
			server := map[string]any{
				higressMCPServerNameKey: name,
				higressMCPServerPathKey: mcpServerPath(name),
				higressMCPServerTypeKey: strings.ToLower(serverType),
				higressMCPServerConfigKey: map[string]any{
					higressMCPServerDSNKey:    stringValue(data["dsn"]),
					higressMCPServerDBTypeKey: stringValue(data["dbType"]),
				},
			}
			upsertNamedMapItem(section, higressMCPServersKey, name, server)
		} else {
			removeNamedMapItem(section, higressMCPServersKey, name)
		}
		return nil
	}); err != nil {
		return err
	}
	if serverType == higressMCPServerTypeOpenAPI {
		if err := c.upsertBuiltinPluginRule(ctx, higressWasmPluginNameMCPServer, map[string][]string{"ingress": {defaultMCPRouteName(name)}}, openAPIConfig, openAPIEnabled, nil); err != nil {
			return err
		}
		for _, routeName := range mcpRouteTargetNames(name)[1:] {
			if err := c.removeBuiltinPluginRule(ctx, higressWasmPluginNameMCPServer, map[string][]string{"ingress": {routeName}}); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
	} else {
		for _, routeName := range mcpRouteTargetNames(name) {
			if err := c.removeBuiltinPluginRule(ctx, higressWasmPluginNameMCPServer, map[string][]string{"ingress": {routeName}}); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
	}
	if err := c.syncRouteAuthRule(ctx, defaultMCPRouteName(name), authInfo, false); err != nil {
		return err
	}
	for _, routeName := range mcpRouteTargetNames(name)[1:] {
		if err := c.syncRouteAuthRule(ctx, routeName, nil, false); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
	}
	return nil
}

func (c *RealClient) mcpServerFromIngress(
	ingress map[string]any,
	serversConfig map[string]map[string]any,
	ssePathSuffix string,
	authRules map[string]map[string]any,
	mcpRules map[string]map[string]any,
	registries map[string]map[string]any,
) (map[string]any, error) {
	route := ingressToRoute(ingress, c.ingressClass)
	name, ok := mcpServerNameFromRouteName(stringValue(route["name"]))
	if !ok {
		return nil, ErrNotFound
	}
	metadata := mapValue(ingress["metadata"])
	annotations := stringMap(metadata["annotations"])
	labels := stringMap(metadata["labels"])
	serverType := strings.ToUpper(firstNonEmpty(labels[higressLabelMCPServerType], higressMCPServerTypeOpenAPI))

	item := map[string]any{
		"name":         name,
		"type":         serverType,
		"description":  annotations[higressAnnotationResourceDescription],
		"domains":      route["domains"],
		"services":     route["services"],
		"ingressClass": route["ingressClass"],
		"routeMetadata": map[string]any{
			"routeName":     route["name"],
			"mcpServerName": name,
			"ingressClass":  route["ingressClass"],
		},
	}
	if authRule := authRules[defaultMCPRouteName(name)]; len(authRule) > 0 {
		authInfo := authInfoFromRule(authRule)
		if levels := normalizeStringSlice(mapValue(route["authConfig"])["allowedConsumerLevels"]); len(levels) > 0 {
			authInfo["allowedConsumerLevels"] = levels
		}
		item["consumerAuthInfo"] = authInfo
	}
	if serverType == higressMCPServerTypeOpenAPI {
		if rule := mcpRules[defaultMCPRouteName(name)]; len(rule) > 0 {
			item["rawConfigurations"] = marshalYAML(rule["config"])
		}
	}
	if serverType == higressMCPServerTypeDatabase {
		if config := serversConfig[name]; len(config) > 0 {
			item["dbType"] = stringValue(config[higressMCPServerDBTypeKey])
			item["dsn"] = stringValue(config[higressMCPServerDSNKey])
			item["rawConfigurations"] = buildMCPDatabaseRawConfig(name, item["dbType"])
		}
	}
	if serverType == higressMCPServerTypeDirectRoute || serverType == higressMCPServerTypeRedirectRoute {
		directConfig := restoreMCPDirectRouteConfig(route, annotations, ssePathSuffix)
		if len(directConfig) > 0 {
			item["directRouteConfig"] = directConfig
			if path := stringValue(directConfig["path"]); path != "" {
				item["upstreamPathPrefix"] = path
			}
			if transport := stringValue(directConfig["transportType"]); transport != "" {
				item["transportType"] = transport
				item["upstreamTransportType"] = transport
			}
		}
		if rewritePrefix := annotations[higressAnnotationMCPPathRewritePrefix]; rewritePrefix != "" {
			item["pathRewritePrefix"] = rewritePrefix
		}
		if host := stringValue(mapValue(route["rewrite"])["host"]); host != "" {
			item["upstreamHost"] = host
		}
	}
	if len(toMapSlice(item["services"])) == 0 {
		if registry := registries[strings.TrimSuffix(stringValue(firstServiceName(route["services"])), "."+higressDNSRegistryType)]; len(registry) > 0 {
			item["services"] = []map[string]any{{
				"name":   registry["name"],
				"port":   registry["port"],
				"weight": 100,
			}}
		}
	}
	return item, nil
}

func (c *RealClient) syncAIRouteRuntime(ctx context.Context, name string, data map[string]any) error {
	publicName := aiRouteIngressName(name)
	internalName := aiRouteInternalIngressName(name)
	services, err := c.aiUpstreamServices(ctx, data["upstreams"])
	if err != nil {
		return err
	}
	publicPayload := buildAIRouteIngressPayload(name, publicName, data, services, false)
	internalPayload := buildAIRouteIngressPayload(name, internalName, data, services, true)
	if _, err := c.upsertIngressResource(ctx, publicName, publicPayload); err != nil {
		return err
	}
	if _, err := c.upsertIngressResource(ctx, internalName, internalPayload); err != nil {
		return err
	}
	if err := c.syncRouteAuthRule(ctx, publicName, mapValue(data["authConfig"]), false); err != nil {
		return err
	}
	if err := c.syncRouteAuthRule(ctx, internalName, mapValue(data["authConfig"]), true); err != nil {
		return err
	}
	if predicates := toMapSlice(data["modelPredicates"]); len(predicates) > 0 {
		if err := c.enableModelRouter(ctx); err != nil {
			return err
		}
	}
	if err := c.syncAIRouteModelMapper(ctx, publicName, data); err != nil {
		return err
	}
	if err := c.syncAIRouteModelMapper(ctx, internalName, data); err != nil {
		return err
	}
	if err := c.syncAIStatisticsRule(ctx, publicName, true); err != nil {
		return err
	}
	if err := c.syncAIStatisticsRule(ctx, internalName, true); err != nil {
		return err
	}
	return c.syncAIRouteFallbackRuntime(ctx, name, data)
}

func (c *RealClient) syncAIRouteFallbackRuntime(ctx context.Context, name string, data map[string]any) error {
	fallback := mapValue(data["fallbackConfig"])
	if !boolValue(fallback["enabled"]) || len(toMapSlice(fallback["upstreams"])) == 0 {
		if err := c.deleteObjectIfExists(ctx, "ingress", aiRouteFallbackIngressName(name)); err != nil {
			return err
		}
		if err := c.deleteObjectIfExists(ctx, "ingress", aiRouteInternalFallbackIngressName(name)); err != nil {
			return err
		}
		_ = c.deleteObjectIfExists(ctx, higressEnvoyFilterResource, aiRouteIngressName(name))
		_ = c.deleteObjectIfExists(ctx, higressEnvoyFilterResource, aiRouteInternalIngressName(name))
		_ = c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {aiRouteFallbackIngressName(name)}})
		_ = c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {aiRouteInternalFallbackIngressName(name)}})
		_ = c.removeModelMapperRulesForIngress(ctx, aiRouteFallbackIngressName(name))
		_ = c.removeModelMapperRulesForIngress(ctx, aiRouteInternalFallbackIngressName(name))
		_ = c.syncRouteAuthRule(ctx, aiRouteFallbackIngressName(name), nil, false)
		_ = c.syncRouteAuthRule(ctx, aiRouteInternalFallbackIngressName(name), nil, true)
		return nil
	}
	fallbackData := cloneMap(data)
	fallbackData["upstreams"] = fallback["upstreams"]
	services, err := c.aiUpstreamServices(ctx, fallbackData["upstreams"])
	if err != nil {
		return err
	}
	publicPayload := buildAIRouteFallbackIngressPayload(name, aiRouteFallbackIngressName(name), aiRouteIngressName(name), fallbackData, services, false)
	internalPayload := buildAIRouteFallbackIngressPayload(name, aiRouteInternalFallbackIngressName(name), aiRouteInternalIngressName(name), fallbackData, services, true)
	if _, err := c.upsertIngressResource(ctx, aiRouteFallbackIngressName(name), publicPayload); err != nil {
		return err
	}
	if _, err := c.upsertIngressResource(ctx, aiRouteInternalFallbackIngressName(name), internalPayload); err != nil {
		return err
	}
	if err := c.syncRouteAuthRule(ctx, aiRouteFallbackIngressName(name), mapValue(data["authConfig"]), false); err != nil {
		return err
	}
	if err := c.syncRouteAuthRule(ctx, aiRouteInternalFallbackIngressName(name), mapValue(data["authConfig"]), true); err != nil {
		return err
	}
	if err := c.syncAIRouteModelMapper(ctx, aiRouteFallbackIngressName(name), fallbackData); err != nil {
		return err
	}
	if err := c.syncAIRouteModelMapper(ctx, aiRouteInternalFallbackIngressName(name), fallbackData); err != nil {
		return err
	}
	if err := c.syncAIStatisticsRule(ctx, aiRouteFallbackIngressName(name), true); err != nil {
		return err
	}
	if err := c.syncAIStatisticsRule(ctx, aiRouteInternalFallbackIngressName(name), true); err != nil {
		return err
	}
	responseCodes := normalizeStringSlice(fallback["responseCodes"])
	if len(responseCodes) == 0 {
		responseCodes = []string{"4xx", "5xx"}
	}
	if err := c.applyEnvoyFilter(ctx, aiRouteIngressName(name), renderFallbackEnvoyFilter(aiRouteIngressName(name), responseCodes)); err != nil {
		return err
	}
	return c.applyEnvoyFilter(ctx, aiRouteInternalIngressName(name), renderFallbackEnvoyFilter(aiRouteInternalIngressName(name), responseCodes))
}

func (c *RealClient) deleteAIRouteResource(ctx context.Context, name string) error {
	_ = c.deleteObjectIfExists(ctx, "configmap", aiRouteConfigMapName(name))
	_ = c.deleteObjectIfExists(ctx, "ingress", aiRouteIngressName(name))
	_ = c.deleteObjectIfExists(ctx, "ingress", aiRouteInternalIngressName(name))
	_ = c.deleteObjectIfExists(ctx, "ingress", aiRouteFallbackIngressName(name))
	_ = c.deleteObjectIfExists(ctx, "ingress", aiRouteInternalFallbackIngressName(name))
	_ = c.deleteObjectIfExists(ctx, higressEnvoyFilterResource, aiRouteIngressName(name))
	_ = c.deleteObjectIfExists(ctx, higressEnvoyFilterResource, aiRouteInternalIngressName(name))
	_ = c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {aiRouteIngressName(name)}})
	_ = c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {aiRouteInternalIngressName(name)}})
	_ = c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {aiRouteFallbackIngressName(name)}})
	_ = c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {aiRouteInternalFallbackIngressName(name)}})
	_ = c.removeModelMapperRulesForIngress(ctx, aiRouteIngressName(name))
	_ = c.removeModelMapperRulesForIngress(ctx, aiRouteInternalIngressName(name))
	_ = c.removeModelMapperRulesForIngress(ctx, aiRouteFallbackIngressName(name))
	_ = c.removeModelMapperRulesForIngress(ctx, aiRouteInternalFallbackIngressName(name))
	_ = c.syncRouteAuthRule(ctx, aiRouteIngressName(name), nil, false)
	_ = c.syncRouteAuthRule(ctx, aiRouteInternalIngressName(name), nil, true)
	_ = c.syncRouteAuthRule(ctx, aiRouteFallbackIngressName(name), nil, false)
	_ = c.syncRouteAuthRule(ctx, aiRouteInternalFallbackIngressName(name), nil, true)
	return nil
}

func (c *RealClient) syncAIProviderRuntime(ctx context.Context, name string, data map[string]any, previous map[string]any, deleting bool) error {
	plan, err := deriveProviderRuntimePlan(name, data)
	if err != nil {
		return err
	}
	previousPlan, err := deriveProviderRuntimePlan(name, previous)
	if err != nil {
		return err
	}
	if deleting {
		if previousPlan.primaryRegistry != nil {
			if err := c.removeMcpBridgeRegistry(ctx, stringValue(previousPlan.primaryRegistry["name"])); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		if previousPlan.primaryServiceRef != nil {
			if err := c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIProxy, map[string][]string{"service": {stringValue(previousPlan.primaryServiceRef["name"])}}); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		for _, registryName := range previousPlan.deletableExtraRegistryNames {
			if err := c.removeMcpBridgeRegistry(ctx, registryName); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
	} else {
		if previousPlan.primaryRegistry != nil && plan.primaryRegistry == nil {
			if err := c.removeMcpBridgeRegistry(ctx, stringValue(previousPlan.primaryRegistry["name"])); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		if previousPlan.primaryServiceRef != nil && (plan.primaryServiceRef == nil || stringValue(previousPlan.primaryServiceRef["name"]) != stringValue(plan.primaryServiceRef["name"])) {
			if err := c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIProxy, map[string][]string{"service": {stringValue(previousPlan.primaryServiceRef["name"])}}); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		nextExtra := map[string]struct{}{}
		for _, registry := range plan.extraRegistries {
			nextExtra[stringValue(registry["name"])] = struct{}{}
		}
		for _, registryName := range previousPlan.deletableExtraRegistryNames {
			if _, ok := nextExtra[registryName]; ok {
				continue
			}
			if err := c.removeMcpBridgeRegistry(ctx, registryName); err != nil && !errors.Is(err, ErrNotFound) {
				return err
			}
		}
		if plan.primaryRegistry != nil {
			if err := c.upsertMcpBridgeRegistry(ctx, plan.primaryRegistry); err != nil {
				return err
			}
		}
		for _, registry := range plan.extraRegistries {
			if err := c.upsertMcpBridgeRegistry(ctx, registry); err != nil {
				return err
			}
		}
		if plan.primaryServiceRef != nil {
			if err := c.upsertBuiltinPluginRule(ctx, higressWasmPluginNameAIProxy, map[string][]string{"service": {stringValue(plan.primaryServiceRef["name"])}}, map[string]any{
				higressAIProxyActiveProviderKey: name,
			}, true, nil); err != nil {
				return err
			}
		}
	}
	if !plan.needsRouteSync && !previousPlan.needsRouteSync {
		return nil
	}
	return c.refreshAIRoutesForProvider(ctx, name)
}

func (c *RealClient) refreshAIRoutesForProvider(ctx context.Context, providerName string) error {
	routes, err := c.listAIRouteResources(ctx)
	if err != nil {
		return err
	}
	for _, route := range routes {
		if !aiRouteUsesProvider(route, providerName) {
			continue
		}
		if _, err := c.upsertAIRouteResource(ctx, stringValue(route["name"]), route); err != nil {
			return err
		}
	}
	return nil
}

func (c *RealClient) syncAIRouteModelMapper(ctx context.Context, ingressName string, data map[string]any) error {
	if err := c.removeModelMapperRulesForIngress(ctx, ingressName); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	for _, upstream := range toMapSlice(data["upstreams"]) {
		modelMapping := mapValue(upstream["modelMapping"])
		if len(modelMapping) == 0 {
			continue
		}
		serviceRef, _, err := c.deriveAIProviderServiceRef(ctx, stringValue(upstream["provider"]))
		if err != nil {
			return err
		}
		if serviceRef == nil {
			continue
		}
		if err := c.upsertBuiltinPluginRule(ctx, higressWasmPluginNameModelMapper, map[string][]string{
			"ingress": {ingressName},
			"service": {stringValue(serviceRef["name"])},
		}, map[string]any{"modelMapping": modelMapping}, true, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *RealClient) enableModelRouter(ctx context.Context) error {
	return c.upsertBuiltinGlobalPlugin(ctx, higressWasmPluginNameModelRouter, map[string]any{
		higressModelRouterHeaderKey: higressAIModelRoutingHeader,
	}, true)
}

func (c *RealClient) syncAIStatisticsRule(ctx context.Context, ingressName string, enabled bool) error {
	if !enabled {
		return c.removeBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {ingressName}})
	}
	return c.upsertBuiltinPluginRule(ctx, higressWasmPluginNameAIStatistics, map[string][]string{"ingress": {ingressName}}, map[string]any{
		higressAIStatisticsDefaultAttrsKey: true,
	}, true, nil)
}

func (c *RealClient) syncRouteAuthRule(ctx context.Context, ingressName string, authConfig map[string]any, internal bool) error {
	enabled := boolValue(firstNonNil(authConfig["enabled"], authConfig["enable"]))
	allow := normalizeStringSlice(authConfig["allowedConsumers"])
	if internal || len(authConfig) == 0 || (!enabled && len(allow) == 0) {
		if err := c.removeBuiltinPluginRule(ctx, higressWasmPluginNameKeyAuth, map[string][]string{"ingress": {ingressName}}); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		return nil
	}
	config := map[string]any{}
	if len(allow) > 0 {
		config[higressKeyAuthAllowKey] = allow
	}
	return c.upsertBuiltinPluginRule(ctx, higressWasmPluginNameKeyAuth, map[string][]string{"ingress": {ingressName}}, config, enabled, nil)
}

func (c *RealClient) upsertBuiltinGlobalPlugin(ctx context.Context, pluginName string, config map[string]any, enabled bool) error {
	return c.mutateBuiltinWasmPlugin(ctx, pluginName, func(plugin map[string]any) error {
		spec := ensureMap(plugin, "spec")
		spec["defaultConfig"] = cloneMap(config)
		spec["defaultConfigDisable"] = !enabled
		return nil
	})
}

func (c *RealClient) upsertBuiltinPluginRule(ctx context.Context, pluginName string, targets map[string][]string, config map[string]any, enabled bool, mutateSpec func(map[string]any)) error {
	return c.mutateBuiltinWasmPlugin(ctx, pluginName, func(plugin map[string]any) error {
		spec := ensureMap(plugin, "spec")
		if mutateSpec != nil {
			mutateSpec(spec)
		}
		matchRules := toMapSlice(spec["matchRules"])
		nextRule := map[string]any{
			"config":        cloneMap(config),
			"configDisable": !enabled,
		}
		applyWasmTargets(nextRule, targets)
		replaced := false
		next := make([]map[string]any, 0, len(matchRules)+1)
		for _, rule := range matchRules {
			if wasmRuleMatchesTargets(rule, targets) {
				if !replaced {
					next = append(next, nextRule)
					replaced = true
				}
				continue
			}
			next = append(next, rule)
		}
		if !replaced {
			next = append(next, nextRule)
		}
		sort.Slice(next, func(i, j int) bool {
			return wasmRuleSortKey(next[i]) < wasmRuleSortKey(next[j])
		})
		spec["matchRules"] = next
		return nil
	})
}

func (c *RealClient) removeBuiltinPluginRule(ctx context.Context, pluginName string, targets map[string][]string) error {
	return c.mutateBuiltinWasmPlugin(ctx, pluginName, func(plugin map[string]any) error {
		spec := ensureMap(plugin, "spec")
		matchRules := toMapSlice(spec["matchRules"])
		next := make([]map[string]any, 0, len(matchRules))
		found := false
		for _, rule := range matchRules {
			if wasmRuleMatchesTargets(rule, targets) {
				found = true
				continue
			}
			next = append(next, rule)
		}
		if !found {
			return ErrNotFound
		}
		spec["matchRules"] = next
		return nil
	})
}

func (c *RealClient) mutateBuiltinWasmPlugin(ctx context.Context, pluginName string, mutate func(map[string]any) error) error {
	plugin, err := c.getBuiltinWasmPlugin(ctx, pluginName)
	if err != nil {
		return err
	}
	working := cloneMap(plugin)
	if err := mutate(working); err != nil {
		return err
	}
	sanitizeObjectForApply(working)
	payload, err := yaml.Marshal(working)
	if err != nil {
		return err
	}
	_, err = c.run(ctx, payload, "apply", "-f", "-")
	return err
}

func (c *RealClient) getBuiltinWasmPlugin(ctx context.Context, pluginName string) (map[string]any, error) {
	items, err := c.listObjects(ctx, higressWasmPluginResource, "-l", buildLabelSelector(higressLabelWasmPluginName, pluginName))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	sort.Slice(items, func(i, j int) bool {
		return stringValue(nestedValue(items[i], "metadata", "name")) < stringValue(nestedValue(items[j], "metadata", "name"))
	})
	return items[0], nil
}

func (c *RealClient) loadBuiltinPluginRules(ctx context.Context, pluginName string) (map[string]map[string]any, error) {
	plugin, err := c.getBuiltinWasmPlugin(ctx, pluginName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return map[string]map[string]any{}, nil
		}
		return nil, err
	}
	result := map[string]map[string]any{}
	for _, rule := range toMapSlice(nestedValue(plugin, "spec", "matchRules")) {
		for _, ingress := range normalizeStringSlice(rule["ingress"]) {
			result[ingress] = rule
		}
	}
	return result, nil
}

func (c *RealClient) loadRouteAuthRules(ctx context.Context) (map[string]map[string]any, error) {
	return c.loadBuiltinPluginRules(ctx, higressWasmPluginNameKeyAuth)
}

func (c *RealClient) removeModelMapperRulesForIngress(ctx context.Context, ingressName string) error {
	return c.mutateBuiltinWasmPlugin(ctx, higressWasmPluginNameModelMapper, func(plugin map[string]any) error {
		spec := ensureMap(plugin, "spec")
		matchRules := toMapSlice(spec["matchRules"])
		next := make([]map[string]any, 0, len(matchRules))
		for _, rule := range matchRules {
			ingresses := normalizeStringSlice(rule["ingress"])
			if len(ingresses) > 0 && ingresses[0] == ingressName {
				continue
			}
			next = append(next, rule)
		}
		spec["matchRules"] = next
		return nil
	})
}

func (c *RealClient) applyEnvoyFilter(ctx context.Context, name, manifest string) error {
	manifest = strings.ReplaceAll(manifest, "${name}", name)
	payload := []byte(manifest)
	_, err := c.run(ctx, payload, "apply", "-f", "-")
	return err
}

func (c *RealClient) deleteObjectIfExists(ctx context.Context, resource, name string) error {
	_, err := c.run(ctx, nil, "delete", resource, name, "--ignore-not-found=true")
	return err
}

func (c *RealClient) ensureMCPRedisConfigured(ctx context.Context) error {
	config, err := c.ReadConfigMap(ctx, higressConfigMapName)
	if err != nil {
		return errors.New("OpenAI to MCP functionality requires Redis configuration, but Redis configuration is missing in higress-config. Please configure correct Redis address first, otherwise OpenAI to MCP functionality will be unavailable.")
	}
	root := map[string]any{}
	if err := yaml.Unmarshal([]byte(config[higressMCPConfigKey]), &root); err != nil {
		return errors.New("OpenAI to MCP functionality requires Redis configuration, but Redis configuration is missing in higress-config. Please configure correct Redis address first, otherwise OpenAI to MCP functionality will be unavailable.")
	}
	mcpSection, _ := ensureMCPServerConfigSection(root)
	redisConfig := mapValue(mcpSection[higressMCPRedisKey])
	return validateMCPRedisConfig(redisConfig)
}

func (c *RealClient) ensureMCPServerConfigInitialized(ctx context.Context) error {
	config, err := c.ReadConfigMap(ctx, higressConfigMapName)
	if err != nil {
		return err
	}
	root := map[string]any{}
	if err := yaml.Unmarshal([]byte(config[higressMCPConfigKey]), &root); err != nil {
		return err
	}
	if _, changed := ensureMCPServerConfigSection(root); !changed {
		return nil
	}
	updated, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	config[higressMCPConfigKey] = string(updated)
	return c.UpsertConfigMap(ctx, higressConfigMapName, config)
}

func (c *RealClient) loadMCPServerConfigMap(ctx context.Context) (map[string]map[string]any, error) {
	servers, _, err := c.loadMCPServerRuntimeState(ctx)
	return servers, err
}

func (c *RealClient) loadMCPServerRuntimeState(ctx context.Context) (map[string]map[string]any, string, error) {
	config, err := c.ReadConfigMap(ctx, higressConfigMapName)
	if err != nil {
		return nil, "", err
	}
	root := map[string]any{}
	if err := yaml.Unmarshal([]byte(config[higressMCPConfigKey]), &root); err != nil {
		return nil, "", err
	}
	section, _ := ensureMCPServerConfigSection(root)
	items := toMapSlice(section[higressMCPServersKey])
	result := make(map[string]map[string]any, len(items))
	for _, item := range items {
		name := stringValue(item[higressMCPServerNameKey])
		configMap := mapValue(item[higressMCPServerConfigKey])
		result[name] = configMap
	}
	return result, normalizeMCPSSEPathSuffix(section[higressMCPSSEPathSuffixKey]), nil
}

func (c *RealClient) updateMCPServerConfigMap(ctx context.Context, mutate func(section map[string]any) error) error {
	config, err := c.ReadConfigMap(ctx, higressConfigMapName)
	if err != nil {
		return err
	}
	root := map[string]any{}
	if err := yaml.Unmarshal([]byte(config[higressMCPConfigKey]), &root); err != nil {
		return err
	}
	section, _ := ensureMCPServerConfigSection(root)
	if err := mutate(section); err != nil {
		return err
	}
	updated, err := yaml.Marshal(root)
	if err != nil {
		return err
	}
	config[higressMCPConfigKey] = string(updated)
	return c.UpsertConfigMap(ctx, higressConfigMapName, config)
}

func (c *RealClient) loadMcpBridgeRegistries(ctx context.Context) (map[string]map[string]any, error) {
	bridge, err := c.getObject(ctx, "mcpbridge.networking.higress.io", higressMcpBridgeDefaultName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return map[string]map[string]any{}, nil
		}
		return nil, err
	}
	registries := toMapSlice(nestedValue(bridge, "spec", "registries"))
	result := map[string]map[string]any{}
	for _, item := range registries {
		result[stringValue(item["name"])] = item
	}
	return result, nil
}

func (c *RealClient) upsertMcpBridgeRegistry(ctx context.Context, registry map[string]any) error {
	bridge, err := c.getObject(ctx, "mcpbridge.networking.higress.io", higressMcpBridgeDefaultName)
	if err != nil {
		return err
	}
	working := cloneMap(bridge)
	spec := ensureMap(working, "spec")
	registries := toMapSlice(spec["registries"])
	name := stringValue(registry["name"])
	updated := false
	for i, item := range registries {
		if strings.EqualFold(stringValue(item["name"]), name) {
			registries[i] = cloneMap(registry)
			updated = true
			break
		}
	}
	if !updated {
		registries = append(registries, cloneMap(registry))
	}
	spec["registries"] = registries
	sanitizeObjectForApply(working)
	payload, err := yaml.Marshal(working)
	if err != nil {
		return err
	}
	_, err = c.run(ctx, payload, "apply", "-f", "-")
	return err
}

func (c *RealClient) removeMcpBridgeRegistry(ctx context.Context, name string) error {
	bridge, err := c.getObject(ctx, "mcpbridge.networking.higress.io", higressMcpBridgeDefaultName)
	if err != nil {
		return err
	}
	working := cloneMap(bridge)
	spec := ensureMap(working, "spec")
	registries := toMapSlice(spec["registries"])
	next := make([]map[string]any, 0, len(registries))
	found := false
	for _, item := range registries {
		if strings.EqualFold(stringValue(item["name"]), name) {
			found = true
			continue
		}
		next = append(next, item)
	}
	if !found {
		return ErrNotFound
	}
	spec["registries"] = next
	sanitizeObjectForApply(working)
	payload, err := yaml.Marshal(working)
	if err != nil {
		return err
	}
	_, err = c.run(ctx, payload, "apply", "-f", "-")
	return err
}

func deriveProviderServiceSource(name string, data map[string]any) (map[string]any, map[string]any, error) {
	plan, err := deriveProviderRuntimePlan(name, data)
	if err != nil {
		return nil, nil, err
	}
	return plan.primaryRegistry, plan.primaryServiceRef, nil
}

type providerRuntimePlan struct {
	primaryRegistry             map[string]any
	primaryServiceRef           map[string]any
	extraRegistries             []map[string]any
	deletableExtraRegistryNames []string
	needsRouteSync              bool
}

func deriveProviderRuntimePlan(name string, data map[string]any) (providerRuntimePlan, error) {
	plan := providerRuntimePlan{}
	if len(data) == 0 {
		return plan, nil
	}
	rawConfigs := cloneMap(mapValue(data["rawConfigs"]))
	providerType := normalizeProviderCatalogKey(firstNonEmpty(stringValue(data["type"]), stringValue(rawConfigs["type"])))
	plan.needsRouteSync = providerNeedsRouteSync(providerType)
	if customService := strings.TrimSpace(stringValue(rawConfigs["openaiCustomServiceName"])); customService != "" {
		plan.primaryServiceRef = map[string]any{"name": customService}
		if port := toInt(rawConfigs["openaiCustomServicePort"]); port > 0 {
			plan.primaryServiceRef["port"] = port
		}
		return plan, nil
	}
	endpoints, err := providerEndpointsForRuntime(providerType, data, rawConfigs)
	if err != nil {
		return plan, err
	}
	registry, serviceRef, err := providerEndpointsToServiceSource(name, endpoints)
	if err != nil {
		return plan, err
	}
	plan.primaryRegistry = registry
	plan.primaryServiceRef = serviceRef
	if providerType == "vertex" && !providerUsesAPITokenAuth(data) {
		plan.extraRegistries = []map[string]any{vertexAuthRegistry()}
	}
	plan.deletableExtraRegistryNames = registryNames(plan.extraRegistries)
	return plan, nil
}

func providerNeedsRouteSync(providerType string) bool {
	switch normalizeProviderCatalogKey(providerType) {
	case "openai", "ollama":
		return true
	default:
		return false
	}
}

func providerEndpointsForRuntime(providerType string, data map[string]any, rawConfigs map[string]any) ([]providerEndpoint, error) {
	if endpoints, err := openAIProviderEndpoints(rawConfigs); err != nil || len(endpoints) > 0 {
		return endpoints, err
	}
	if domain := strings.TrimSpace(stringValue(rawConfigs["providerDomain"])); domain != "" {
		endpoint, err := providerEndpointFromDomain(domain)
		if err != nil {
			return nil, err
		}
		return []providerEndpoint{endpoint}, nil
	}
	if endpoints, ok, err := providerCustomServiceEndpoints(providerType, data, rawConfigs); ok || err != nil {
		return endpoints, err
	}
	if raw := firstNonEmpty(stringValue(rawConfigs["baseUrl"]), stringValue(rawConfigs["endpoint"])); raw != "" {
		endpoint, err := providerEndpointFromURL(raw)
		if err != nil {
			return nil, err
		}
		return []providerEndpoint{endpoint}, nil
	}
	if defaultEndpoint, ok := providerDefaultEndpoints[providerType]; ok {
		return []providerEndpoint{defaultEndpoint}, nil
	}
	return nil, nil
}

func providerCustomServiceEndpoints(providerType string, data map[string]any, rawConfigs map[string]any) ([]providerEndpoint, bool, error) {
	switch providerType {
	case "openai":
		endpoints, err := openAIProviderEndpoints(rawConfigs)
		return endpoints, len(endpoints) > 0, err
	case "azure":
		if endpoint, ok, err := providerEndpointFromRawURL(rawConfigs["azureServiceUrl"], true); ok || err != nil {
			return []providerEndpoint{endpoint}, ok, err
		}
	case "qwen":
		if domain := strings.TrimSpace(stringValue(rawConfigs["qwenDomain"])); domain != "" {
			endpoint, err := providerEndpointFromDomain(domain)
			return []providerEndpoint{endpoint}, true, err
		}
	case "zhipuai":
		if domain := strings.TrimSpace(stringValue(rawConfigs["zhipuDomain"])); domain != "" {
			endpoint, err := providerEndpointFromDomain(domain)
			return []providerEndpoint{endpoint}, true, err
		}
	case "vertex":
		if providerUsesAPITokenAuth(data) {
			return []providerEndpoint{{
				Type:        higressDNSRegistryType,
				Protocol:    higressTransportHTTPS,
				Domain:      "aiplatform.googleapis.com",
				Port:        443,
				ContextPath: "/",
			}}, true, nil
		}
		region := strings.TrimSpace(stringValue(rawConfigs["vertexRegion"]))
		if region == "" {
			return nil, false, errors.New("vertexRegion cannot be empty")
		}
		return []providerEndpoint{{Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: fmt.Sprintf("%s-aiplatform.googleapis.com", strings.ToLower(region)), Port: 443, ContextPath: "/"}}, true, nil
	case "bedrock":
		region := strings.TrimSpace(stringValue(rawConfigs["awsRegion"]))
		if region == "" {
			return nil, false, errors.New("awsRegion cannot be empty")
		}
		return []providerEndpoint{{Type: higressDNSRegistryType, Protocol: higressTransportHTTPS, Domain: fmt.Sprintf("bedrock-runtime.%s.amazonaws.com", region), Port: 443, ContextPath: "/"}}, true, nil
	case "ollama":
		host := strings.TrimSpace(stringValue(rawConfigs["ollamaServerHost"]))
		if host == "" {
			return nil, false, errors.New("ollamaServerHost cannot be empty")
		}
		port := firstPositiveInt(toInt(rawConfigs["ollamaServerPort"]), 11434)
		endpoint := providerEndpoint{Type: higressDNSRegistryType, Protocol: higressTransportHTTP, Domain: host, Port: port, ContextPath: "/"}
		if ip := net.ParseIP(host); ip != nil {
			endpoint.Type = higressStaticRegistryType
			endpoint.Domain = net.JoinHostPort(host, strconv.Itoa(port))
			endpoint.Port = higressMCPServerStaticPort
		}
		return []providerEndpoint{endpoint}, true, nil
	}
	return nil, false, nil
}

func providerUsesAPITokenAuth(data map[string]any) bool {
	return len(normalizeStringSlice(data["tokens"])) > 0
}

func openAIProviderEndpoints(rawConfigs map[string]any) ([]providerEndpoint, error) {
	customURL := strings.TrimSpace(stringValue(rawConfigs["openaiCustomUrl"]))
	if customURL == "" {
		return nil, nil
	}
	rawURLs := []string{customURL}
	switch typed := rawConfigs["openaiExtraCustomUrls"].(type) {
	case nil:
	case []any:
		for _, item := range typed {
			value := strings.TrimSpace(fmt.Sprint(item))
			if value == "" {
				return nil, errors.New("openaiExtraCustomUrls must contain non-empty strings")
			}
			rawURLs = append(rawURLs, value)
		}
	case []string:
		for _, item := range typed {
			value := strings.TrimSpace(item)
			if value == "" {
				return nil, errors.New("openaiExtraCustomUrls must contain non-empty strings")
			}
			rawURLs = append(rawURLs, value)
		}
	default:
		return nil, errors.New("openaiExtraCustomUrls must be an array")
	}
	endpoints := make([]providerEndpoint, 0, len(rawURLs))
	for _, raw := range rawURLs {
		endpoint, err := providerEndpointFromStrictURL(raw, true)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}
	return endpoints, nil
}

func providerEndpointFromRawURL(value any, requireScheme bool) (providerEndpoint, bool, error) {
	raw := strings.TrimSpace(fmt.Sprint(value))
	if raw == "" || raw == "<nil>" {
		return providerEndpoint{}, false, nil
	}
	endpoint, err := providerEndpointFromStrictURL(raw, requireScheme)
	if err != nil {
		return providerEndpoint{}, false, err
	}
	return endpoint, true, nil
}

func providerEndpointFromDomain(domain string) (providerEndpoint, error) {
	parsed, err := url.Parse("https://" + strings.TrimSpace(domain))
	if err != nil {
		return providerEndpoint{}, err
	}
	if strings.TrimSpace(parsed.Hostname()) == "" || parsed.Hostname() != strings.TrimSpace(domain) {
		return providerEndpoint{}, fmt.Errorf("invalid provider domain %s", domain)
	}
	return providerEndpoint{
		Type:        higressDNSRegistryType,
		Protocol:    higressTransportHTTPS,
		Domain:      strings.TrimSpace(domain),
		Port:        443,
		ContextPath: "/",
	}, nil
}

func providerEndpointsToServiceSource(name string, endpoints []providerEndpoint) (map[string]any, map[string]any, error) {
	if len(endpoints) == 0 {
		return nil, nil, nil
	}
	endpointType := ""
	protocol := ""
	contextPath := ""
	port := 0
	domains := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if protocol != "" && protocol != endpoint.Protocol {
			return nil, nil, fmt.Errorf("multiple protocols found in the endpoints of provider: %s", name)
		}
		protocol = endpoint.Protocol
		if contextPath != "" && contextPath != endpoint.ContextPath {
			return nil, nil, fmt.Errorf("multiple context paths found in the endpoints of provider: %s", name)
		}
		contextPath = endpoint.ContextPath
		if endpointType != "" && endpointType != endpoint.Type {
			return nil, nil, fmt.Errorf("multiple types of endpoints found for provider: %s", name)
		}
		endpointType = endpoint.Type
		switch endpoint.Type {
		case higressStaticRegistryType:
			domains = append(domains, endpoint.Domain)
			port = higressMCPServerStaticPort
		case higressDNSRegistryType:
			if len(endpoints) > 1 {
				return nil, nil, fmt.Errorf("multiple endpoints only work with static IP addresses, provider: %s", name)
			}
			domains = append(domains, endpoint.Domain)
			port = endpoint.Port
		default:
			return nil, nil, fmt.Errorf("unsupported endpoint type %s", endpoint.Type)
		}
	}
	endpoint := providerEndpoint{Type: endpointType, Protocol: protocol, Domain: strings.Join(domains, ","), Port: port}
	return endpoint.toRegistry(name), endpoint.toServiceRef(name), nil
}

func vertexAuthRegistry() map[string]any {
	return map[string]any{
		"name":      "vertex-auth" + consts.InternalResourceNameSuffix,
		"type":      higressDNSRegistryType,
		"protocol":  higressTransportHTTPS,
		"domain":    "oauth2.googleapis.com",
		"port":      443,
		"proxyName": "",
	}
}

func registryNames(items []map[string]any) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if name := stringValue(item["name"]); name != "" {
			result = append(result, name)
		}
	}
	return result
}

func (c *RealClient) deriveAIProviderServiceRef(ctx context.Context, providerName string) (map[string]any, map[string]any, error) {
	if strings.TrimSpace(providerName) == "" {
		return nil, nil, nil
	}
	provider, err := c.getAIProviderResource(ctx, providerName)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	registry, serviceRef, err := deriveProviderServiceSource(providerName, provider)
	if err != nil {
		return nil, nil, err
	}
	return serviceRef, registry, nil
}

func (e providerEndpoint) toRegistry(providerName string) map[string]any {
	name := llmRegistryName(providerName)
	return map[string]any{
		"name":      name,
		"type":      e.Type,
		"protocol":  e.Protocol,
		"domain":    e.Domain,
		"port":      e.Port,
		"proxyName": "",
	}
}

func (e providerEndpoint) toServiceRef(providerName string) map[string]any {
	return map[string]any{
		"name":   llmServiceName(providerName, e.Type),
		"port":   e.Port,
		"weight": 100,
	}
}

func providerEndpointFromURL(raw string) (providerEndpoint, error) {
	return providerEndpointFromStrictURL(raw, false)
}

func providerEndpointFromStrictURL(raw string, requireScheme bool) (providerEndpoint, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return providerEndpoint{}, err
	}
	port := 443
	protocol := strings.ToLower(parsed.Scheme)
	if requireScheme && protocol == "" {
		return providerEndpoint{}, errors.New("provider URL must have a scheme")
	}
	if protocol == "" {
		protocol = higressTransportHTTPS
	}
	if protocol != higressTransportHTTP && protocol != higressTransportHTTPS {
		return providerEndpoint{}, fmt.Errorf("provider URL must use http or https scheme, got %s", protocol)
	}
	if protocol == higressTransportHTTP {
		port = 80
	}
	if parsed.Port() != "" {
		parsedPort, convErr := strconv.Atoi(parsed.Port())
		if convErr == nil {
			port = parsedPort
		}
	}
	host := parsed.Hostname()
	registryType := higressDNSRegistryType
	if ip := net.ParseIP(host); ip != nil {
		registryType = higressStaticRegistryType
		if parsed.Port() == "" {
			host = net.JoinHostPort(host, strconv.Itoa(port))
		} else {
			host = parsed.Host
		}
		port = higressMCPServerStaticPort
	}
	return providerEndpoint{
		Type:        registryType,
		Protocol:    protocol,
		Domain:      host,
		Port:        port,
		ContextPath: firstNonEmpty(strings.TrimSpace(parsed.EscapedPath()), "/"),
	}, nil
}

func buildAIRouteIngressPayload(name, ingressName string, data map[string]any, services []map[string]any, internal bool) map[string]any {
	payload := map[string]any{
		"name":         ingressName,
		"domains":      normalizeStringSlice(data["domains"]),
		"ingressClass": firstNonEmpty(stringValue(data["ingressClass"]), "aigateway"),
		"path":         aiRoutePathForIngress(name, data, internal),
		"services":     services,
		"headers":      aiIngressHeaders(data, nil),
		"urlParams":    toMapSlice(data["urlParamPredicates"]),
		"methods":      normalizeStringSlice(data["methods"]),
		"authConfig":   mapValue(data["authConfig"]),
	}
	if internal {
		payload["domains"] = []string{}
	}
	if proxyNext := mapValue(data["proxyNextUpstream"]); len(proxyNext) > 0 {
		payload["proxyNextUpstream"] = proxyNext
	}
	return payload
}

func buildAIRouteFallbackIngressPayload(name, ingressName, originalRouteName string, data map[string]any, services []map[string]any, internal bool) map[string]any {
	payload := buildAIRouteIngressPayload(name, ingressName, data, services, internal)
	payload["headers"] = aiIngressHeaders(data, map[string]any{
		"key":           higressAIFallbackHeader,
		"matchType":     "EQUAL",
		"matchValue":    originalRouteName,
		"caseSensitive": true,
	})
	return payload
}

func aiIngressHeaders(data map[string]any, extra map[string]any) []map[string]any {
	headers := toMapSlice(data["headerPredicates"])
	modelPredicates := toMapSlice(data["modelPredicates"])
	for _, predicate := range modelPredicates {
		headers = append(headers, map[string]any{
			"key":           higressAIModelRoutingHeader,
			"matchType":     firstNonEmpty(stringValue(predicate["matchType"]), "EQUAL"),
			"matchValue":    stringValue(predicate["matchValue"]),
			"caseSensitive": true,
		})
	}
	if extra != nil {
		headers = append(headers, extra)
	}
	return headers
}

func aiRoutePathForIngress(name string, data map[string]any, internal bool) map[string]any {
	if internal {
		return map[string]any{
			"matchType":     "PRE",
			"matchValue":    higressAIRouteInternalPathPrefix + name,
			"caseSensitive": true,
		}
	}
	path := mapValue(firstNonNil(data["pathPredicate"], data["path"]))
	if len(path) == 0 {
		return map[string]any{"matchType": "PRE", "matchValue": "/" + name}
	}
	return cloneMap(path)
}

func (c *RealClient) aiUpstreamServices(ctx context.Context, value any) ([]map[string]any, error) {
	upstreams := toMapSlice(value)
	services := make([]map[string]any, 0, len(upstreams))
	for _, upstream := range upstreams {
		provider := stringValue(upstream["provider"])
		if provider == "" {
			continue
		}
		serviceRef, _, err := c.deriveAIProviderServiceRef(ctx, provider)
		if err != nil {
			return nil, err
		}
		service := map[string]any{
			"name":   llmServiceName(provider, higressDNSRegistryType),
			"weight": firstPositiveInt(toInt(upstream["weight"]), 100),
			"port":   443,
		}
		if serviceRef != nil {
			service["name"] = stringValue(serviceRef["name"])
			if port := toInt(serviceRef["port"]); port > 0 {
				service["port"] = port
			}
		}
		services = append(services, service)
	}
	return services, nil
}

func aiRouteIngressName(name string) string {
	return "ai-route-" + strings.TrimSpace(name) + consts.InternalResourceNameSuffix
}

func aiRouteInternalIngressName(name string) string {
	return aiRouteIngressName(name) + "-internal"
}

func aiRouteFallbackIngressName(name string) string {
	return "ai-route-" + strings.TrimSpace(name) + higressAIRouteFallbackSuffix + consts.InternalResourceNameSuffix
}

func aiRouteInternalFallbackIngressName(name string) string {
	return aiRouteFallbackIngressName(name) + "-internal"
}

func defaultMCPRouteName(name string) string {
	return "mcp-server-" + strings.TrimSpace(name) + consts.InternalResourceNameSuffix
}

func mcpServerNameFromRouteName(routeName string) (string, bool) {
	trimmed := strings.TrimSpace(routeName)
	if !strings.HasPrefix(trimmed, "mcp-server-") || !strings.HasSuffix(trimmed, consts.InternalResourceNameSuffix) {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(trimmed, "mcp-server-"), consts.InternalResourceNameSuffix)
	if name == "" {
		return "", false
	}
	return name, true
}

func mcpServerPath(name string) string {
	return higressMCPServerPathPrefix + "/" + strings.TrimSpace(name)
}

func authInfoFromRule(rule map[string]any) map[string]any {
	config := mapValue(rule["config"])
	allow := normalizeStringSlice(config[higressKeyAuthAllowKey])
	return map[string]any{
		"type":             "key-auth",
		"enable":           !boolValue(rule["configDisable"]),
		"allowedConsumers": allow,
	}
}

func parseMCPRawConfig(name, description, raw string) (map[string]any, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{
			"server": map[string]any{
				"name":        name,
				"description": firstNonEmpty(description, "Nothing"),
			},
		}, false, nil
	}
	result := map[string]any{}
	if err := yaml.Unmarshal([]byte(raw), &result); err != nil {
		return nil, false, err
	}
	enabled := len(toMapSlice(result["tools"])) > 0 || len(sliceValue(result["tools"])) > 0
	return result, enabled, nil
}

func validateMCPRedisConfig(redisConfig map[string]any) error {
	if len(redisConfig) == 0 {
		return errors.New("OpenAI to MCP functionality requires Redis configuration, but Redis configuration is missing in higress-config. Please configure correct Redis address first, otherwise OpenAI to MCP functionality will be unavailable.")
	}
	address := stringValue(redisConfig[higressMCPRedisAddressKey])
	if address == "" {
		return errors.New("OpenAI to MCP functionality requires Redis configuration, but Redis address is not configured. Please modify Redis configuration in higress-config, otherwise OpenAI to MCP functionality will be unavailable.")
	}
	if address == higressMCPRedisAddressPlaceholder {
		return errors.New("OpenAI to MCP functionality requires Redis configuration, but Redis address is still a placeholder. Please modify Redis configuration in higress-config, otherwise OpenAI to MCP functionality will be unavailable.")
	}
	return nil
}

func ensureMCPServerConfigSection(root map[string]any) (map[string]any, bool) {
	section := ensureMap(root, higressMCPConfigSectionKey)
	changed := false
	if section[higressMCPServerConfigSectionEnabledKey] == nil {
		section[higressMCPServerConfigSectionEnabledKey] = true
		changed = true
	}
	if strings.TrimSpace(stringValue(section[higressMCPSSEPathSuffixKey])) == "" {
		section[higressMCPSSEPathSuffixKey] = higressMCPSSEPathSuffixDefault
		changed = true
	}
	if _, ok := section[higressMCPServersKey]; !ok {
		section[higressMCPServersKey] = []map[string]any{}
		changed = true
	}
	if len(mapValue(section[higressMCPRedisKey])) == 0 {
		section[higressMCPRedisKey] = map[string]any{
			higressMCPRedisDBKey:       0,
			higressMCPRedisAddressKey:  higressMCPRedisAddressPlaceholder,
			higressMCPRedisPasswordKey: higressMCPRedisPasswordPlaceholder,
			higressMCPRedisUsernameKey: higressMCPRedisUsernamePlaceholder,
		}
		changed = true
	}
	return section, changed
}

func buildMCPDatabaseRawConfig(name string, dbType any) string {
	config := map[string]any{
		"server": name,
		"tools": []map[string]any{
			{"name": "query", "description": fmt.Sprintf("Run a read-only SQL query in database %v.", dbType)},
			{"name": "execute", "description": fmt.Sprintf("Execute an insert, update, or delete SQL in database %v.", dbType)},
			{"name": "list tables", "description": fmt.Sprintf("List all tables in database %v.", dbType)},
			{"name": "describe table", "description": fmt.Sprintf("Get the structure of a specific table in database %v.", dbType)},
		},
	}
	return marshalYAML(config)
}

func mcpRouteTargetNames(name string) []string {
	result := []string{defaultMCPRouteName(name)}
	trimmed := strings.TrimSpace(name)
	if trimmed != "" && trimmed != defaultMCPRouteName(name) {
		result = append(result, trimmed)
	}
	return result
}

func buildMCPDirectRouteRewrite(data map[string]any, registries map[string]map[string]any, ssePathSuffix string) (map[string]any, map[string]any, error) {
	directConfig := cloneMap(mapValue(data["directRouteConfig"]))
	if len(directConfig) == 0 {
		directConfig = map[string]any{}
		if path := stringValue(data["upstreamPathPrefix"]); path != "" {
			directConfig["path"] = path
		}
		if transport := firstNonEmpty(stringValue(data["transportType"]), stringValue(data["upstreamTransportType"]), stringValue(data["transport"])); transport != "" {
			directConfig["transportType"] = strings.ToLower(strings.TrimSpace(transport))
		}
	}
	transportType := strings.ToLower(strings.TrimSpace(stringValue(directConfig["transportType"])))
	directPath := strings.TrimSpace(stringValue(directConfig["path"]))
	rewritePath := directPath
	if transportType == "sse" && directPath != "" {
		ssePathSuffix = normalizeMCPSSEPathSuffix(ssePathSuffix)
		if !strings.HasSuffix(directPath, ssePathSuffix) {
			return nil, nil, fmt.Errorf("The path of direct route config must end with %s", ssePathSuffix)
		}
		rewritePath = strings.TrimSuffix(directPath, ssePathSuffix)
		if rewritePath == "" {
			rewritePath = "/"
		}
	}
	rewrite := map[string]any{}
	if rewritePath != "" {
		rewrite["enabled"] = true
		rewrite["path"] = rewritePath
		rewrite["prefix"] = "/"
	}
	if host := mcpDirectRouteRewriteHost(data["services"], registries); host != "" {
		rewrite["host"] = host
	}
	return rewrite, directConfig, nil
}

func mcpDirectRouteRewriteHost(value any, registries map[string]map[string]any) string {
	services := toMapSlice(value)
	if len(services) == 0 {
		return ""
	}
	name := stringValue(services[0]["name"])
	if strings.HasSuffix(name, "."+higressDNSRegistryType) {
		name = strings.TrimSuffix(name, "."+higressDNSRegistryType)
	}
	registry := registries[name]
	if stringValue(registry["type"]) != higressDNSRegistryType {
		return ""
	}
	return stringValue(registry["domain"])
}

func restoreMCPDirectRouteConfig(route map[string]any, annotations map[string]string, ssePathSuffix string) map[string]any {
	transportType := strings.ToLower(strings.TrimSpace(annotations[higressAnnotationMCPUpstreamTransport]))
	rewrite := mapValue(route["rewrite"])
	rewritePath := strings.TrimSpace(stringValue(rewrite["path"]))
	if transportType == "" && rewritePath == "" {
		return nil
	}
	config := map[string]any{}
	if transportType != "" {
		config["transportType"] = transportType
	}
	switch transportType {
	case "sse":
		ssePathSuffix = normalizeMCPSSEPathSuffix(ssePathSuffix)
		config["path"] = rewritePath + ssePathSuffix
	default:
		config["path"] = rewritePath
	}
	if stringValue(config["path"]) == "" {
		delete(config, "path")
	}
	return config
}

func normalizeMCPSSEPathSuffix(value any) string {
	suffix := strings.TrimSpace(stringValue(value))
	if suffix == "" {
		return higressMCPSSEPathSuffixDefault
	}
	return suffix
}

func marshalYAML(value any) string {
	payload, err := yaml.Marshal(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(payload))
}

func renderFallbackEnvoyFilter(routeName string, responseCodes []string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(`apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: ${name}
spec:
  configPatches:
    - applyTo: HTTP_ROUTE
      match:
        context: GATEWAY
        routeConfiguration:
          vhost:
            route:
              name: ${routeName}
      patch:
        operation: MERGE
        value:
          typed_per_filter_config:
            envoy.filters.http.custom_response:
              "@type": type.googleapis.com/udpa.type.v1.TypedStruct
              type_url: type.googleapis.com/envoy.extensions.filters.http.custom_response.v3.CustomResponse
              value:
                custom_response_matcher:
                  matcher_list:
                    matchers:
                      - predicate:
                          or_matcher:
                            predicate:
${predicates}
                        on_match:
                          action:
                            name: action
                            typed_config:
                              "@type": type.googleapis.com/udpa.type.v1.TypedStruct
                              type_url: type.googleapis.com/envoy.extensions.http.custom_response.redirect_policy.v3.RedirectPolicy
                              value:
                                max_internal_redirects: 10
                                use_original_request_uri: true
                                keep_original_response_code: false
                                use_original_request_body: true
                                only_redirect_upstream_code: false
                                request_headers_to_add:
                                  - header:
                                      key: ${fallbackHeader}
                                      value: ${routeName}
                                    append: false
                                response_headers_to_add:
                                  - header:
                                      key: ${fallbackHeader}
                                      value: ${routeName}
                                    append: false
                with_request_body:
                  max_request_bytes: 1024000
`, "${routeName}", routeName), "${fallbackHeader}", higressAIFallbackHeader), "${predicates}", renderResponseCodePredicates(responseCodes))
}

func renderResponseCodePredicates(responseCodes []string) string {
	lines := make([]string, 0, len(responseCodes))
	for _, code := range responseCodes {
		lines = append(lines, fmt.Sprintf(`                              - single_predicate:
                                  input:
                                    name: "%s_response"
                                    typed_config:
                                      "@type": type.googleapis.com/envoy.type.matcher.v3.HttpResponseStatusCodeClassMatchInput
                                  value_match:
                                    exact: "%s"`, code, code))
	}
	return strings.Join(lines, "\n")
}

func wasmRuleMatchesTargets(rule map[string]any, targets map[string][]string) bool {
	for key, expected := range targets {
		if !stringSlicesEqual(normalizeStringSlice(rule[key]), normalizeStringSlice(expected)) {
			return false
		}
	}
	for _, key := range []string{"domain", "ingress", "service"} {
		if _, ok := targets[key]; ok {
			continue
		}
		if len(normalizeStringSlice(rule[key])) > 0 {
			return false
		}
	}
	return true
}

func wasmRuleSortKey(rule map[string]any) string {
	return strings.Join(normalizeStringSlice(rule["service"]), ",") + "|" + strings.Join(normalizeStringSlice(rule["ingress"]), ",") + "|" + strings.Join(normalizeStringSlice(rule["domain"]), ",")
}

func applyWasmTargets(rule map[string]any, targets map[string][]string) {
	for key, values := range targets {
		if len(values) == 0 {
			delete(rule, key)
			continue
		}
		rule[key] = normalizeStringSlice(values)
	}
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func upsertNamedMapItem(section map[string]any, key, name string, item map[string]any) {
	items := toMapSlice(section[key])
	next := make([]map[string]any, 0, len(items)+1)
	replaced := false
	for _, current := range items {
		if strings.EqualFold(stringValue(current[higressMCPServerNameKey]), name) {
			next = append(next, mergeMCPConfigItem(current, item))
			replaced = true
			continue
		}
		next = append(next, current)
	}
	if !replaced {
		next = append(next, cloneMap(item))
	}
	section[key] = next
}

func removeNamedMapItem(section map[string]any, key, name string) {
	items := toMapSlice(section[key])
	next := make([]map[string]any, 0, len(items))
	for _, current := range items {
		if strings.EqualFold(stringValue(current[higressMCPServerNameKey]), name) {
			continue
		}
		next = append(next, current)
	}
	section[key] = next
}

func upsertMCPMatchRule(section map[string]any, item map[string]any) {
	items := toMapSlice(section[higressMCPMatchListKey])
	path := stringValue(item[higressMCPMatchRulePathKey])
	next := make([]map[string]any, 0, len(items)+1)
	replaced := false
	for _, current := range items {
		if strings.EqualFold(stringValue(current[higressMCPMatchRulePathKey]), path) {
			next = append(next, mergeMCPConfigItem(current, item))
			replaced = true
			continue
		}
		next = append(next, current)
	}
	if !replaced {
		next = append(next, cloneMap(item))
	}
	section[higressMCPMatchListKey] = next
}

func mergeMCPConfigItem(current, desired map[string]any) map[string]any {
	merged := cloneMap(current)
	for key, value := range desired {
		merged[key] = cloneValue(value)
	}
	return merged
}

func cloneValue(value any) any {
	if value == nil {
		return nil
	}
	bytes, err := yaml.Marshal(value)
	if err != nil {
		return value
	}
	var cloned any
	if err := yaml.Unmarshal(bytes, &cloned); err != nil {
		return value
	}
	return cloned
}

func removeMCPMatchRule(section map[string]any, path string) {
	items := toMapSlice(section[higressMCPMatchListKey])
	next := make([]map[string]any, 0, len(items))
	for _, current := range items {
		if strings.EqualFold(stringValue(current[higressMCPMatchRulePathKey]), path) {
			continue
		}
		next = append(next, current)
	}
	section[higressMCPMatchListKey] = next
}

func aiRouteUsesProvider(route map[string]any, providerName string) bool {
	for _, upstream := range toMapSlice(route["upstreams"]) {
		if strings.EqualFold(stringValue(upstream["provider"]), providerName) {
			return true
		}
	}
	fallback := mapValue(route["fallbackConfig"])
	for _, upstream := range toMapSlice(fallback["upstreams"]) {
		if strings.EqualFold(stringValue(upstream["provider"]), providerName) {
			return true
		}
	}
	return false
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return strings.EqualFold(strings.TrimSpace(fmt.Sprint(value)), "true")
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && strings.TrimSpace(text) == "" {
			continue
		}
		return value
	}
	return nil
}

func firstServiceName(value any) any {
	services := toMapSlice(value)
	if len(services) == 0 {
		return nil
	}
	return services[0]["name"]
}

func firstPositiveInt(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func llmRegistryName(providerName string) string {
	return "llm-" + strings.TrimSpace(providerName) + consts.InternalResourceNameSuffix
}

func llmServiceName(providerName, sourceType string) string {
	return llmRegistryName(providerName) + "." + sourceType
}
