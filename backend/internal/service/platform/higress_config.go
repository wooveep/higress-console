package platform

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
)

const (
	maxHigressRequestHeadersKB            = 8192
	minHigressConcurrentStreams           = 1
	maxHigressConcurrentStreams           = 2147483647
	minHigressInitialWindowSize           = 65535
	maxHigressInitialWindowSize           = 2147483647
	minHigressGzipMemoryLevel             = 1
	maxHigressGzipMemoryLevel             = 9
	minHigressGzipWindowBits              = 9
	maxHigressGzipWindowBits              = 15
	defaultHigressUpstreamBufferLimit int = 10485760
)

var (
	validHigressCompressionLevels = map[string]struct{}{
		"BEST_COMPRESSION":    {},
		"BEST_SPEED":          {},
		"COMPRESSION_LEVEL_1": {},
		"COMPRESSION_LEVEL_2": {},
		"COMPRESSION_LEVEL_3": {},
		"COMPRESSION_LEVEL_4": {},
		"COMPRESSION_LEVEL_5": {},
		"COMPRESSION_LEVEL_6": {},
		"COMPRESSION_LEVEL_7": {},
		"COMPRESSION_LEVEL_8": {},
		"COMPRESSION_LEVEL_9": {},
	}
	validHigressCompressionStrategies = map[string]struct{}{
		"DEFAULT_STRATEGY": {},
		"FILTERED":         {},
		"HUFFMAN_ONLY":     {},
		"RLE":              {},
		"FIXED":            {},
	}
)

func (s *Service) GetAIGatewayConfig(ctx context.Context) (string, error) {
	data, err := s.readHigressConfigMap(ctx)
	if err != nil {
		return "", err
	}
	if raw := strings.TrimSpace(data[consts.DefaultHigressConfigDataKey]); raw != "" {
		return ensureTrailingNewline(data[consts.DefaultHigressConfigDataKey]), nil
	}
	return defaultHigressConfigYAML(), nil
}

func (s *Service) SetAIGatewayConfig(ctx context.Context, raw string) (string, error) {
	normalized, err := normalizeAndValidateHigressConfig(raw)
	if err != nil {
		return "", err
	}

	data, err := s.readHigressConfigMap(ctx)
	if err != nil {
		return "", err
	}
	data[consts.DefaultHigressConfigDataKey] = normalized
	if err := s.k8sClient.UpsertConfigMap(ctx, consts.DefaultHigressConfigMapName, data); err != nil {
		return "", err
	}
	return normalized, nil
}

func (s *Service) readHigressConfigMap(ctx context.Context) (map[string]string, error) {
	data, err := s.k8sClient.ReadConfigMap(ctx, consts.DefaultHigressConfigMapName)
	if err != nil {
		if errors.Is(err, k8sclient.ErrNotFound) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	return data, nil
}

func defaultHigressConfigYAML() string {
	payload := map[string]any{
		"tracing": map[string]any{
			"enable":   false,
			"sampling": 100,
			"timeout":  500,
		},
		"gzip": map[string]any{
			"enable":              true,
			"minContentLength":    1024,
			"contentType":         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
			"disableOnEtagHeader": true,
			"memoryLevel":         5,
			"windowBits":          12,
			"chunkSize":           4096,
			"compressionLevel":    "BEST_COMPRESSION",
			"compressionStrategy": "DEFAULT_STRATEGY",
		},
		"downstream": map[string]any{
			"idleTimeout":            180,
			"maxRequestHeadersKb":    60,
			"connectionBufferLimits": 32768,
			"http2": map[string]any{
				"maxConcurrentStreams":        100,
				"initialStreamWindowSize":     65535,
				"initialConnectionWindowSize": 1048576,
			},
			"routeTimeout": 0,
		},
		"upstream": map[string]any{
			"idleTimeout":            10,
			"connectionBufferLimits": defaultHigressUpstreamBufferLimit,
		},
		"addXRealIpHeader":     false,
		"disableXEnvoyHeaders": false,
	}
	bytes, err := yaml.Marshal(payload)
	if err != nil {
		return ""
	}
	return string(bytes)
}

func normalizeAndValidateHigressConfig(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.New("higress config is empty")
	}

	var root map[string]any
	if err := yaml.Unmarshal([]byte(trimmed), &root); err != nil {
		return "", fmt.Errorf("invalid higress config YAML: %w", err)
	}
	if root == nil {
		return "", errors.New("higress config must be a YAML object")
	}
	if err := validateHigressConfigRoot(root); err != nil {
		return "", err
	}
	return ensureTrailingNewline(raw), nil
}

func validateHigressConfigRoot(root map[string]any) error {
	if err := validateKnownBoolean(root, "addXRealIpHeader"); err != nil {
		return err
	}
	if err := validateKnownBoolean(root, "disableXEnvoyHeaders"); err != nil {
		return err
	}
	if err := validateTracingConfig(root); err != nil {
		return err
	}
	if err := validateGzipConfig(root); err != nil {
		return err
	}
	if err := validateDownstreamConfig(root); err != nil {
		return err
	}
	if err := validateUpstreamConfig(root); err != nil {
		return err
	}
	return nil
}

func validateTracingConfig(root map[string]any) error {
	value, ok := root["tracing"]
	if !ok || value == nil {
		return nil
	}
	section, ok := value.(map[string]any)
	if !ok {
		return errors.New("tracing must be an object")
	}

	enable, err := readOptionalBool(section, "enable")
	if err != nil {
		return err
	}
	if _, err := readOptionalNumber(section, "sampling"); err != nil {
		return err
	}
	if sampling, ok := optionalNumberValue(section, "sampling"); ok && (sampling < 0 || sampling > 100) {
		return errors.New("tracing.sampling must be between 0 and 100")
	}
	timeout, ok := optionalIntegerValue(section, "timeout")
	if _, err := readOptionalInteger(section, "timeout"); err != nil {
		return err
	}
	if ok && timeout <= 0 {
		return errors.New("tracing.timeout must be greater than 0")
	}

	backendCount := 0
	for _, backend := range []string{"skywalking", "zipkin", "opentelemetry"} {
		if !hasNonNilValue(section, backend) {
			continue
		}
		backendCount++
		config, ok := section[backend].(map[string]any)
		if !ok {
			return fmt.Errorf("tracing.%s must be an object", backend)
		}
		if backend == "skywalking" {
			if err := validateRequiredString(config, "service", "tracing.skywalking.service"); err != nil {
				return err
			}
			if err := validateRequiredString(config, "port", "tracing.skywalking.port"); err != nil {
				return err
			}
			if err := validateOptionalString(config, "access_token", "tracing.skywalking.access_token"); err != nil {
				return err
			}
			continue
		}
		if err := validateRequiredString(config, "service", fmt.Sprintf("tracing.%s.service", backend)); err != nil {
			return err
		}
		if err := validateRequiredString(config, "port", fmt.Sprintf("tracing.%s.port", backend)); err != nil {
			return err
		}
	}

	if enable && backendCount != 1 {
		return errors.New("tracing requires exactly one backend when enabled")
	}
	return nil
}

func validateGzipConfig(root map[string]any) error {
	value, ok := root["gzip"]
	if !ok || value == nil {
		return nil
	}
	section, ok := value.(map[string]any)
	if !ok {
		return errors.New("gzip must be an object")
	}

	if err := validateKnownBoolean(section, "enable"); err != nil {
		return wrapSectionError("gzip", err)
	}
	if minLength, ok := optionalIntegerValue(section, "minContentLength"); ok && minLength <= 0 {
		return errors.New("gzip.minContentLength must be greater than 0")
	}
	if _, err := readOptionalInteger(section, "minContentLength"); err != nil {
		return err
	}
	if err := validateStringArray(section, "contentType", "gzip.contentType"); err != nil {
		return err
	}
	if hasNonNilValue(section, "contentType") {
		values := toStringSlice(section["contentType"])
		if len(values) == 0 {
			return errors.New("gzip.contentType must not be empty")
		}
	}
	if err := validateKnownBoolean(section, "disableOnEtagHeader"); err != nil {
		return wrapSectionError("gzip", err)
	}

	if level, ok := optionalIntegerValue(section, "memoryLevel"); ok && (level < minHigressGzipMemoryLevel || level > maxHigressGzipMemoryLevel) {
		return errors.New("gzip.memoryLevel must be between 1 and 9")
	}
	if _, err := readOptionalInteger(section, "memoryLevel"); err != nil {
		return err
	}
	if bits, ok := optionalIntegerValue(section, "windowBits"); ok && (bits < minHigressGzipWindowBits || bits > maxHigressGzipWindowBits) {
		return errors.New("gzip.windowBits must be between 9 and 15")
	}
	if _, err := readOptionalInteger(section, "windowBits"); err != nil {
		return err
	}
	if size, ok := optionalIntegerValue(section, "chunkSize"); ok && size <= 0 {
		return errors.New("gzip.chunkSize must be greater than 0")
	}
	if _, err := readOptionalInteger(section, "chunkSize"); err != nil {
		return err
	}

	level, err := readOptionalString(section, "compressionLevel")
	if err != nil {
		return err
	}
	if level != "" {
		if _, ok := validHigressCompressionLevels[level]; !ok {
			return fmt.Errorf("gzip.compressionLevel must be one of %s", strings.Join(mapKeys(validHigressCompressionLevels), ","))
		}
	}
	strategy, err := readOptionalString(section, "compressionStrategy")
	if err != nil {
		return err
	}
	if strategy != "" {
		if _, ok := validHigressCompressionStrategies[strategy]; !ok {
			return fmt.Errorf("gzip.compressionStrategy must be one of %s", strings.Join(mapKeys(validHigressCompressionStrategies), ","))
		}
	}
	return nil
}

func validateDownstreamConfig(root map[string]any) error {
	value, ok := root["downstream"]
	if !ok || value == nil {
		return nil
	}
	section, ok := value.(map[string]any)
	if !ok {
		return errors.New("downstream must be an object")
	}

	if _, err := readOptionalInteger(section, "connectionBufferLimits"); err != nil {
		return err
	}
	if connectionBufferLimits, ok := optionalIntegerValue(section, "connectionBufferLimits"); ok && connectionBufferLimits < 0 {
		return errors.New("downstream.connectionBufferLimits must be greater than or equal to 0")
	}
	if _, err := readOptionalInteger(section, "idleTimeout"); err != nil {
		return err
	}
	if idleTimeout, ok := optionalIntegerValue(section, "idleTimeout"); ok && idleTimeout < 0 {
		return errors.New("downstream.idleTimeout must be greater than or equal to 0")
	}
	if _, err := readOptionalInteger(section, "maxRequestHeadersKb"); err != nil {
		return err
	}
	if maxRequestHeadersKb, ok := optionalIntegerValue(section, "maxRequestHeadersKb"); ok && maxRequestHeadersKb > maxHigressRequestHeadersKB {
		return fmt.Errorf("downstream.maxRequestHeadersKb must be less than or equal to %d", maxHigressRequestHeadersKB)
	}
	if maxRequestHeadersKb, ok := optionalIntegerValue(section, "maxRequestHeadersKb"); ok && maxRequestHeadersKb < 0 {
		return errors.New("downstream.maxRequestHeadersKb must be greater than or equal to 0")
	}
	if _, err := readOptionalInteger(section, "routeTimeout"); err != nil {
		return err
	}
	if routeTimeout, ok := optionalIntegerValue(section, "routeTimeout"); ok && routeTimeout < 0 {
		return errors.New("downstream.routeTimeout must be greater than or equal to 0")
	}

	http2Value, ok := section["http2"]
	if !ok || http2Value == nil {
		return nil
	}
	http2, ok := http2Value.(map[string]any)
	if !ok {
		return errors.New("downstream.http2 must be an object")
	}

	if _, err := readOptionalInteger(http2, "maxConcurrentStreams"); err != nil {
		return err
	}
	if streams, ok := optionalIntegerValue(http2, "maxConcurrentStreams"); ok && (streams < minHigressConcurrentStreams || streams > maxHigressConcurrentStreams) {
		return fmt.Errorf("downstream.http2.maxConcurrentStreams must be between %d and %d", minHigressConcurrentStreams, maxHigressConcurrentStreams)
	}
	if _, err := readOptionalInteger(http2, "initialStreamWindowSize"); err != nil {
		return err
	}
	if window, ok := optionalIntegerValue(http2, "initialStreamWindowSize"); ok && (window < minHigressInitialWindowSize || window > maxHigressInitialWindowSize) {
		return fmt.Errorf("downstream.http2.initialStreamWindowSize must be between %d and %d", minHigressInitialWindowSize, maxHigressInitialWindowSize)
	}
	if _, err := readOptionalInteger(http2, "initialConnectionWindowSize"); err != nil {
		return err
	}
	if window, ok := optionalIntegerValue(http2, "initialConnectionWindowSize"); ok && (window < minHigressInitialWindowSize || window > maxHigressInitialWindowSize) {
		return fmt.Errorf("downstream.http2.initialConnectionWindowSize must be between %d and %d", minHigressInitialWindowSize, maxHigressInitialWindowSize)
	}
	return nil
}

func validateUpstreamConfig(root map[string]any) error {
	value, ok := root["upstream"]
	if !ok || value == nil {
		return nil
	}
	section, ok := value.(map[string]any)
	if !ok {
		return errors.New("upstream must be an object")
	}

	if _, err := readOptionalInteger(section, "connectionBufferLimits"); err != nil {
		return err
	}
	if connectionBufferLimits, ok := optionalIntegerValue(section, "connectionBufferLimits"); ok && connectionBufferLimits < 0 {
		return errors.New("upstream.connectionBufferLimits must be greater than or equal to 0")
	}
	if _, err := readOptionalInteger(section, "idleTimeout"); err != nil {
		return err
	}
	if idleTimeout, ok := optionalIntegerValue(section, "idleTimeout"); ok && idleTimeout < 0 {
		return errors.New("upstream.idleTimeout must be greater than or equal to 0")
	}
	return nil
}

func validateKnownBoolean(root map[string]any, key string) error {
	if !hasNonNilValue(root, key) {
		return nil
	}
	if _, ok := root[key].(bool); !ok {
		return fmt.Errorf("%s must be a boolean", key)
	}
	return nil
}

func validateRequiredString(root map[string]any, key, path string) error {
	value, err := readOptionalString(root, key)
	if err != nil {
		return err
	}
	if value == "" {
		return fmt.Errorf("%s is required", path)
	}
	return nil
}

func validateOptionalString(root map[string]any, key, path string) error {
	if !hasNonNilValue(root, key) {
		return nil
	}
	if _, err := readOptionalString(root, key); err != nil {
		return fmt.Errorf("%s must be a string", path)
	}
	return nil
}

func validateStringArray(root map[string]any, key, path string) error {
	if !hasNonNilValue(root, key) {
		return nil
	}
	if _, ok := root[key].([]any); !ok {
		if _, ok := root[key].([]string); !ok {
			return fmt.Errorf("%s must be an array of strings", path)
		}
	}
	for _, item := range toStringSlice(root[key]) {
		if strings.TrimSpace(item) == "" {
			return fmt.Errorf("%s must not contain empty items", path)
		}
	}
	return nil
}

func readOptionalString(root map[string]any, key string) (string, error) {
	if !hasNonNilValue(root, key) {
		return "", nil
	}
	value, ok := root[key].(string)
	if !ok {
		return "", fmt.Errorf("%s must be a string", key)
	}
	return strings.TrimSpace(value), nil
}

func readOptionalBool(root map[string]any, key string) (bool, error) {
	if !hasNonNilValue(root, key) {
		return false, nil
	}
	value, ok := root[key].(bool)
	if !ok {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return value, nil
}

func readOptionalNumber(root map[string]any, key string) (float64, error) {
	if !hasNonNilValue(root, key) {
		return 0, nil
	}
	value, ok := toFloat64(root[key])
	if !ok {
		return 0, fmt.Errorf("%s must be a number", key)
	}
	return value, nil
}

func readOptionalInteger(root map[string]any, key string) (int64, error) {
	if !hasNonNilValue(root, key) {
		return 0, nil
	}
	value, ok := toInt64(root[key])
	if !ok {
		return 0, fmt.Errorf("%s must be an integer", key)
	}
	return value, nil
}

func optionalNumberValue(root map[string]any, key string) (float64, bool) {
	if !hasNonNilValue(root, key) {
		return 0, false
	}
	return toFloat64(root[key])
}

func optionalIntegerValue(root map[string]any, key string) (int64, bool) {
	if !hasNonNilValue(root, key) {
		return 0, false
	}
	return toInt64(root[key])
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				return nil
			}
			items = append(items, text)
		}
		return items
	default:
		return nil
	}
}

func toFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func toInt64(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > maxHigressConcurrentStreams {
			return 0, false
		}
		return int64(typed), true
	case float64:
		if typed != float64(int64(typed)) {
			return 0, false
		}
		return int64(typed), true
	case float32:
		if typed != float32(int64(typed)) {
			return 0, false
		}
		return int64(typed), true
	case string:
		value, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return value, true
	default:
		return 0, false
	}
}

func ensureTrailingNewline(raw string) string {
	trimmed := strings.TrimRight(raw, "\n")
	if trimmed == "" {
		return ""
	}
	return trimmed + "\n"
}

func hasNonNilValue(root map[string]any, key string) bool {
	value, ok := root[key]
	return ok && value != nil
}

func mapKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	return keys
}

func wrapSectionError(section string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s.%s", section, err.Error())
}
