package user

import (
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	"github.com/wooveep/aigateway-console/backend/internal/model/response"
	"github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func Bind(group *ghttp.RouterGroup, platformService *platform.Service) {
	group.GET("/user/info", func(r *ghttp.Request) {
		user, _ := r.GetCtxVar(consts.CtxKeyCurrentUser).Val().(*response.User)
		if user == nil {
			r.Response.WriteStatusExit(401, g.Map{"success": false, "message": "unauthorized"})
			return
		}
		r.Response.WriteJsonExit(g.Map{"success": true, "data": user})
	})

	group.POST("/user/changePassword", func(r *ghttp.Request) {
		user, _ := r.GetCtxVar(consts.CtxKeyCurrentUser).Val().(*response.User)
		if user == nil {
			r.Response.WriteStatusExit(401, g.Map{"success": false, "message": "unauthorized"})
			return
		}
		var req ChangePasswordRequest
		if err := r.Parse(&req); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		if strings.TrimSpace(req.OldPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "missing password fields"})
			return
		}
		if err := platformService.ChangePassword(r.Context(), firstNonEmpty(user.Username, user.Name), req.OldPassword, req.NewPassword); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		r.Cookie.Remove(consts.DefaultAdminCookieName)
		r.Response.WriteJsonExit(g.Map{"success": true, "data": true})
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
