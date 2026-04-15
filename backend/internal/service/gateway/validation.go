package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	neturl "net/url"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	rfc1123NamePattern = regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`)
	httpMethods        = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	matchTypes         = []string{"EQUAL", "PRE", "REGULAR"}
	wasmPhases         = []string{"UNSPECIFIED_PHASE", "AUTHN", "AUTHZ", "STATS"}
	mcpServerTypes     = []string{"OPEN_API", "DATABASE", "DIRECT_ROUTE", "REDIRECT_ROUTE"}
	fallbackRespCodes  = []string{"4xx", "5xx"}
)

func (s *Service) hydrateResources(kind string, items []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, s.hydrateResource(kind, item))
	}
	return result
}

func (s *Service) hydrateResource(kind string, item map[string]any) map[string]any {
	clone := clonePayload(item)
	if clone == nil {
		return nil
	}
	if shouldExposeIngressClass(kind) && stringValue(clone["ingressClass"]) == "" {
		clone["ingressClass"] = s.k8sClient.IngressClass()
	}
	switch kind {
	case "mcp-servers":
		name := strings.TrimSpace(fmt.Sprint(clone["name"]))
		if name != "" {
			clone["routeMetadata"] = normalizeMCPRouteMetadata(name, clone["routeMetadata"], s.k8sClient.IngressClass())
		}
	case "ai-routes":
		if pathPredicate, _ := clone["pathPredicate"].(map[string]any); len(pathPredicate) == 0 {
			if path, ok := clone["path"].(map[string]any); ok && len(path) > 0 {
				clone["pathPredicate"] = clonePayload(path)
			}
		}
		if len(toMapSlice(clone["headerPredicates"])) == 0 && len(toMapSlice(clone["headers"])) > 0 {
			clone["headerPredicates"] = clone["headers"]
		}
		if len(toMapSlice(clone["urlParamPredicates"])) == 0 && len(toMapSlice(clone["urlParams"])) > 0 {
			clone["urlParamPredicates"] = clone["urlParams"]
		}
	}
	if kind == "routes" || kind == "ai-routes" {
		name := strings.TrimSpace(fmt.Sprint(clone["name"]))
		if name != "" {
			clone["mcpRouteMetadata"] = normalizeRouteBindingMetadata(name, clone["mcpRouteMetadata"], s.k8sClient.IngressClass())
		}
	}
	return clone
}

func (s *Service) normalizeForSave(ctx context.Context, kind string, payload map[string]any) (map[string]any, error) {
	normalized := clonePayload(payload)
	name := strings.TrimSpace(fmt.Sprint(normalized["name"]))
	if err := validateResourceName(name); err != nil {
		return nil, err
	}

	if shouldExposeIngressClass(kind) {
		normalized["ingressClass"] = firstNonEmpty(stringValue(normalized["ingressClass"]), s.k8sClient.IngressClass())
	}

	switch kind {
	case "routes":
		return s.normalizeRouteLike(kind, normalized)
	case "ai-routes":
		return s.normalizeAIRoute(ctx, normalized)
	case "services":
		return normalizeGatewayService(normalized)
	case "ai-providers":
		return normalizeAIProvider(normalized)
	case "wasm-plugins":
		return s.normalizeWasmPlugin(normalized)
	case "mcp-servers":
		return normalizeMCPServer(normalized, s.k8sClient.IngressClass())
	default:
		return normalized, nil
	}
}

func (s *Service) normalizeRouteLike(kind string, payload map[string]any) (map[string]any, error) {
	payload["ingressClass"] = firstNonEmpty(stringValue(payload["ingressClass"]), s.k8sClient.IngressClass())
	path, err := normalizeRoutePredicate(payload["path"], false)
	if err != nil {
		return nil, err
	}
	payload["path"] = path

	services, err := normalizeUpstreamServices(payload["services"])
	if err != nil {
		return nil, err
	}
	payload["services"] = services
	payload["domains"] = normalizeStringSlice(payload["domains"])

	if methods, err := normalizeMethods(payload["methods"]); err != nil {
		return nil, err
	} else if len(methods) > 0 {
		payload["methods"] = methods
	} else {
		delete(payload, "methods")
	}

	if headers, err := normalizeKeyedPredicates(payload["headers"]); err != nil {
		return nil, err
	} else if len(headers) > 0 {
		payload["headers"] = headers
	} else {
		delete(payload, "headers")
	}

	if params, err := normalizeKeyedPredicates(payload["urlParams"]); err != nil {
		return nil, err
	} else if len(params) > 0 {
		payload["urlParams"] = params
	} else {
		delete(payload, "urlParams")
	}

	if authConfig, err := normalizeAuthConfig(payload["authConfig"]); err != nil {
		return nil, err
	} else if len(authConfig) > 0 {
		payload["authConfig"] = authConfig
	} else {
		delete(payload, "authConfig")
	}

	payload["mcpRouteMetadata"] = normalizeRouteBindingMetadata(strings.TrimSpace(fmt.Sprint(payload["name"])), payload["mcpRouteMetadata"], strings.TrimSpace(fmt.Sprint(payload["ingressClass"])))
	return payload, nil
}

func (s *Service) normalizeAIRoute(ctx context.Context, payload map[string]any) (map[string]any, error) {
	payload["ingressClass"] = firstNonEmpty(stringValue(payload["ingressClass"]), s.k8sClient.IngressClass())
	payload["domains"] = normalizeStringSlice(payload["domains"])

	pathSource := firstNonNil(payload["pathPredicate"], payload["path"])
	if pathSource == nil {
		pathSource = map[string]any{
			"matchType":  "PRE",
			"matchValue": "/",
		}
	}
	pathPredicate, err := normalizeRoutePredicate(pathSource, false)
	if err != nil {
		return nil, err
	}
	if stringValue(pathPredicate["matchType"]) != "PRE" {
		return nil, errors.New("pathPredicate must be of type PRE")
	}
	payload["pathPredicate"] = pathPredicate
	delete(payload, "path")

	if headers, err := normalizeKeyedPredicates(firstNonNil(payload["headerPredicates"], payload["headers"])); err != nil {
		return nil, err
	} else if len(headers) > 0 {
		for _, predicate := range headers {
			if strings.EqualFold(stringValue(predicate["key"]), "x-higress-llm-model") {
				return nil, errors.New("headerPredicates cannot contain the model routing header")
			}
		}
		payload["headerPredicates"] = headers
	} else {
		delete(payload, "headerPredicates")
	}
	delete(payload, "headers")

	if params, err := normalizeKeyedPredicates(firstNonNil(payload["urlParamPredicates"], payload["urlParams"])); err != nil {
		return nil, err
	} else if len(params) > 0 {
		payload["urlParamPredicates"] = params
	} else {
		delete(payload, "urlParamPredicates")
	}
	delete(payload, "urlParams")

	if methods, err := normalizeMethods(payload["methods"]); err != nil {
		return nil, err
	} else if len(methods) > 0 {
		payload["methods"] = methods
	} else {
		delete(payload, "methods")
	}

	upstreams, err := s.normalizeAIUpstreams(ctx, payload["upstreams"], true)
	if err != nil {
		return nil, err
	}
	payload["upstreams"] = upstreams
	weightSum := 0
	for _, upstream := range upstreams {
		weightSum += toInt(upstream["weight"])
	}
	if weightSum != 100 {
		return nil, errors.New("the sum of upstream weights must be 100")
	}

	if modelPredicates, err := normalizeAIModelPredicates(payload["modelPredicates"]); err != nil {
		return nil, err
	} else if len(modelPredicates) > 0 {
		payload["modelPredicates"] = modelPredicates
	} else {
		delete(payload, "modelPredicates")
	}

	if authConfig, err := normalizeAuthConfig(payload["authConfig"]); err != nil {
		return nil, err
	} else if len(authConfig) > 0 {
		payload["authConfig"] = authConfig
	} else {
		delete(payload, "authConfig")
	}

	if fallbackConfig, err := s.normalizeAIRouteFallbackConfig(ctx, payload["fallbackConfig"]); err != nil {
		return nil, err
	} else if len(fallbackConfig) > 0 {
		payload["fallbackConfig"] = fallbackConfig
	} else {
		delete(payload, "fallbackConfig")
	}

	return payload, nil
}

func (s *Service) normalizeAIRouteFallbackConfig(ctx context.Context, value any) (map[string]any, error) {
	item, _ := value.(map[string]any)
	if len(item) == 0 {
		return nil, nil
	}

	result := clonePayload(item)
	enabled := boolFromAny(item["enabled"], false)
	result["enabled"] = enabled

	if !enabled {
		if upstreams, err := s.normalizeAIUpstreams(ctx, item["upstreams"], false); err == nil && len(upstreams) > 0 {
			result["upstreams"] = upstreams
		}
		if responseCodes := normalizeFallbackResponseCodes(item["responseCodes"]); len(responseCodes) > 0 {
			result["responseCodes"] = responseCodes
		}
		if strategy := strings.ToUpper(firstNonEmpty(stringValue(item["fallbackStrategy"]), stringValue(item["strategy"]))); strategy != "" {
			result["fallbackStrategy"] = strategy
		}
		delete(result, "strategy")
		return result, nil
	}

	upstreams, err := s.normalizeAIUpstreams(ctx, item["upstreams"], true)
	if err != nil {
		return nil, err
	}
	if len(upstreams) == 0 {
		return nil, errors.New("upstreams cannot be empty when fallback is enabled")
	}
	result["upstreams"] = upstreams

	strategy := strings.ToUpper(firstNonEmpty(stringValue(item["fallbackStrategy"]), stringValue(item["strategy"])))
	if strategy == "" {
		strategy = "RANDOM"
	}
	if strategy != "RANDOM" && strategy != "SEQUENCE" {
		return nil, fmt.Errorf("unknown fallback strategy: %s", strategy)
	}
	result["fallbackStrategy"] = strategy
	delete(result, "strategy")

	responseCodes := normalizeFallbackResponseCodes(item["responseCodes"])
	if len(responseCodes) == 0 {
		return nil, errors.New("response codes cannot be empty when fallback is enabled")
	}
	for _, code := range responseCodes {
		if !slices.Contains(fallbackRespCodes, code) {
			return nil, fmt.Errorf("invalid response code:%s", code)
		}
	}
	result["responseCodes"] = responseCodes
	return result, nil
}

func (s *Service) normalizeAIUpstreams(ctx context.Context, value any, validateProvider bool) ([]map[string]any, error) {
	rawItems := toAnySlice(value)
	if len(rawItems) == 0 {
		return nil, nil
	}

	result := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, _ := rawItem.(map[string]any)
		provider := strings.TrimSpace(fmt.Sprint(item["provider"]))
		if provider == "" {
			return nil, errors.New("provider cannot be null or empty")
		}
		if validateProvider {
			if _, err := s.k8sClient.GetResource(ctx, "ai-providers", provider); err != nil {
				return nil, fmt.Errorf("unknown provider: %s", provider)
			}
		}
		weight := toInt(item["weight"])
		if weight < 0 {
			return nil, fmt.Errorf("provider %s weight must be greater than or equal to 0", provider)
		}
		if weight == 0 && len(rawItems) == 1 {
			weight = 100
		}

		normalized := map[string]any{
			"provider": provider,
			"weight":   weight,
		}
		if modelMapping, ok := item["modelMapping"].(map[string]any); ok && len(modelMapping) > 0 {
			normalized["modelMapping"] = clonePayload(modelMapping)
		}
		result = append(result, normalized)
	}
	return result, nil
}

func normalizeAIModelPredicates(value any) ([]map[string]any, error) {
	items := toAnySlice(value)
	if len(items) == 0 {
		return nil, nil
	}

	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		predicate, _ := item.(map[string]any)
		if len(predicate) == 0 {
			return nil, errors.New("model predicate is required")
		}
		matchType := strings.ToUpper(strings.TrimSpace(fmt.Sprint(predicate["matchType"])))
		if matchType == "" {
			return nil, errors.New("matchType is required")
		}
		if matchType == "REGULAR" {
			return nil, errors.New("AiModelPredicate does not support regular expression matchType")
		}
		if !slices.Contains(matchTypes, matchType) {
			return nil, fmt.Errorf("unsupported matchType %s", matchType)
		}
		matchValue := strings.TrimSpace(fmt.Sprint(predicate["matchValue"]))
		if matchValue == "" {
			return nil, errors.New("matchValue is required")
		}
		result = append(result, map[string]any{
			"matchType":  matchType,
			"matchValue": matchValue,
		})
	}
	return result, nil
}

func normalizeFallbackResponseCodes(value any) []string {
	items := normalizeStringSlice(value)
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, strings.ToLower(strings.TrimSpace(item)))
	}
	return uniqueStrings(result)
}

func normalizeGatewayService(payload map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(fmt.Sprint(payload["name"]))
	if name == "" {
		return nil, errors.New("service name is required")
	}

	endpoints := normalizeStringSlice(payload["endpoints"])
	namespace := strings.TrimSpace(fmt.Sprint(payload["namespace"]))
	port := toInt(payload["port"])
	if len(endpoints) == 0 && namespace == "" {
		return nil, errors.New("service endpoints or namespace is required")
	}
	if port != 0 && (port < 1 || port > 65535) {
		return nil, fmt.Errorf("service %s port must be between 1 and 65535", name)
	}
	if len(endpoints) > 0 {
		payload["endpoints"] = endpoints
	}
	if namespace != "" {
		payload["namespace"] = namespace
	}
	if port != 0 {
		payload["port"] = port
	}
	return payload, nil
}

func normalizeAIProvider(payload map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(fmt.Sprint(payload["name"]))
	if err := validateResourceName(name); err != nil {
		return nil, fmt.Errorf("invalid provider name: %w", err)
	}

	rawConfigValue, _ := payload["rawConfigs"].(map[string]any)
	rawConfigs := clonePayload(rawConfigValue)
	if rawConfigs == nil {
		rawConfigs = map[string]any{}
	}
	providerType := normalizeProviderType(firstNonEmpty(stringValue(payload["type"]), stringValue(rawConfigs["type"])))
	if providerType == "" {
		return nil, errors.New("provider type is required")
	}
	rawConfigs["type"] = providerType
	payload["type"] = providerType
	payload["rawConfigs"] = rawConfigs

	if protocol := strings.TrimSpace(fmt.Sprint(payload["protocol"])); protocol != "" {
		payload["protocol"] = protocol
	}
	tokens := normalizeStringSlice(payload["tokens"])
	if len(tokens) > 0 {
		payload["tokens"] = tokens
	} else {
		delete(payload, "tokens")
	}

	if failover, ok := payload["tokenFailoverConfig"].(map[string]any); ok && len(failover) > 0 {
		payload["tokenFailoverConfig"] = clonePayload(failover)
	}
	if err := normalizeProviderAdvancedConfigs(rawConfigs); err != nil {
		return nil, err
	}

	switch providerType {
	case "openai":
		if err := normalizeOpenAIProviderConfigs(rawConfigs); err != nil {
			return nil, err
		}
	case "azure":
		if err := normalizeAzureProviderConfigs(rawConfigs); err != nil {
			return nil, err
		}
	case "qwen":
		if err := normalizeQwenProviderConfigs(rawConfigs); err != nil {
			return nil, err
		}
	case "zhipuai":
		if err := normalizeZhipuAIProviderConfigs(rawConfigs); err != nil {
			return nil, err
		}
	case "claude":
		normalizeClaudeProviderConfigs(rawConfigs)
	case "vertex":
		if err := normalizeVertexProviderConfigs(rawConfigs, tokens); err != nil {
			return nil, err
		}
	case "bedrock":
		if err := normalizeBedrockProviderConfigs(rawConfigs); err != nil {
			return nil, err
		}
	case "ollama":
		if err := normalizeOllamaProviderConfigs(rawConfigs); err != nil {
			return nil, err
		}
	}

	return payload, nil
}

func normalizeProviderAdvancedConfigs(rawConfigs map[string]any) error {
	if domain := strings.TrimSpace(stringValue(rawConfigs["providerDomain"])); domain != "" {
		if err := validateProviderDomain(domain); err != nil {
			return fmt.Errorf("providerDomain contains an invalid domain name: %s", domain)
		}
		rawConfigs["providerDomain"] = domain
	}
	if rawConfigs["promoteThinkingOnEmpty"] != nil {
		rawConfigs["promoteThinkingOnEmpty"] = boolFromAny(rawConfigs["promoteThinkingOnEmpty"], false)
	}
	if rawConfigs["hiclawMode"] != nil {
		rawConfigs["hiclawMode"] = boolFromAny(rawConfigs["hiclawMode"], false)
		if boolFromAny(rawConfigs["hiclawMode"], false) {
			rawConfigs["promoteThinkingOnEmpty"] = true
		}
	}
	if rawConfigs["providerBasePath"] != nil {
		basePath := strings.TrimSpace(stringValue(rawConfigs["providerBasePath"]))
		if basePath == "" {
			delete(rawConfigs, "providerBasePath")
		} else {
			if !strings.HasPrefix(basePath, "/") {
				return errors.New("providerBasePath must start with '/'")
			}
			rawConfigs["providerBasePath"] = basePath
		}
	}
	if rawConfigs["bedrockPromptCachePointPositions"] != nil {
		positions, err := normalizeBedrockPromptCachePointPositions(rawConfigs["bedrockPromptCachePointPositions"])
		if err != nil {
			return err
		}
		rawConfigs["bedrockPromptCachePointPositions"] = positions
	}
	if rawConfigs["promptCacheRetention"] != nil {
		retention, err := normalizePromptCacheRetentionConfig(rawConfigs["promptCacheRetention"])
		if err != nil {
			return err
		}
		if retention == "" {
			delete(rawConfigs, "promptCacheRetention")
		} else {
			rawConfigs["promptCacheRetention"] = retention
		}
	}
	return nil
}

func normalizeMCPServer(payload map[string]any, ingressClass string) (map[string]any, error) {
	name := strings.TrimSpace(fmt.Sprint(payload["name"]))
	if err := validateResourceName(name); err != nil {
		return nil, fmt.Errorf("invalid mcp server name: %w", err)
	}
	serverType := strings.ToUpper(strings.TrimSpace(fmt.Sprint(payload["type"])))
	if serverType == "" {
		serverType = "OPEN_API"
	}
	if !slices.Contains(mcpServerTypes, serverType) {
		return nil, fmt.Errorf("unsupported mcp server type %s", serverType)
	}
	payload["type"] = serverType
	payload["domains"] = normalizeStringSlice(payload["domains"])
	if services, err := normalizeOptionalUpstreamServices(payload["services"]); err != nil {
		return nil, err
	} else if len(services) > 0 {
		payload["services"] = services
	}
	switch serverType {
	case "OPEN_API":
		if rawConfigurations := stringValue(payload["rawConfigurations"]); rawConfigurations != "" {
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(rawConfigurations), &parsed); err != nil {
				return nil, fmt.Errorf("invalid rawConfigurations yaml: %w", err)
			}
			payload["rawConfigurations"] = rawConfigurations
		}
	case "DATABASE":
		dbType := stringValue(payload["dbType"])
		dsn := stringValue(payload["dsn"])
		if dbType == "" {
			return nil, errors.New("dbType is required for DATABASE mcp server")
		}
		if dsn == "" {
			return nil, errors.New("dsn is required for DATABASE mcp server")
		}
		payload["dbType"] = dbType
		payload["dsn"] = dsn
	case "DIRECT_ROUTE", "REDIRECT_ROUTE":
		directRouteConfig, _ := payload["directRouteConfig"].(map[string]any)
		directRouteConfig = clonePayload(directRouteConfig)
		if len(directRouteConfig) == 0 {
			directRouteConfig = map[string]any{
				"path":          strings.TrimSpace(fmt.Sprint(payload["upstreamPathPrefix"])),
				"transportType": firstNonEmpty(stringValue(payload["transportType"]), stringValue(payload["upstreamTransportType"])),
			}
		}
		if transportType := strings.ToLower(strings.TrimSpace(fmt.Sprint(directRouteConfig["transportType"]))); transportType != "" {
			if !slices.Contains([]string{"streamable", "sse"}, transportType) {
				return nil, fmt.Errorf("unsupported directRoute transportType %s", transportType)
			}
			directRouteConfig["transportType"] = transportType
		}
		if path := strings.TrimSpace(fmt.Sprint(directRouteConfig["path"])); path != "" {
			if !strings.HasPrefix(path, "/") {
				return nil, errors.New("directRouteConfig.path must start with '/'")
			}
			directRouteConfig["path"] = path
		}
		if len(directRouteConfig) > 0 {
			payload["directRouteConfig"] = directRouteConfig
		}
	}
	payload["routeMetadata"] = normalizeMCPRouteMetadata(name, payload["routeMetadata"], firstNonEmpty(stringValue(payload["ingressClass"]), ingressClass))
	return payload, nil
}

func normalizeProviderType(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	return normalized
}

func normalizeOpenAIProviderConfigs(rawConfigs map[string]any) error {
	if customURL := strings.TrimSpace(stringValue(rawConfigs["openaiCustomUrl"])); customURL != "" {
		if _, err := validateProviderURL(customURL, true); err != nil {
			return fmt.Errorf("openaiCustomUrl %w", err)
		}
		rawConfigs["openaiCustomUrl"] = customURL
		extraURLs, err := normalizeStringArray(rawConfigs["openaiExtraCustomUrls"], "openaiExtraCustomUrls")
		if err != nil {
			return err
		}
		for _, extraURL := range extraURLs {
			if _, err := validateProviderURL(extraURL, true); err != nil {
				return fmt.Errorf("openaiExtraCustomUrls %w", err)
			}
		}
		if len(extraURLs) > 0 {
			rawConfigs["openaiExtraCustomUrls"] = extraURLs
		} else {
			delete(rawConfigs, "openaiExtraCustomUrls")
		}
	}

	customService := strings.TrimSpace(stringValue(rawConfigs["openaiCustomServiceName"]))
	if customService != "" {
		if err := validateResourceName(customService); err != nil {
			return fmt.Errorf("invalid openaiCustomServiceName: %w", err)
		}
		rawConfigs["openaiCustomServiceName"] = customService
		port := toInt(rawConfigs["openaiCustomServicePort"])
		if port <= 0 || port > 65535 {
			return errors.New("openaiCustomServicePort must be a valid port number")
		}
		rawConfigs["openaiCustomServicePort"] = port
	}
	return nil
}

func normalizeAzureProviderConfigs(rawConfigs map[string]any) error {
	serviceURL := strings.TrimSpace(stringValue(rawConfigs["azureServiceUrl"]))
	if serviceURL == "" {
		return errors.New("azureServiceUrl cannot be empty")
	}
	if _, err := validateProviderURL(serviceURL, true); err != nil {
		return fmt.Errorf("azureServiceUrl %w", err)
	}
	rawConfigs["azureServiceUrl"] = serviceURL
	return nil
}

func normalizeQwenProviderConfigs(rawConfigs map[string]any) error {
	rawConfigs["qwenEnableSearch"] = boolFromAny(rawConfigs["qwenEnableSearch"], false)
	rawConfigs["qwenEnableCompatible"] = boolFromAny(rawConfigs["qwenEnableCompatible"], true)
	if rawConfigs["qwenFileIds"] != nil {
		fileIDs, err := normalizeStringArray(rawConfigs["qwenFileIds"], "qwenFileIds")
		if err != nil {
			return errors.New("invalid configuration: qwenFileIds")
		}
		rawConfigs["qwenFileIds"] = fileIDs
	}
	if domain := strings.TrimSpace(stringValue(rawConfigs["qwenDomain"])); domain != "" {
		if err := validateProviderDomain(domain); err != nil {
			return fmt.Errorf("qwenDomain contains an invalid domain name: %s", domain)
		}
		rawConfigs["qwenDomain"] = domain
	}
	return nil
}

func normalizeZhipuAIProviderConfigs(rawConfigs map[string]any) error {
	rawConfigs["zhipuCodePlanMode"] = boolFromAny(rawConfigs["zhipuCodePlanMode"], true)
	if domain := strings.TrimSpace(stringValue(rawConfigs["zhipuDomain"])); domain != "" {
		if err := validateProviderDomain(domain); err != nil {
			return fmt.Errorf("zhipuDomain contains an invalid domain name: %s", domain)
		}
		rawConfigs["zhipuDomain"] = domain
	}
	return nil
}

func normalizeClaudeProviderConfigs(rawConfigs map[string]any) {
	if rawConfigs["claudeCodeMode"] != nil {
		rawConfigs["claudeCodeMode"] = boolFromAny(rawConfigs["claudeCodeMode"], false)
	}
	if strings.TrimSpace(stringValue(rawConfigs["claudeVersion"])) == "" {
		rawConfigs["claudeVersion"] = "2023-06-01"
	}
}

func normalizeVertexProviderConfigs(rawConfigs map[string]any, tokens []string) error {
	region := strings.TrimSpace(stringValue(rawConfigs["vertexRegion"]))
	if region == "" && len(tokens) == 0 {
		return errors.New("vertexRegion cannot be empty")
	}
	if region != "" {
		rawConfigs["vertexRegion"] = strings.ToLower(region)
	}

	if strings.TrimSpace(stringValue(rawConfigs["vertexProjectId"])) == "" && len(tokens) == 0 {
		return errors.New("vertexProjectId cannot be empty")
	}
	authKey := strings.TrimSpace(stringValue(rawConfigs["vertexAuthKey"]))
	if authKey == "" && len(tokens) == 0 {
		return errors.New("vertexAuthKey cannot be empty")
	}
	if authKey != "" {
		authKeyJSON := map[string]any{}
		if err := json.Unmarshal([]byte(authKey), &authKeyJSON); err != nil {
			return fmt.Errorf("vertexAuthKey must contain a valid JSON object: %w", err)
		}
		for _, key := range []string{"client_email", "private_key_id", "private_key", "token_uri"} {
			if strings.TrimSpace(stringValue(authKeyJSON[key])) == "" {
				return fmt.Errorf("vertexAuthKey must contain a valid JSON object with a string property: %s", key)
			}
		}
		rawConfigs["vertexAuthKey"] = authKey
	}

	if rawConfigs["vertexTokenRefreshAhead"] != nil {
		refreshAhead := toInt(rawConfigs["vertexTokenRefreshAhead"])
		if refreshAhead < 0 {
			return errors.New("vertexTokenRefreshAhead must be a non-negative number")
		}
		rawConfigs["vertexTokenRefreshAhead"] = refreshAhead
	}
	if rawSafetySetting := rawConfigs["geminiSafetySetting"]; rawSafetySetting != nil {
		safetySetting, ok := rawSafetySetting.(map[string]any)
		if !ok {
			return errors.New("geminiSafetySetting must be an object")
		}
		for key, value := range safetySetting {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(stringValue(value)) == "" {
				return errors.New("geminiSafetySetting must be an object with string key-value pairs")
			}
		}
	}
	if len(tokens) == 0 {
		rawConfigs["vertexAuthServiceName"] = "vertex-auth.internal"
	} else {
		delete(rawConfigs, "vertexAuthServiceName")
	}
	return nil
}

func normalizeBedrockProviderConfigs(rawConfigs map[string]any) error {
	if strings.TrimSpace(stringValue(rawConfigs["awsRegion"])) == "" {
		return errors.New("awsRegion cannot be empty")
	}
	if strings.TrimSpace(stringValue(rawConfigs["awsAccessKey"])) == "" {
		return errors.New("awsAccessKey cannot be empty")
	}
	if strings.TrimSpace(stringValue(rawConfigs["awsSecretKey"])) == "" {
		return errors.New("awsSecretKey cannot be empty")
	}
	return nil
}

func normalizeBedrockPromptCachePointPositions(value any) (map[string]bool, error) {
	switch typed := value.(type) {
	case map[string]bool:
		result := make(map[string]bool, len(typed))
		for key, enabled := range typed {
			normalizedKey := normalizeBedrockPromptCachePointPositionKey(key)
			if normalizedKey == "" {
				return nil, fmt.Errorf("bedrockPromptCachePointPositions contains an empty key")
			}
			result[normalizedKey] = enabled
		}
		return result, nil
	case map[string]any:
		result := make(map[string]bool, len(typed))
		for key, raw := range typed {
			normalizedKey := normalizeBedrockPromptCachePointPositionKey(key)
			if normalizedKey == "" {
				return nil, fmt.Errorf("bedrockPromptCachePointPositions contains an empty key")
			}
			boolValue, err := normalizeBoolFieldValue(raw, "bedrockPromptCachePointPositions."+normalizedKey)
			if err != nil {
				return nil, err
			}
			result[normalizedKey] = boolValue
		}
		return result, nil
	default:
		return nil, errors.New("bedrockPromptCachePointPositions must be an object")
	}
}

func normalizePromptCacheRetentionConfig(value any) (string, error) {
	retention := strings.ToLower(strings.TrimSpace(stringValue(value)))
	retention = strings.ReplaceAll(retention, "-", "_")
	retention = strings.ReplaceAll(retention, " ", "_")
	switch retention {
	case "", "<nil>":
		return "", nil
	case "inmemory":
		return "in_memory", nil
	case "in_memory", "24h":
		return retention, nil
	default:
		return "", errors.New("promptCacheRetention must be in_memory or 24h")
	}
}

func normalizeBedrockPromptCachePointPositionKey(raw string) string {
	key := strings.TrimSpace(raw)
	if key == "" {
		return ""
	}
	normalized := strings.ToLower(key)
	normalized = strings.ReplaceAll(normalized, "_", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	switch normalized {
	case "systemprompt":
		return "systemPrompt"
	case "lastusermessage":
		return "lastUserMessage"
	case "lastmessage":
		return "lastMessage"
	default:
		return key
	}
}

func normalizeBoolFieldValue(value any, field string) (bool, error) {
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		switch normalized {
		case "true", "1", "yes":
			return true, nil
		case "false", "0", "no":
			return false, nil
		}
	}
	return false, fmt.Errorf("%s must be a boolean", field)
}

func normalizeOllamaProviderConfigs(rawConfigs map[string]any) error {
	host := strings.TrimSpace(stringValue(rawConfigs["ollamaServerHost"]))
	if host == "" {
		return errors.New("ollamaServerHost cannot be empty")
	}
	if !isValidHostnameOrIP(host) {
		return errors.New("ollamaServerHost must be a valid hostname or IP")
	}
	rawConfigs["ollamaServerHost"] = host
	port := toInt(rawConfigs["ollamaServerPort"])
	if port == 0 {
		port = 11434
	}
	if port < 1 || port > 65535 {
		return errors.New("ollamaServerPort must be a valid port number")
	}
	rawConfigs["ollamaServerPort"] = port
	return nil
}

func normalizeStringArray(value any, field string) ([]string, error) {
	switch typed := value.(type) {
	case nil:
		return nil, nil
	case []string:
		for _, item := range typed {
			if strings.TrimSpace(item) == "" {
				return nil, fmt.Errorf("%s must contain non-empty strings", field)
			}
		}
		return uniqueStrings(typed), nil
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(fmt.Sprint(item))
			if text == "" || text == "<nil>" {
				return nil, fmt.Errorf("%s must contain non-empty strings", field)
			}
			result = append(result, text)
		}
		return uniqueStrings(result), nil
	default:
		return nil, fmt.Errorf("%s must be an array", field)
	}
}

func validateProviderURL(raw string, requireScheme bool) (*neturl.URL, error) {
	parsed, err := neturl.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("contains an invalid URL: %s", raw)
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if requireScheme && scheme == "" {
		return nil, errors.New("must have a scheme")
	}
	if scheme != "" && scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("must have a valid scheme: %s", raw)
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return nil, fmt.Errorf("contains an invalid URL: %s", raw)
	}
	return parsed, nil
}

func validateProviderDomain(domain string) error {
	parsed, err := neturl.Parse("https://" + strings.TrimSpace(domain))
	if err != nil {
		return err
	}
	if strings.TrimSpace(parsed.Hostname()) == "" {
		return errors.New("domain is empty")
	}
	if parsed.Hostname() != strings.TrimSpace(domain) {
		return errors.New("domain is invalid")
	}
	return nil
}

func boolFromAny(value any, fallback bool) bool {
	switch typed := value.(type) {
	case nil:
		return fallback
	case bool:
		return typed
	case string:
		normalized := strings.TrimSpace(strings.ToLower(typed))
		if normalized == "" {
			return fallback
		}
		return normalized == "true" || normalized == "1" || normalized == "yes"
	default:
		return strings.TrimSpace(strings.ToLower(fmt.Sprint(value))) == "true"
	}
}

func isValidHostnameOrIP(value string) bool {
	if net.ParseIP(value) != nil {
		return true
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || strings.ContainsAny(trimmed, "/:") {
		return false
	}
	return regexp.MustCompile(`^[a-zA-Z0-9]([-.a-zA-Z0-9]*[a-zA-Z0-9])?$`).MatchString(trimmed)
}

func (s *Service) normalizeWasmPlugin(payload map[string]any) (map[string]any, error) {
	name := strings.TrimSpace(fmt.Sprint(payload["name"]))
	if name == "" {
		return nil, errors.New("plugin name is required")
	}

	legacy, _ := s.loadBuiltinWasmPlugin(name)
	if phase := strings.ToUpper(strings.TrimSpace(fmt.Sprint(payload["phase"]))); phase != "" {
		if !slices.Contains(wasmPhases, phase) {
			return nil, fmt.Errorf("unsupported wasm plugin phase %s", phase)
		}
		payload["phase"] = phase
	} else if legacy != nil && legacy["phase"] != nil {
		payload["phase"] = legacy["phase"]
	}

	if priority := toInt(payload["priority"]); priority != 0 {
		payload["priority"] = priority
	} else if legacy != nil && legacy["priority"] != nil {
		payload["priority"] = legacy["priority"]
	}

	if payload["configSchema"] == nil && payload["schema"] == nil && legacy != nil {
		if schema := firstNonNil(legacy["configSchema"], legacy["schema"]); schema != nil {
			payload["configSchema"] = schema
		}
	}
	if payload["readme"] == nil && legacy != nil && strings.TrimSpace(fmt.Sprint(legacy["readme"])) != "" {
		payload["readme"] = legacy["readme"]
	}
	if payload["description"] == nil && legacy != nil && strings.TrimSpace(fmt.Sprint(legacy["description"])) != "" {
		payload["description"] = legacy["description"]
	}
	if payload["title"] == nil && legacy != nil && strings.TrimSpace(fmt.Sprint(legacy["title"])) != "" {
		payload["title"] = legacy["title"]
	}

	if schema := firstNonNil(payload["configSchema"], payload["schema"]); schema != nil {
		if _, err := extractJSONSchema(schema); err != nil {
			return nil, fmt.Errorf("invalid wasm plugin schema: %w", err)
		}
	}
	return payload, nil
}

func (s *Service) validatePluginInstance(ctx context.Context, scope, target, pluginName string, payload map[string]any) error {
	normalizedScope := strings.TrimSpace(scope)
	if normalizedScope == "" {
		return errors.New("plugin scope is required")
	}
	if normalizedScope != "global" && strings.TrimSpace(target) == "" {
		return errors.New("plugin target is required")
	}
	if normalizedScope != "global" {
		targetKind := normalizedScope + "s"
		if normalizedScope == "service" {
			targetKind = "services"
		}
		if _, err := s.k8sClient.GetResource(ctx, targetKind, target); err != nil {
			return fmt.Errorf("%s %s not found", normalizedScope, target)
		}
	}

	rawConfig, _ := payload["config"].(map[string]any)
	if rawConfig == nil {
		rawConfig = map[string]any{}
	}
	if rawConfigurations := strings.TrimSpace(fmt.Sprint(payload["rawConfigurations"])); rawConfigurations != "" {
		payload["rawConfigurations"] = rawConfigurations
	}

	item, err := s.Get(ctx, "wasm-plugins", pluginName)
	if err != nil {
		return nil
	}
	schema, err := extractJSONSchema(firstNonNil(item["configSchema"], item["schema"]))
	if err != nil || schema == nil {
		return nil
	}
	if err := validateBySchema(rawConfig, schema, "config"); err != nil {
		return err
	}
	return nil
}

func validateResourceName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("name is required")
	}
	if !rfc1123NamePattern.MatchString(trimmed) {
		return errors.New("name must be a lowercase RFC1123 subdomain")
	}
	return nil
}

func normalizeMethods(value any) ([]string, error) {
	items := normalizeStringSlice(value)
	result := make([]string, 0, len(items))
	for _, item := range items {
		method := strings.ToUpper(strings.TrimSpace(item))
		if method == "" {
			continue
		}
		if !slices.Contains(httpMethods, method) {
			return nil, fmt.Errorf("unsupported HTTP method %s", method)
		}
		result = append(result, method)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeRoutePredicate(value any, requireKey bool) (map[string]any, error) {
	predicate, _ := value.(map[string]any)
	if len(predicate) == 0 {
		return nil, errors.New("route predicate is required")
	}
	matchType := strings.ToUpper(strings.TrimSpace(fmt.Sprint(predicate["matchType"])))
	if !slices.Contains(matchTypes, matchType) {
		return nil, fmt.Errorf("unsupported matchType %s", matchType)
	}
	matchValue := strings.TrimSpace(fmt.Sprint(predicate["matchValue"]))
	if matchValue == "" {
		return nil, errors.New("matchValue is required")
	}
	if requireKey && strings.TrimSpace(fmt.Sprint(predicate["key"])) == "" {
		return nil, errors.New("predicate key is required")
	}
	if matchType != "REGULAR" && !strings.HasPrefix(matchValue, "/") && !requireKey {
		return nil, errors.New("path matchValue must start with '/'")
	}
	result := clonePayload(predicate)
	result["matchType"] = matchType
	result["matchValue"] = matchValue
	if requireKey {
		result["key"] = strings.TrimSpace(fmt.Sprint(predicate["key"]))
	}
	return result, nil
}

func normalizeKeyedPredicates(value any) ([]map[string]any, error) {
	items := toAnySlice(value)
	if len(items) == 0 {
		return nil, nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		predicate, err := normalizeRoutePredicate(item, true)
		if err != nil {
			return nil, err
		}
		result = append(result, predicate)
	}
	return result, nil
}

func normalizeUpstreamServices(value any) ([]map[string]any, error) {
	rawItems := toAnySlice(value)
	if len(rawItems) == 0 {
		return nil, errors.New("services cannot be empty")
	}
	return normalizeUpstreamServicesItems(rawItems)
}

func normalizeOptionalUpstreamServices(value any) ([]map[string]any, error) {
	rawItems := toAnySlice(value)
	if len(rawItems) == 0 {
		return nil, nil
	}
	return normalizeUpstreamServicesItems(rawItems)
}

func normalizeUpstreamServicesItems(rawItems []any) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(rawItems))
	for _, rawItem := range rawItems {
		item, _ := rawItem.(map[string]any)
		name := strings.TrimSpace(fmt.Sprint(item["name"]))
		if name == "" {
			return nil, errors.New("service name is required")
		}
		port := toInt(item["port"])
		if port != 0 && (port < 1 || port > 65535) {
			return nil, fmt.Errorf("service %s port must be between 1 and 65535", name)
		}
		weight := toInt(item["weight"])
		if weight != 0 && weight < 0 {
			return nil, fmt.Errorf("service %s weight must be greater than or equal to 0", name)
		}
		normalized := map[string]any{"name": name}
		if port != 0 {
			normalized["port"] = port
		}
		if version := strings.TrimSpace(fmt.Sprint(item["version"])); version != "" {
			normalized["version"] = version
		}
		if weight != 0 {
			normalized["weight"] = weight
		}
		result = append(result, normalized)
	}
	return result, nil
}

func normalizeAuthConfig(value any) (map[string]any, error) {
	item, _ := value.(map[string]any)
	if len(item) == 0 {
		return nil, nil
	}
	result := map[string]any{
		"enabled": fmt.Sprint(item["enabled"]) == "true",
	}
	allowedConsumers := normalizeStringSlice(item["allowedConsumers"])
	if len(allowedConsumers) > 0 {
		result["allowedConsumers"] = allowedConsumers
	}
	levels := normalizeStringSlice(item["allowedConsumerLevels"])
	for _, level := range levels {
		if !slices.Contains([]string{"normal", "plus", "pro", "ultra"}, level) {
			return nil, fmt.Errorf("unsupported consumer level %s", level)
		}
	}
	if len(levels) > 0 {
		result["allowedConsumerLevels"] = levels
	}
	return result, nil
}

func normalizeMCPRouteMetadata(serverName string, value any, ingressClass string) map[string]any {
	item, _ := value.(map[string]any)
	result := clonePayload(item)
	if result == nil {
		result = map[string]any{}
	}
	result["managedBy"] = "aigateway-console"
	result["routeName"] = defaultMCPRouteName(serverName)
	result["mcpServerName"] = serverName
	if ingressClass != "" {
		result["ingressClass"] = ingressClass
	}
	return result
}

func normalizeRouteBindingMetadata(routeName string, value any, ingressClass string) map[string]any {
	item, _ := value.(map[string]any)
	result := clonePayload(item)
	if result == nil {
		result = map[string]any{}
	}
	serverName, ok := mcpServerNameFromRouteName(routeName)
	if !ok {
		if len(result) == 0 {
			return nil
		}
		if ingressClass != "" && stringValue(result["ingressClass"]) == "" {
			result["ingressClass"] = ingressClass
		}
		return result
	}
	result["managedBy"] = "aigateway-console"
	result["mcpServerName"] = serverName
	result["routeName"] = routeName
	if ingressClass != "" {
		result["ingressClass"] = ingressClass
	}
	return result
}

func defaultMCPRouteName(serverName string) string {
	return fmt.Sprintf("mcp-server-%s.internal", strings.TrimSpace(serverName))
}

func mcpServerNameFromRouteName(routeName string) (string, bool) {
	trimmed := strings.TrimSpace(routeName)
	if !strings.HasPrefix(trimmed, "mcp-server-") || !strings.HasSuffix(trimmed, ".internal") {
		return "", false
	}
	name := strings.TrimSuffix(strings.TrimPrefix(trimmed, "mcp-server-"), ".internal")
	if name == "" {
		return "", false
	}
	return name, true
}

func shouldExposeIngressClass(kind string) bool {
	switch kind {
	case "routes", "ai-routes", "domains", "services", "mcp-servers":
		return true
	default:
		return false
	}
}

func extractJSONSchema(value any) (map[string]any, error) {
	item, _ := value.(map[string]any)
	if len(item) == 0 {
		return nil, nil
	}
	if schema, ok := item["openAPIV3Schema"].(map[string]any); ok {
		return schema, nil
	}
	if spec, ok := item["spec"].(map[string]any); ok {
		if schema, ok := spec["configSchema"].(map[string]any); ok {
			return extractJSONSchema(schema)
		}
		if schema, ok := spec["routeConfigSchema"].(map[string]any); ok {
			return extractJSONSchema(schema)
		}
	}
	if _, ok := item["type"]; ok {
		return item, nil
	}
	return nil, nil
}

func validateBySchema(value any, schema map[string]any, path string) error {
	if len(schema) == 0 {
		return nil
	}
	schemaType := strings.TrimSpace(fmt.Sprint(schema["type"]))
	switch schemaType {
	case "", "object":
		objectValue, _ := value.(map[string]any)
		if objectValue == nil {
			return fmt.Errorf("%s must be an object", path)
		}
		required := normalizeStringSlice(schema["required"])
		for _, key := range required {
			if _, ok := objectValue[key]; !ok {
				return fmt.Errorf("%s.%s is required", path, key)
			}
		}
		properties, _ := schema["properties"].(map[string]any)
		for key, rawSchema := range properties {
			childSchema, _ := rawSchema.(map[string]any)
			if childSchema == nil {
				continue
			}
			childValue, ok := objectValue[key]
			if !ok {
				continue
			}
			if err := validateBySchema(childValue, childSchema, path+"."+key); err != nil {
				return err
			}
		}
	case "array":
		items := toAnySlice(value)
		if len(items) == 0 && value != nil {
			return fmt.Errorf("%s must be an array", path)
		}
		itemSchema, _ := schema["items"].(map[string]any)
		for index, item := range items {
			if err := validateBySchema(item, itemSchema, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	case "string":
		if _, ok := value.(string); !ok {
			return fmt.Errorf("%s must be a string", path)
		}
	case "number", "integer":
		if !isNumeric(value) {
			return fmt.Errorf("%s must be a number", path)
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s must be a boolean", path)
		}
	}
	return nil
}

func normalizeStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return uniqueStrings(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, strings.TrimSpace(fmt.Sprint(item)))
		}
		return uniqueStrings(items)
	default:
		raw := strings.TrimSpace(fmt.Sprint(value))
		if raw == "" || raw == "<nil>" {
			return []string{}
		}
		return []string{raw}
	}
}

func toAnySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	case []map[string]any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, item)
		}
		return items
	default:
		return nil
	}
}

func toMapSlice(value any) []map[string]any {
	items := toAnySlice(value)
	if len(items) == 0 {
		return nil
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		typed, _ := item.(map[string]any)
		if len(typed) == 0 {
			continue
		}
		result = append(result, typed)
	}
	return result
}

func uniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func toInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		number, _ := typed.Int64()
		return int(number)
	default:
		number, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
		return number
	}
}

func isNumeric(value any) bool {
	switch value.(type) {
	case int, int32, int64, float32, float64, json.Number:
		return true
	default:
		_, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(value)), 64)
		return err == nil
	}
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func clonePayloadMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	bytes, _ := json.Marshal(src)
	dst := map[string]any{}
	_ = json.Unmarshal(bytes, &dst)
	return dst
}
