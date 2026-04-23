package session

import (
	"net/http"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	"github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

type LoginRequest struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	AutoLogin bool   `json:"autoLogin"`
}

func Bind(group *ghttp.RouterGroup, platformService *platform.Service) {
	group.POST("/session/login", func(r *ghttp.Request) {
		var req LoginRequest
		if err := r.Parse(&req); err != nil {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": err.Error()})
			return
		}
		if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" {
			r.Response.WriteStatusExit(400, g.Map{"success": false, "message": "missing user name or password"})
			return
		}

		user, token, err := platformService.Login(r.Context(), req.Username, req.Password)
		if err != nil {
			r.Response.WriteStatusExit(401, g.Map{"success": false, "message": err.Error()})
			return
		}
		maxAge := time.Duration(0)
		if req.AutoLogin {
			maxAge = time.Duration(consts.DefaultAdminCookieMaxAge) * time.Second
		}
		r.Cookie.SetHttpCookie(&http.Cookie{
			Name:     consts.DefaultAdminCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			MaxAge:   int(maxAge / time.Second),
		})
		r.Response.WriteJsonExit(g.Map{"success": true, "data": user})
	})

	group.GET("/session/logout", func(r *ghttp.Request) {
		r.Cookie.SetHttpCookie(&http.Cookie{
			Name:     consts.DefaultAdminCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})
		r.Response.WriteJsonExit(g.Map{"success": true, "data": true})
	})
}
