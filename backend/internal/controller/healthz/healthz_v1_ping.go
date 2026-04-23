package healthz

import (
	"context"

	v1 "github.com/wooveep/aigateway-console/backend/api/healthz/v1"
)

func (c *ControllerV1) Ping(ctx context.Context, req *v1.PingReq) (res *v1.PingRes, err error) {
	status, err := c.platformService.Healthz(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.PingRes{
		Status:        status.Status,
		Service:       status.Service,
		Phase:         status.Phase,
		LegacyBackend: status.LegacyBackend,
	}, nil
}
