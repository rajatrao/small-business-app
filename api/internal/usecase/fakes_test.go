package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

type fakeUsers struct {
	mu    sync.Mutex
	byID  map[string]domain.User
	email map[string]string
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[string]domain.User{}, email: map[string]string{}}
}

func (f *fakeUsers) Create(_ context.Context, u domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.email[u.Email]; ok {
		return domain.ErrConflict
	}
	f.byID[u.ID] = u
	f.email[u.Email] = u.ID
	return nil
}

func (f *fakeUsers) GetByID(_ context.Context, id string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.email[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return f.byID[id], nil
}

func (f *fakeUsers) UpdateRole(_ context.Context, id string, role domain.Role) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.Role = role
	f.byID[id] = u
	return nil
}

type fakeOIDCIdent struct {
	mu   sync.Mutex
	byKey map[string]domain.OIDCIdentity
}

func newFakeOIDCIdent() *fakeOIDCIdent {
	return &fakeOIDCIdent{byKey: map[string]domain.OIDCIdentity{}}
}

func (f *fakeOIDCIdent) GetByIssuerSubject(_ context.Context, issuer, subject string) (domain.OIDCIdentity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byKey[issuer+"|"+subject]
	if !ok {
		return domain.OIDCIdentity{}, domain.ErrNotFound
	}
	return id, nil
}

func (f *fakeOIDCIdent) Create(_ context.Context, ident domain.OIDCIdentity) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byKey[ident.Issuer+"|"+ident.Subject] = ident
	return nil
}

type fakeAffinity struct {
	mu   sync.Mutex
	rows []domain.CategoryAffinity
}

func (f *fakeAffinity) Bump(_ context.Context, userID, guestID, category string, delta float64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, r := range f.rows {
		if r.Category != category {
			continue
		}
		if userID != "" && r.UserID != nil && *r.UserID == userID {
			f.rows[i].Score += delta
			return nil
		}
		if guestID != "" && r.GuestID != nil && *r.GuestID == guestID {
			f.rows[i].Score += delta
			return nil
		}
	}
	row := domain.CategoryAffinity{ID: category + userID + guestID, Category: category, Score: delta}
	if userID != "" {
		row.UserID = &userID
	} else {
		row.GuestID = &guestID
	}
	f.rows = append(f.rows, row)
	return nil
}

func (f *fakeAffinity) List(_ context.Context, userID, guestID string) ([]domain.CategoryAffinity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.CategoryAffinity
	for _, r := range f.rows {
		if userID != "" && r.UserID != nil && *r.UserID == userID {
			out = append(out, r)
		}
		if guestID != "" && r.GuestID != nil && *r.GuestID == guestID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (f *fakeAffinity) MergeGuestIntoUser(_ context.Context, guestID, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, r := range f.rows {
		if r.GuestID != nil && *r.GuestID == guestID {
			uid := userID
			f.rows[i].UserID = &uid
			f.rows[i].GuestID = nil
		}
	}
	return nil
}

type fakeTokens struct{}

func (fakeTokens) Issue(p domain.Principal, _ time.Duration) (string, error) {
	return "tok:" + p.UserID + ":" + string(p.Role), nil
}
func (fakeTokens) Parse(token string) (domain.Principal, error) {
	parts := strings.Split(token, ":")
	if len(parts) != 3 || parts[0] != "tok" {
		return domain.Principal{}, errors.New("bad token")
	}
	return domain.Principal{UserID: parts[1], Role: domain.Role(parts[2])}, nil
}

type unconfiguredOIDC struct{}

func (unconfiguredOIDC) Configured() bool { return false }
func (unconfiguredOIDC) Start(context.Context, ports.OIDCStartParams) (ports.OIDCStartResult, error) {
	return ports.OIDCStartResult{}, domain.ErrOIDCNotConfigured
}
func (unconfiguredOIDC) Exchange(context.Context, string, string, string, string) (ports.OIDCClaims, error) {
	return ports.OIDCClaims{}, domain.ErrOIDCNotConfigured
}

type fakeStores struct {
	mu    sync.Mutex
	items []domain.Store
}

func (f *fakeStores) Create(_ context.Context, s domain.Store) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, s)
	return nil
}
func (f *fakeStores) Update(_ context.Context, s domain.Store) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, x := range f.items {
		if x.ID == s.ID {
			f.items[i] = s
			return nil
		}
	}
	return domain.ErrNotFound
}
func (f *fakeStores) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i, x := range f.items {
		if x.ID == id {
			f.items = append(f.items[:i], f.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}
func (f *fakeStores) GetByID(_ context.Context, id string) (domain.Store, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.items {
		if x.ID == id {
			return x, nil
		}
	}
	return domain.Store{}, domain.ErrNotFound
}
func (f *fakeStores) GetBySlug(_ context.Context, slug string) (domain.Store, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.items {
		if x.Slug == slug {
			return x, nil
		}
	}
	return domain.Store{}, domain.ErrNotFound
}
func (f *fakeStores) ListByCity(_ context.Context, city, category string) ([]domain.Store, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Store
	for _, x := range f.items {
		if !strings.EqualFold(x.City, city) {
			continue
		}
		if category != "" && x.Category != category {
			continue
		}
		out = append(out, x)
	}
	return out, nil
}
func (f *fakeStores) ListByOwner(_ context.Context, ownerID string) ([]domain.Store, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Store
	for _, x := range f.items {
		if x.OwnerID == ownerID {
			out = append(out, x)
		}
	}
	return out, nil
}
func (f *fakeStores) SlugExists(_ context.Context, slug string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.items {
		if x.Slug == slug {
			return true, nil
		}
	}
	return false, nil
}

type fakeProducts struct {
	mu    sync.Mutex
	items []domain.Product
}

func (f *fakeProducts) Create(_ context.Context, p domain.Product) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.items = append(f.items, p)
	return nil
}
func (f *fakeProducts) Update(_ context.Context, p domain.Product) error { return nil }
func (f *fakeProducts) GetByID(_ context.Context, id string) (domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, x := range f.items {
		if x.ID == id {
			return x, nil
		}
	}
	return domain.Product{}, domain.ErrNotFound
}
func (f *fakeProducts) GetBySlug(_ context.Context, slug string) (domain.Product, error) {
	return domain.Product{}, domain.ErrNotFound
}
func (f *fakeProducts) ListByStore(_ context.Context, storeID string) ([]domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Product
	for _, x := range f.items {
		if x.StoreID == storeID {
			out = append(out, x)
		}
	}
	return out, nil
}
func (f *fakeProducts) ListByCity(_ context.Context, city, category string) ([]domain.Product, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.Product{}, f.items...), nil
}
func (f *fakeProducts) SlugExists(_ context.Context, slug string) (bool, error) {
	return false, nil
}

type fakeVotes struct {
	mu    sync.Mutex
	store map[string]int
	prod  map[string]int
}

func newFakeVotes() *fakeVotes {
	return &fakeVotes{store: map[string]int{}, prod: map[string]int{}}
}
func (f *fakeVotes) Toggle(context.Context, string, *string, *string) (bool, error) { return true, nil }
func (f *fakeVotes) HasVoted(context.Context, string, *string, *string) (bool, error) {
	return false, nil
}
func (f *fakeVotes) CountByStoreIDs(_ context.Context, ids []string, _ *time.Time) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]int{}
	for _, id := range ids {
		out[id] = f.store[id]
	}
	return out, nil
}
func (f *fakeVotes) CountByProductIDs(_ context.Context, ids []string, _ *time.Time) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string]int{}
	for _, id := range ids {
		out[id] = f.prod[id]
	}
	return out, nil
}

type fakeReviews struct {
	mu     sync.Mutex
	stats  map[string]domain.ReviewStats
	recent []domain.Review
	pstats map[string]domain.ReviewStats
}

func newFakeReviews() *fakeReviews {
	return &fakeReviews{stats: map[string]domain.ReviewStats{}, pstats: map[string]domain.ReviewStats{}}
}
func (f *fakeReviews) Create(context.Context, domain.Review) error { return nil }
func (f *fakeReviews) ListByStore(context.Context, string) ([]domain.Review, error) {
	return nil, nil
}
func (f *fakeReviews) ListByProduct(context.Context, string) ([]domain.Review, error) {
	return nil, nil
}
func (f *fakeReviews) StatsByStoreIDs(_ context.Context, ids []string, _ *time.Time) (map[string]domain.ReviewStats, error) {
	out := map[string]domain.ReviewStats{}
	for _, id := range ids {
		out[id] = f.stats[id]
	}
	return out, nil
}
func (f *fakeReviews) StatsByProductIDs(_ context.Context, ids []string, _ *time.Time) (map[string]domain.ReviewStats, error) {
	out := map[string]domain.ReviewStats{}
	for _, id := range ids {
		out[id] = f.pstats[id]
	}
	return out, nil
}
func (f *fakeReviews) ListRecent(context.Context, string, string, *time.Time, int) ([]domain.Review, error) {
	return f.recent, nil
}

// silence unused io in this file (blob tests live elsewhere)
var _ io.Reader = strings.NewReader("")
