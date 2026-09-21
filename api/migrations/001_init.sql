CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT,
    name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('CUSTOMER', 'OWNER', 'ADMIN')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS oidc_identities (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject)
);

CREATE INDEX IF NOT EXISTS oidc_identities_user_id_idx ON oidc_identities (user_id);

CREATE TABLE IF NOT EXISTS stores (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL,
    city TEXT NOT NULL,
    phone TEXT,
    address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS stores_city_idx ON stores (city);
CREATE INDEX IF NOT EXISTS stores_category_idx ON stores (category);
CREATE INDEX IF NOT EXISTS stores_owner_id_idx ON stores (owner_id);

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY,
    store_id UUID NOT NULL REFERENCES stores (id) ON DELETE CASCADE,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_cents INTEGER NOT NULL CHECK (price_cents >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS products_store_id_idx ON products (store_id);

CREATE TABLE IF NOT EXISTS photos (
    id UUID PRIMARY KEY,
    store_id UUID REFERENCES stores (id) ON DELETE CASCADE,
    product_id UUID REFERENCES products (id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (store_id IS NOT NULL OR product_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS photos_store_id_idx ON photos (store_id);
CREATE INDEX IF NOT EXISTS photos_product_id_idx ON photos (product_id);

CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    store_id UUID REFERENCES stores (id) ON DELETE CASCADE,
    product_id UUID REFERENCES products (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (store_id IS NOT NULL AND product_id IS NULL)
        OR (store_id IS NULL AND product_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS votes_user_store_uidx ON votes (user_id, store_id) WHERE store_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS votes_user_product_uidx ON votes (user_id, product_id) WHERE product_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    store_id UUID REFERENCES stores (id) ON DELETE CASCADE,
    product_id UUID REFERENCES products (id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (store_id IS NOT NULL AND product_id IS NULL)
        OR (store_id IS NULL AND product_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS reviews_store_id_idx ON reviews (store_id);
CREATE INDEX IF NOT EXISTS reviews_product_id_idx ON reviews (product_id);
CREATE INDEX IF NOT EXISTS reviews_created_at_idx ON reviews (created_at DESC);

CREATE TABLE IF NOT EXISTS category_affinities (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users (id) ON DELETE CASCADE,
    guest_id TEXT,
    category TEXT NOT NULL,
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (user_id IS NOT NULL AND guest_id IS NULL)
        OR (user_id IS NULL AND guest_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS category_affinities_user_cat_uidx
    ON category_affinities (user_id, category) WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS category_affinities_guest_cat_uidx
    ON category_affinities (guest_id, category) WHERE guest_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS chat_threads (
    id UUID PRIMARY KEY,
    store_id UUID REFERENCES stores (id) ON DELETE CASCADE,
    product_id UUID REFERENCES products (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY,
    thread_id UUID NOT NULL REFERENCES chat_threads (id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant')),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS chat_messages_thread_id_idx ON chat_messages (thread_id);
