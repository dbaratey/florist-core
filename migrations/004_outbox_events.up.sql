-- Migration 004: Transactional Outbox Events
-- Pattern: write events in same TX as domain changes, worker dispatches async

CREATE TYPE outbox_status AS ENUM ('pending', 'processing', 'done', 'failed');

CREATE TABLE outbox_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type  TEXT NOT NULL,          -- e.g. 'Batch', 'Order'
    aggregate_id    UUID NOT NULL,
    event_type      TEXT NOT NULL,          -- e.g. 'BatchConsumed', 'OrderConfirmed'
    payload         JSONB NOT NULL,
    status          outbox_status NOT NULL DEFAULT 'pending',
    attempts        SMALLINT NOT NULL DEFAULT 0,
    max_attempts    SMALLINT NOT NULL DEFAULT 5,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at    TIMESTAMPTZ
);

CREATE INDEX idx_outbox_events_status ON outbox_events (status) WHERE status IN ('pending', 'processing');
CREATE INDEX idx_outbox_events_created_at ON outbox_events (created_at);
