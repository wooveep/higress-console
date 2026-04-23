package dashboard

import (
	"strconv"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

type dashboardURLRequest struct {
	URL string `json:"url"`
}

func Bind(group *ghttp.RouterGroup, platformService *platform.Service) {
	group.GET("/dashboard/init", func(r *ghttp.Request) {
		info, err := platformService.InitializeDashboard(r.Context(), normalizeDashboardType(r.GetQuery("type").String()), r.GetQuery("force").Bool())
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": info})
	})

	group.GET("/dashboard/info", func(r *ghttp.Request) {
		info, err := platformService.DashboardInfo(r.Context(), normalizeDashboardType(r.GetQuery("type").String()))
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": info})
	})

	group.PUT("/dashboard/info", func(r *ghttp.Request) {
		var req dashboardURLRequest
		if err := r.Parse(&req); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		if strings.TrimSpace(req.URL) == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "missing required parameter: url"})
			return
		}
		info, err := platformService.SetDashboardURL(r.Context(), normalizeDashboardType(r.GetQuery("type").String()), req.URL)
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": info})
	})

	group.GET("/dashboard/configData", func(r *ghttp.Request) {
		dataSourceUID := r.GetQuery("dataSourceUid").String()
		if strings.TrimSpace(dataSourceUID) == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "missing required parameter: dataSourceUid"})
			return
		}
		data, err := platformService.BuildDashboardConfigData(r.Context(), normalizeDashboardType(r.GetQuery("type").String()), dataSourceUID)
		if err != nil {
			r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": data})
	})

	group.GET("/dashboard/native", func(r *ghttp.Request) {
		from, _ := strconv.ParseInt(r.GetQuery("from").String(), 10, 64)
		to, _ := strconv.ParseInt(r.GetQuery("to").String(), 10, 64)
		data, err := platformService.NativeDashboard(
			r.Context(),
			normalizeDashboardType(r.GetQuery("type").String()),
			from,
			to,
			r.GetQuery("gateway").String(),
			r.GetQuery("namespace").String(),
		)
		if err != nil {
			r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": data})
	})
}

func normalizeDashboardType(value string) string {
	value = strings.TrimSpace(strings.ToUpper(value))
	if value == "" {
		return "MAIN"
	}
	return value
}
