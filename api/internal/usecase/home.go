package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type HomeService struct {
	stores   ports.StoreRepository
	products ports.ProductRepository
	votes    ports.VoteRepository
	reviews  ports.ReviewRepository
	affinity ports.AffinityRepository
	clock    ports.Clock
}

func NewHomeService(
	stores ports.StoreRepository,
	products ports.ProductRepository,
	votes ports.VoteRepository,
	reviews ports.ReviewRepository,
	affinity ports.AffinityRepository,
	clock ports.Clock,
) *HomeService {
	return &HomeService{
		stores: stores, products: products, votes: votes,
		reviews: reviews, affinity: affinity, clock: clock,
	}
}

func (s *HomeService) Home(ctx context.Context, city, category, window string, principal *domain.Principal, guestID string, cookieCats []string) (domain.HomePage, error) {
	if city == "" {
		city = domain.DefaultCity
	}
	city = strings.TrimSpace(city)

	category = strings.ToLower(strings.TrimSpace(category))
	if category != "" && !domain.ValidCategory(category) {
		category = ""
	}
	w := domain.ParseWindow(window)
	now := s.clock.Now()
	since := w.Since(now)

	stores, err := s.stores.ListByCity(ctx, city, category)
	if err != nil {
		return domain.HomePage{}, err
	}
	storeIDs := make([]string, len(stores))
	storeByID := make(map[string]domain.Store, len(stores))
	for i, st := range stores {
		storeIDs[i] = st.ID
		storeByID[st.ID] = st
	}

	upvotes, err := s.votes.CountByStoreIDs(ctx, storeIDs, since)
	if err != nil {
		return domain.HomePage{}, err
	}
	revStats, err := s.reviews.StatsByStoreIDs(ctx, storeIDs, since)
	if err != nil {
		return domain.HomePage{}, err
	}

	affMap, favored := s.loadAffinity(ctx, principal, guestID, cookieCats)

	ranked := make([]domain.RankedStore, 0, len(stores))
	communityByStore := make(map[string]float64, len(stores))
	for _, st := range stores {
		if !strings.EqualFold(st.City, city) {
			continue
		}
		hours := now.Sub(st.CreatedAt).Hours()
		uv := upvotes[st.ID]
		wr := revStats[st.ID].Weighted()
		cs := domain.CommunityScore(uv, wr, hours, domain.Gravity)
		communityByStore[st.ID] = cs
		boost := domain.AffinityBoost(affMap[st.Category])
		ranked = append(ranked, domain.RankedStore{
			Store:           st,
			CommunityScore:  cs,
			FeedScore:       domain.FeedScore(cs, boost),
			Upvotes:         uv,
			WeightedReviews: wr,
		})
	}
	sortRanked(ranked)

	pageSize := 12
	if category == "" {
		ranked = domain.ApplyDiversityFloor(ranked, favored, pageSize)
	} else if len(ranked) > pageSize {
		ranked = ranked[:pageSize]
	}
	for i := range ranked {
		ranked[i].Rank = i + 1
	}

	products, err := s.products.ListByCity(ctx, city, category)
	if err != nil {
		return domain.HomePage{}, err
	}
	prodIDs := make([]string, len(products))
	for i, p := range products {
		prodIDs[i] = p.ID
	}
	pVotes, err := s.votes.CountByProductIDs(ctx, prodIDs, since)
	if err != nil {
		return domain.HomePage{}, err
	}
	pRev, err := s.reviews.StatsByProductIDs(ctx, prodIDs, since)
	if err != nil {
		return domain.HomePage{}, err
	}

	trendingP := make([]domain.TrendingProduct, 0, len(products))
	for _, p := range products {
		st, ok := storeByID[p.StoreID]
		if !ok {
			got, err := s.stores.GetByID(ctx, p.StoreID)
			if err != nil {
				continue
			}
			st = got
			storeByID[st.ID] = st
		}
		if !strings.EqualFold(st.City, city) {
			continue
		}
		last := p.CreatedAt
		hours := now.Sub(last).Hours()
		heat := domain.ProductHeat(pVotes[p.ID], pRev[p.ID].Count, hours, domain.Gravity)
		heat *= 1 + domain.AffinityBoost(affMap[st.Category])*0.5
		trendingP = append(trendingP, domain.TrendingProduct{Product: p, Store: st, Heat: heat})
	}
	sortTrendingProducts(trendingP)
	if len(trendingP) > 8 {
		trendingP = trendingP[:8]
	}

	recent, err := s.reviews.ListRecent(ctx, city, category, since, 24)
	if err != nil {
		return domain.HomePage{}, err
	}
	trendingR := make([]domain.TrendingReview, 0, len(recent))
	for _, rv := range recent {
		parent := 0.0
		if rv.StoreID != nil {
			parent = communityByStore[*rv.StoreID]
		}
		heat := domain.ReviewHeat(now.Sub(rv.CreatedAt).Hours(), float64(rv.Rating), parent)
		trendingR = append(trendingR, domain.TrendingReview{Review: rv, Heat: heat})
	}
	sortTrendingReviews(trendingR)
	if len(trendingR) > 8 {
		trendingR = trendingR[:8]
	}

	return domain.HomePage{
		City:             city,
		RankedStores:     ranked,
		TrendingProducts: trendingP,
		TrendingReviews:  trendingR,
	}, nil
}

func (s *HomeService) loadAffinity(ctx context.Context, principal *domain.Principal, guestID string, cookieCats []string) (map[string]float64, map[string]bool) {
	scores := map[string]float64{}
	userID := ""
	if principal != nil {
		userID = principal.UserID
	}
	if rows, err := s.affinity.List(ctx, userID, guestID); err == nil {
		for _, r := range rows {
			scores[r.Category] += r.Score
		}
	}
	for i, c := range cookieCats {
		c = strings.ToLower(strings.TrimSpace(c))
		if !domain.ValidCategory(c) {
			continue
		}
		// cookie is a weak signal; earlier chips count a bit more
		scores[c] += 1.2 - 0.1*float64(i)
	}
	favored := map[string]bool{}
	for cat, sc := range scores {
		if sc >= 1.0 {
			favored[cat] = true
		}
	}
	return scores, favored
}

func sortRanked(items []domain.RankedStore) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].FeedScore > items[i].FeedScore ||
				(items[j].FeedScore == items[i].FeedScore && items[j].Store.CreatedAt.After(items[i].Store.CreatedAt)) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortTrendingProducts(items []domain.TrendingProduct) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Heat > items[i].Heat {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortTrendingReviews(items []domain.TrendingReview) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Heat > items[i].Heat {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

type SearchService struct {
	search ports.SearchRepository
}

func NewSearchService(search ports.SearchRepository) *SearchService {
	return &SearchService{search: search}
}

type SearchResult struct {
	Stores   []domain.Store
	Products []domain.Product
}

func (s *SearchService) Search(ctx context.Context, q, city string) (SearchResult, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return SearchResult{}, nil
	}
	stores, err := s.search.SearchStores(ctx, q, city, 20)
	if err != nil {
		return SearchResult{}, err
	}
	products, err := s.search.SearchProducts(ctx, q, city, 20)
	if err != nil {
		return SearchResult{}, err
	}
	return SearchResult{Stores: stores, Products: products}, nil
}

type AffinityService struct {
	affinity ports.AffinityRepository
}

func NewAffinityService(a ports.AffinityRepository) *AffinityService {
	return &AffinityService{affinity: a}
}

func (s *AffinityService) Record(ctx context.Context, principal *domain.Principal, guestID, category string, kind domain.AffinityKind) error {
	category = strings.ToLower(strings.TrimSpace(category))
	if !domain.ValidCategory(category) {
		return domain.ErrInvalid
	}
	userID := ""
	if principal != nil {
		userID = principal.UserID
	}
	if userID == "" && guestID == "" {
		return nil
	}
	return s.affinity.Bump(ctx, userID, guestID, category, kind.Weight())
}

func HoursSince(now, then time.Time) float64 {
	return now.Sub(then).Hours()
}
