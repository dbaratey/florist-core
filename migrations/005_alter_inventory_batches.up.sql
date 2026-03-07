-- Migration 005: extend inventory_batches table with all Batch aggregate fields
-- and add indices for freshness recalc and store queries.

ALTER TABLE inventory_batches
    ADD COLUMN IF NOT EXISTS store_id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    ADD COLUMN IF NOT EXISTS ingredient_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    ADD COLUMN IF NOT EXISTS received_qty     INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS remaining_qty    INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS received_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS freshness        TEXT        NOT NULL DEFAULT 'fresh'
                                              CHECK (freshness IN ('fresh','aging','critical','expired')),
    ADD COLUMN IF NOT EXISTS version          INTEGER     NOT NULL DEFAULT 0;

-- Rename legacy columns to match domain model
ALTER TABLE inventory_batches
    RENAME COLUMN sku         TO ingredient_sku;

ALTER TABLE inventory_batches
    RENAME COLUMN expiry_date TO expires_at;

ALTER TABLE inventory_batches
    RENAME COLUMN quantity    TO legacy_quantity;

-- Remove defaults that were only needed for the migration
ALTER TABLE inventory_batches
    ALTER COLUMN store_id      DROP DEFAULT,
    ALTER COLUMN ingredient_id DROP DEFAULT;

-- Indices for common query patterns
CREATE INDEX IF NOT EXISTS idx_inventory_batches_store_id
    ON inventory_batches (store_id);

CREATE INDEX IF NOT EXISTS idx_inventory_batches_active
    ON inventory_batches (freshness, remaining_qty)
    WHERE freshness <> 'expired' AND remaining_qty > 0;

CREATE INDEX IF NOT EXISTS idx_inventory_batches_expires_at
    ON inventory_batches (expires_at)
    WHERE freshness <> 'expired';
