package usecase

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/ports"
)

type ChatService struct {
	chats    ports.ChatRepository
	stores   ports.StoreRepository
	products ports.ProductRepository
	llm      ports.LLMClient
	clock    ports.Clock
}

func NewChatService(
	chats ports.ChatRepository,
	stores ports.StoreRepository,
	products ports.ProductRepository,
	llm ports.LLMClient,
	clock ports.Clock,
) *ChatService {
	return &ChatService{chats: chats, stores: stores, products: products, llm: llm, clock: clock}
}

type ChatReply struct {
	ThreadID string
	Message  string
	Phone    *string
}

func (s *ChatService) Chat(ctx context.Context, storeID, productID, threadID, message string) (ChatReply, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return ChatReply{}, domain.ErrInvalid
	}

	var thread domain.ChatThread
	var err error
	if threadID != "" {
		thread, err = s.chats.GetThread(ctx, threadID)
		if err != nil {
			return ChatReply{}, err
		}
	} else {
		thread = domain.ChatThread{ID: uuid.NewString(), CreatedAt: s.clock.Now()}
		if productID != "" {
			thread.ProductID = &productID
		}
		if storeID != "" {
			thread.StoreID = &storeID
		}
		if err := s.chats.CreateThread(ctx, thread); err != nil {
			return ChatReply{}, err
		}
	}

	ground, err := s.ground(ctx, thread)
	if err != nil {
		return ChatReply{}, err
	}

	_ = s.chats.AddMessage(ctx, domain.ChatMessage{
		ID: uuid.NewString(), ThreadID: thread.ID, Role: "user", Body: message, CreatedAt: s.clock.Now(),
	})

	history, _ := s.chats.ListMessages(ctx, thread.ID, 12)
	reply := s.templateAnswer(message, ground)
	if s.llm != nil {
		if out, err := s.llm.Complete(ctx, groundingSystem(ground), historyPrompt(history, message)); err == nil && strings.TrimSpace(out) != "" {
			reply = strings.TrimSpace(out)
		}
	}

	_ = s.chats.AddMessage(ctx, domain.ChatMessage{
		ID: uuid.NewString(), ThreadID: thread.ID, Role: "assistant", Body: reply, CreatedAt: s.clock.Now(),
	})
	return ChatReply{ThreadID: thread.ID, Message: reply, Phone: ground.Phone}, nil
}

func (s *ChatService) ground(ctx context.Context, thread domain.ChatThread) (domain.CatalogGrounding, error) {
	g := domain.CatalogGrounding{}
	var store domain.Store
	var err error
	if thread.ProductID != nil {
		p, err := s.products.GetByID(ctx, *thread.ProductID)
		if err != nil {
			return g, err
		}
		g.ProductName = p.Name
		g.ProductDesc = p.Description
		g.ProductPrice = domain.DisplayPrice(p.PriceCents)
		store, err = s.stores.GetByID(ctx, p.StoreID)
		if err != nil {
			return g, err
		}
	} else if thread.StoreID != nil {
		store, err = s.stores.GetByID(ctx, *thread.StoreID)
		if err != nil {
			return g, err
		}
	} else {
		return g, domain.ErrInvalid
	}
	g.StoreName = store.Name
	g.StoreDescription = store.Description
	g.Category = store.Category
	g.City = store.City
	g.Phone = store.Phone
	g.Address = store.Address
	prods, err := s.products.ListByStore(ctx, store.ID)
	if err != nil {
		return g, err
	}
	for _, p := range prods {
		g.Products = append(g.Products, domain.GroundedProduct{
			Name: p.Name, Price: domain.DisplayPrice(p.PriceCents), Desc: p.Description,
		})
	}
	return g, nil
}

func (s *ChatService) templateAnswer(message string, g domain.CatalogGrounding) string {
	q := strings.ToLower(message)
	phone := "the shop"
	if g.Phone != nil && *g.Phone != "" {
		phone = *g.Phone
	}

	switch {
	case containsAny(q, "price", "cost", "how much", "expensive", "cheap"):
		if g.ProductPrice != "" {
			return fmt.Sprintf("%s is listed at %s (display price — confirm in person). For the latest, call %s.", g.ProductName, g.ProductPrice, phone)
		}
		if len(g.Products) == 0 {
			return fmt.Sprintf("%s has not listed menu prices yet. Call %s to ask.", g.StoreName, phone)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "Listed prices at %s (display only):\n", g.StoreName)
		for i, p := range g.Products {
			if i >= 8 {
				break
			}
			fmt.Fprintf(&b, "• %s — %s\n", p.Name, p.Price)
		}
		fmt.Fprintf(&b, "Confirm in person or call %s.", phone)
		return b.String()
	case containsAny(q, "phone", "call", "contact", "number"):
		if g.Phone != nil && *g.Phone != "" {
			return fmt.Sprintf("Call %s at %s.", g.StoreName, *g.Phone)
		}
		return fmt.Sprintf("%s has not published a phone number yet.", g.StoreName)
	case containsAny(q, "where", "address", "located", "direction"):
		if g.Address != nil && *g.Address != "" {
			return fmt.Sprintf("%s is at %s in %s.", g.StoreName, *g.Address, g.City)
		}
		return fmt.Sprintf("%s is a %s shop in %s. Address is not listed — call %s.", g.StoreName, g.Category, g.City, phone)
	case containsAny(q, "hour", "open", "close", "when"):
		return fmt.Sprintf("Hours are not in the catalog. %s: %s Call %s to confirm they are open.", g.StoreName, g.StoreDescription, phone)
	case containsAny(q, "menu", "what do you sell", "product", "offer"):
		if g.ProductName != "" {
			return fmt.Sprintf("%s — %s. Listed at %s. From %s in %s.", g.ProductName, g.ProductDesc, g.ProductPrice, g.StoreName, g.City)
		}
		if len(g.Products) == 0 {
			return fmt.Sprintf("%s (%s) in %s. %s", g.StoreName, g.Category, g.City, g.StoreDescription)
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%s currently lists:\n", g.StoreName)
		for i, p := range g.Products {
			if i >= 8 {
				break
			}
			fmt.Fprintf(&b, "• %s (%s) — %s\n", p.Name, p.Price, p.Desc)
		}
		return b.String()
	default:
		focus := g.StoreDescription
		if g.ProductName != "" {
			focus = fmt.Sprintf("%s (%s, %s). %s", g.ProductName, g.ProductPrice, g.StoreName, g.ProductDesc)
		}
		return fmt.Sprintf("I can only answer from the catalog listing.\n\n%s\n\nAsk about prices, address, or tap call %s.", focus, phone)
	}
}

func groundingSystem(g domain.CatalogGrounding) string {
	var b strings.Builder
	b.WriteString("You are a catalog-grounded assistant for a local discovery app. ")
	b.WriteString("Only use the facts below. Never invent prices, hours, or promotions. ")
	b.WriteString("If something is missing, say so and suggest calling the store. Keep answers short.\n\n")
	fmt.Fprintf(&b, "Store: %s\nCategory: %s\nCity: %s\nDescription: %s\n", g.StoreName, g.Category, g.City, g.StoreDescription)
	if g.Phone != nil {
		fmt.Fprintf(&b, "Phone: %s\n", *g.Phone)
	}
	if g.Address != nil {
		fmt.Fprintf(&b, "Address: %s\n", *g.Address)
	}
	if g.ProductName != "" {
		fmt.Fprintf(&b, "Focus product: %s — %s — %s\n", g.ProductName, g.ProductPrice, g.ProductDesc)
	}
	if len(g.Products) > 0 {
		b.WriteString("Products:\n")
		for _, p := range g.Products {
			fmt.Fprintf(&b, "- %s | %s | %s\n", p.Name, p.Price, p.Desc)
		}
	}
	return b.String()
}

func historyPrompt(msgs []domain.ChatMessage, latest string) string {
	var b strings.Builder
	for _, m := range msgs {
		fmt.Fprintf(&b, "%s: %s\n", m.Role, m.Body)
	}
	if !strings.Contains(b.String(), latest) {
		fmt.Fprintf(&b, "user: %s\n", latest)
	}
	return b.String()
}

func containsAny(q string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(q, w) {
			return true
		}
	}
	return false
}

func IsLetterOrSpace(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsSpace(r)
}
