package k8s

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
)

const (
	higressAnnotationDestination             = "higress.io/destination"
	higressAnnotationUseRegex                = "higress.io/use-regex"
	higressAnnotationIgnorePathCase          = "higress.io/ignore-path-case"
	higressAnnotationMatchMethod             = "higress.io/match-method"
	higressAnnotationResourceDescription     = "higress.io/resource-description"
	higressAnnotationProxyNextUpstream       = "higress.io/proxy-next-upstream"
	higressAnnotationProxyNextTries          = "higress.io/proxy-next-upstream-tries"
	higressAnnotationProxyNextTimeout        = "higress.io/proxy-next-upstream-timeout"
	higressAnnotationProxyNextEnabled        = "higress.io/enable-proxy-next-upstream"
	higressAnnotationRewriteEnabled          = "higress.io/enable-rewrite"
	higressAnnotationRewritePath             = "higress.io/rewrite-path"
	higressAnnotationRewriteTarget           = "higress.io/rewrite-target"
	higressAnnotationUpstreamVHost           = "higress.io/upstream-vhost"
	higressAnnotationAuthConsumerDepartments = "higress.io/auth-consumer-departments"
	higressAnnotationAuthConsumerLevels      = "higress.io/auth-consumer-levels"
	higressAnnotationComment                 = "higress.io/comment"
	higressAnnotationHeaderMatch             = "-match-header-"
	higressAnnotationQueryMatch              = "-match-query-"
	higressAnnotationPseudoHeaderMatch       = "-match-pseudo-header-"
	higressLabelDomainPrefix                 = "higress.io/domain_"
	higressLabelConfigMapType                = "higress.io/config-map-type"
	higressLabelConfigMapTypeDomain          = "domain"
	higressLabelConfigMapTypeAIRoute         = "ai-route"
	higressLabelResourceDefiner              = "higress.io/resource-definer"
	higressLabelResourceDefinerValue         = "higress"
	higressLabelInternal                     = "higress.io/internal"
	higressLabelBizType                      = "higress.io/biz-type"
	higressLabelBizTypeMCPServer             = "mcp-server"
	higressLabelMCPServerType                = "higress.io/mcp-server-type"
	higressLabelWasmPluginName               = "higress.io/wasm-plugin-name"
	higressLabelWasmPluginVersion            = "higress.io/wasm-plugin-version"
	higressAnnotationTrueValue               = "true"
	higressSecretTLS                         = "kubernetes.io/tls"
	higressSecretTLSCRT                      = "tls.crt"
	higressSecretTLSKey                      = "tls.key"
	higressDataField                         = "data"
	higressMcpBridgeKind                     = "McpBridge"
	higressMcpBridgeAPIGroup                 = "networking.higress.io"
	higressMcpBridgeDefaultName              = "default"
	higressConfigMapName                     = "higress-config"
	higressWasmPluginResource                = "wasmplugin.extensions.higress.io"
	higressEnvoyFilterResource               = "envoyfilters.networking.istio.io"
	higressAIRoutePrefix                     = "ai-route-"
	higressDomainPrefix                      = "domain-"
	higressProviderDefaultProtocol           = "openai/v1"
	higressProviderPluginProtocolOpenAI      = "openai"
	higressMCPConfigKey                      = "higress"
	higressMCPServerPathPrefix               = "/mcp-servers"
)

func isControlPlaneKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "routes", "domains", "tls-certificates", "ai-routes", "ai-providers", "mcp-servers", "service-sources":
		return true
	default:
		return false
	}
}

func (c *RealClient) listControlPlaneResources(ctx context.Context, kind string) ([]map[string]any, error) {
	switch kind {
	case "routes":
		return c.listIngressResources(ctx)
	case "domains":
		return c.listDomainResources(ctx)
	case "tls-certificates":
		return c.listTLSCertificateResources(ctx)
	case "ai-routes":
		return c.listAIRouteResources(ctx)
	case "ai-providers":
		return c.listAIProviderResources(ctx)
	case "mcp-servers":
		return c.listMCPServerResources(ctx)
	case "service-sources":
		return c.listServiceSourceResources(ctx)
	default:
		return nil, ErrNotFound
	}
}

func (c *RealClient) getControlPlaneResource(ctx context.Context, kind, name string) (map[string]any, error) {
	switch kind {
	case "routes":
		return c.getIngressResource(ctx, name)
	case "domains":
		return c.getDomainResource(ctx, name)
	case "tls-certificates":
		return c.getTLSCertificateResource(ctx, name)
	case "ai-routes":
		return c.getAIRouteResource(ctx, name)
	case "ai-providers":
		return c.getAIProviderResource(ctx, name)
	case "mcp-servers":
		return c.getMCPServerResource(ctx, name)
	case "service-sources":
		return c.getServiceSourceResource(ctx, name)
	default:
		return nil, ErrNotFound
	}
}

func (c *RealClient) upsertControlPlaneResource(ctx context.Context, kind, name string, data map[string]any) (map[string]any, error) {
	switch kind {
	case "routes":
		return c.upsertIngressResource(ctx, name, data)
	case "domains":
		return c.upsertDomainResource(ctx, name, data)
	case "tls-certificates":
		return c.upsertTLSCertificateResource(ctx, name, data)
	case "ai-routes":
		return c.upsertAIRouteResource(ctx, name, data)
	case "ai-providers":
		return c.upsertAIProviderResource(ctx, name, data)
	case "mcp-servers":
		return c.upsertMCPServerResource(ctx, name, data)
	case "service-sources":
		return c.upsertServiceSourceResource(ctx, name, data)
	default:
		return nil, ErrNotFound
	}
}

func (c *RealClient) deleteControlPlaneResource(ctx context.Context, kind, name string) error {
	switch kind {
	case "routes":
		_, err := c.run(ctx, nil, "delete", "ingress", name, "--ignore-not-found=false")
		return err
	case "domains":
		_, err := c.run(ctx, nil, "delete", "configmap", domainConfigMapName(name), "--ignore-not-found=false")
		return err
	case "tls-certificates":
		_, err := c.run(ctx, nil, "delete", "secret", name, "--ignore-not-found=false")
		return err
	case "ai-routes":
		return c.deleteAIRouteResource(ctx, name)
	case "ai-providers":
		return c.deleteAIProviderResource(ctx, name)
	case "mcp-servers":
		return c.deleteMCPServerResource(ctx, name)
	case "service-sources":
		return c.deleteServiceSourceResource(ctx, name)
	default:
		return ErrNotFound
	}
}

func (c *RealClient) listIngressResources(ctx context.Context) ([]map[string]any, error) {
	items, err := c.listObjects(ctx, "ingress", "-l", buildLabelSelector(higressLabelResourceDefiner, higressLabelResourceDefinerValue))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, ingressToRoute(item, c.ingressClass))
	}
	sortResourcesByName(result)
	return result, nil
}

func (c *RealClient) getIngressResource(ctx context.Context, name string) (map[string]any, error) {
	item, err := c.getObject(ctx, "ingress", name)
	if err != nil {
		return nil, err
	}
	return ingressToRoute(item, c.ingressClass), nil
}

func (c *RealClient) upsertIngressResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	metadata := map[string]any{
		"name":      name,
		"namespace": c.namespace,
		"labels":    routeLabels(data),
	}
	annotations := routeAnnotations(data)
	if len(annotations) > 0 {
		metadata["annotations"] = annotations
	}

	manifest := map[string]any{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "Ingress",
		"metadata":   metadata,
		"spec": map[string]any{
			"ingressClassName": firstNonEmpty(stringValue(data["ingressClass"]), c.ingressClass),
			"rules":            routeRules(data),
		},
	}
	if tls := c.routeTLS(data); len(tls) > 0 {
		manifest["spec"].(map[string]any)["tls"] = tls
	}
	payload, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if _, err := c.run(ctx, payload, "apply", "-f", "-"); err != nil {
		return nil, err
	}
	return c.getIngressResource(ctx, name)
}

func (c *RealClient) listDomainResources(ctx context.Context) ([]map[string]any, error) {
	items, err := c.listObjects(ctx, "configmap", "-l", strings.Join([]string{
		buildLabelSelector(higressLabelResourceDefiner, higressLabelResourceDefinerValue),
		buildLabelSelector(higressLabelConfigMapType, higressLabelConfigMapTypeDomain),
	}, ","))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, configMapToDomain(item))
	}
	sortResourcesByName(result)
	return result, nil
}

func (c *RealClient) getDomainResource(ctx context.Context, name string) (map[string]any, error) {
	item, err := c.getObject(ctx, "configmap", domainConfigMapName(name))
	if err != nil {
		return nil, err
	}
	return configMapToDomain(item), nil
}

func (c *RealClient) upsertDomainResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	manifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      domainConfigMapName(name),
			"namespace": c.namespace,
			"labels": map[string]string{
				higressLabelConfigMapType:   higressLabelConfigMapTypeDomain,
				higressLabelResourceDefiner: higressLabelResourceDefinerValue,
			},
		},
		"data": map[string]string{
			"domain":      name,
			"cert":        stringValue(data["certIdentifier"]),
			"enableHttps": firstNonEmpty(stringValue(data["enableHttps"]), "off"),
		},
	}
	payload, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if _, err := c.run(ctx, payload, "apply", "-f", "-"); err != nil {
		return nil, err
	}
	return c.getDomainResource(ctx, name)
}

func (c *RealClient) listTLSCertificateResources(ctx context.Context) ([]map[string]any, error) {
	items, err := c.listObjects(ctx, "secret")
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if stringValue(nestedValue(item, "type")) != higressSecretTLS {
			continue
		}
		result = append(result, secretToTLSCertificate(item))
	}
	sortResourcesByName(result)
	return result, nil
}

func (c *RealClient) getTLSCertificateResource(ctx context.Context, name string) (map[string]any, error) {
	item, err := c.getObject(ctx, "secret", name)
	if err != nil {
		return nil, err
	}
	if stringValue(item["type"]) != higressSecretTLS {
		return nil, ErrNotFound
	}
	return secretToTLSCertificate(item), nil
}

func (c *RealClient) upsertTLSCertificateResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	manifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{
			"name":      name,
			"namespace": c.namespace,
		},
		"type": higressSecretTLS,
		"stringData": map[string]string{
			higressSecretTLSCRT: stringValue(data["cert"]),
			higressSecretTLSKey: stringValue(data["key"]),
		},
	}
	payload, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if _, err := c.run(ctx, payload, "apply", "-f", "-"); err != nil {
		return nil, err
	}
	return c.getTLSCertificateResource(ctx, name)
}

func (c *RealClient) listAIRouteResources(ctx context.Context) ([]map[string]any, error) {
	items, err := c.listObjects(ctx, "configmap", "-l", strings.Join([]string{
		buildLabelSelector(higressLabelResourceDefiner, higressLabelResourceDefinerValue),
		buildLabelSelector(higressLabelConfigMapType, higressLabelConfigMapTypeAIRoute),
	}, ","))
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		resource, err := configMapToAIRoute(item)
		if err != nil {
			return nil, err
		}
		result = append(result, resource)
	}
	sortResourcesByName(result)
	return result, nil
}

func (c *RealClient) getAIRouteResource(ctx context.Context, name string) (map[string]any, error) {
	item, err := c.getObject(ctx, "configmap", aiRouteConfigMapName(name))
	if err != nil {
		return nil, err
	}
	return configMapToAIRoute(item)
}

func (c *RealClient) upsertAIRouteResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	payloadData := cloneMap(data)
	delete(payloadData, "version")
	rawData, err := json.Marshal(payloadData)
	if err != nil {
		return nil, err
	}
	manifest := map[string]any{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]any{
			"name":      aiRouteConfigMapName(name),
			"namespace": c.namespace,
			"labels": map[string]string{
				higressLabelConfigMapType:   higressLabelConfigMapTypeAIRoute,
				higressLabelResourceDefiner: higressLabelResourceDefinerValue,
			},
		},
		"data": map[string]string{
			higressDataField: string(rawData),
		},
	}
	payload, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	if _, err := c.run(ctx, payload, "apply", "-f", "-"); err != nil {
		return nil, err
	}
	if err := c.syncAIRouteRuntime(ctx, name, payloadData); err != nil {
		return nil, err
	}
	if err := c.SyncAIDataMaskingRuntime(ctx); err != nil {
		return nil, err
	}
	return c.getAIRouteResource(ctx, name)
}

func (c *RealClient) listAIProviderResources(ctx context.Context) ([]map[string]any, error) {
	plugin, err := c.getAIProxyWasmPlugin(ctx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	items := wasmPluginToProviders(plugin)
	sortResourcesByName(items)
	return items, nil
}

func (c *RealClient) getAIProviderResource(ctx context.Context, name string) (map[string]any, error) {
	items, err := c.listAIProviderResources(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.EqualFold(stringValue(item["name"]), strings.TrimSpace(name)) {
			return item, nil
		}
	}
	return nil, ErrNotFound
}

func (c *RealClient) upsertAIProviderResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	previous, lookupErr := c.getAIProviderResource(ctx, name)
	if lookupErr != nil && !errors.Is(lookupErr, ErrNotFound) {
		return nil, lookupErr
	}
	item, err := c.updateAIProxyWasmPlugin(ctx, func(plugin map[string]any) error {
		spec := ensureMap(plugin, "spec")
		defaultConfig := ensureMap(spec, "defaultConfig")
		providers := toMapSlice(defaultConfig["providers"])
		next := providerPayloadFromResource(name, data)
		updated := false
		for index, item := range providers {
			if strings.EqualFold(stringValue(item["id"]), name) {
				providers[index] = next
				updated = true
				break
			}
		}
		if !updated {
			providers = append(providers, next)
		}
		defaultConfig["providers"] = providers
		return nil
	}, name)
	if err != nil {
		return nil, err
	}
	if errors.Is(lookupErr, ErrNotFound) {
		previous = nil
	}
	if err := c.syncAIProviderRuntime(ctx, name, data, previous, false); err != nil {
		return nil, err
	}
	return item, nil
}

func (c *RealClient) deleteAIProviderResource(ctx context.Context, name string) error {
	provider, lookupErr := c.getAIProviderResource(ctx, name)
	if lookupErr != nil && !errors.Is(lookupErr, ErrNotFound) {
		return lookupErr
	}
	_, err := c.updateAIProxyWasmPlugin(ctx, func(plugin map[string]any) error {
		spec := ensureMap(plugin, "spec")
		defaultConfig := ensureMap(spec, "defaultConfig")
		providers := toMapSlice(defaultConfig["providers"])
		next := make([]map[string]any, 0, len(providers))
		found := false
		for _, item := range providers {
			if strings.EqualFold(stringValue(item["id"]), name) {
				found = true
				continue
			}
			next = append(next, item)
		}
		if !found {
			return ErrNotFound
		}
		defaultConfig["providers"] = next
		return nil
	}, "")
	if err != nil {
		return err
	}
	if lookupErr == nil {
		return c.syncAIProviderRuntime(ctx, name, nil, provider, true)
	}
	return nil
}

func (c *RealClient) listServiceSourceResources(ctx context.Context) ([]map[string]any, error) {
	registries, err := c.loadMcpBridgeRegistries(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(registries))
	for _, registry := range registries {
		result = append(result, serviceSourceFromRegistry(registry))
	}
	sortResourcesByName(result)
	return result, nil
}

func (c *RealClient) getServiceSourceResource(ctx context.Context, name string) (map[string]any, error) {
	registries, err := c.loadMcpBridgeRegistries(ctx)
	if err != nil {
		return nil, err
	}
	item, ok := registries[strings.TrimSpace(name)]
	if !ok {
		return nil, ErrNotFound
	}
	return serviceSourceFromRegistry(item), nil
}

func (c *RealClient) upsertServiceSourceResource(ctx context.Context, name string, data map[string]any) (map[string]any, error) {
	if err := c.upsertMcpBridgeRegistry(ctx, serviceSourceToRegistry(name, data)); err != nil {
		return nil, err
	}
	return c.getServiceSourceResource(ctx, name)
}

func (c *RealClient) deleteServiceSourceResource(ctx context.Context, name string) error {
	return c.removeMcpBridgeRegistry(ctx, name)
}

func (c *RealClient) updateAIProxyWasmPlugin(ctx context.Context, mutate func(map[string]any) error, providerName string) (map[string]any, error) {
	plugin, err := c.getAIProxyWasmPlugin(ctx)
	if err != nil {
		return nil, err
	}
	working := cloneMap(plugin)
	if err := mutate(working); err != nil {
		return nil, err
	}
	sanitizeObjectForApply(working)
	payload, err := yaml.Marshal(working)
	if err != nil {
		return nil, err
	}
	if _, err := c.run(ctx, payload, "apply", "-f", "-"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(providerName) == "" {
		return map[string]any{}, nil
	}
	return c.getAIProviderResource(ctx, providerName)
}

func (c *RealClient) getAIProxyWasmPlugin(ctx context.Context) (map[string]any, error) {
	name := builtinWasmPluginResourceName(higressWasmPluginNameAIProxy)
	plugin, err := c.getObject(ctx, higressWasmPluginResource, name)
	if err == nil {
		return plugin, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	if err := c.ensureBuiltinWasmPlugin(ctx, higressWasmPluginNameAIProxy); err != nil {
		return nil, err
	}
	return c.getObject(ctx, higressWasmPluginResource, name)
}

func (c *RealClient) listObjects(ctx context.Context, resource string, extraArgs ...string) ([]map[string]any, error) {
	args := append([]string{"get", resource}, extraArgs...)
	args = append(args, "-o", "json")
	body, err := c.run(ctx, nil, args...)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func (c *RealClient) getObject(ctx context.Context, resource, name string) (map[string]any, error) {
	body, err := c.run(ctx, nil, "get", resource, name, "-o", "json")
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func ingressToRoute(item map[string]any, defaultIngressClass string) map[string]any {
	metadata := mapValue(item["metadata"])
	spec := mapValue(item["spec"])
	annotations := stringMap(metadata["annotations"])
	labels := stringMap(metadata["labels"])

	route := map[string]any{
		"name":         stringValue(metadata["name"]),
		"version":      stringValue(metadata["resourceVersion"]),
		"domains":      ingressDomains(spec, labels),
		"ingressClass": firstNonEmpty(stringValue(spec["ingressClassName"]), defaultIngressClass),
		"readonly":     stringValue(labels[higressLabelResourceDefiner]) != higressLabelResourceDefinerValue,
	}
	if path := ingressPath(spec, annotations); len(path) > 0 {
		route["path"] = path
	}
	if services := parseDestinationServices(annotations[higressAnnotationDestination]); len(services) > 0 {
		route["services"] = services
	}
	if methods := splitAndTrim(annotations[higressAnnotationMatchMethod], ","); len(methods) > 0 {
		route["methods"] = methods
	}
	if headers := ingressKeyedPredicates(annotations, higressAnnotationHeaderMatch, higressAnnotationPseudoHeaderMatch); len(headers) > 0 {
		route["headers"] = headers
	}
	if params := ingressKeyedPredicates(annotations, higressAnnotationQueryMatch); len(params) > 0 {
		route["urlParams"] = params
	}
	if rewrite := ingressRewrite(annotations); len(rewrite) > 0 {
		route["rewrite"] = rewrite
	}
	if authConfig := ingressAuthConfig(annotations); len(authConfig) > 0 {
		route["authConfig"] = authConfig
	}
	if proxyNext := ingressProxyNextUpstreamConfig(annotations); len(proxyNext) > 0 {
		route["proxyNextUpstream"] = proxyNext
	}
	if description := strings.TrimSpace(annotations[higressAnnotationResourceDescription]); description != "" {
		route["description"] = description
	}
	return route
}

func configMapToDomain(item map[string]any) map[string]any {
	metadata := mapValue(item["metadata"])
	data := stringMap(item["data"])
	return map[string]any{
		"name":           firstNonEmpty(data["domain"], strings.TrimPrefix(stringValue(metadata["name"]), higressDomainPrefix)),
		"version":        stringValue(metadata["resourceVersion"]),
		"certIdentifier": data["cert"],
		"enableHttps":    data["enableHttps"],
	}
}

func secretToTLSCertificate(item map[string]any) map[string]any {
	metadata := mapValue(item["metadata"])
	data := stringMap(item["data"])
	return map[string]any{
		"name":    stringValue(metadata["name"]),
		"version": stringValue(metadata["resourceVersion"]),
		"cert":    decodeBase64Value(data[higressSecretTLSCRT]),
		"key":     decodeBase64Value(data[higressSecretTLSKey]),
	}
}

func configMapToAIRoute(item map[string]any) (map[string]any, error) {
	metadata := mapValue(item["metadata"])
	data := stringMap(item["data"])
	raw := strings.TrimSpace(data[higressDataField])
	if raw == "" {
		return nil, errors.New("ai route configmap data is empty")
	}
	payload := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	payload["version"] = stringValue(metadata["resourceVersion"])
	return payload, nil
}

func wasmPluginToProviders(plugin map[string]any) []map[string]any {
	spec := mapValue(plugin["spec"])
	defaultConfig := mapValue(spec["defaultConfig"])
	providers := toMapSlice(defaultConfig["providers"])
	result := make([]map[string]any, 0, len(providers))
	for _, provider := range providers {
		name := stringValue(provider["id"])
		if name == "" {
			continue
		}
		providerType := canonicalProviderType(stringValue(provider["type"]))
		item := map[string]any{
			"name":       name,
			"type":       providerType,
			"protocol":   providerProtocolValue(provider["protocol"]),
			"tokens":     normalizeStringSlice(provider["apiTokens"]),
			"rawConfigs": normalizeProviderRawConfigsForConsole(cloneMap(provider), providerType),
		}
		if failover := mapValue(provider["failover"]); len(failover) > 0 {
			item["tokenFailoverConfig"] = cloneMap(failover)
		}
		if models := providerModelCatalog(name, providerType); len(models) > 0 {
			item["models"] = models
		}
		result = append(result, item)
	}
	return result
}

func serviceSourceFromRegistry(registry map[string]any) map[string]any {
	item := map[string]any{
		"name":       stringValue(registry["name"]),
		"type":       stringValue(registry["type"]),
		"domain":     stringValue(registry["domain"]),
		"protocol":   stringValue(registry["protocol"]),
		"proxyName":  stringValue(registry["proxyName"]),
		"properties": map[string]any{},
	}
	if port := toInt(registry["port"]); port > 0 {
		item["port"] = port
	}
	if sni := stringValue(registry["sni"]); sni != "" {
		item["sni"] = sni
	}
	if vport := toInt(registry["vport"]); vport > 0 {
		item["vport"] = vport
	}
	properties := map[string]any{}
	for key, value := range registry {
		switch key {
		case "name", "type", "domain", "port", "protocol", "proxyName", "sni", "vport":
			continue
		default:
			properties[key] = value
		}
	}
	item["properties"] = properties
	return item
}

func serviceSourceToRegistry(name string, data map[string]any) map[string]any {
	registry := map[string]any{
		"name":      strings.TrimSpace(name),
		"type":      stringValue(data["type"]),
		"domain":    stringValue(data["domain"]),
		"protocol":  stringValue(data["protocol"]),
		"proxyName": stringValue(data["proxyName"]),
	}
	if port := toInt(data["port"]); port > 0 {
		registry["port"] = port
	}
	if sni := stringValue(data["sni"]); sni != "" {
		registry["sni"] = sni
	}
	if vport := toInt(data["vport"]); vport > 0 {
		registry["vport"] = vport
	}
	for key, value := range mapValue(data["properties"]) {
		if key == "" {
			continue
		}
		registry[key] = value
	}
	return registry
}

func providerPayloadFromResource(name string, data map[string]any) map[string]any {
	providerType := canonicalProviderType(firstNonEmpty(stringValue(data["type"]), stringValue(mapValue(data["rawConfigs"])["type"])))
	payload := normalizeProviderRawConfigsForConsole(cloneMap(mapValue(data["rawConfigs"])), providerType)
	if payload == nil {
		payload = map[string]any{}
	}
	payload["id"] = name
	if providerType != "" {
		payload["type"] = providerType
	}
	if value := providerProtocolPluginValue(data["protocol"]); value != "" {
		payload["protocol"] = value
	}
	if tokens := normalizeStringSlice(data["tokens"]); len(tokens) > 0 {
		payload["apiTokens"] = tokens
	} else {
		delete(payload, "apiTokens")
	}
	if failover := mapValue(data["tokenFailoverConfig"]); len(failover) > 0 {
		payload["failover"] = failover
	}
	return payload
}

func routeLabels(data map[string]any) map[string]string {
	labels := map[string]string{
		higressLabelResourceDefiner: higressLabelResourceDefinerValue,
	}
	for _, domain := range normalizeStringSlice(data["domains"]) {
		labels[higressLabelDomainPrefix+normalizeDomainName(domain)] = higressAnnotationTrueValue
	}
	if strings.HasSuffix(stringValue(data["name"]), consts.InternalResourceNameSuffix) {
		labels[higressLabelInternal] = higressAnnotationTrueValue
	}
	if metadata := mapValue(firstNonNil(data["routeMetadata"], data["mcpRouteMetadata"])); stringValue(metadata["mcpServerName"]) != "" {
		labels[higressLabelBizType] = higressLabelBizTypeMCPServer
	}
	if serverType := strings.ToUpper(stringValue(data["type"])); serverType != "" {
		labels[higressLabelMCPServerType] = serverType
	}
	return labels
}

func routeAnnotations(data map[string]any) map[string]string {
	annotations := map[string]string{}
	if path := mapValue(data["path"]); strings.EqualFold(stringValue(path["matchType"]), "REGULAR") {
		annotations[higressAnnotationUseRegex] = higressAnnotationTrueValue
	}
	if path := mapValue(data["path"]); path != nil {
		caseSensitive, ok := path["caseSensitive"].(bool)
		if ok {
			annotations[higressAnnotationIgnorePathCase] = strconv.FormatBool(!caseSensitive)
		}
	}
	if services := normalizeDestinationString(data["services"]); services != "" {
		annotations[higressAnnotationDestination] = services
	}
	if methods := normalizeStringSlice(data["methods"]); len(methods) > 0 {
		annotations[higressAnnotationMatchMethod] = strings.Join(methods, " ")
	}
	if rewrite := mapValue(data["rewrite"]); len(rewrite) > 0 {
		enabled := true
		if value, ok := rewrite["enabled"].(bool); ok {
			enabled = value
		}
		annotations[higressAnnotationRewriteEnabled] = strconv.FormatBool(enabled)
		if path := stringValue(rewrite["path"]); path != "" {
			matchType := strings.ToUpper(stringValue(mapValue(data["path"])["matchType"]))
			if matchType == "REGULAR" {
				annotations[higressAnnotationRewriteTarget] = path
			} else {
				annotations[higressAnnotationRewritePath] = path
			}
		}
		if host := stringValue(rewrite["host"]); host != "" {
			annotations[higressAnnotationUpstreamVHost] = host
		}
	}
	if authConfig := mapValue(data["authConfig"]); len(authConfig) > 0 {
		if departments := normalizeStringSlice(authConfig["allowedDepartments"]); len(departments) > 0 {
			annotations[higressAnnotationAuthConsumerDepartments] = strings.Join(departments, ",")
		}
		if levels := normalizeStringSlice(authConfig["allowedConsumerLevels"]); len(levels) > 0 {
			annotations[higressAnnotationAuthConsumerLevels] = strings.Join(levels, ",")
		}
	}
	if description := stringValue(data["description"]); description != "" {
		annotations[higressAnnotationResourceDescription] = description
	}
	for _, predicate := range toMapSlice(data["headers"]) {
		setPredicateAnnotation(annotations, predicate, higressAnnotationHeaderMatch, higressAnnotationPseudoHeaderMatch)
	}
	for _, predicate := range toMapSlice(data["urlParams"]) {
		setPredicateAnnotation(annotations, predicate, higressAnnotationQueryMatch)
	}
	if proxyNext := mapValue(data["proxyNextUpstream"]); len(proxyNext) > 0 {
		enabled := true
		if raw, ok := proxyNext["enabled"].(bool); ok {
			enabled = raw
		}
		annotations[higressAnnotationProxyNextEnabled] = strconv.FormatBool(enabled)
		if conditions := normalizeStringSlice(proxyNext["conditions"]); len(conditions) > 0 {
			annotations[higressAnnotationProxyNextUpstream] = strings.Join(conditions, ",")
		}
		if attempts := toInt(proxyNext["attempts"]); attempts > 0 {
			annotations[higressAnnotationProxyNextTries] = strconv.Itoa(attempts)
		}
		if timeout := toInt(proxyNext["timeout"]); timeout > 0 {
			annotations[higressAnnotationProxyNextTimeout] = strconv.Itoa(timeout)
		}
	}
	if metadata := mapValue(firstNonNil(data["routeMetadata"], data["mcpRouteMetadata"])); stringValue(metadata["mcpServerName"]) != "" {
		annotations[higressAnnotationMCPServer] = higressAnnotationTrueValue
		matchDomains := normalizeStringSlice(data["domains"])
		if len(matchDomains) == 0 {
			matchDomains = []string{"*"}
		}
		annotations[higressAnnotationMCPMatchRuleDomains] = strings.Join(matchDomains, ",")
		path := mapValue(firstNonNil(data["path"], data["pathPredicate"]))
		if matchType := predicateAnnotationPrefix(path["matchType"]); matchType != "" {
			annotations[higressAnnotationMCPMatchRuleType] = matchType
		}
		if matchValue := stringValue(path["matchValue"]); matchValue != "" {
			annotations[higressAnnotationMCPMatchRuleValue] = matchValue
		}
		if upstreamType := inferMCPUpstreamType(data); upstreamType != "" {
			annotations[higressAnnotationMCPUpstreamType] = upstreamType
		}
		directRouteConfig := mapValue(data["directRouteConfig"])
		if transport := firstNonEmpty(stringValue(data["upstreamTransportType"]), stringValue(directRouteConfig["transportType"]), stringValue(data["transport"]), stringValue(data["protocol"])); transport != "" {
			annotations[higressAnnotationMCPUpstreamTransport] = strings.ToLower(transport)
		}
		prefix := firstNonEmpty(stringValue(data["pathRewritePrefix"]), stringValue(data["upstreamPathPrefix"]))
		if prefix == "" && len(mapValue(data["rewrite"])) > 0 && (strings.EqualFold(stringValue(data["type"]), "DIRECT_ROUTE") || strings.EqualFold(stringValue(data["type"]), "REDIRECT_ROUTE")) {
			prefix = "/"
		}
		if prefix != "" {
			annotations[higressAnnotationMCPPathRewriteEnabled] = higressAnnotationTrueValue
			annotations[higressAnnotationMCPPathRewritePrefix] = prefix
		}
	}
	return annotations
}

func routeRules(data map[string]any) []map[string]any {
	domains := normalizeStringSlice(data["domains"])
	if len(domains) == 0 {
		domains = []string{""}
	}
	path := mapValue(data["path"])
	matchValue := firstNonEmpty(stringValue(path["matchValue"]), "/")
	matchType := strings.ToUpper(firstNonEmpty(stringValue(path["matchType"]), "PRE"))
	pathType := "Prefix"
	if matchType == "EQUAL" {
		pathType = "Exact"
	}
	rules := make([]map[string]any, 0, len(domains))
	for _, domain := range domains {
		rule := map[string]any{
			"http": map[string]any{
				"paths": []map[string]any{{
					"path":     matchValue,
					"pathType": pathType,
					"backend": map[string]any{
						"resource": map[string]any{
							"apiGroup": higressMcpBridgeAPIGroup,
							"kind":     higressMcpBridgeKind,
							"name":     higressMcpBridgeDefaultName,
						},
					},
				}},
			},
		}
		if strings.TrimSpace(domain) != "" {
			rule["host"] = strings.TrimSpace(domain)
		}
		rules = append(rules, rule)
	}
	return rules
}

func (c *RealClient) routeTLS(data map[string]any) []map[string]any {
	domains := normalizeStringSlice(data["domains"])
	if len(domains) == 0 {
		return nil
	}
	tls := make([]map[string]any, 0, len(domains))
	for _, domain := range domains {
		record, err := c.getDomainResource(context.Background(), domain)
		if err != nil {
			continue
		}
		enableHTTPS := strings.ToLower(strings.TrimSpace(fmt.Sprint(record["enableHttps"])))
		certIdentifier := strings.TrimSpace(fmt.Sprint(record["certIdentifier"]))
		if certIdentifier == "" || enableHTTPS == "" || enableHTTPS == "off" {
			continue
		}
		entry := map[string]any{
			"secretName": certIdentifier,
		}
		if domain != "" {
			entry["hosts"] = []string{domain}
		}
		tls = append(tls, entry)
	}
	return tls
}

func ingressDomains(spec map[string]any, labels map[string]string) []string {
	domains := make([]string, 0)
	seen := map[string]struct{}{}
	for _, rawRule := range sliceValue(spec["rules"]) {
		rule := mapValue(rawRule)
		host := stringValue(rule["host"])
		if host != "" {
			if _, ok := seen[host]; !ok {
				domains = append(domains, host)
				seen[host] = struct{}{}
			}
		}
	}
	for _, rawTLS := range sliceValue(spec["tls"]) {
		tls := mapValue(rawTLS)
		for _, rawHost := range sliceValue(tls["hosts"]) {
			host := strings.TrimSpace(fmt.Sprint(rawHost))
			if host == "" {
				continue
			}
			if _, ok := seen[host]; !ok {
				domains = append(domains, host)
				seen[host] = struct{}{}
			}
		}
	}
	for key, value := range labels {
		if value != higressAnnotationTrueValue || !strings.HasPrefix(key, higressLabelDomainPrefix) {
			continue
		}
		domain := strings.TrimPrefix(key, higressLabelDomainPrefix)
		if domain == "" {
			continue
		}
		if _, ok := seen[domain]; !ok {
			domains = append(domains, domain)
			seen[domain] = struct{}{}
		}
	}
	sort.Strings(domains)
	return domains
}

func ingressPath(spec map[string]any, annotations map[string]string) map[string]any {
	for _, rawRule := range sliceValue(spec["rules"]) {
		rule := mapValue(rawRule)
		httpRule := mapValue(rule["http"])
		for _, rawPath := range sliceValue(httpRule["paths"]) {
			path := mapValue(rawPath)
			matchType := "PRE"
			switch strings.TrimSpace(fmt.Sprint(path["pathType"])) {
			case "Exact":
				matchType = "EQUAL"
			case "Prefix":
				if strings.EqualFold(annotations[higressAnnotationUseRegex], higressAnnotationTrueValue) {
					matchType = "REGULAR"
				}
			}
			result := map[string]any{
				"matchType":  matchType,
				"matchValue": firstNonEmpty(stringValue(path["path"]), "/"),
			}
			if raw := strings.TrimSpace(annotations[higressAnnotationIgnorePathCase]); raw != "" {
				result["caseSensitive"] = !strings.EqualFold(raw, higressAnnotationTrueValue)
			}
			return result
		}
	}
	return nil
}

func parseDestinationServices(raw string) []map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	lines := strings.Split(raw, "\n")
	result := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue
		}
		weight := 100
		addressIndex := 0
		if strings.HasSuffix(fields[0], "%") {
			parsed, err := strconv.Atoi(strings.TrimSuffix(fields[0], "%"))
			if err == nil && parsed > 0 {
				weight = parsed
				addressIndex = 1
			}
		}
		if len(fields) <= addressIndex {
			continue
		}
		host := fields[addressIndex]
		port := 0
		if colon := strings.LastIndex(host, ":"); colon > 0 {
			if parsed, err := strconv.Atoi(host[colon+1:]); err == nil && parsed > 0 && parsed < 65536 {
				port = parsed
				host = host[:colon]
			}
		}
		service := map[string]any{"name": host, "weight": weight}
		if port > 0 {
			service["port"] = port
		}
		if len(fields) > addressIndex+1 {
			service["version"] = fields[addressIndex+1]
		}
		result = append(result, service)
	}
	return result
}

func normalizeDestinationString(value any) string {
	services := toMapSlice(value)
	if len(services) == 0 {
		return ""
	}
	if len(services) == 1 {
		service := services[0]
		address := stringValue(service["name"])
		if port := toInt(service["port"]); port > 0 {
			address = fmt.Sprintf("%s:%d", address, port)
		}
		return address
	}
	lines := make([]string, 0, len(services))
	for _, service := range services {
		weight := toInt(service["weight"])
		if weight <= 0 {
			weight = 100
		}
		address := stringValue(service["name"])
		if port := toInt(service["port"]); port > 0 {
			address = fmt.Sprintf("%s:%d", address, port)
		}
		line := fmt.Sprintf("%d%% %s", weight, address)
		if version := stringValue(service["version"]); version != "" {
			line = line + " " + version
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ingressRewrite(annotations map[string]string) map[string]any {
	path := firstNonEmpty(annotations[higressAnnotationRewritePath], annotations[higressAnnotationRewriteTarget])
	host := annotations[higressAnnotationUpstreamVHost]
	enabledRaw := annotations[higressAnnotationRewriteEnabled]
	if path == "" && host == "" && enabledRaw == "" {
		return nil
	}
	enabled := true
	if strings.TrimSpace(enabledRaw) != "" {
		enabled = strings.EqualFold(enabledRaw, higressAnnotationTrueValue)
	}
	result := map[string]any{"enabled": enabled}
	if path != "" {
		result["path"] = path
	}
	if host != "" {
		result["host"] = host
	}
	return result
}

func ingressAuthConfig(annotations map[string]string) map[string]any {
	departments := splitAndTrim(annotations[higressAnnotationAuthConsumerDepartments], ",")
	levels := splitAndTrim(annotations[higressAnnotationAuthConsumerLevels], ",")
	if len(departments) == 0 && len(levels) == 0 {
		return nil
	}
	result := map[string]any{
		"enabled": true,
	}
	if len(departments) > 0 {
		result["allowedDepartments"] = departments
	}
	if len(levels) > 0 {
		result["allowedConsumerLevels"] = levels
	}
	return result
}

func ingressProxyNextUpstreamConfig(annotations map[string]string) map[string]any {
	rawEnabled := strings.TrimSpace(annotations[higressAnnotationProxyNextEnabled])
	tries := toInt(annotations[higressAnnotationProxyNextTries])
	timeout := toInt(annotations[higressAnnotationProxyNextTimeout])
	conditions := splitAndTrim(annotations[higressAnnotationProxyNextUpstream], ",")
	if rawEnabled == "" && tries == 0 && timeout == 0 && len(conditions) == 0 {
		return nil
	}
	enabled := true
	if rawEnabled != "" {
		enabled = strings.EqualFold(rawEnabled, higressAnnotationTrueValue)
	}
	result := map[string]any{
		"enabled": enabled,
	}
	if tries > 0 {
		result["attempts"] = tries
	}
	if timeout > 0 {
		result["timeout"] = timeout
	}
	if len(conditions) > 0 {
		result["conditions"] = conditions
	}
	return result
}

func ingressKeyedPredicates(annotations map[string]string, suffixes ...string) []map[string]any {
	result := make([]map[string]any, 0)
	for key, value := range annotations {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		predicate, ok := parsePredicateAnnotation(key, value, suffixes...)
		if ok {
			result = append(result, predicate)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return stringValue(result[i]["key"]) < stringValue(result[j]["key"])
	})
	return result
}

func parsePredicateAnnotation(annotationKey, value string, suffixes ...string) (map[string]any, bool) {
	trimmedKey := strings.TrimSpace(annotationKey)
	enabled := true
	if strings.HasPrefix(trimmedKey, "disabled.") {
		enabled = false
		trimmedKey = strings.TrimPrefix(trimmedKey, "disabled.")
	}
	if !strings.HasPrefix(trimmedKey, "higress.io/") {
		return nil, false
	}
	trimmedKey = strings.TrimPrefix(trimmedKey, "higress.io/")
	for _, suffix := range suffixes {
		idx := strings.Index(trimmedKey, suffix)
		if idx <= 0 {
			continue
		}
		prefix := trimmedKey[:idx]
		matchType := predicateTypeFromAnnotationPrefix(prefix)
		if matchType == "" {
			return nil, false
		}
		key := trimmedKey[idx+len(suffix):]
		if key == "" {
			return nil, false
		}
		if suffix == higressAnnotationPseudoHeaderMatch {
			key = ":" + key
		}
		return map[string]any{
			"key":        key,
			"matchType":  matchType,
			"matchValue": value,
			"enabled":    enabled,
		}, true
	}
	return nil, false
}

func setPredicateAnnotation(annotations map[string]string, predicate map[string]any, suffixes ...string) {
	key := stringValue(predicate["key"])
	matchValue := stringValue(predicate["matchValue"])
	matchType := predicateAnnotationPrefix(predicate["matchType"])
	if key == "" || matchValue == "" || matchType == "" {
		return
	}
	suffix := higressAnnotationHeaderMatch
	actualKey := key
	if len(suffixes) > 0 {
		suffix = suffixes[0]
	}
	if strings.HasPrefix(key, ":") {
		suffix = higressAnnotationPseudoHeaderMatch
		actualKey = strings.TrimPrefix(key, ":")
	}
	if len(suffixes) > 1 && strings.HasPrefix(key, ":") {
		suffix = suffixes[1]
	}
	annotationKey := fmt.Sprintf("higress.io/%s%s%s", matchType, suffix, actualKey)
	if enabled, ok := predicate["enabled"].(bool); ok && !enabled {
		annotationKey = "disabled." + annotationKey
	}
	annotations[annotationKey] = matchValue
}

func predicateTypeFromAnnotationPrefix(prefix string) string {
	switch strings.ToLower(strings.TrimSpace(prefix)) {
	case "exact":
		return "EQUAL"
	case "prefix":
		return "PRE"
	case "regex":
		return "REGULAR"
	default:
		return ""
	}
}

func predicateAnnotationPrefix(value any) string {
	switch strings.ToUpper(strings.TrimSpace(fmt.Sprint(value))) {
	case "EQUAL":
		return "exact"
	case "PRE":
		return "prefix"
	case "REGULAR":
		return "regex"
	default:
		return ""
	}
}

func inferMCPUpstreamType(data map[string]any) string {
	if value := strings.TrimSpace(stringValue(data["upstreamType"])); value != "" {
		return strings.ToLower(value)
	}
	services := toMapSlice(data["services"])
	if len(services) == 0 {
		return ""
	}
	name := stringValue(services[0]["name"])
	switch {
	case strings.HasSuffix(name, "."+higressDNSRegistryType):
		return higressDNSRegistryType
	case strings.HasSuffix(name, "."+higressStaticRegistryType):
		return higressStaticRegistryType
	default:
		return ""
	}
}

func providerProtocolValue(value any) string {
	switch strings.TrimSpace(fmt.Sprint(value)) {
	case "", higressProviderPluginProtocolOpenAI:
		return higressProviderDefaultProtocol
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func providerProtocolPluginValue(value any) string {
	switch strings.TrimSpace(fmt.Sprint(value)) {
	case "", higressProviderDefaultProtocol:
		return higressProviderPluginProtocolOpenAI
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}

func providerModelCatalog(name, providerType string) []map[string]any {
	key := normalizeProviderCatalogKey(firstNonEmpty(name, providerType))
	if key == "" {
		return nil
	}
	for catalogKey, models := range legacyProviderModelCatalog {
		if key == catalogKey || strings.Contains(key, catalogKey) || strings.Contains(catalogKey, key) {
			result := make([]map[string]any, 0, len(models))
			for _, model := range models {
				result = append(result, map[string]any{
					"modelId":     model,
					"targetModel": model,
					"label":       model,
				})
			}
			return result
		}
	}
	return nil
}

var legacyProviderModelCatalog = map[string][]string{
	"openai":     {"gpt-3", "gpt-35-turbo", "gpt-4", "gpt-4o", "gpt-4o-mini"},
	"qwen":       {"qwen-max", "qwen-plus", "qwen-turbo", "qwen-long"},
	"moonshot":   {"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
	"azure":      {"gpt-3", "gpt-35-turbo", "gpt-4", "gpt-4o", "gpt-4o-mini"},
	"claude":     {"claude-3-haiku", "claude-3-sonnet", "claude-3-opus", "claude-3-5-sonnet"},
	"baichuan":   {"Baichuan2-Turbo", "Baichuan2-Turbo-192K", "Baichuan3-Turbo", "Baichuan4"},
	"yi":         {"yi-lightning", "yi-medium", "yi-large"},
	"zhipuai":    {"glm-4", "glm-4-air", "glm-4-plus", "glm-4v"},
	"baidu":      {"ERNIE-Speed-128K", "ERNIE-Lite-8K", "ERNIE-4.0-8K"},
	"stepfun":    {"step-1-8k", "step-1-32k", "step-1-128k"},
	"minimax":    {"abab6.5s-chat", "abab6.5g-chat"},
	"gemini":     {"gemini-1.5-flash", "gemini-1.5-pro"},
	"volcengine": {"doubao-pro-32k", "doubao-lite-32k", "doubao-seed-2-0-pro-260215"},
	"openrouter": {"openrouter/auto"},
	"grok":       {"grok-2-1212", "grok-2-vision-1212"},
}

func normalizeProviderCatalogKey(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "doubao" {
		return "volcengine"
	}
	return value
}

func canonicalProviderType(value string) string {
	return normalizeProviderCatalogKey(value)
}

func normalizeProviderRawConfigsForConsole(rawConfigs map[string]any, providerType string) map[string]any {
	if rawConfigs == nil {
		return map[string]any{}
	}
	rawConfigs["type"] = providerType
	if providerType == "volcengine" {
		if stringValue(rawConfigs["providerDomain"]) == "" {
			if legacyDomain := stringValue(rawConfigs["doubaoDomain"]); legacyDomain != "" {
				rawConfigs["providerDomain"] = legacyDomain
			}
		}
		delete(rawConfigs, "doubaoDomain")
	}
	return rawConfigs
}

func sanitizeObjectForApply(item map[string]any) {
	delete(item, "status")
	metadata := mapValue(item["metadata"])
	delete(metadata, "creationTimestamp")
	delete(metadata, "generation")
	delete(metadata, "managedFields")
	delete(metadata, "uid")
	delete(metadata, "selfLink")
}

func domainConfigMapName(name string) string {
	return higressDomainPrefix + normalizeDomainName(name)
}

func aiRouteConfigMapName(name string) string {
	return higressAIRoutePrefix + strings.TrimSpace(name)
}

func normalizeDomainName(name string) string {
	trimmed := strings.TrimSpace(name)
	if strings.HasPrefix(trimmed, "*") {
		return "wildcard" + strings.TrimPrefix(trimmed, "*")
	}
	return trimmed
}

func buildLabelSelector(key, value string) string {
	return key + "=" + value
}

func mapValue(value any) map[string]any {
	item, _ := value.(map[string]any)
	if item == nil {
		return map[string]any{}
	}
	return item
}

func stringMap(value any) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	switch typed := value.(type) {
	case map[string]string:
		return cloneStringMap(typed)
	case map[string]any:
		result := make(map[string]string, len(typed))
		for key, raw := range typed {
			result[key] = strings.TrimSpace(fmt.Sprint(raw))
		}
		return result
	default:
		return map[string]string{}
	}
}

func sliceValue(value any) []any {
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case []any:
		return typed
	case []map[string]any:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result
	case []string:
		result := make([]any, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result
	default:
		return nil
	}
}

func toMapSlice(value any) []map[string]any {
	items := sliceValue(value)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mapped, ok := item.(map[string]any); ok {
			result = append(result, mapped)
		}
	}
	return result
}

func splitAndTrim(raw, sep string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, sep)
	result := make([]string, 0, len(parts))
	for _, item := range parts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func sortResourcesByName(items []map[string]any) {
	sort.Slice(items, func(i, j int) bool {
		return stringValue(items[i]["name"]) < stringValue(items[j]["name"])
	})
}

func nestedValue(item map[string]any, keys ...string) any {
	current := any(item)
	for _, key := range keys {
		mapped, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = mapped[key]
	}
	return current
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return map[string]any{}
	}
	if current, ok := parent[key].(map[string]any); ok {
		return current
	}
	current := map[string]any{}
	parent[key] = current
	return current
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func normalizeStringSlice(value any) []string {
	items := sliceValue(value)
	result := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(fmt.Sprint(item))
		if text == "" {
			continue
		}
		result = append(result, text)
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
	case string:
		number, _ := strconv.Atoi(strings.TrimSpace(typed))
		return number
	default:
		number, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(value)))
		return number
	}
}

func decodeBase64Value(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return ""
	}
	return string(decoded)
}
