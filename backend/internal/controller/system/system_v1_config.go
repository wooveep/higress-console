package system

import (
	"context"

	v1 "github.com/wooveep/aigateway-console/backend/api/system/v1"
)

func (c *ControllerV1) Config(ctx context.Context, req *v1.ConfigReq) (res *v1.ConfigRes, err error) {
	config, err := c.platformService.SystemConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ConfigRes{
		Module:                config.Module,
		ServerAddress:         config.ServerAddress,
		ExplicitRenameTargets: config.ExplicitRenameTargets,
		ContractDirectories:   config.ContractDirectories,
		Properties:            config.Properties,
	}, nil
}
