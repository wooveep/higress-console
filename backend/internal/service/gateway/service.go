package gateway

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

type Service struct {
	k8sClient k8sclient.Client
	portal    portalReader
	hook      Hook
}

type builtinPluginRuleLoader interface {
	LoadBuiltinPluginRules(ctx context.Context, pluginName string) (map[string]map[string]any, error)
}

type portalReader interface {
	ListAccounts(ctx context.Context) ([]portalsvc.OrgAccountRecord, error)
	ListDepartmentTree(ctx context.Context) ([]*portalsvc.OrgDepartmentNode, error)
}

type Hook interface {
	AfterWrite(ctx context.Context, trigger string) error
}

type noopHook struct{}

func New(k8sClient k8sclient.Client, portals ...portalReader) *Service {
	var portal portalReader
	if len(portals) > 0 {
		portal = portals[0]
	}
	svc := &Service{k8sClient: k8sClient, portal: portal, hook: noopHook{}}
	svc.bootstrapDefaults(context.Background())
	return svc
}

func (s *Service) List(ctx context.Context, kind string) ([]map[string]any, error) {
	items, err := s.k8sClient.ListResources(ctx, kind)
	if err != nil {
		return nil, err
	}
	if kind == "wasm-plugins" {
		return s.mergeWasmPlugins(items), nil
	}
	return s.hydrateResources(kind, items), nil
}

func (s *Service) Get(ctx context.Context, kind, name string) (map[string]any, error) {
	item, err := s.k8sClient.GetResource(ctx, kind, name)
	if err == nil {
		return s.hydrateResource(kind, item), nil
	}
	if kind == "wasm-plugins" {
		if fallback, ok := s.loadBuiltinWasmPlugin(name); ok {
			return fallback, nil
		}
	}
	return nil, err
}

func (s *Service) Save(ctx context.Context, kind string, payload map[string]any) (map[string]any, error) {
	payload = clonePayload(payload)
	name := strings.TrimSpace(fmt.Sprint(payload["name"]))
	if name == "" {
		return nil, errors.New("name is required")
	}
	if s.isProtected(kind, name) {
		return nil, fmt.Errorf("%s %s is protected", kind, name)
	}
	if s.isInternalWriteBlocked(kind, name) {
		return nil, fmt.Errorf("%s %s is an internal resource", kind, name)
	}
	normalized, err := s.normalizeForSave(ctx, kind, payload)
	if err != nil {
		return nil, err
	}
	item, err := s.k8sClient.UpsertResource(ctx, kind, name, normalized)
	if err != nil {
		return nil, err
	}
	if err := s.afterWrite(ctx, kind, "save"); err != nil {
		return nil, err
	}
	return s.hydrateResource(kind, item), nil
}

func (s *Service) Delete(ctx context.Context, kind, name string) error {
	if s.isProtected(kind, name) {
		return fmt.Errorf("%s %s is protected", kind, name)
	}
	if s.isInternalWriteBlocked(kind, name) {
		return fmt.Errorf("%s %s is an internal resource", kind, name)
	}
	if err := s.k8sClient.DeleteResource(ctx, kind, name); err != nil {
		return err
	}
	return s.afterWrite(ctx, kind, "delete")
}

func (s *Service) ListMcpConsumers(ctx context.Context, serverName string) ([]map[string]any, error) {
	item, err := s.Get(ctx, "mcp-servers", serverName)
	if err != nil {
		return nil, err
	}
	info, _ := item["consumerAuthInfo"].(map[string]any)
	allowed, _ := info["allowedConsumers"].([]any)

	result := make([]map[string]any, 0, len(allowed))
	for _, consumer := range allowed {
		result = append(result, map[string]any{
			"mcpServerName": serverName,
			"consumerName":  fmt.Sprint(consumer),
			"type":          fmt.Sprint(info["type"]),
		})
	}
	return result, nil
}

func (s *Service) SwaggerToMCPConfig(content string) string {
	return strings.TrimSpace(fmt.Sprintf("type: OPEN_API\nrawConfigurations: |\n  %s\n", strings.ReplaceAll(content, "\n", "\n  ")))
}

func (s *Service) GetPluginInstance(ctx context.Context, scope, target, pluginName string, aliases ...string) (map[string]any, error) {
	item, err := s.k8sClient.GetResource(ctx, pluginInstanceKind(scope, target), pluginName)
	if err == nil {
		return item, nil
	}
	if scope == "route" {
		return s.getBuiltinRuntimePluginInstance(ctx, pluginName, append([]string{target}, aliases...))
	}
	if scope == "global" {
		return s.getBuiltinGlobalPluginInstance(ctx, pluginName)
	}
	return nil, err
}

func (s *Service) SavePluginInstance(ctx context.Context, scope, target, pluginName string, payload map[string]any) (map[string]any, error) {
	if err := s.validatePluginInstance(ctx, scope, target, pluginName, payload); err != nil {
		return nil, err
	}
	payload["name"] = pluginName
	return s.k8sClient.UpsertResource(ctx, pluginInstanceKind(scope, target), pluginName, payload)
}

func (s *Service) DeletePluginInstance(ctx context.Context, scope, target, pluginName string) error {
	return s.k8sClient.DeleteResource(ctx, pluginInstanceKind(scope, target), pluginName)
}

func (s *Service) ListPluginInstances(ctx context.Context, scope, target string, aliases ...string) ([]map[string]any, error) {
	items, err := s.k8sClient.ListResources(ctx, pluginInstanceKind(scope, target))
	if err != nil {
		return nil, err
	}
	if scope != "route" {
		return items, nil
	}
	return s.mergeBuiltinRuntimePluginInstances(ctx, items, append([]string{target}, aliases...))
}

func (s *Service) GetWasmPluginConfig(ctx context.Context, name string) (map[string]any, error) {
	item, err := s.Get(ctx, "wasm-plugins", name)
	if err != nil {
		return nil, err
	}
	config := map[string]any{
		"name": name,
		"type": "yaml",
	}
	if schema, ok := item["configSchema"]; ok {
		config["schema"] = schema
	} else if schema, ok := item["schema"]; ok {
		config["schema"] = schema
	} else {
		config["schema"] = map[string]any{}
	}
	return config, nil
}

func (s *Service) GetWasmPluginReadme(ctx context.Context, name string) (string, error) {
	item, err := s.Get(ctx, "wasm-plugins", name)
	if err != nil {
		return "", err
	}
	readme := stringValue(item["readme"])
	if readme == "" {
		readme = stringValue(item["documentation"])
	}
	if readme == "" {
		description := stringValue(item["description"])
		if description != "" {
			readme = "# " + name + "\n\n" + description
		}
	}
	if readme == "" {
		readme = "# " + name + "\n\nNo readme metadata is available yet for this plugin."
	}
	return readme, nil
}

func (s *Service) bootstrapDefaults(ctx context.Context) {
	_, _ = s.k8sClient.UpsertResource(ctx, "services", "aigateway-console.dns", map[string]any{
		"name":         "aigateway-console.dns",
		"namespace":    "aigateway-system",
		"port":         8080,
		"endpoints":    []string{"aigateway-console.aigateway-system.svc.cluster.local:8080"},
		"ingressClass": s.k8sClient.IngressClass(),
	})
	_, _ = s.k8sClient.UpsertResource(ctx, "proxy-servers", "default-http-proxy", map[string]any{
		"name":           "default-http-proxy",
		"type":           "HTTP",
		"serverAddress":  "127.0.0.1",
		"serverPort":     3128,
		"connectTimeout": 5000,
	})
}

func (s *Service) isProtected(kind, name string) bool {
	switch kind {
	case "domains":
		return name == consts.DefaultDomainName
	case "routes":
		return name == consts.DefaultRouteName
	case "tls-certificates":
		return name == consts.DefaultTLSCertificateName
	default:
		return false
	}
}

func (s *Service) isInternalWriteBlocked(kind, name string) bool {
	switch kind {
	case "routes", "service-sources", "proxy-servers":
		return strings.HasSuffix(name, consts.InternalResourceNameSuffix)
	default:
		return false
	}
}

func pluginInstanceKind(scope, target string) string {
	scope = strings.TrimSpace(scope)
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Sprintf("%s-plugin-instances", scope)
	}
	return fmt.Sprintf("%s-plugin-instances:%s", scope, target)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func (noopHook) AfterWrite(ctx context.Context, trigger string) error { return nil }

func (s *Service) SetHook(hook Hook) {
	if hook == nil {
		s.hook = noopHook{}
		return
	}
	s.hook = hook
}

func (s *Service) afterWrite(ctx context.Context, kind, action string) error {
	if strings.TrimSpace(kind) != "ai-routes" {
		return nil
	}
	return s.hook.AfterWrite(ctx, "ai-route-"+strings.TrimSpace(action))
}

func clonePayload(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	return clonePayloadMap(src)
}

func (s *Service) mergeBuiltinRuntimePluginInstances(
	ctx context.Context,
	items []map[string]any,
	targets []string,
) ([]map[string]any, error) {
	loader, ok := s.k8sClient.(builtinPluginRuleLoader)
	if !ok {
		return items, nil
	}

	result := make([]map[string]any, 0, len(items))
	seen := make(map[string]int, len(items))
	for _, item := range items {
		cloned := clonePayload(item)
		pluginName := strings.TrimSpace(fmt.Sprint(cloned["pluginName"]))
		if pluginName == "" {
			pluginName = strings.TrimSpace(fmt.Sprint(cloned["name"]))
		}
		if pluginName != "" {
			seen[pluginName] = len(result)
		}
		result = append(result, cloned)
	}

	for _, builtin := range s.listBuiltinWasmPlugins() {
		pluginName := strings.TrimSpace(fmt.Sprint(builtin["name"]))
		if pluginName == "" {
			continue
		}
		rules, err := loader.LoadBuiltinPluginRules(ctx, pluginName)
		if err != nil {
			return nil, err
		}
		rule, matchedTarget := findBuiltinPluginRuleForTargets(rules, targets)
		if rule == nil {
			continue
		}
		runtimeItem := builtinRuntimePluginInstance(pluginName, matchedTarget, rule)
		if index, exists := seen[pluginName]; exists {
			result[index] = mergePluginInstanceRuntime(result[index], runtimeItem)
			continue
		}
		seen[pluginName] = len(result)
		result = append(result, runtimeItem)
	}

	return result, nil
}

func (s *Service) getBuiltinRuntimePluginInstance(
	ctx context.Context,
	pluginName string,
	targets []string,
) (map[string]any, error) {
	loader, ok := s.k8sClient.(builtinPluginRuleLoader)
	if !ok {
		return nil, k8sclient.ErrNotFound
	}
	rules, err := loader.LoadBuiltinPluginRules(ctx, pluginName)
	if err != nil {
		return nil, err
	}
	rule, matchedTarget := findBuiltinPluginRuleForTargets(rules, targets)
	if rule == nil {
		return nil, k8sclient.ErrNotFound
	}
	return builtinRuntimePluginInstance(pluginName, matchedTarget, rule), nil
}

func (s *Service) getBuiltinGlobalPluginInstance(ctx context.Context, pluginName string) (map[string]any, error) {
	loader, ok := s.k8sClient.(builtinPluginRuleLoader)
	if !ok {
		return nil, k8sclient.ErrNotFound
	}
	rules, err := loader.LoadBuiltinPluginRules(ctx, pluginName)
	if err != nil {
		return nil, err
	}
	targets := make([]string, 0, len(rules))
	config := map[string]any{}
	for target, rule := range rules {
		if boolValue(rule["configDisable"]) {
			continue
		}
		targets = append(targets, target)
		if len(config) == 0 {
			config = mapValue(rule["config"])
		}
	}
	if len(targets) == 0 {
		return nil, k8sclient.ErrNotFound
	}
	sort.Strings(targets)
	item := map[string]any{
		"name":           pluginName,
		"pluginName":     pluginName,
		"enabled":        true,
		"runtimeEnabled": true,
		"runtimeSource":  "builtin-rule-global",
		"runtimeTargets": targets,
		"config":         config,
	}
	if len(config) > 0 {
		if payload, err := yaml.Marshal(config); err == nil {
			item["rawConfigurations"] = string(payload)
		}
	}
	return item, nil
}

func findBuiltinPluginRuleForTargets(
	rules map[string]map[string]any,
	targets []string,
) (map[string]any, string) {
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "" {
			continue
		}
		if rule, ok := rules[trimmed]; ok {
			return rule, trimmed
		}
	}
	return nil, ""
}

func builtinRuntimePluginInstance(pluginName, target string, rule map[string]any) map[string]any {
	enabled := !boolValue(rule["configDisable"])
	config := mapValue(rule["config"])
	item := map[string]any{
		"name":           pluginName,
		"pluginName":     pluginName,
		"enabled":        enabled,
		"runtimeEnabled": enabled,
		"runtimeSource":  "builtin-rule",
		"runtimeTarget":  target,
		"config":         config,
	}
	if len(config) > 0 {
		if payload, err := yaml.Marshal(config); err == nil {
			item["rawConfigurations"] = string(payload)
		}
	}
	return item
}

func mergePluginInstanceRuntime(base, runtime map[string]any) map[string]any {
	merged := clonePayload(base)
	for key, value := range runtime {
		switch key {
		case "name", "pluginName":
			if stringValue(merged[key]) == "" {
				merged[key] = value
			}
		case "enabled":
			if _, exists := merged[key]; !exists || merged[key] == nil {
				merged[key] = value
			}
		default:
			merged[key] = value
		}
	}
	return merged
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}
