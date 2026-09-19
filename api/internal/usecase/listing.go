package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type ListingService struct {
	stores   ports.StoreRepository
	products ports.ProductRepository
	photos   ports.PhotoRepository
	votes    ports.VoteRepository
	reviews  ports.ReviewRepository
	clock    ports.Clock
}

func NewListingService(
	stores ports.StoreRepository,
	products ports.ProductRepository,
	photos ports.PhotoRepository,
	votes ports.VoteRepository,
	reviews ports.ReviewRepository,
	clock ports.Clock,
) *ListingService {
	return &ListingService{
		stores: stores, products: products, photos: photos,
		votes: votes, reviews: reviews, clock: clock,
	}
}

type StoreDetail struct {
	Store         domain.Store
	Photos        []domain.Photo
	Products      []domain.Product
	ProductPhotos map[string][]domain.Photo
	Reviews       []domain.Review
	Upvotes       int
	ViewerVoted   bool
}

type ProductDetail struct {
	Product     domain.Product
	Store       domain.Store
	Photos      []domain.Photo
	Reviews     []domain.Review
	Upvotes     int
	ViewerVoted bool
}

func (s *ListingService) StoreBySlug(ctx context.Context, slug string, principal *domain.Principal) (StoreDetail, error) {
	st, err := s.stores.GetBySlug(ctx, slug)
	if err != nil {
		return StoreDetail{}, err
	}
	return s.hydrateStore(ctx, st, principal)
}

func (s *ListingService) StoreByID(ctx context.Context, id string, principal *domain.Principal) (StoreDetail, error) {
	st, err := s.stores.GetByID(ctx, id)
	if err != nil {
		return StoreDetail{}, err
	}
	return s.hydrateStore(ctx, st, principal)
}

func (s *ListingService) hydrateStore(ctx context.Context, st domain.Store, principal *domain.Principal) (StoreDetail, error) {
	photos, err := s.photos.ListByStoreIDs(ctx, []string{st.ID})
	if err != nil {
		return StoreDetail{}, err
	}
	products, err := s.products.ListByStore(ctx, st.ID)
	if err != nil {
		return StoreDetail{}, err
	}
	pIDs := make([]string, len(products))
	for i, p := range products {
		pIDs[i] = p.ID
	}
	pphotos, err := s.photos.ListByProductIDs(ctx, pIDs)
	if err != nil {
		return StoreDetail{}, err
	}
	reviews, err := s.reviews.ListByStore(ctx, st.ID)
	if err != nil {
		return StoreDetail{}, err
	}
	up, err := s.votes.CountByStoreIDs(ctx, []string{st.ID}, nil)
	if err != nil {
		return StoreDetail{}, err
	}
	voted := false
	if principal != nil {
		voted, _ = s.votes.HasVoted(ctx, principal.UserID, &st.ID, nil)
	}
	return StoreDetail{
		Store:         st,
		Photos:        photos[st.ID],
		Products:      products,
		ProductPhotos: pphotos,
		Reviews:       reviews,
		Upvotes:       up[st.ID],
		ViewerVoted:   voted,
	}, nil
}

func (s *ListingService) ProductBySlug(ctx context.Context, slug string, principal *domain.Principal) (ProductDetail, error) {
	p, err := s.products.GetBySlug(ctx, slug)
	if err != nil {
		return ProductDetail{}, err
	}
	st, err := s.stores.GetByID(ctx, p.StoreID)
	if err != nil {
		return ProductDetail{}, err
	}
	photos, err := s.photos.ListByProductIDs(ctx, []string{p.ID})
	if err != nil {
		return ProductDetail{}, err
	}
	reviews, err := s.reviews.ListByProduct(ctx, p.ID)
	if err != nil {
		return ProductDetail{}, err
	}
	up, err := s.votes.CountByProductIDs(ctx, []string{p.ID}, nil)
	if err != nil {
		return ProductDetail{}, err
	}
	voted := false
	if principal != nil {
		voted, _ = s.votes.HasVoted(ctx, principal.UserID, nil, &p.ID)
	}
	return ProductDetail{
		Product:     p,
		Store:       st,
		Photos:      photos[p.ID],
		Reviews:     reviews,
		Upvotes:     up[p.ID],
		ViewerVoted: voted,
	}, nil
}

func (s *ListingService) PhotosForStores(ctx context.Context, ids []string) (map[string][]domain.Photo, error) {
	return s.photos.ListByStoreIDs(ctx, ids)
}

func (s *ListingService) PhotosForProducts(ctx context.Context, ids []string) (map[string][]domain.Photo, error) {
	return s.photos.ListByProductIDs(ctx, ids)
}

func (s *ListingService) OwnerStores(ctx context.Context, principal *domain.Principal) ([]domain.Store, error) {
	if principal == nil {
		return nil, domain.ErrUnauthorized
	}
	if principal.Role != domain.RoleOwner && principal.Role != domain.RoleAdmin {
		return nil, domain.ErrForbidden
	}
	return s.stores.ListByOwner(ctx, principal.UserID)
}

type StoreInput struct {
	Name, Description, Category, City string
	Phone, Address                    *string
}

func (s *ListingService) CreateStore(ctx context.Context, principal *domain.Principal, in StoreInput) (domain.Store, error) {
	if principal == nil {
		return domain.Store{}, domain.ErrUnauthorized
	}
	if principal.Role != domain.RoleOwner && principal.Role != domain.RoleAdmin {
		return domain.Store{}, domain.ErrForbidden
	}
	if err := validateStoreInput(in, true); err != nil {
		return domain.Store{}, err
	}
	slug, err := s.uniqueStoreSlug(ctx, in.Name)
	if err != nil {
		return domain.Store{}, err
	}
	st := domain.Store{
		ID:          uuid.NewString(),
		OwnerID:     principal.UserID,
		Slug:        slug,
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		Category:    strings.ToLower(in.Category),
		City:        strings.TrimSpace(in.City),
		Phone:       emptyToNil(in.Phone),
		Address:     emptyToNil(in.Address),
		CreatedAt:   s.clock.Now(),
	}
	if err := s.stores.Create(ctx, st); err != nil {
		return domain.Store{}, err
	}
	return st, nil
}

func (s *ListingService) UpdateStore(ctx context.Context, principal *domain.Principal, id string, in StoreInput) (domain.Store, error) {
	st, err := s.requireOwnerStore(ctx, principal, id)
	if err != nil {
		return domain.Store{}, err
	}
	if strings.TrimSpace(in.Name) != "" {
		st.Name = strings.TrimSpace(in.Name)
	}
	if in.Description != "" {
		st.Description = strings.TrimSpace(in.Description)
	}
	if in.Category != "" {
		if !domain.ValidCategory(in.Category) {
			return domain.Store{}, domain.ErrInvalid
		}
		st.Category = strings.ToLower(in.Category)
	}
	if strings.TrimSpace(in.City) != "" {
		st.City = strings.TrimSpace(in.City)
	}
	if in.Phone != nil {
		st.Phone = emptyToNil(in.Phone)
	}
	if in.Address != nil {
		st.Address = emptyToNil(in.Address)
	}
	if err := s.stores.Update(ctx, st); err != nil {
		return domain.Store{}, err
	}
	return st, nil
}

func (s *ListingService) DeleteStore(ctx context.Context, principal *domain.Principal, id string) error {
	if _, err := s.requireOwnerStore(ctx, principal, id); err != nil {
		return err
	}
	return s.stores.Delete(ctx, id)
}

type ProductInput struct {
	Name, Description string
	PriceCents        int
}

func (s *ListingService) CreateProduct(ctx context.Context, principal *domain.Principal, storeID string, in ProductInput) (domain.Product, error) {
	if _, err := s.requireOwnerStore(ctx, principal, storeID); err != nil {
		return domain.Product{}, err
	}
	if strings.TrimSpace(in.Name) == "" || in.PriceCents < 0 {
		return domain.Product{}, domain.ErrInvalid
	}
	slug, err := s.uniqueProductSlug(ctx, in.Name)
	if err != nil {
		return domain.Product{}, err
	}
	p := domain.Product{
		ID:          uuid.NewString(),
		StoreID:     storeID,
		Slug:        slug,
		Name:        strings.TrimSpace(in.Name),
		Description: strings.TrimSpace(in.Description),
		PriceCents:  in.PriceCents,
		CreatedAt:   s.clock.Now(),
	}
	if err := s.products.Create(ctx, p); err != nil {
		return domain.Product{}, err
	}
	return p, nil
}

func (s *ListingService) UpdateProduct(ctx context.Context, principal *domain.Principal, id string, in ProductInput) (domain.Product, error) {
	p, err := s.products.GetByID(ctx, id)
	if err != nil {
		return domain.Product{}, err
	}
	if _, err := s.requireOwnerStore(ctx, principal, p.StoreID); err != nil {
		return domain.Product{}, err
	}
	if strings.TrimSpace(in.Name) != "" {
		p.Name = strings.TrimSpace(in.Name)
	}
	if in.Description != "" {
		p.Description = strings.TrimSpace(in.Description)
	}
	if in.PriceCents >= 0 {
		p.PriceCents = in.PriceCents
	}
	if err := s.products.Update(ctx, p); err != nil {
		return domain.Product{}, err
	}
	return p, nil
}

func (s *ListingService) AddPhoto(ctx context.Context, principal *domain.Principal, storeID, productID *string, url string) (domain.Photo, error) {
	if url == "" {
		return domain.Photo{}, domain.ErrInvalid
	}
	var sid, pid *string
	if productID != nil && *productID != "" {
		p, err := s.products.GetByID(ctx, *productID)
		if err != nil {
			return domain.Photo{}, err
		}
		if _, err := s.requireOwnerStore(ctx, principal, p.StoreID); err != nil {
			return domain.Photo{}, err
		}
		pid = &p.ID
		sidCopy := p.StoreID
		sid = &sidCopy
	} else if storeID != nil && *storeID != "" {
		if _, err := s.requireOwnerStore(ctx, principal, *storeID); err != nil {
			return domain.Photo{}, err
		}
		sid = storeID
	} else {
		return domain.Photo{}, domain.ErrInvalid
	}
	ph := domain.Photo{
		ID:        uuid.NewString(),
		StoreID:   sid,
		ProductID: pid,
		URL:       url,
		SortOrder: 0,
		CreatedAt: s.clock.Now(),
	}
	if err := s.photos.Create(ctx, ph); err != nil {
		return domain.Photo{}, err
	}
	return ph, nil
}

func (s *ListingService) requireOwnerStore(ctx context.Context, principal *domain.Principal, storeID string) (domain.Store, error) {
	if principal == nil {
		return domain.Store{}, domain.ErrUnauthorized
	}
	st, err := s.stores.GetByID(ctx, storeID)
	if err != nil {
		return domain.Store{}, err
	}
	if principal.Role == domain.RoleAdmin {
		return st, nil
	}
	if principal.Role != domain.RoleOwner || st.OwnerID != principal.UserID {
		return domain.Store{}, domain.ErrForbidden
	}
	return st, nil
}

func (s *ListingService) uniqueStoreSlug(ctx context.Context, name string) (string, error) {
	base := domain.Slugify(name)
	slug := base
	for i := 0; i < 8; i++ {
		exists, err := s.stores.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%s", base, uuid.NewString()[:6])
	}
	return base + "-" + uuid.NewString()[:8], nil
}

func (s *ListingService) uniqueProductSlug(ctx context.Context, name string) (string, error) {
	base := domain.Slugify(name)
	slug := base
	for i := 0; i < 8; i++ {
		exists, err := s.products.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%s", base, uuid.NewString()[:6])
	}
	return base + "-" + uuid.NewString()[:8], nil
}

func validateStoreInput(in StoreInput, requireAll bool) error {
	if requireAll {
		if strings.TrimSpace(in.Name) == "" || strings.TrimSpace(in.City) == "" {
			return domain.ErrInvalid
		}
		if !domain.ValidCategory(in.Category) {
			return domain.ErrInvalid
		}
	}
	return nil
}

func emptyToNil(p *string) *string {
	if p == nil {
		return nil
	}
	v := strings.TrimSpace(*p)
	if v == "" {
		return nil
	}
	return &v
}

type VoteService struct {
	votes    ports.VoteRepository
	stores   ports.StoreRepository
	products ports.ProductRepository
	affinity ports.AffinityRepository
}

func NewVoteService(v ports.VoteRepository, s ports.StoreRepository, p ports.ProductRepository, a ports.AffinityRepository) *VoteService {
	return &VoteService{votes: v, stores: s, products: p, affinity: a}
}

func (s *VoteService) Vote(ctx context.Context, principal *domain.Principal, storeID, productID *string) (bool, error) {
	if principal == nil {
		return false, domain.ErrUnauthorized
	}
	if (storeID == nil || *storeID == "") && (productID == nil || *productID == "") {
		return false, domain.ErrInvalid
	}
	var cat string
	if productID != nil && *productID != "" {
		p, err := s.products.GetByID(ctx, *productID)
		if err != nil {
			return false, err
		}
		st, err := s.stores.GetByID(ctx, p.StoreID)
		if err != nil {
			return false, err
		}
		cat = st.Category
		storeID = nil
	} else {
		st, err := s.stores.GetByID(ctx, *storeID)
		if err != nil {
			return false, err
		}
		cat = st.Category
		productID = nil
	}
	voted, err := s.votes.Toggle(ctx, principal.UserID, storeID, productID)
	if err != nil {
		return false, err
	}
	if voted && cat != "" {
		_ = s.affinity.Bump(ctx, principal.UserID, "", cat, domain.AffinityVote.Weight())
	}
	return voted, nil
}

type ReviewService struct {
	reviews  ports.ReviewRepository
	stores   ports.StoreRepository
	products ports.ProductRepository
	affinity ports.AffinityRepository
	clock    ports.Clock
}

func NewReviewService(
	r ports.ReviewRepository,
	s ports.StoreRepository,
	p ports.ProductRepository,
	a ports.AffinityRepository,
	c ports.Clock,
) *ReviewService {
	return &ReviewService{reviews: r, stores: s, products: p, affinity: a, clock: c}
}

func (s *ReviewService) Create(ctx context.Context, principal *domain.Principal, storeID, productID *string, rating int, body string) (domain.Review, error) {
	if principal == nil {
		return domain.Review{}, domain.ErrUnauthorized
	}
	body = strings.TrimSpace(body)
	if rating < 1 || rating > 5 || body == "" {
		return domain.Review{}, domain.ErrInvalid
	}
	var cat string
	rv := domain.Review{
		ID:        uuid.NewString(),
		UserID:    principal.UserID,
		Rating:    rating,
		Body:      body,
		CreatedAt: s.clock.Now(),
		AuthorName: principal.Email,
	}
	if productID != nil && *productID != "" {
		p, err := s.products.GetByID(ctx, *productID)
		if err != nil {
			return domain.Review{}, err
		}
		st, err := s.stores.GetByID(ctx, p.StoreID)
		if err != nil {
			return domain.Review{}, err
		}
		cat = st.Category
		id := p.ID
		rv.ProductID = &id
		rv.ProductName = p.Name
		rv.ProductSlug = p.Slug
		rv.StoreName = st.Name
		rv.StoreSlug = st.Slug
	} else if storeID != nil && *storeID != "" {
		st, err := s.stores.GetByID(ctx, *storeID)
		if err != nil {
			return domain.Review{}, err
		}
		cat = st.Category
		id := st.ID
		rv.StoreID = &id
		rv.StoreName = st.Name
		rv.StoreSlug = st.Slug
	} else {
		return domain.Review{}, domain.ErrInvalid
	}
	if err := s.reviews.Create(ctx, rv); err != nil {
		return domain.Review{}, err
	}
	if cat != "" {
		_ = s.affinity.Bump(ctx, principal.UserID, "", cat, domain.AffinityReview.Weight())
	}
	uName := strings.Split(principal.Email, "@")[0]
	rv.AuthorName = uName
	return rv, nil
}
