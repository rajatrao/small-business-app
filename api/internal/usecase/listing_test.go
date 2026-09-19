package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
)

func TestDeleteStoreOwnerOnly(t *testing.T) {
	stores := &fakeStores{items: []domain.Store{{
		ID: "st1", OwnerID: "owner-1", Name: "Kayaks", Category: "outdoor", City: "Austin",
	}}}
	svc := NewListingService(stores, &fakeProducts{}, nil, nil, nil, fakeClock{t: time.Now()})
	ctx := context.Background()

	if err := svc.DeleteStore(ctx, nil, "st1"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("unauthed: %v", err)
	}
	if err := svc.DeleteStore(ctx, &domain.Principal{UserID: "other", Role: domain.RoleOwner}, "st1"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("other owner: %v", err)
	}
	if err := svc.DeleteStore(ctx, &domain.Principal{UserID: "owner-1", Role: domain.RoleOwner}, "st1"); err != nil {
		t.Fatal(err)
	}
	if _, err := stores.GetByID(ctx, "st1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected store gone, got %v", err)
	}
}
