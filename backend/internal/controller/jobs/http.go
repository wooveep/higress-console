package jobs

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	jobssvc "github.com/wooveep/aigateway-console/backend/internal/service/jobs"
)

func Bind(group *ghttp.RouterGroup, jobsService *jobssvc.Service) {
	group.GET("/internal/jobs", func(r *ghttp.Request) {
		items, err := jobsService.ListJobs(r.Context())
		if err != nil {
			r.Response.WriteStatusExit(500, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": items})
	})

	group.GET("/internal/jobs/:name", func(r *ghttp.Request) {
		item, err := jobsService.GetJob(r.Context(), r.GetRouter("name").String())
		if err != nil {
			status := 400
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				status = 404
			}
			r.Response.WriteStatusExit(status, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
	})

	group.POST("/internal/jobs/:name/trigger", func(r *ghttp.Request) {
		var req jobssvc.TriggerInput
		if err := r.Parse(&req); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		item, err := jobsService.Trigger(r.Context(), r.GetRouter("name").String(), req)
		if err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": item})
	})
}
