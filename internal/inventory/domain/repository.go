package domain

import "context"

// BatchRepository defines persistence operations for the Batch aggregate.
type BatchRepository interface {
	// Save persists a new or updated Batch (upsert by ID + version check).
	Save(ctx context.Context, b *Batch) error

	// FindByID returns the batch with the given ID or an error if not found.
	FindByID(ctx context.Context, id interface{}) (*Batch, error)

	// FindActive returns all batches that are not expired and have remaining qty > 0.
	FindActive(ctx context.Context) ([]*Batch, error)

	// FindByStore returns all active batches for a given store.
	FindByStore(ctx context.Context, storeID interface{}) ([]*Batch, error)
}
