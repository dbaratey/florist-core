-- migrations/001_init.up.sql
-- Инициализация схемы florist-core
-- Боундед контексты: inventory, ordering

BEGIN;

-- ===========================================================
-- INVENTORY
-- ===========================================================

CREATE TABLE IF NOT EXISTS inventory_batches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID        NOT NULL,
    ingredient_id   UUID        NOT NULL,
    received_qty    INT         NOT NULL CHECK (received_qty > 0),
    remaining_qty   INT         NOT NULL CHECK (remaining_qty >= 0),
    purchase_price  BIGINT      NOT NULL,  -- в копейках
    currency        VARCHAR(3)  NOT NULL DEFAULT 'RUB',
    received_at     TIMESTAMPTZ NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    freshness       VARCHAR(20) NOT NULL DEFAULT 'fresh'
                    CHECK (freshness IN ('fresh', 'aging', 'critical', 'expired')),
    written_off     BOOLEAN     NOT NULL DEFAULT FALSE,
    write_off_reason TEXT,
    version         INT         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_batches_store_ingredient
    ON inventory_batches (store_id, ingredient_id)
    WHERE NOT written_off AND freshness != 'expired';

CREATE INDEX idx_batches_expires_at
    ON inventory_batches (expires_at)
    WHERE NOT written_off;

-- ===========================================================
-- ORDERING
-- ===========================================================

CREATE TABLE IF NOT EXISTS orders (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id    UUID        NOT NULL,
    customer_id UUID        NOT NULL,
    status      VARCHAR(20) NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending','confirmed','cancelled','shipped','completed')),
    total_price BIGINT      NOT NULL DEFAULT 0,  -- в копейках
    currency    VARCHAR(3)  NOT NULL DEFAULT 'RUB',
    notes       TEXT,
    version     INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id            UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID    NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id    UUID    NOT NULL,
    ingredient_id UUID    NOT NULL,
    qty           INT     NOT NULL CHECK (qty > 0),
    unit          VARCHAR(10) NOT NULL DEFAULT 'шт',
    unit_price    BIGINT  NOT NULL,
    currency      VARCHAR(3) NOT NULL DEFAULT 'RUB',
    batch_id      UUID    REFERENCES inventory_batches(id) ON DELETE SET NULL,
    reserved      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_orders_store_status
    ON orders (store_id, status);

CREATE INDEX idx_order_items_order
    ON order_items (order_id);

-- ===========================================================
-- OUTBOX (гарантированная доставка доменных событий)
-- ===========================================================

CREATE TABLE IF NOT EXISTS domain_events_outbox (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_name   VARCHAR(100) NOT NULL,
    aggregate_id UUID        NOT NULL,
    payload      JSONB       NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    published    BOOLEAN     NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_outbox_unpublished
    ON domain_events_outbox (created_at)
    WHERE NOT published;

COMMIT;
