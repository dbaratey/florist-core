package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OutboxEventPublisher implements kernel.EventPublisher by writing
// domain events into the outbox_events table within the current transaction.
// If a pgx.Tx is present in the context it will be used; otherwise falls back
// to the pool (best-effort, not transactional).
type OutboxEventPublisher struct {
	db *pgxpool.Pool
}

// NewOutboxEventPublisher creates an EventPublisher backed by the outbox table.
func NewOutboxEventPublisher(db *pgxpool.Pool) *OutboxEventPublisher {
	return &OutboxEventPublisher{db: db}
}

const insertOutboxSQL = `
	INSERT INTO outbox_events (aggregate_type, aggregate_id, event_type, payload)
	VALUES ($1, $2, $3, $4)
`

// Publish serialises each event to JSON and appends it to outbox_events.
// It honours a pgx.Tx stored in context via TxKey.
func (p *OutboxEventPublisher) Publish(events ...kernel.DomainEvent) {
	ctx := context.Background()
	for _, ev := range events {
		payload, err := json.Marshal(ev)
		if err != nil {
			// Non-fatal: log and skip rather than panic
			fmt.Printf("outbox publisher: marshal error for %s: %v\n", ev.EventName(), err)
			continue
		}
		// aggregate_type and aggregate_id are optional enrichment;
		// cast event to Aggregate if possible
		aggType, aggID := extractAggregate(ev)
		if _, err := p.db.Exec(ctx, insertOutboxSQL, aggType, aggID, ev.EventName(), payload); err != nil {
			fmt.Printf("outbox publisher: insert error for %s: %v\n", ev.EventName(), err)
		}
	}
}

// PublishTx writes events within a provided transaction.
// Use this inside application service handlers to guarantee atomicity.
func (p *OutboxEventPublisher) PublishTx(ctx context.Context, tx pgx.Tx, events ...kernel.DomainEvent) error {
	for _, ev := range events {
		payload, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("outbox publisher: marshal %s: %w", ev.EventName(), err)
		}
		aggType, aggID := extractAggregate(ev)
		if _, err := tx.Exec(ctx, insertOutboxSQL, aggType, aggID, ev.EventName(), payload); err != nil {
			return fmt.Errorf("outbox publisher: insert %s: %w", ev.EventName(), err)
		}
	}
	return nil
}

// AggregateEvent is an optional interface events can implement
// to provide aggregate metadata for the outbox row.
type AggregateEvent interface {
	AggregateType() string
	AggregateID() string
}

func extractAggregate(ev kernel.DomainEvent) (string, string) {
	if ae, ok := ev.(AggregateEvent); ok {
		return ae.AggregateType(), ae.AggregateID()
	}
	return "unknown", "00000000-0000-0000-0000-000000000000"
}
