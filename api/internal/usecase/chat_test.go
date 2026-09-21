package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
)

type fakeChats struct {
	threads map[string]domain.ChatThread
	msgs    map[string][]domain.ChatMessage
}

func newFakeChats() *fakeChats {
	return &fakeChats{threads: map[string]domain.ChatThread{}, msgs: map[string][]domain.ChatMessage{}}
}
func (f *fakeChats) CreateThread(_ context.Context, t domain.ChatThread) error {
	f.threads[t.ID] = t
	return nil
}
func (f *fakeChats) GetThread(_ context.Context, id string) (domain.ChatThread, error) {
	t, ok := f.threads[id]
	if !ok {
		return domain.ChatThread{}, domain.ErrNotFound
	}
	return t, nil
}
func (f *fakeChats) AddMessage(_ context.Context, m domain.ChatMessage) error {
	f.msgs[m.ThreadID] = append(f.msgs[m.ThreadID], m)
	return nil
}
func (f *fakeChats) ListMessages(_ context.Context, threadID string, limit int) ([]domain.ChatMessage, error) {
	return f.msgs[threadID], nil
}

func TestChatTemplatePricesWithoutLLM(t *testing.T) {
	phone := "512-555-0100"
	stores := &fakeStores{items: []domain.Store{{
		ID: "st1", Name: "Franklin", Description: "Brisket line.", Category: "food", City: "Austin", Phone: &phone,
	}}}
	products := &fakeProducts{items: []domain.Product{{
		ID: "p1", StoreID: "st1", Name: "Brisket plate", Description: "Half pound", PriceCents: 1899,
	}}}
	svc := NewChatService(newFakeChats(), stores, products, nil, fakeClock{t: time.Now()})
	reply, err := svc.Chat(context.Background(), "st1", "", "", "how much is it?")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reply.Message, "18.99") && !strings.Contains(reply.Message, "Brisket") {
		t.Fatalf("expected catalog price, got %q", reply.Message)
	}
	if reply.Phone == nil || *reply.Phone != phone {
		t.Fatalf("phone: %v", reply.Phone)
	}
}
