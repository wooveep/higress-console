package system

import platformsvc "github.com/wooveep/aigateway-console/backend/internal/service/platform"

type ControllerV1 struct {
	platformService *platformsvc.Service
}
