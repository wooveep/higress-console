package gateway

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var builtinPluginResourceDirs = []string{
	"backend/resource/public/plugin",
	"backend-java-legacy/sdk/src/main/resources/plugins",
	"backend/sdk/src/main/resources/plugins",
}

func (s *Service) mergeWasmPlugins(items []map[string]any) []map[string]any {
	index := map[string]map[string]any{}
	for _, item := range items {
		hydrated := s.hydrateResource("wasm-plugins", item)
		index[strings.TrimSpace(strings.ToLower(stringValue(hydrated["name"])))] = hydrated
	}
	for _, builtin := range s.listBuiltinWasmPlugins() {
		key := strings.TrimSpace(strings.ToLower(stringValue(builtin["name"])))
		if key == "" {
			continue
		}
		if existing, ok := index[key]; ok {
			index[key] = mergeMaps(builtin, existing)
			continue
		}
		index[key] = builtin
	}
	result := make([]map[string]any, 0, len(index))
	for _, item := range index {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		return stringValue(result[i]["name"]) < stringValue(result[j]["name"])
	})
	return result
}

func (s *Service) listBuiltinWasmPlugins() []map[string]any {
	roots := resolveBuiltinPluginRoots()
	if len(roots) == 0 {
		return []map[string]any{}
	}
	index := map[string]map[string]any{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			key := strings.TrimSpace(strings.ToLower(entry.Name()))
			if key == "" || index[key] != nil {
				continue
			}
			if item, ok := loadBuiltinWasmPluginFromRoot(root, entry.Name()); ok {
				index[key] = item
			}
		}
	}
	items := make([]map[string]any, 0, len(index))
	for _, item := range index {
		items = append(items, item)
	}
	return items
}

func (s *Service) loadBuiltinWasmPlugin(name string) (map[string]any, bool) {
	for _, root := range resolveBuiltinPluginRoots() {
		if item, ok := loadBuiltinWasmPluginFromRoot(root, name); ok {
			return item, true
		}
	}
	return nil, false
}

func loadBuiltinWasmPluginFromRoot(base, name string) (map[string]any, bool) {
	root := filepath.Join(base, name)
	specBytes, err := os.ReadFile(filepath.Join(root, "spec.yaml"))
	if err != nil {
		return nil, false
	}
	var spec map[string]any
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		return nil, false
	}

	info, _ := spec["info"].(map[string]any)
	specNode, _ := spec["spec"].(map[string]any)
	item := map[string]any{
		"name":        name,
		"builtIn":     true,
		"internal":    true,
		"title":       stringValue(info["title"]),
		"description": stringValue(info["description"]),
		"category":    stringValue(info["category"]),
		"icon":        stringValue(info["iconUrl"]),
		"iconUrl":     stringValue(info["iconUrl"]),
		"version":     stringValue(info["version"]),
		"phase":       strings.ToUpper(stringValue(specNode["phase"])),
		"priority":    specNode["priority"],
	}
	if schema, ok := specNode["configSchema"]; ok {
		item["configSchema"] = schema
	}
	if schema, ok := specNode["routeConfigSchema"]; ok {
		item["routeConfigSchema"] = schema
		if item["configSchema"] == nil {
			item["configSchema"] = schema
		}
	}
	if readme := firstExistingTextFile(filepath.Join(root, "README_EN.md"), filepath.Join(root, "README.md")); strings.TrimSpace(readme) != "" {
		item["readme"] = readme
		item["documentation"] = readme
	}
	return item, true
}

func resolveBuiltinPluginRoots() []string {
	roots := make([]string, 0, len(builtinPluginResourceDirs))
	seen := map[string]struct{}{}
	for _, resourceDir := range builtinPluginResourceDirs {
		for _, candidate := range relativeResourceCandidates(resourceDir) {
			if isBuiltinPluginRoot(candidate) {
				cleaned := filepath.Clean(candidate)
				if _, ok := seen[cleaned]; ok {
					continue
				}
				seen[cleaned] = struct{}{}
				roots = append(roots, cleaned)
				break
			}
		}
	}
	return roots
}

func relativeResourceCandidates(resourceDir string) []string {
	candidates := []string{
		resourceDir,
		filepath.Join("..", resourceDir),
		filepath.Join("..", "..", resourceDir),
		filepath.Join("..", "..", "..", resourceDir),
		filepath.Join("..", "..", "..", "..", resourceDir),
		filepath.Join("..", "..", "..", "..", "..", resourceDir),
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		base := filepath.Dir(file)
		candidates = append(candidates,
			filepath.Join(base, resourceDir),
			filepath.Join(base, "..", resourceDir),
			filepath.Join(base, "..", "..", resourceDir),
			filepath.Join(base, "..", "..", "..", resourceDir),
			filepath.Join(base, "..", "..", "..", "..", resourceDir),
			filepath.Join(base, "..", "..", "..", "..", "..", resourceDir),
		)
	}
	return candidates
}

func isBuiltinPluginRoot(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(path, entry.Name(), "spec.yaml")); err == nil {
			return true
		}
	}
	return false
}

func firstExistingTextFile(paths ...string) string {
	for _, path := range paths {
		if bytes, err := os.ReadFile(path); err == nil {
			content := strings.TrimSpace(string(bytes))
			if content != "" {
				return content
			}
		}
	}
	return ""
}

func mergeMaps(base, override map[string]any) map[string]any {
	result := clonePayload(base)
	for key, value := range override {
		result[key] = value
	}
	return result
}
