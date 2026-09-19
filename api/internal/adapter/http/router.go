package httpadp

import (
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rajat/localdiscovery/internal/adapter/blob"
	gqladapter "github.com/rajat/localdiscovery/internal/adapter/graphql"
	"github.com/rajat/localdiscovery/internal/adapter/graphql/generated"
	"github.com/rajat/localdiscovery/internal/config"
	"github.com/rajat/localdiscovery/internal/domain"
	"github.com/rajat/localdiscovery/internal/usecase"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	sessionCookie = "ld_session"
	guestCookie   = "ld_guest"
	oidcCookie    = "ld_oidc"
)

func NewRouter(cfg config.Config, resolver *gqladapter.Resolver, auth *usecase.AuthService, disk *blob.Disk) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{cfg.CORSOrigin, "http://127.0.0.1:5173"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Guest-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(sessionMiddleware(cfg, auth))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	r.Handle("/media/*", disk.Handler())

	gqlSrv := handler.New(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	gqlSrv.AddTransport(transport.Options{})
	gqlSrv.AddTransport(transport.GET{})
	gqlSrv.AddTransport(transport.POST{})
	gqlSrv.SetQueryCache(lru.New[*ast.QueryDocument](1000))
	gqlSrv.Use(extension.Introspection{})

	r.Handle("/graphql", gqlSrv)
	r.Handle("/playground", playground.Handler("Local Discovery", "/graphql"))

	r.Get("/auth/oidc/start", oidcStart(cfg, auth))
	r.Get("/auth/oidc/callback", oidcCallback(cfg, auth))
	return r
}

func sessionMiddleware(cfg config.Config, auth *usecase.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := gqladapter.WithWriter(r.Context(), w, cfg.CookieSecure)
			if c, err := r.Cookie(sessionCookie); err == nil && c.Value != "" {
				if p, err := auth.ParseToken(c.Value); err == nil {
					ctx = gqladapter.WithPrincipal(ctx, &p)
				}
			} else if h := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(h), "bearer ") {
				if p, err := auth.ParseToken(strings.TrimSpace(h[7:])); err == nil {
					ctx = gqladapter.WithPrincipal(ctx, &p)
				}
			}
			guest := ""
			if c, err := r.Cookie(guestCookie); err == nil {
				guest = c.Value
			}
			if guest == "" {
				guest = usecase.RandomGuestID()
				http.SetCookie(w, cookie(guestCookie, guest, 365*24*time.Hour, cfg.CookieSecure))
			}
			ctx = gqladapter.WithGuest(ctx, guest)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func oidcStart(cfg config.Config, auth *usecase.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := domain.ParseRole(r.URL.Query().Get("role"))
		next := r.URL.Query().Get("next")
		authURL, stateCookie, err := auth.StartOIDC(r.Context(), role, next, cfg.OIDCRedirectURI)
		if err != nil {
			if errors.Is(err, domain.ErrOIDCNotConfigured) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotImplemented)
				_, _ = w.Write([]byte(oidcMissingHTML(cfg.CORSOrigin)))
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.SetCookie(w, cookie(oidcCookie, stateCookie, 10*time.Minute, cfg.CookieSecure))
		http.Redirect(w, r, authURL, http.StatusFound)
	}
}

func oidcCallback(cfg config.Config, auth *usecase.AuthService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if msg := r.URL.Query().Get("error"); msg != "" {
			http.Error(w, "oidc: "+msg, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		sc := ""
		if c, err := r.Cookie(oidcCookie); err == nil {
			sc = c.Value
		}
		guest := ""
		if c, err := r.Cookie(guestCookie); err == nil {
			guest = c.Value
		}
		sess, next, err := auth.HandleOIDCCallback(r.Context(), code, state, sc, cfg.OIDCRedirectURI, guest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.SetCookie(w, cookie(sessionCookie, sess.Token, 7*24*time.Hour, cfg.CookieSecure))
		http.SetCookie(w, expired(oidcCookie, cfg.CookieSecure))
		dest := cfg.CORSOrigin + next
		if next == "" {
			dest = cfg.CORSOrigin + "/"
		}
		http.Redirect(w, r, dest, http.StatusFound)
	}
}

func cookie(name, value string, ttl time.Duration, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
	}
}

func expired(name string, secure bool) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: secure,
	}
}

func oidcMissingHTML(web string) string {
	esc := html.EscapeString(web + "/login")
	return `<!doctype html><meta charset="utf-8"><title>OIDC not configured</title>
<body style="font-family:system-ui;max-width:40rem;margin:4rem auto;line-height:1.5">
<h1>OpenID Connect is not configured</h1>
<p>Set <code>OIDC_ISSUER</code>, <code>OIDC_CLIENT_ID</code>, <code>OIDC_CLIENT_SECRET</code>, and
<code>OIDC_REDIRECT_URI</code> on the API. Email/password login still works.</p>
<p><a href="` + esc + `">Back to login</a></p>
</body>`
}
