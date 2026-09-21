package usecase

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
)

func TestCommunityScoreGravity(t *testing.T) {
	hot := domain.CommunityScore(50, 10, 2, domain.Gravity)
	old := domain.CommunityScore(50, 10, 48, domain.Gravity)
	if hot <= old {
		t.Fatalf("newer should rank higher: %f vs %f", hot, old)
	}
	zero := domain.CommunityScore(0, 0, 1, domain.Gravity)
	if zero != 0 {
		t.Fatalf("zero votes: %f", zero)
	}
}

func TestFeedScoreAffinity(t *testing.T) {
	base := domain.FeedScore(10, 0)
	boosted := domain.FeedScore(10, 0.5)
	if boosted <= base {
		t.Fatalf("affinity should boost: %f vs %f", boosted, base)
	}
}

func TestHomeRanksByCommunityThenAffinity(t *testing.T) {
	now := time.Date(2026, 9, 19, 18, 0, 0, 0, time.UTC)
	stores := &fakeStores{items: []domain.Store{
		{ID: "s-food", OwnerID: "o", Slug: "tacos", Name: "Tacos", Category: "food", City: "Austin", CreatedAt: now.Add(-6 * time.Hour)},
		{ID: "s-retail", OwnerID: "o", Slug: "books", Name: "Books", Category: "retail", City: "Austin", CreatedAt: now.Add(-6 * time.Hour)},
		{ID: "s-coffee", OwnerID: "o", Slug: "beans", Name: "Beans", Category: "coffee", City: "Austin", CreatedAt: now.Add(-6 * time.Hour)},
		{ID: "s-health", OwnerID: "o", Slug: "yoga", Name: "Yoga", Category: "health", City: "Austin", CreatedAt: now.Add(-6 * time.Hour)},
	}}
	votes := newFakeVotes()
	votes.store["s-food"] = 12
	votes.store["s-retail"] = 10
	votes.store["s-coffee"] = 9
	votes.store["s-health"] = 7

	aff := &fakeAffinity{}
	_ = aff.Bump(context.Background(), "user-1", "", "food", 20)

	svc := NewHomeService(stores, &fakeProducts{}, votes, newFakeReviews(), aff, fakeClock{t: now})
	page, err := svc.Home(context.Background(), "Austin", "", "ALL", &domain.Principal{UserID: "user-1"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.RankedStores) < 4 {
		t.Fatalf("expected 4 stores, got %d", len(page.RankedStores))
	}
	// retail has the raw community lead (20 votes) but food has a huge affinity bump.
	// With boost 0.6, food feed = community * 1.6. Check food is not last.
	if page.RankedStores[0].Store.Category == "health" {
		t.Fatalf("lowest-vote health should not lead: %+v", cats(page))
	}
	foodFeed, retailFeed := 0.0, 0.0
	for _, r := range page.RankedStores {
		if r.Store.ID == "s-food" {
			foodFeed = r.FeedScore
		}
		if r.Store.ID == "s-retail" {
			retailFeed = r.FeedScore
		}
	}
	if foodFeed <= retailFeed {
		t.Fatalf("food affinity should beat retail community: food=%f retail=%f cats=%v", foodFeed, retailFeed, cats(page))
	}
}

func TestHomeDiversityFloor(t *testing.T) {
	now := time.Date(2026, 9, 19, 18, 0, 0, 0, time.UTC)
	items := make([]domain.Store, 0, 10)
	for i := 0; i < 8; i++ {
		items = append(items, domain.Store{
			ID: "food-" + string(rune('a'+i)), OwnerID: "o", Slug: "f", Name: "F",
			Category: "food", City: "Austin", CreatedAt: now.Add(-time.Duration(i) * time.Hour),
		})
	}
	items = append(items,
		domain.Store{ID: "out-1", OwnerID: "o", Slug: "kayak", Name: "Kayak", Category: "outdoor", City: "Austin", CreatedAt: now.Add(-3 * time.Hour)},
		domain.Store{ID: "home-1", OwnerID: "o", Slug: "hw", Name: "Hardware", Category: "home", City: "Austin", CreatedAt: now.Add(-3 * time.Hour)},
	)
	stores := &fakeStores{items: items}
	votes := newFakeVotes()
	for _, st := range items {
		if st.Category == "food" {
			votes.store[st.ID] = 50
		} else {
			votes.store[st.ID] = 1
		}
	}
	aff := &fakeAffinity{}
	_ = aff.Bump(context.Background(), "u", "", "food", 30)

	svc := NewHomeService(stores, &fakeProducts{}, votes, newFakeReviews(), aff, fakeClock{t: now})
	page, err := svc.Home(context.Background(), "Austin", "", "ALL", &domain.Principal{UserID: "u"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	other := 0
	for _, r := range page.RankedStores {
		if r.Store.Category != "food" {
			other++
		}
	}
	need := int(math.Ceil(float64(len(page.RankedStores)) * domain.DiversityFloor))
	if other < need && other < 2 {
		t.Fatalf("diversity floor: others=%d need~%d page=%v", other, need, cats(page))
	}
}

func TestHomeCategoryFilterSkipsDiversity(t *testing.T) {
	now := time.Now()
	stores := &fakeStores{items: []domain.Store{
		{ID: "1", Slug: "a", Name: "A", Category: "food", City: "Austin", CreatedAt: now},
		{ID: "2", Slug: "b", Name: "B", Category: "coffee", City: "Austin", CreatedAt: now},
	}}
	svc := NewHomeService(stores, &fakeProducts{}, newFakeVotes(), newFakeReviews(), &fakeAffinity{}, fakeClock{t: now})
	page, err := svc.Home(context.Background(), "Austin", "food", "ALL", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.RankedStores) != 1 || page.RankedStores[0].Store.Category != "food" {
		t.Fatalf("filter failed: %+v", page.RankedStores)
	}
}

func TestHomeCityFilter(t *testing.T) {
	now := time.Now()
	stores := &fakeStores{items: []domain.Store{
		{ID: "atx", Slug: "austin-shop", Name: "Austin Shop", Category: "food", City: "Austin", CreatedAt: now},
		{ID: "hou", Slug: "houston-shop", Name: "Houston Shop", Category: "food", City: "Houston", CreatedAt: now},
	}}
	products := &fakeProducts{items: []domain.Product{
		{ID: "p-atx", StoreID: "atx", Name: "Tacos", CreatedAt: now},
		{ID: "p-hou", StoreID: "hou", Name: "Kolaches", CreatedAt: now},
	}}
	svc := NewHomeService(stores, products, newFakeVotes(), newFakeReviews(), &fakeAffinity{}, fakeClock{t: now})
	page, err := svc.Home(context.Background(), "Houston", "", "ALL", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.RankedStores) != 1 || page.RankedStores[0].Store.ID != "hou" {
		t.Fatalf("ranked should be Houston only: %+v", page.RankedStores)
	}
	if len(page.TrendingProducts) != 1 || page.TrendingProducts[0].Product.ID != "p-hou" {
		t.Fatalf("products should be Houston only: %+v", page.TrendingProducts)
	}
}

func cats(page domain.HomePage) []string {
	out := make([]string, len(page.RankedStores))
	for i, r := range page.RankedStores {
		out[i] = r.Store.Category
	}
	return out
}
