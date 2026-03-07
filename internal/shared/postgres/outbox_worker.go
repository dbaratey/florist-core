package postgres

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EventHandler is called for each outbox event.
// Implementations should publish the event to a bus or process it directly.
type EventHandler func(ctx context.Context, eventType string, payload json.RawMessage) error

// OutboxWorker polls the outbox_events table and dispatches pending events.
type OutboxWorker struct {
	db          *pgxpool.Pool
	handler     EventHandler
	interval    time.Duration
	maxAttempts int
	logger      *slog.Logger
}

// NewOutboxWorker creates a new OutboxWorker.
// interval: how often to poll (e.g. 2*time.Second)
// maxAttempts: give up after this many failures per event
func NewOutboxWorker(db *pgxpool.Pool, handler EventHandler, interval time.Duration, maxAttempts int, logger *slog.Logger) *OutboxWorker {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &OutboxWorker{
		db:          db,
		handler:     handler,
		interval:    interval,
		maxAttempts: maxAttempts,
		logger:      logger,
	}
}

// Run starts the polling loop. Cancel ctx to stop.
func (w *OutboxWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("outbox worker stopped")
			return
		case <-ticker.C:
			w.processOnce(ctx)
		}
	}
}

type outboxRow struct {
	ID          string
	EventType   string
	Payload     json.RawMessage
	Attempts    int
	MaxAttempts int
}

func (w *OutboxWorker) processOnce(ctx context.Context) {
	// Claim a batch of pending events atomically
	rows, err := w.db.Query(ctx, `
		UPDATE outbox_events
		SET status = 'processing', attempts = attempts + 1
		WHERE id IN (
			SELECT id FROM outbox_events
			WHERE status = 'pending'
			  AND attempts < max_attempts
			ORDER BY created_at
			LIMIT 50
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, event_type, payload, attempts, max_attempts
	`)
	if err != nil {
		w.logger.Error("outbox: query failed", "error", err)
		return
	}
	defer rows.Close()

	var events []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.ID, &r.EventType, &r.Payload, &r.Attempts, &r.MaxAttempts); err != nil {
			w.logger.Error("outbox: scan failed", "error", err)
			continue
		}
		events = append(events, r)
	}
	rows.Close()

	for _, ev := range events {
		if err := w.handler(ctx, ev.EventType, ev.Payload); err != nil {
			w.logger.Warn("outbox: handler failed", "event_id", ev.ID, "attempts", ev.Attempts, "error", err)
			newStatus := "pending"
			if ev.Attempts >= ev.MaxAttempts {
				newStatus = "failed"
			}
			w.db.Exec(ctx, `UPDATE outbox_events SET status = $1 WHERE id = $2`, newStatus, ev.ID) //nolint:errcheck
		} else {
			w.db.Exec(ctx, `UPDATE outbox_events SET status = 'done', processed_at = NOW() WHERE id = $1`, ev.ID) //nolint:errcheck
		}
	}
}
