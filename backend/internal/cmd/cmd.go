package cmd

import (
	"context"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gcmd"

	dashboardcontroller "github.com/wooveep/aigateway-console/backend/internal/controller/dashboard"
	gatewaycontroller "github.com/wooveep/aigateway-console/backend/internal/controller/gateway"
	"github.com/wooveep/aigateway-console/backend/internal/controller/healthz"
	jobscontroller "github.com/wooveep/aigateway-console/backend/internal/controller/jobs"
	portalcontroller "github.com/wooveep/aigateway-console/backend/internal/controller/portal"
	sessioncontroller "github.com/wooveep/aigateway-console/backend/internal/controller/session"
	"github.com/wooveep/aigateway-console/backend/internal/controller/system"
	usercontroller "github.com/wooveep/aigateway-console/backend/internal/controller/user"
	"github.com/wooveep/aigateway-console/backend/internal/middleware"
	gatewaysvc "github.com/wooveep/aigateway-console/backend/internal/service/gateway"
	jobssvc "github.com/wooveep/aigateway-console/backend/internal/service/jobs"
	platformsvc "github.com/wooveep/aigateway-console/backend/internal/service/platform"
	portalsvc "github.com/wooveep/aigateway-console/backend/internal/service/portal"
	grafanaclient "github.com/wooveep/aigateway-console/backend/utility/clients/grafana"
	k8sclient "github.com/wooveep/aigateway-console/backend/utility/clients/k8s"
	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			deps := loadRuntimeDependencies(ctx)
			k8sService := k8sclient.New(deps.K8s)
			grafanaService := grafanaclient.New(deps.Grafana)
			portalService := portaldbclient.New(deps.Portal)

			platformService := platformsvc.New(k8sService, grafanaService, portalService)
			portalDomainService := portalsvc.New(portalService, k8sService)
			gatewayService := gatewaysvc.New(k8sService, portalDomainService)
			jobsService := jobssvc.New(portalService, portalDomainService, gatewayService, k8sService)
			portalDomainService.SetHook(jobsService)
			if err := jobsService.Start(ctx); err != nil {
				return err
			}
			s := g.Server()
			s.Use(middleware.Trace, middleware.AccessLog, middleware.Auth(platformService))
			s.AddStaticPath("/assets", "resource/public/html/assets")
			s.AddStaticPath("/mcp-templates", "resource/public/html/mcp-templates")
			s.BindHandler("/", serveFrontendApp)
			s.BindHandler("/landing", serveFrontendApp)
			s.BindStatusHandler(http.StatusForbidden, serveFrontendStatusFallback)
			s.BindStatusHandler(http.StatusNotFound, serveFrontendStatusFallback)
			s.Group("/", func(group *ghttp.RouterGroup) {
				group.Middleware(ghttp.MiddlewareHandlerResponse)
				group.Bind(
					healthz.NewV1(platformService),
					system.NewV1(platformService),
				)
				sessioncontroller.Bind(group, platformService)
				usercontroller.Bind(group, platformService)
				system.BindHTTP(group, platformService)
				dashboardcontroller.Bind(group, platformService)
				gatewaycontroller.Bind(group, gatewayService)
				portalcontroller.Bind(group, portalDomainService)
				jobscontroller.Bind(group, jobsService)
				group.GET("/healthz/ready", func(r *ghttp.Request) {
					r.Response.WriteJsonExit(g.Map{"success": true, "data": "ok"})
				})
			})
			s.Run()
			return nil
		},
	}
)

const (
	frontendPublicDir = "resource/public/html"
	frontendEntryFile = frontendPublicDir + "/index.html"
)

func serveFrontendApp(r *ghttp.Request) {
	r.Response.ServeFile(frontendEntryFile)
}

func serveFrontendStatusFallback(r *ghttp.Request) {
	if assetFile, ok := resolveFrontendPublicFile(r.URL.Path); ok {
		r.Response.ClearBuffer()
		r.Response.ServeFile(assetFile)
		return
	}
	if middleware.IsFrontendPagePath(r.URL.Path) {
		r.Response.ClearBuffer()
		r.Response.ServeFile(frontendEntryFile)
		return
	}
	r.Response.WriteStatus(r.Response.Status)
}

func resolveFrontendPublicFile(requestPath string) (string, bool) {
	requestPath = strings.TrimSpace(requestPath)
	if !middleware.IsFrontendStaticAssetPath(requestPath) {
		return "", false
	}

	cleanPath := path.Clean("/" + requestPath)
	if cleanPath == "/" {
		return "", false
	}

	relativePath := strings.TrimPrefix(cleanPath, "/")
	if relativePath == "" {
		return "", false
	}

	candidates := []string{
		filepath.Join(frontendPublicDir, filepath.FromSlash(relativePath)),
		filepath.Join("..", "..", frontendPublicDir, filepath.FromSlash(relativePath)),
	}
	for _, filePath := range candidates {
		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			return filePath, true
		}
	}
	return "", false
}
