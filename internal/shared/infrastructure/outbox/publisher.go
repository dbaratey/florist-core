package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher — реализует kernel.EventPublisher через Outbox-таблицу
// Событие записывается в ту же транзакцию, что и сам агрегат — at-least-once
type Publisher struct {
	pool *pgxpool.Pool
}

func NewPublisher(pool *pgxpool.Pool) *Publisher {
	return &Publisher{pool: pool}
}

// Publish записывает события в domain_events_outbox.
// Вызывать после сохранения агрегата в рамках одной транзакции.
func (p *Publisher) Publish(events ...kernel.DomainEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return p.PublishTx(ctx, nil, events...)
}

// PublishTx записывает события в рамках существующей pgx-транзакции.
// tx = nil — запускается без транзакции (редкий случай).
func (p *Publisher) PublishTx(ctx context.Context, tx pgx.Tx, events ...kernel.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	const query = `
		INSERT INTO domain_events_outbox
			(event_name, aggregate_id, payload, occurred_at)
		VALUES ($1, $2, $3, $4)
	`

	var querier interface {
		Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	}

	if tx != nil {
		querier = tx
	} else {
		querier = p.pool
	}

	for _, evt := range events {
		payload, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("outbox: marshal event %s: %w", evt.EventName(), err)
		}

		_, err = querier.Exec(ctx, query,
			evt.EventName(),
			evt.AggregateID(),
			payload,
			evt.OccurredAt(),
		)
		if err != nil {
			return fmt.Errorf("outbox: insert event %s: %w", evt.EventName(), err)
		}

		slog.Debug("outbox: event queued",
			"event", evt.EventName(),
			"aggregate_id", evt.AggregateID(),
		)
	}

	return nil
}
