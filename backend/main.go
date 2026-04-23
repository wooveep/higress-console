package main

import (
	_ "github.com/wooveep/aigateway-console/backend/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"

	"github.com/wooveep/aigateway-console/backend/internal/cmd"
)

func main() {
	cmd.Main.Run(gctx.GetInitCtx())
}
