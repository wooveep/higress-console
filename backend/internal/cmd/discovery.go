package cmd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"

	grafanaclient "github.com/wooveep/aigateway-console/backend/utility/clients/grafana"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

const (
	defaultClusterDomain   = "cluster.local"
	defaultControllerSvc   = "aigateway-controller"
	defaultPluginServerSvc = "aigateway-plugin-server"
)

type runtimeConfigValues struct {
	K8SEnabled        bool
	K8SNamespace      string
	K8SKubectlBin     string
	K8SKubeconfigPath string
	K8SResourcePrefix string
	K8SIngressClass   string

	PortalDBEnabled     bool
	PortalDBDriver      string
	PortalDBDSN         string
	PortalDBAutoMigrate bool

	GrafanaEnabled  bool
	GrafanaBaseURL  string
	GrafanaUsername string
	GrafanaPassword string

	ControllerService   string
	PluginServerService string
	ClusterDomain       string
}

type runtimeDependencies struct {
	K8s     k8sclient.Config
	Grafana grafanaclient.Config
	Portal  portaldbclient.Config
}

func loadRuntimeDependencies(ctx context.Context) runtimeDependencies {
	values := runtimeConfigValues{
		K8SEnabled:        g.Cfg().MustGet(ctx, "clients.k8s.enabled", false).Bool(),
		K8SNamespace:      g.Cfg().MustGet(ctx, "clients.k8s.namespace", "aigateway-system").String(),
		K8SKubectlBin:     g.Cfg().MustGet(ctx, "clients.k8s.kubectlBin", "kubectl").String(),
		K8SKubeconfigPath: g.Cfg().MustGet(ctx, "clients.k8s.kubeconfig", "").String(),
		K8SResourcePrefix: g.Cfg().MustGet(ctx, "clients.k8s.resourcePrefix", "aigw-console").String(),
		K8SIngressClass:   g.Cfg().MustGet(ctx, "clients.k8s.ingressClass", "aigateway").String(),

		PortalDBEnabled:     g.Cfg().MustGet(ctx, "clients.portaldb.enabled", false).Bool(),
		PortalDBDriver:      g.Cfg().MustGet(ctx, "clients.portaldb.driver", "").String(),
		PortalDBDSN:         g.Cfg().MustGet(ctx, "clients.portaldb.dsn", "").String(),
		PortalDBAutoMigrate: g.Cfg().MustGet(ctx, "clients.portaldb.autoMigrate", false).Bool(),

		GrafanaEnabled:  g.Cfg().MustGet(ctx, "clients.grafana.enabled", false).Bool(),
		GrafanaBaseURL:  g.Cfg().MustGet(ctx, "clients.grafana.baseURL", "").String(),
		GrafanaUsername: g.Cfg().MustGet(ctx, "clients.grafana.username", "").String(),
		GrafanaPassword: g.Cfg().MustGet(ctx, "clients.grafana.password", "").String(),

		ControllerService:   g.Cfg().MustGet(ctx, "controller.serviceName", "").String(),
		PluginServerService: g.Cfg().MustGet(ctx, "pluginServer.serviceName", "").String(),
		ClusterDomain:       g.Cfg().MustGet(ctx, "global.proxy.clusterDomain", defaultClusterDomain).String(),
	}

	return resolveRuntimeDependencies(values, os.Getenv)
}

func resolveRuntimeDependencies(values runtimeConfigValues, lookup func(string) string) runtimeDependencies {
	namespace := firstNonEmpty(
		lookup("AIGATEWAY_CONSOLE_NAMESPACE"),
		lookup("AIGATEWAY_CONSOLE_K8S_NAMESPACE"),
		lookup("HIGRESS_CONSOLE_CONTROLLER_WATCHED_NAMESPACE"),
		values.K8SNamespace,
		"aigateway-system",
	)
	clusterDomain := firstNonEmpty(
		lookup("AIGATEWAY_CONSOLE_CLUSTER_DOMAIN"),
		lookup("CLUSTER_DOMAIN_SUFFIX"),
		values.ClusterDomain,
		defaultClusterDomain,
	)
	controllerService := firstNonEmpty(
		lookup("AIGATEWAY_CONSOLE_CONTROLLER_SERVICE"),
		lookup("HIGRESS_CONSOLE_CONTROLLER_SERVICE_NAME"),
		values.ControllerService,
		defaultControllerSvc,
	)
	_ = controllerService
	pluginServerService := firstNonEmpty(
		lookup("AIGATEWAY_CONSOLE_PLUGIN_SERVER_SERVICE"),
		values.PluginServerService,
		defaultPluginServerSvc,
	)
	_ = pluginServerService

	k8sEnabled := values.K8SEnabled
	if enabled, ok := parseOptionalBool(lookup("AIGATEWAY_CONSOLE_K8S_ENABLED")); ok {
		k8sEnabled = enabled
	}
	k8sConfig := k8sclient.Config{
		Enabled:        k8sEnabled,
		Namespace:      namespace,
		KubectlBin:     firstNonEmpty(lookup("AIGATEWAY_CONSOLE_KUBECTL_BIN"), values.K8SKubectlBin, "kubectl"),
		KubeconfigPath: firstNonEmpty(lookup("KUBECONFIG"), lookup("AIGATEWAY_CONSOLE_KUBECONFIG"), values.K8SKubeconfigPath),
		ResourcePrefix: firstNonEmpty(lookup("AIGATEWAY_CONSOLE_RESOURCE_PREFIX"), values.K8SResourcePrefix, "aigw-console"),
		IngressClass: firstNonEmpty(
			lookup("AIGATEWAY_CONSOLE_INGRESS_CLASS"),
			lookup("HIGRESS_CONSOLE_CONTROLLER_INGRESS_CLASS_NAME"),
			values.K8SIngressClass,
			"aigateway",
		),
	}

	portalConfig := resolvePortalDBConfig(values, lookup)
	grafanaConfig := resolveGrafanaConfig(values, lookup, namespace, clusterDomain)

	return runtimeDependencies{
		K8s:     k8sConfig,
		Grafana: grafanaConfig,
		Portal:  portalConfig,
	}
}

func resolvePortalDBConfig(values runtimeConfigValues, lookup func(string) string) portaldbclient.Config {
	driver := firstNonEmpty(lookup("AIGATEWAY_CONSOLE_PORTALDB_DRIVER"), values.PortalDBDriver, "mysql")
	dsn, discovered := resolvePortalDBDSN(values, lookup)
	enabled := values.PortalDBEnabled
	if override, ok := parseOptionalBool(lookup("AIGATEWAY_CONSOLE_PORTALDB_ENABLED")); ok {
		enabled = override
	} else if discovered {
		enabled = true
	}
	autoMigrate := values.PortalDBAutoMigrate
	if override, ok := parseOptionalBool(lookup("AIGATEWAY_CONSOLE_PORTALDB_AUTO_MIGRATE")); ok {
		autoMigrate = override
	}

	return portaldbclient.Config{
		Enabled:     enabled,
		Driver:      driver,
		DSN:         dsn,
		AutoMigrate: autoMigrate,
	}
}

func resolvePortalDBDSN(values runtimeConfigValues, lookup func(string) string) (string, bool) {
	if dsn := firstNonEmpty(
		lookup("AIGATEWAY_CONSOLE_PORTALDB_DSN"),
		lookup("PORTAL_MYSQL_DSN"),
		values.PortalDBDSN,
	); dsn != "" {
		return dsn, true
	}

	if hasPortalMySQLConnEnv(lookup) {
		dsn := buildMySQLDSN(
			firstNonEmpty(lookup("PORTAL_MYSQL_HOST"), "127.0.0.1"),
			firstNonEmpty(lookup("PORTAL_MYSQL_PORT"), "3306"),
			firstNonEmpty(lookup("PORTAL_MYSQL_USER"), "root"),
			firstNonEmpty(lookup("PORTAL_MYSQL_PASSWORD"), "root"),
			firstNonEmpty(lookup("PORTAL_MYSQL_DATABASE"), "aigateway_portal"),
			firstNonEmpty(lookup("PORTAL_MYSQL_PARAMS"), "parseTime=true&charset=utf8mb4&loc=UTC"),
		)
		return dsn, true
	}

	rawURL := firstNonEmpty(lookup("PORTAL_CORE_DB_URL"), lookup("HIGRESS_PORTAL_DB_URL"))
	if strings.TrimSpace(rawURL) == "" {
		return "", false
	}
	host, port, database, params, err := parseMySQLJDBCURL(rawURL)
	if err != nil {
		return "", false
	}
	dsn := buildMySQLDSN(
		host,
		port,
		firstNonEmpty(lookup("PORTAL_CORE_DB_USERNAME"), lookup("HIGRESS_PORTAL_DB_USERNAME"), "root"),
		firstNonEmpty(lookup("PORTAL_CORE_DB_PASSWORD"), lookup("HIGRESS_PORTAL_DB_PASSWORD"), "root"),
		database,
		params,
	)
	return dsn, true
}

func resolveGrafanaConfig(values runtimeConfigValues, lookup func(string) string, namespace, clusterDomain string) grafanaclient.Config {
	baseURL := strings.TrimSpace(values.GrafanaBaseURL)
	if service := strings.TrimSpace(lookup("AIGATEWAY_CONSOLE_GRAFANA_SERVICE")); service != "" {
		baseURL = buildClusterServiceURL(
			firstNonEmpty(lookup("AIGATEWAY_CONSOLE_GRAFANA_SCHEME"), "http"),
			service,
			namespace,
			clusterDomain,
			firstNonEmpty(lookup("AIGATEWAY_CONSOLE_GRAFANA_PORT"), "3000"),
			firstNonEmpty(lookup("AIGATEWAY_CONSOLE_GRAFANA_PATH"), "/grafana"),
		)
	} else if legacy := strings.TrimSpace(lookup("HIGRESS_CONSOLE_DASHBOARD_BASE_URL")); legacy != "" {
		baseURL = legacy
	}

	enabled := values.GrafanaEnabled
	if override, ok := parseOptionalBool(lookup("AIGATEWAY_CONSOLE_GRAFANA_ENABLED")); ok {
		enabled = override
	} else if baseURL != "" {
		enabled = true
	}

	return grafanaclient.Config{
		Enabled:  enabled,
		BaseURL:  baseURL,
		Username: values.GrafanaUsername,
		Password: values.GrafanaPassword,
	}
}

func hasPortalMySQLConnEnv(lookup func(string) string) bool {
	keys := []string{
		"PORTAL_MYSQL_HOST",
		"PORTAL_MYSQL_PORT",
		"PORTAL_MYSQL_USER",
		"PORTAL_MYSQL_PASSWORD",
		"PORTAL_MYSQL_DATABASE",
		"PORTAL_MYSQL_PARAMS",
	}
	for _, key := range keys {
		if strings.TrimSpace(lookup(key)) != "" {
			return true
		}
	}
	return false
}

func buildMySQLDSN(host, port, user, password, database, params string) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, password, host, port, database, normalizeMySQLParams(params))
}

func buildClusterServiceURL(scheme, service, namespace, clusterDomain, port, path string) string {
	scheme = firstNonEmpty(scheme, "http")
	service = strings.TrimSpace(service)
	namespace = firstNonEmpty(namespace, "aigateway-system")
	clusterDomain = firstNonEmpty(clusterDomain, defaultClusterDomain)
	port = strings.TrimSpace(port)
	path = strings.TrimSpace(path)
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s://%s.%s.svc.%s:%s%s", scheme, service, namespace, clusterDomain, port, path)
}

func parseOptionalBool(raw string) (bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return parsed, true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseMySQLJDBCURL(raw string) (host string, port string, database string, params string, err error) {
	const prefix = "jdbc:mysql://"
	if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(raw)), prefix) {
		return "", "", "", "", fmt.Errorf("unsupported jdbc mysql url")
	}
	parsed, err := url.Parse("mysql://" + strings.TrimSpace(raw)[len(prefix):])
	if err != nil {
		return "", "", "", "", err
	}
	host = parsed.Hostname()
	port = parsed.Port()
	if port == "" {
		port = "3306"
	}
	database = strings.TrimPrefix(parsed.Path, "/")
	if database == "" {
		return "", "", "", "", fmt.Errorf("missing database in jdbc mysql url")
	}
	params = normalizeMySQLParams(parsed.RawQuery)
	return host, port, database, params, nil
}

func normalizeMySQLParams(raw string) string {
	const defaultParams = "parseTime=true&charset=utf8mb4&loc=UTC"
	if strings.TrimSpace(raw) == "" {
		return defaultParams
	}

	query, err := url.ParseQuery(raw)
	if err != nil {
		return raw
	}

	if query.Get("parseTime") == "" {
		query.Set("parseTime", "true")
	}
	if query.Get("charset") == "" {
		if encoding := strings.TrimSpace(query.Get("characterEncoding")); encoding != "" {
			query.Set("charset", encoding)
		} else {
			query.Set("charset", "utf8mb4")
		}
	}
	query.Set("loc", "UTC")
	query.Del("serverTimezone")
	return query.Encode()
}
