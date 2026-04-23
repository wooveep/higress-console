package system

import (
	"context"

	v1 "github.com/wooveep/aigateway-console/backend/api/system/v1"
)

func (c *ControllerV1) Info(ctx context.Context, req *v1.InfoReq) (res *v1.InfoRes, err error) {
	info, err := c.platformService.SystemInfo(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.InfoRes{
		Service:         info.Service,
		Version:         info.Version,
		Phase:           info.Phase,
		PreferredNaming: info.PreferredNaming,
		LegacyNaming:    info.LegacyNaming,
		LegacyBackend:   info.LegacyBackend,
		BusinessLines:   info.BusinessLines,
	}, nil
}
