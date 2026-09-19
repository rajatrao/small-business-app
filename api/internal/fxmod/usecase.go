package fxmod

import (
	"context"
	"errors"

	"github.com/rajat/localdiscovery/internal/adapter/jwt"
	openaiadp "github.com/rajat/localdiscovery/internal/adapter/openai"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
	"github.com/rajat/localdiscovery/internal/usecase"
	"go.uber.org/fx"
)

func NewClock() ports.Clock { return domain.SystemClock{} }

func NewTokens(cfg Config) ports.TokenIssuer {
	return jwt.NewIssuer(cfg.JWTSecret)
}

type disabledLLM struct{}

func (disabledLLM) Complete(context.Context, string, string) (string, error) {
	return "", errors.New("llm disabled")
}

func NewLLM(cfg Config) ports.LLMClient {
	if c := openaiadp.NewOpenAI(cfg.OpenAIKey); c != nil {
		return c
	}
	return disabledLLM{}
}

func NewAuth(
	users ports.UserRepository,
	oids ports.OIDCIdentityRepository,
	aff ports.AffinityRepository,
	tokens ports.TokenIssuer,
	oidc ports.OIDCClient,
	clock ports.Clock,
	cfg Config,
) *usecase.AuthService {
	return usecase.NewAuthService(users, oids, aff, tokens, oidc, clock, cfg.CORSOrigin)
}

var UsecaseModule = fx.Module("usecase",
	fx.Provide(
		NewClock,
		NewTokens,
		NewLLM,
		NewAuth,
		usecase.NewHomeService,
		usecase.NewSearchService,
		usecase.NewListingService,
		usecase.NewReviewService,
		usecase.NewVoteService,
		usecase.NewAffinityService,
		usecase.NewChatService,
	),
)
