package middleware

import (
	"net/http"
	"path"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
	"github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

func Auth(platformService *platform.Service) func(r *ghttp.Request) {
	return func(r *ghttp.Request) {
		if isAnonymousRequest(r.Method, r.URL.Path) {
			r.Middleware.Next()
			return
		}

		token := ""
		if cookie := r.Cookie.Get(consts.DefaultAdminCookieName); cookie != nil {
			token = cookie.String()
		}
		if token == "" {
			header := strings.TrimSpace(r.Header.Get(consts.AuthorizationHeader))
			if strings.HasPrefix(strings.ToLower(header), "basic ") {
				token = strings.TrimSpace(header[6:])
			} else {
				token = header
			}
		}

		user, err := platformService.ValidateSessionToken(r.Context(), token)
		if err != nil {
			g.Log().Warningf(r.Context(), "auth rejected path=%s err=%v", r.URL.Path, err)
			r.Response.WriteStatusExit(401, g.Map{
				"success": false,
				"message": "unauthorized",
			})
			return
		}
		r.SetCtxVar(consts.CtxKeyCurrentUser, user)
		r.Middleware.Next()
	}
}

func isAnonymousRequest(method, requestPath string) bool {
	requestPath = strings.TrimSpace(requestPath)
	method = strings.ToUpper(strings.TrimSpace(method))

	if requestPath == "/healthz" || requestPath == "/healthz/ready" || requestPath == "/landing" {
		return true
	}
	if strings.HasPrefix(requestPath, "/swagger") ||
		requestPath == "/api.json" ||
		strings.HasPrefix(requestPath, "/mcp-templates/") ||
		strings.HasPrefix(requestPath, "/assets/") ||
		IsFrontendStaticAssetPath(requestPath) {
		return true
	}
	if (method == http.MethodGet || method == http.MethodHead) && IsFrontendPagePath(requestPath) {
		return true
	}
	switch requestPath {
	case "/session/login", "/system/info", "/system/config", "/system/init":
		return true
	default:
		return false
	}
}

func IsFrontendPagePath(requestPath string) bool {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" || requestPath == "/" || requestPath == "/landing" {
		return true
	}

	base := path.Base(requestPath)
	if strings.Contains(base, ".") {
		return false
	}

	if exactAPIPath(requestPath) {
		return false
	}

	switch requestPath {
	case "/login", "/init", "/dashboard":
		return true
	}
	for _, prefix := range []string{
		"/session",
		"/v1",
		"/internal",
		"/healthz",
		"/swagger",
		"/mcp-templates",
		"/assets",
	} {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return false
		}
	}
	return true
}

func IsFrontendStaticAssetPath(requestPath string) bool {
	requestPath = strings.TrimSpace(requestPath)
	if requestPath == "" || requestPath == "/" {
		return false
	}
	base := path.Base(requestPath)
	if !strings.Contains(base, ".") {
		return false
	}
	for _, prefix := range []string{
		"/session/",
		"/user/",
		"/system/",
		"/dashboard/",
		"/v1/",
		"/internal/",
		"/healthz/",
		"/swagger/",
	} {
		if strings.HasPrefix(requestPath, prefix) {
			return false
		}
	}
	return true
}

func exactAPIPath(requestPath string) bool {
	switch requestPath {
	case "/api.json",
		"/user/info",
		"/system/info",
		"/system/config",
		"/system/init",
		"/system/aigateway-config",
		"/dashboard/init",
		"/dashboard/info",
		"/dashboard/configData",
		"/dashboard/native":
		return true
	default:
		return false
	}
}
