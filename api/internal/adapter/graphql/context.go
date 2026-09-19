package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/rajat/localdiscovery/internal/domain"
)

type ctxKey int

const (
	principalKey ctxKey = iota
	guestKey
	writerKey
	secureKey
)

func WithPrincipal(ctx context.Context, p *domain.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

func PrincipalFrom(ctx context.Context) *domain.Principal {
	p, _ := ctx.Value(principalKey).(*domain.Principal)
	return p
}

func WithGuest(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, guestKey, id)
}

func GuestFrom(ctx context.Context) string {
	s, _ := ctx.Value(guestKey).(string)
	return s
}

func WithWriter(ctx context.Context, w http.ResponseWriter, secure bool) context.Context {
	ctx = context.WithValue(ctx, writerKey, w)
	return context.WithValue(ctx, secureKey, secure)
}

func SetSessionCookie(ctx context.Context, token string) {
	w, _ := ctx.Value(writerKey).(http.ResponseWriter)
	if w == nil {
		return
	}
	secure, _ := ctx.Value(secureKey).(bool)
	http.SetCookie(w, &http.Cookie{
		Name:     "ld_session",
		Value:    token,
		Path:     "/",
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	})
}

func ClearSessionCookie(ctx context.Context) {
	w, _ := ctx.Value(writerKey).(http.ResponseWriter)
	if w == nil {
		return
	}
	secure, _ := ctx.Value(secureKey).(bool)
	http.SetCookie(w, &http.Cookie{
		Name: "ld_session", Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure,
	})
}
