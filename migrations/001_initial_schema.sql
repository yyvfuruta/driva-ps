CREATE TABLE IF NOT EXISTS orders (
    id uuid PRIMARY KEY,
    customer_id TEXT NOT NULL,
    status TEXT NOT NULL,
    total_amount NUMERIC(12, 2),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders(id),
    sku TEXT NOT NULL,
    qty INT NOT NULL
);

CREATE TABLE IF NOT EXISTS order_enrichments (
    id SERIAL PRIMARY KEY,
    order_id uuid NOT NULL REFERENCES orders(id),
    data JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    order_id uuid NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
