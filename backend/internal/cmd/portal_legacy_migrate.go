package cmd

import (
	"context"
	"github.com/gogf/gf/v2/os/gcmd"

	portaldbclient "github.com/wooveep/aigateway-console/backend/utility/clients/portaldb"
)

var PortalLegacyMigrate = &gcmd.Command{
	Name:  "portal-legacy-migrate",
	Usage: "portal-legacy-migrate",
	Brief: "migrate legacy console portal tables into shared portal schema",
	Func: func(ctx context.Context, parser *gcmd.Parser) error {
		portalConfig := loadRuntimeDependencies(ctx).Portal
		portalConfig.AutoMigrate = false
		client := portaldbclient.New(portalConfig)
		if err := client.Healthy(ctx); err != nil {
			return err
		}
		return client.MigrateLegacyData(ctx)
	},
}

func init() {
	_ = Main.AddCommand(PortalLegacyMigrate)
}
