package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/stretchr/testify/require"
)

func TestManifestConfigLoads(t *testing.T) {
	ctx := context.Background()

	require.Equal(t, ":8080", g.Cfg().MustGet(ctx, "server.address").String())
	require.Equal(t, "aigateway-console", g.Cfg().MustGet(ctx, "app.name").String())
	require.NotEmpty(t, g.Cfg().MustGet(ctx, "naming.preferred").String())
}

func TestResolveFrontendPublicFile(t *testing.T) {
	filePath, ok := resolveFrontendPublicFile("/logo-ai.svg")
	require.True(t, ok)
	require.True(t, strings.HasSuffix(filePath, "resource/public/html/logo-ai.svg"))

	filePath, ok = resolveFrontendPublicFile("/banner.png")
	require.True(t, ok)
	require.True(t, strings.HasSuffix(filePath, "resource/public/html/banner.png"))

	_, ok = resolveFrontendPublicFile("/system/config")
	require.False(t, ok)
}

func TestResolveRuntimeDependenciesPrefersStructuredEnv(t *testing.T) {
	values := runtimeConfigValues{
		K8SEnabled:          true,
		K8SNamespace:        "cfg-ns",
		K8SKubectlBin:       "kubectl",
		K8SResourcePrefix:   "cfg-prefix",
		K8SIngressClass:     "cfg-ingress",
		PortalDBEnabled:     false,
		PortalDBDriver:      "mysql",
		PortalDBAutoMigrate: true,
		GrafanaEnabled:      false,
		GrafanaBaseURL:      "http://cfg-grafana",
		ControllerService:   "cfg-controller",
		PluginServerService: "cfg-plugin-server",
		ClusterDomain:       "cfg.local",
	}
	env := map[string]string{
		"AIGATEWAY_CONSOLE_NAMESPACE":             "runtime-ns",
		"AIGATEWAY_CONSOLE_CLUSTER_DOMAIN":        "cluster.local",
		"AIGATEWAY_CONSOLE_CONTROLLER_SERVICE":    "runtime-controller",
		"AIGATEWAY_CONSOLE_PLUGIN_SERVER_SERVICE": "runtime-plugin-server",
		"PORTAL_MYSQL_HOST":                       "mysql-server",
		"PORTAL_MYSQL_PORT":                       "3306",
		"PORTAL_MYSQL_USER":                       "portal",
		"PORTAL_MYSQL_PASSWORD":                   "secret",
		"PORTAL_MYSQL_DATABASE":                   "aigateway_portal",
		"PORTAL_MYSQL_PARAMS":                     "parseTime=true",
		"AIGATEWAY_CONSOLE_GRAFANA_ENABLED":       "true",
		"AIGATEWAY_CONSOLE_GRAFANA_SCHEME":        "http",
		"AIGATEWAY_CONSOLE_GRAFANA_SERVICE":       "aigateway-console-grafana",
		"AIGATEWAY_CONSOLE_GRAFANA_PORT":          "3000",
		"AIGATEWAY_CONSOLE_GRAFANA_PATH":          "/grafana",
	}

	deps := resolveRuntimeDependencies(values, lookupEnvMap(env))

	require.Equal(t, "runtime-ns", deps.K8s.Namespace)
	require.Equal(t, "cfg-prefix", deps.K8s.ResourcePrefix)
	require.Equal(t, "cfg-ingress", deps.K8s.IngressClass)
	require.True(t, deps.Portal.Enabled)
	require.Equal(t, "portal:secret@tcp(mysql-server:3306)/aigateway_portal?charset=utf8mb4&loc=UTC&parseTime=true", deps.Portal.DSN)
	require.True(t, deps.Portal.AutoMigrate)
	require.True(t, deps.Grafana.Enabled)
	require.Equal(t, "http://aigateway-console-grafana.runtime-ns.svc.cluster.local:3000/grafana", deps.Grafana.BaseURL)
}

func TestResolveRuntimeDependenciesFallsBackToLegacyEnv(t *testing.T) {
	values := runtimeConfigValues{
		K8SNamespace: "cfg-ns",
	}
	env := map[string]string{
		"HIGRESS_PORTAL_DB_URL":              "jdbc:mysql://mysql-server:3306/aigateway_portal?useSSL=false",
		"HIGRESS_PORTAL_DB_USERNAME":         "portal",
		"HIGRESS_PORTAL_DB_PASSWORD":         "legacy-secret",
		"HIGRESS_CONSOLE_DASHBOARD_BASE_URL": "http://legacy-grafana/grafana",
	}

	deps := resolveRuntimeDependencies(values, lookupEnvMap(env))

	require.True(t, deps.Portal.Enabled)
	require.Equal(t, "portal:legacy-secret@tcp(mysql-server:3306)/aigateway_portal?charset=utf8mb4&loc=UTC&parseTime=true&useSSL=false", deps.Portal.DSN)
	require.True(t, deps.Grafana.Enabled)
	require.Equal(t, "http://legacy-grafana/grafana", deps.Grafana.BaseURL)
}

func TestResolveRuntimeDependenciesNormalizesJDBCStyleMySQLParams(t *testing.T) {
	values := runtimeConfigValues{}
	env := map[string]string{
		"PORTAL_MYSQL_HOST":     "mysql-server",
		"PORTAL_MYSQL_PORT":     "3306",
		"PORTAL_MYSQL_USER":     "portal",
		"PORTAL_MYSQL_PASSWORD": "secret",
		"PORTAL_MYSQL_DATABASE": "aigateway_portal",
		"PORTAL_MYSQL_PARAMS":   "useUnicode=true&characterEncoding=UTF-8&useSSL=false&allowPublicKeyRetrieval=true&serverTimezone=Asia/Shanghai",
	}

	deps := resolveRuntimeDependencies(values, lookupEnvMap(env))

	require.True(t, deps.Portal.Enabled)
	require.Equal(
		t,
		"portal:secret@tcp(mysql-server:3306)/aigateway_portal?allowPublicKeyRetrieval=true&characterEncoding=UTF-8&charset=UTF-8&loc=UTC&parseTime=true&useSSL=false&useUnicode=true",
		deps.Portal.DSN,
	)
}

func TestResolveRuntimeDependenciesAllowsExplicitDisable(t *testing.T) {
	values := runtimeConfigValues{
		PortalDBEnabled: true,
		GrafanaEnabled:  true,
	}
	env := map[string]string{
		"AIGATEWAY_CONSOLE_PORTALDB_ENABLED": "false",
		"AIGATEWAY_CONSOLE_PORTALDB_DSN":     "root:root@tcp(db:3306)/manual?parseTime=true",
		"AIGATEWAY_CONSOLE_GRAFANA_ENABLED":  "false",
		"AIGATEWAY_CONSOLE_GRAFANA_SERVICE":  "aigateway-console-grafana",
		"AIGATEWAY_CONSOLE_GRAFANA_PORT":     "3000",
	}

	deps := resolveRuntimeDependencies(values, lookupEnvMap(env))

	require.False(t, deps.Portal.Enabled)
	require.Equal(t, "root:root@tcp(db:3306)/manual?parseTime=true", deps.Portal.DSN)
	require.False(t, deps.Grafana.Enabled)
}

func TestResolveRuntimeDependenciesAllowsPortalAutoMigrateOverride(t *testing.T) {
	values := runtimeConfigValues{
		PortalDBEnabled:     true,
		PortalDBDriver:      "mysql",
		PortalDBAutoMigrate: false,
	}
	env := map[string]string{
		"PORTAL_MYSQL_HOST":                       "mysql-server",
		"PORTAL_MYSQL_PORT":                       "3306",
		"PORTAL_MYSQL_USER":                       "portal",
		"PORTAL_MYSQL_PASSWORD":                   "secret",
		"PORTAL_MYSQL_DATABASE":                   "aigateway_portal",
		"AIGATEWAY_CONSOLE_PORTALDB_AUTO_MIGRATE": "true",
	}

	deps := resolveRuntimeDependencies(values, lookupEnvMap(env))

	require.True(t, deps.Portal.Enabled)
	require.True(t, deps.Portal.AutoMigrate)
}

func lookupEnvMap(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
