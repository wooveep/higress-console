package middleware

import (
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
)

func AccessLog(r *ghttp.Request) {
	startedAt := time.Now()
	r.Middleware.Next()

	requestID := r.GetCtxVar(consts.CtxKeyRequestID).String()
	g.Log().Infof(
		r.Context(),
		"request_id=%s method=%s path=%s status=%d duration=%s",
		requestID,
		r.Method,
		r.URL.Path,
		r.Response.Status,
		time.Since(startedAt),
	)
}
