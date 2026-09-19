package graphql

import (
	"errors"
	"strings"
	"time"

	"github.com/rajat/localdiscovery/internal/adapter/graphql/model"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/usecase"
)

func mapUser(u domain.User) *model.User {
	return &model.User{
		ID:    u.ID,
		Email: u.Email,
		Name:  u.Name,
		Role:  model.Role(u.Role),
	}
}

func mapPhoto(p domain.Photo) *model.Photo {
	return &model.Photo{ID: p.ID, URL: p.URL, SortOrder: p.SortOrder}
}

func mapPhotos(ps []domain.Photo) []*model.Photo {
	out := make([]*model.Photo, 0, len(ps))
	for _, p := range ps {
		out = append(out, mapPhoto(p))
	}
	return out
}

func mapStore(s domain.Store, photos []domain.Photo, products []*model.Product, reviews []*model.Review, upvotes int, voted bool, score float64) *model.Store {
	if products == nil {
		products = []*model.Product{}
	}
	if reviews == nil {
		reviews = []*model.Review{}
	}
	return &model.Store{
		ID:             s.ID,
		Slug:           s.Slug,
		Name:           s.Name,
		Description:    s.Description,
		Category:       s.Category,
		City:           s.City,
		Phone:          s.Phone,
		Address:        s.Address,
		Photos:         mapPhotos(photos),
		Products:       products,
		Reviews:        reviews,
		UpvoteCount:    upvotes,
		CommunityScore: score,
		ViewerHasVoted: voted,
		CreatedAt:      s.CreatedAt,
	}
}

func mapStoreLite(s domain.Store, photos []domain.Photo) *model.Store {
	return mapStore(s, photos, nil, nil, 0, false, 0)
}

func mapProduct(p domain.Product, store *model.Store, photos []domain.Photo, reviews []*model.Review, upvotes int, voted bool) *model.Product {
	if reviews == nil {
		reviews = []*model.Review{}
	}
	if store == nil {
		store = &model.Store{Photos: []*model.Photo{}, Products: []*model.Product{}, Reviews: []*model.Review{}}
	}
	return &model.Product{
		ID:             p.ID,
		Slug:           p.Slug,
		Name:           p.Name,
		Description:    p.Description,
		PriceCents:     p.PriceCents,
		DisplayPrice:   domain.DisplayPrice(p.PriceCents),
		Store:          store,
		Photos:         mapPhotos(photos),
		Reviews:        reviews,
		UpvoteCount:    upvotes,
		ViewerHasVoted: voted,
		CreatedAt:      p.CreatedAt,
	}
}

func firstName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Neighbor"
	}
	if i := strings.IndexByte(name, ' '); i > 0 {
		return name[:i]
	}
	return name
}

func mapReview(rv domain.Review) *model.Review {
	out := &model.Review{
		ID:              rv.ID,
		Rating:          rv.Rating,
		Body:            rv.Body,
		AuthorFirstName: firstName(rv.AuthorName),
		CreatedAt:       rv.CreatedAt,
	}
	if rv.StoreSlug != "" {
		out.Store = &model.Store{Slug: rv.StoreSlug, Name: rv.StoreName, Category: rv.StoreCategory, City: rv.StoreCity}
	}
	if rv.ProductSlug != "" {
		out.Product = &model.Product{Slug: rv.ProductSlug, Name: rv.ProductName, DisplayPrice: ""}
	}
	return out
}

func mapStoreDetail(d usecase.StoreDetail) *model.Store {
	prods := make([]*model.Product, 0, len(d.Products))
	stLite := mapStoreLite(d.Store, d.Photos)
	for _, p := range d.Products {
		prods = append(prods, mapProduct(p, stLite, d.ProductPhotos[p.ID], nil, 0, false))
	}
	revs := make([]*model.Review, 0, len(d.Reviews))
	for _, rv := range d.Reviews {
		revs = append(revs, mapReview(rv))
	}
	return mapStore(d.Store, d.Photos, prods, revs, d.Upvotes, d.ViewerVoted, 0)
}

func mapProductDetail(d usecase.ProductDetail) *model.Product {
	st := mapStoreLite(d.Store, nil)
	revs := make([]*model.Review, 0, len(d.Reviews))
	for _, rv := range d.Reviews {
		revs = append(revs, mapReview(rv))
	}
	return mapProduct(d.Product, st, d.Photos, revs, d.Upvotes, d.ViewerVoted)
}

func gqlErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		return errors.New("unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		return errors.New("forbidden")
	case errors.Is(err, domain.ErrNotFound):
		return errors.New("not found")
	case errors.Is(err, domain.ErrDuplicateEmail):
		return errors.New("email already registered")
	case errors.Is(err, domain.ErrInvalidCredentials):
		return errors.New("invalid email or password")
	case errors.Is(err, domain.ErrOIDCNotConfigured):
		return errors.New("openid connect is not configured")
	case errors.Is(err, domain.ErrConflict):
		return errors.New("conflict")
	case errors.Is(err, domain.ErrInvalid):
		return err
	default:
		return err
	}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func cookieCats(_ time.Time) []string { return nil }
