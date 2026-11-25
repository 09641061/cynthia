SET LOCAL lock_timeout = '2s';
SET LOCAL statement_timeout = '10s';

CREATE TABLE IF NOT EXISTS catalog_categories (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    sort_order INTEGER NOT NULL CHECK (sort_order > 0),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_categories_sort_order_key UNIQUE (sort_order)
        DEFERRABLE INITIALLY DEFERRED
);

CREATE TABLE IF NOT EXISTS catalog_services (
    id UUID PRIMARY KEY,
    category_id UUID NOT NULL REFERENCES catalog_categories(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(btrim(name)) BETWEEN 1 AND 120),
    duration_minutes INTEGER NOT NULL CHECK (duration_minutes BETWEEN 1 AND 1440),
    price_minor_units BIGINT NOT NULL CHECK (price_minor_units >= 0),
    currency_code TEXT NOT NULL CHECK (currency_code ~ '^[A-Z]{3}$'),
    sort_order INTEGER NOT NULL CHECK (sort_order > 0),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT catalog_services_category_sort_order_key
        UNIQUE (category_id, sort_order) DEFERRABLE INITIALLY DEFERRED
);

