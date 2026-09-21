package fxmod

import (
	"github.com/rajat/localdiscovery/internal/adapter/oidc"
	"github.com/rajat/localdiscovery/internal/ports"
	"go.uber.org/fx"
)

type OIDCLabel string

func NewOIDC(cfg Config) ports.OIDCClient {
	return oidc.NewClient(oidc.Config{
		Issuer:       cfg.OIDCIssuer,
		ClientID:     cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret,
		RedirectURI:  cfg.OIDCRedirectURI,
	})
}

func oidcLabel(cfg Config) OIDCLabel {
	if cfg.OIDCLabel != "" {
		return OIDCLabel(cfg.OIDCLabel)
	}
	return "Continue with OpenID"
}

var OIDCModule = fx.Module("oidc",
	fx.Provide(NewOIDC, oidcLabel),
)
