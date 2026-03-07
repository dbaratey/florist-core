package application

import (
	"context"
	"time"

	"github.com/dbaratey/florist-core/internal/inventory/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// RecalcFreshnessHandler listens for a scheduled trigger and updates
// freshness state for all batches that have passed age thresholds.
type RecalcFreshnessHandler struct {
	repo domain.BatchRepository
}

func NewRecalcFreshnessHandler(repo domain.BatchRepository) *RecalcFreshnessHandler {
	return &RecalcFreshnessHandler{repo: repo}
}

// Handle is invoked periodically (e.g. by a cron job or scheduler).
// It loads all active batches and transitions freshness where needed.
func (h *RecalcFreshnessHandler) Handle(ctx context.Context) error {
	batches, err := h.repo.FindActive(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, b := range batches {
		updated := false

		switch {
		case now.After(b.ExpiresAt()):
			if err := b.Expire(); err != nil {
				// already expired – skip
				continue
			}
			updated = true

		case b.Freshness() == domain.FreshnessFresh &&
			now.After(b.ExpiresAt().Add(-kernel.AgingThreshold)):
			b.MarkAging()
			updated = true

		case b.Freshness() == domain.FreshnessAging &&
			now.After(b.ExpiresAt().Add(-kernel.CriticalThreshold)):
			b.MarkCritical()
			updated = true
		}

		if updated {
			if err := h.repo.Save(ctx, b); err != nil {
				return err
			}
		}
	}
	return nil
}
