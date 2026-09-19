package fxmod

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/rajat/localdiscovery/internal/adapter/blob"
	gqladapter "github.com/rajat/localdiscovery/internal/adapter/graphql"
	httpadp "github.com/rajat/localdiscovery/internal/adapter/http"
	"github.com/rajat/localdiscovery/internal/usecase"
	"go.uber.org/fx"
)

func NewHTTPServer(cfg Config, resolver *gqladapter.Resolver, auth *usecase.AuthService, disk *blob.Disk) *http.Server {
	h := httpadp.NewRouter(cfg, resolver, auth, disk)
	return &http.Server{
		Addr:              cfg.APIAddr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}
}

func RegisterHTTP(lc fx.Lifecycle, srv *http.Server) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			go func() { _ = srv.Serve(ln) }()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
}

var HTTPModule = fx.Module("http",
	fx.Provide(NewHTTPServer),
	fx.Invoke(RegisterHTTP),
)
