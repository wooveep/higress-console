package middleware

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/google/uuid"

	"github.com/wooveep/aigateway-console/backend/internal/consts"
)

func Trace(r *ghttp.Request) {
	requestID := r.Header.Get(consts.RequestIDHeader)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	r.SetCtxVar(consts.CtxKeyRequestID, requestID)
	r.Response.Header().Set(consts.RequestIDHeader, requestID)
	g.Log().Debugf(r.Context(), "request_id=%s method=%s path=%s", requestID, r.Method, r.URL.Path)
	r.Middleware.Next()
}
