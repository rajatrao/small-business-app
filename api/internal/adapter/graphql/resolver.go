package graphql

import (
	"github.com/rajat/localdiscovery/internal/ports"
	"github.com/rajat/localdiscovery/internal/usecase"
)

// Resolver is wired by Fx. Resolvers only call use cases.
type Resolver struct {
	Auth      *usecase.AuthService
	Homes     *usecase.HomeService
	Searcher  *usecase.SearchService
	Listing   *usecase.ListingService
	Reviews   *usecase.ReviewService
	Votes     *usecase.VoteService
	Affinity  *usecase.AffinityService
	Chats     *usecase.ChatService
	Blob      ports.BlobStore
	OIDCLabel string
}

func NewResolver(
	auth *usecase.AuthService,
	home *usecase.HomeService,
	search *usecase.SearchService,
	listing *usecase.ListingService,
	review *usecase.ReviewService,
	vote *usecase.VoteService,
	affinity *usecase.AffinityService,
	chat *usecase.ChatService,
	blobs ports.BlobStore,
) *Resolver {
	return &Resolver{
		Auth: auth, Homes: home, Searcher: search, Listing: listing,
		Reviews: review, Votes: vote, Affinity: affinity, Chats: chat,
		Blob: blobs,
	}
}

func (r *Resolver) SetOIDCLabel(label string) {
	r.OIDCLabel = label
}
