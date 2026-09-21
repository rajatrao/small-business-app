package main

import (
	"github.com/rajat/localdiscovery/internal/fxmod"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.NopLogger,
		fxmod.ConfigModule,
		fxmod.PostgresModule,
		fxmod.OIDCModule,
		fxmod.BlobModule,
		fxmod.UsecaseModule,
		fxmod.GraphQLModule,
		fxmod.HTTPModule,
	).Run()
}
