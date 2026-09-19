# Local Discovery

Public, discovery-first marketplace for a neighborhood. Anyone can browse ranked shops, trending products, and reviews. Login is required only to upvote, write a review, or manage a storefront.

This MVP does **not** include Stripe, ads, or cart email.

## Stack

- `web/` — Vite + React + TypeScript + Apollo Client
- `api/` — Go, gqlgen, chi, Uber Fx, pgx, clean architecture
- Postgres via `DATABASE_URL` (local Docker or a hosted URL such as Supabase)

## Run locally

```bash
cp .env.example .env
make up          # Postgres on :5432
make seed        # migrate + Austin catalog + demo users
make api         # GraphQL at http://localhost:8080/graphql  (playground /playground)
make web         # http://localhost:5173
```

First web run: `cd web && npm install`.

### Demo accounts

| Email | Password | Role |
| --- | --- | --- |
| `customer@demo.local` | `demo1234` | neighbor |
| `owner@demo.local` | `demo1234` | owner (seeded Austin shops) |

City seeded: **Austin**. Categories: food, coffee, retail, services, health, nightlife, outdoor, home.

## What you can do

- Home: ranked businesses (HN gravity + category affinity), trending products, trending reviews. Filter chips and today/week/all.
- Search stores and products (`pg_trgm` when the extension is available, otherwise `ILIKE`).
- Store/product pages: photo strip, display prices, click-to-call, catalog-grounded chat drawer.
- Owner dashboard (`/dashboard`): create/edit store, photos, products.

## Auth

- GraphQL `signup` / `login` / `logout` / `me` with httpOnly JWT cookie `ld_session`.
- OpenID Connect: `GET /auth/oidc/start?role=CUSTOMER|OWNER&next=` then `GET /auth/oidc/callback`.
- If `OIDC_ISSUER` / client env vars are unset, password login still works. The OpenID button hits the start URL and returns a clear HTML error.
- Owner signup: `/signup?role=owner` (also passes `role=OWNER` into OIDC start).

## Chat

Catalog-grounded. With `OPENAI_API_KEY`, answers go through the model with a grounding packet. Without it, FAQ/template replies are built from store/product fields. Always offers `tel:`.

## Tests / build

```bash
cd api && go test ./... && go build -o /tmp/ld-api ./cmd/api
cd web && npm install && npx tsc --noEmit && npm run build
```

GraphQL smoke (after api is up):

```bash
curl -s http://localhost:8080/graphql -H 'content-type: application/json' \
  -d '{"query":"{ home(city:\"Austin\", window: ALL) { city rankedStores { rank store { name category upvoteCount } } trendingProducts { product { name displayPrice } } trendingReviews { review { body authorFirstName } } } }"}'
```

## Layout

- `api/cmd/api` — `fx.New` only here
- `api/cmd/seed` — migrate + seed (`-migrate-only` for migrations alone)
- `api/internal/{domain,ports,usecase}`
- `api/internal/adapter/{graphql,postgres,http,oidc,blob,jwt,openai}`
- `api/internal/fxmod` — config, postgres, oidc, blob, usecase, graphql, http
- `api/migrations` — portable SQL, applied on API start and by seed

## Gaps

- OIDC needs a real issuer (`OIDC_ISSUER`, `OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `OIDC_REDIRECT_URI=http://localhost:8080/auth/oidc/callback`).
- Chat LLM is optional; template answers are the default.
- Photos in seed are remote placeholders (`picsum.photos`). Owner uploads go to local disk (`BLOB_DIR`) and are served at `/media/`.
- One city in seed data. Phase 2 (not built): Stripe, sponsored search, abandoned-cart email.
