package fxmod

import (
	gqladapter "github.com/rajat/localdiscovery/internal/adapter/graphql"
	"github.com/rajat/localdiscovery/internal/ports"
	"github.com/rajat/localdiscovery/internal/usecase"
	"go.uber.org/fx"
)

func NewGQLResolver(
	auth *usecase.AuthService,
	home *usecase.HomeService,
	search *usecase.SearchService,
	listing *usecase.ListingService,
	review *usecase.ReviewService,
	vote *usecase.VoteService,
	affinity *usecase.AffinityService,
	chat *usecase.ChatService,
	blobs ports.BlobStore,
	label OIDCLabel,
) *gqladapter.Resolver {
	r := gqladapter.NewResolver(auth, home, search, listing, review, vote, affinity, chat, blobs)
	r.SetOIDCLabel(string(label))
	return r
}

var GraphQLModule = fx.Module("graphql",
	fx.Provide(NewGQLResolver),
)
