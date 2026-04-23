package healthz

import (
	"github.com/wooveep/aigateway-console/backend/api/healthz"
	platformsvc "github.com/wooveep/aigateway-console/backend/internal/service/platform"
)

func NewV1(platformService *platformsvc.Service) healthz.IHealthzV1 {
	return &ControllerV1{
		platformService: platformService,
	}
}
