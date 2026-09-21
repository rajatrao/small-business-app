package ports

import (
	"context"
	"io"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
)

type Clock interface {
	Now() time.Time
}

type UserRepository interface {
	Create(ctx context.Context, u domain.User) error
	GetByID(ctx context.Context, id string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	UpdateRole(ctx context.Context, id string, role domain.Role) error
}

type OIDCIdentityRepository interface {
	GetByIssuerSubject(ctx context.Context, issuer, subject string) (domain.OIDCIdentity, error)
	Create(ctx context.Context, ident domain.OIDCIdentity) error
}

type StoreRepository interface {
	Create(ctx context.Context, s domain.Store) error
	Update(ctx context.Context, s domain.Store) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (domain.Store, error)
	GetBySlug(ctx context.Context, slug string) (domain.Store, error)
	ListByCity(ctx context.Context, city, category string) ([]domain.Store, error)
	ListByOwner(ctx context.Context, ownerID string) ([]domain.Store, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
}

type ProductRepository interface {
	Create(ctx context.Context, p domain.Product) error
	Update(ctx context.Context, p domain.Product) error
	GetByID(ctx context.Context, id string) (domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (domain.Product, error)
	ListByStore(ctx context.Context, storeID string) ([]domain.Product, error)
	ListByCity(ctx context.Context, city, category string) ([]domain.Product, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
}

type PhotoRepository interface {
	Create(ctx context.Context, p domain.Photo) error
	ListByStoreIDs(ctx context.Context, ids []string) (map[string][]domain.Photo, error)
	ListByProductIDs(ctx context.Context, ids []string) (map[string][]domain.Photo, error)
}

type VoteRepository interface {
	Toggle(ctx context.Context, userID string, storeID, productID *string) (voted bool, err error)
	HasVoted(ctx context.Context, userID string, storeID, productID *string) (bool, error)
	CountByStoreIDs(ctx context.Context, ids []string, since *time.Time) (map[string]int, error)
	CountByProductIDs(ctx context.Context, ids []string, since *time.Time) (map[string]int, error)
}

type ReviewRepository interface {
	Create(ctx context.Context, r domain.Review) error
	ListByStore(ctx context.Context, storeID string) ([]domain.Review, error)
	ListByProduct(ctx context.Context, productID string) ([]domain.Review, error)
	StatsByStoreIDs(ctx context.Context, ids []string, since *time.Time) (map[string]domain.ReviewStats, error)
	StatsByProductIDs(ctx context.Context, ids []string, since *time.Time) (map[string]domain.ReviewStats, error)
	ListRecent(ctx context.Context, city, category string, since *time.Time, limit int) ([]domain.Review, error)
}

type AffinityRepository interface {
	Bump(ctx context.Context, userID, guestID, category string, delta float64) error
	List(ctx context.Context, userID, guestID string) ([]domain.CategoryAffinity, error)
	MergeGuestIntoUser(ctx context.Context, guestID, userID string) error
}

type ChatRepository interface {
	CreateThread(ctx context.Context, t domain.ChatThread) error
	GetThread(ctx context.Context, id string) (domain.ChatThread, error)
	AddMessage(ctx context.Context, m domain.ChatMessage) error
	ListMessages(ctx context.Context, threadID string, limit int) ([]domain.ChatMessage, error)
}

type SearchRepository interface {
	SearchStores(ctx context.Context, q, city string, limit int) ([]domain.Store, error)
	SearchProducts(ctx context.Context, q, city string, limit int) ([]domain.Product, error)
}

type BlobStore interface {
	Put(ctx context.Context, key string, r io.Reader, contentType string) (url string, err error)
	Get(ctx context.Context, key string) (io.ReadCloser, string, error)
}

type TokenIssuer interface {
	Issue(p domain.Principal, ttl time.Duration) (string, error)
	Parse(token string) (domain.Principal, error)
}

type OIDCStartParams struct {
	Role        domain.Role
	Next        string
	RedirectURI string
}

type OIDCStartResult struct {
	AuthURL      string
	State        string
	Nonce        string
	CodeVerifier string
}

type OIDCClaims struct {
	Issuer          string
	Subject         string
	Email           string
	EmailVerified   bool
	Name            string
	Nonce           string
}

type OIDCClient interface {
	Configured() bool
	Start(ctx context.Context, p OIDCStartParams) (OIDCStartResult, error)
	Exchange(ctx context.Context, code, codeVerifier, redirectURI, nonce string) (OIDCClaims, error)
}

type LLMClient interface {
	Complete(ctx context.Context, system, user string) (string, error)
}
