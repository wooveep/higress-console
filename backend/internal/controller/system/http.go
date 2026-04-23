package system

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/wooveep/aigateway-console/backend/internal/model/response"
	"github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

type systemInitRequest struct {
	AdminUser *response.User `json:"adminUser"`
	Configs   map[string]any `json:"configs"`
}

type updateConfigRequest struct {
	Config string `json:"config"`
}

func BindHTTP(group *ghttp.RouterGroup, platformService *platform.Service) {
	group.POST("/system/init", func(r *ghttp.Request) {
		var req systemInitRequest
		if err := r.Parse(&req); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		if err := platformService.InitializeSystem(r.Context(), req.AdminUser, req.Configs); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": true})
	})

	group.GET("/system/aigateway-config", func(r *ghttp.Request) {
		content, err := platformService.GetAIGatewayConfig(r.Context())
		if err != nil {
			r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": content})
	})

	group.PUT("/system/aigateway-config", func(r *ghttp.Request) {
		var req updateConfigRequest
		if err := r.Parse(&req); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		if strings.TrimSpace(req.Config) == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "missing required parameter: config"})
			return
		}
		content, err := platformService.SetAIGatewayConfig(r.Context(), req.Config)
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": content})
	})
}
