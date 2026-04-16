package gateway

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	gatewaysvc "github.com/wooveep/aigateway-console/backend/internal/service/gateway"
)

func Bind(group *ghttp.RouterGroup, gatewayService *gatewaysvc.Service) {
	bindCollection(group, gatewayService, "routes", "/v1/routes", true, true)
	bindCollection(group, gatewayService, "domains", "/v1/domains", false, true)
	bindCollection(group, gatewayService, "tls-certificates", "/v1/tls-certificates", false, true)
	bindCollection(group, gatewayService, "service-sources", "/v1/service-sources", false, true)
	bindCollection(group, gatewayService, "proxy-servers", "/v1/proxy-servers", false, true)
	bindCollection(group, gatewayService, "ai-routes", "/v1/ai/routes", true, true)
	bindCollection(group, gatewayService, "ai-providers", "/v1/ai/providers", false, false)
	bindCollection(group, gatewayService, "mcp-servers", "/v1/mcpServer", true, false)
	bindCollection(group, gatewayService, "wasm-plugins", "/v1/wasm-plugins", false, false)

	group.GET("/v1/services", func(r *ghttp.Request) {
		items, err := gatewayService.List(r.Context(), "services")
		if err != nil {
			r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": items})
	})

	group.GET("/v1/mcpServer/consumers", func(r *ghttp.Request) {
		serverName := strings.TrimSpace(r.GetQuery("mcpServerName").String())
		if serverName == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "mcpServerName is empty"})
			return
		}
		items, err := gatewayService.ListMcpConsumers(r.Context(), serverName)
		if err != nil {
			r.Response.WriteStatusExit(404, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": pageResult(items)})
	})

	group.POST("/v1/mcpServer/swaggerToMcpConfig", func(r *ghttp.Request) {
		content := r.Get("content").String()
		if strings.TrimSpace(content) == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "content is required"})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": gatewayService.SwaggerToMCPConfig(content)})
	})

	group.GET("/v1/wasm-plugins/:name/config", func(r *ghttp.Request) {
		item, err := gatewayService.GetWasmPluginConfig(r.Context(), r.GetRouter("name").String())
		if err != nil {
			r.Response.WriteStatusExit(404, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
	})
	group.GET("/v1/wasm-plugins/:name/readme", func(r *ghttp.Request) {
		item, err := gatewayService.GetWasmPluginReadme(r.Context(), r.GetRouter("name").String())
		if err != nil {
			r.Response.WriteStatusExit(404, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
	})

	group.GET("/v1/global/plugin-instances/:pluginName", func(r *ghttp.Request) {
		getPluginInstance(r, gatewayService, "global", "", r.GetRouter("pluginName").String())
	})
	group.PUT("/v1/global/plugin-instances/:pluginName", func(r *ghttp.Request) {
		savePluginInstance(r, gatewayService, "global", "", r.GetRouter("pluginName").String())
	})
	group.DELETE("/v1/global/plugin-instances/:pluginName", func(r *ghttp.Request) {
		deletePluginInstance(r, gatewayService, "global", "", r.GetRouter("pluginName").String())
	})

	group.GET("/v1/routes/:name/plugin-instances", func(r *ghttp.Request) {
		listPluginInstances(r, gatewayService, "route", r.GetRouter("name").String())
	})
	group.GET("/v1/routes/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		getPluginInstance(r, gatewayService, "route", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
	group.PUT("/v1/routes/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		savePluginInstance(r, gatewayService, "route", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
	group.DELETE("/v1/routes/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		deletePluginInstance(r, gatewayService, "route", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})

	group.GET("/v1/domains/:name/plugin-instances", func(r *ghttp.Request) {
		listPluginInstances(r, gatewayService, "domain", r.GetRouter("name").String())
	})
	group.GET("/v1/domains/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		getPluginInstance(r, gatewayService, "domain", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
	group.PUT("/v1/domains/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		savePluginInstance(r, gatewayService, "domain", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
	group.DELETE("/v1/domains/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		deletePluginInstance(r, gatewayService, "domain", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})

	group.GET("/v1/services/:name/plugin-instances", func(r *ghttp.Request) {
		listPluginInstances(r, gatewayService, "service", r.GetRouter("name").String())
	})
	group.GET("/v1/services/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		getPluginInstance(r, gatewayService, "service", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
	group.PUT("/v1/services/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		savePluginInstance(r, gatewayService, "service", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
	group.DELETE("/v1/services/:name/plugin-instances/:pluginName", func(r *ghttp.Request) {
		deletePluginInstance(r, gatewayService, "service", r.GetRouter("name").String(), r.GetRouter("pluginName").String())
	})
}

func bindCollection(group *ghttp.RouterGroup, gatewayService *gatewaysvc.Service, kind, path string, paginated bool, hasGet bool) {
	group.GET(path, func(r *ghttp.Request) {
		items, err := gatewayService.List(r.Context(), kind)
		if err != nil {
			r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
			return
		}
		if paginated {
			r.Response.WriteJsonExit(g.Map{"success": true, "data": pageResult(items)})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": items})
	})

	if hasGet {
		group.GET(path+"/:name", func(r *ghttp.Request) {
			item, err := gatewayService.Get(r.Context(), kind, r.GetRouter("name").String())
			if err != nil {
				r.Response.WriteStatusExit(404, g.Map{"success": false, "message": err.Error()})
				return
			}
			r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
		})
	}

	saveFn := func(r *ghttp.Request) {
		body, err := r.GetJson()
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		payload := body.Map()
		if name := r.GetRouter("name").String(); strings.TrimSpace(name) != "" {
			payload["name"] = name
		}
		item, err := gatewayService.Save(r.Context(), kind, payload)
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
	}
	group.POST(path, saveFn)
	group.PUT(path+"/:name", saveFn)

	group.DELETE(path+"/:name", func(r *ghttp.Request) {
		if err := gatewayService.Delete(r.Context(), kind, r.GetRouter("name").String()); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteStatus(204)
		r.ExitAll()
	})
}

func pageResult(items []map[string]any) map[string]any {
	return map[string]any{
		"data":     items,
		"pageNum":  0,
		"pageSize": len(items),
		"total":    len(items),
	}
}

func listPluginInstances(r *ghttp.Request, gatewayService *gatewaysvc.Service, scope, target string) {
	aliases := splitAliases(r.GetQuery("aliases").String())
	items, err := gatewayService.ListPluginInstances(r.Context(), scope, target, aliases...)
	if err != nil {
		r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"success": true, "data": items})
}

func getPluginInstance(r *ghttp.Request, gatewayService *gatewaysvc.Service, scope, target, pluginName string) {
	aliases := splitAliases(r.GetQuery("aliases").String())
	item, err := gatewayService.GetPluginInstance(r.Context(), scope, target, pluginName, aliases...)
	if err != nil {
		r.Response.WriteStatusExit(404, g.Map{"success": false, "message": err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
}

func splitAliases(value string) []string {
	raw := strings.Split(strings.TrimSpace(value), ",")
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	return items
}

func savePluginInstance(r *ghttp.Request, gatewayService *gatewaysvc.Service, scope, target, pluginName string) {
	body, err := r.GetJson()
	if err != nil {
		r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
		return
	}
	payload := body.Map()
	item, err := gatewayService.SavePluginInstance(r.Context(), scope, target, pluginName, payload)
	if err != nil {
		r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
		return
	}
	r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
}

func deletePluginInstance(r *ghttp.Request, gatewayService *gatewaysvc.Service, scope, target, pluginName string) {
	if err := gatewayService.DeletePluginInstance(r.Context(), scope, target, pluginName); err != nil {
		r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
		return
	}
	r.Response.WriteStatus(204)
	r.ExitAll()
}
