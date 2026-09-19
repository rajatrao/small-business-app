package fxmod

import (
	"github.com/rajat/localdiscovery/internal/config"
	"go.uber.org/fx"
)

type Config = config.Config

func LoadConfig() Config { return config.Load() }

var ConfigModule = fx.Module("config",
	fx.Provide(LoadConfig),
)
