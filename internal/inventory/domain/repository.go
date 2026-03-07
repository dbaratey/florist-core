package domain

import (
	"context"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// BatchRepository defines persistence operations for the Batch aggregate.
type BatchRepository interface {
	// Save upserts a Batch (insert or update by ID with optimistic lock on version).
	Save(ctx context.Context, b *Batch) error

	// FindByID returns the batch with the given ID or ErrBatchNotFound.
	FindByID(ctx context.Context, id kernel.BatchID) (*Batch, error)

	// FindActive returns all batches that are not expired and have remaining qty > 0.
	FindActive(ctx context.Context) ([]*Batch, error)

	// FindByStore returns all active batches for a given store.
	FindByStore(ctx context.Context, storeID kernel.StoreID) ([]*Batch, error)

	// FindAvailableByIngredient returns usable batches for a store+ingredient ordered
	// by expiry ASC (FEFO — First Expired, First Out).
	FindAvailableByIngredient(
		ctx context.Context,
		storeID kernel.StoreID,
		ingredientID kernel.IngredientID,
	) ([]*Batch, error)
}
