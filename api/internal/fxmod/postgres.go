package fxmod

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rajat/localdiscovery/internal/adapter/postgres"
	"github.com/rajat/localdiscovery/internal/ports"
	"go.uber.org/fx"
)

func NewPool(lc fx.Lifecycle, cfg Config) (*pgxpool.Pool, error) {
	pcfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	pcfg.MaxConns = 10
	pcfg.MinConns = 0
	pcfg.MaxConnLifetime = time.Hour
	pool, err := pgxpool.NewWithConfig(context.Background(), pcfg)
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pool.Ping(ctx); err != nil {
				return err
			}
			return postgres.Migrate(ctx, pool)
		},
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})
	return pool, nil
}

func provideRepos(pool *pgxpool.Pool) (
	ports.UserRepository,
	ports.OIDCIdentityRepository,
	ports.StoreRepository,
	ports.ProductRepository,
	ports.PhotoRepository,
	ports.VoteRepository,
	ports.ReviewRepository,
	ports.AffinityRepository,
	ports.ChatRepository,
	ports.SearchRepository,
) {
	trgm := postgres.HasTrgm(context.Background(), pool)
	return postgres.NewUserRepo(pool),
		postgres.NewOIDCIdentRepo(pool),
		postgres.NewStoreRepo(pool),
		postgres.NewProductRepo(pool),
		postgres.NewPhotoRepo(pool),
		postgres.NewVoteRepo(pool),
		postgres.NewReviewRepo(pool),
		postgres.NewAffinityRepo(pool),
		postgres.NewChatRepo(pool),
		postgres.NewSearchRepo(pool, trgm)
}

var PostgresModule = fx.Module("postgres",
	fx.Provide(NewPool, provideRepos),
)
