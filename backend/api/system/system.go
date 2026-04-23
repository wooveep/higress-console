package system

import (
	"context"

	v1 "github.com/wooveep/aigateway-console/backend/api/system/v1"
)

type ISystemV1 interface {
	Info(ctx context.Context, req *v1.InfoReq) (res *v1.InfoRes, err error)
	Config(ctx context.Context, req *v1.ConfigReq) (res *v1.ConfigRes, err error)
}
