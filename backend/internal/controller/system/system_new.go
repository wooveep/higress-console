package system

import (
	"github.com/wooveep/aigateway-console/backend/api/system"
	platformsvc "github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

func NewV1(platformService *platformsvc.Service) system.ISystemV1 {
	return &ControllerV1{
		platformService: platformService,
	}
}
