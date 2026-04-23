package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMemoryClientResourceCRUD(t *testing.T) {
	client := NewMemoryClient()

	item, err := client.UpsertResource(context.Background(), "routes", "demo", map[string]any{
		"name": "demo",
		"path": map[string]any{"matchValue": "/"},
	})
	require.NoError(t, err)
	require.Equal(t, "demo", item["name"])

	loaded, err := client.GetResource(context.Background(), "routes", "demo")
	require.NoError(t, err)
	require.Equal(t, "demo", loaded["name"])

	list, err := client.ListResources(context.Background(), "routes")
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, client.DeleteResource(context.Background(), "routes", "demo"))
}

func TestConfigMapRenderRoundTrip(t *testing.T) {
	rendered, err := RenderConfigMapYAML("demo", map[string]string{
		"resourceVersion": "2",
		"aigateway":       "enabled: true\n",
		"mesh":            "defaultConfig: {}\n",
		"meshNetworks":    "networks: {}\n",
	})
	require.NoError(t, err)
	require.Contains(t, rendered, "ConfigMap")

	parsed, err := ParseConfigMapYAML(rendered)
	require.NoError(t, err)
	require.Equal(t, "2", parsed["resourceVersion"])
}

func TestLabelValueNormalizesSpecialCharacters(t *testing.T) {
	require.Equal(t, "mcp-servers-demo-route", labelValue("mcp-servers:demo_route"))
}

func TestIncrementVersion(t *testing.T) {
	require.Equal(t, "2", incrementVersion("1"))
	require.Equal(t, "1", incrementVersion("bad"))
}
