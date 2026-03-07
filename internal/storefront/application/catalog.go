package application

import (
	"context"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// CatalogItem represents a single available batch in the storefront catalog.
type CatalogItem struct {
	BatchID      string `json:"batch_id"`
	IngredientID string `json:"ingredient_id"`
	StoreID      string `json:"store_id"`
	RemainingQty int    `json:"remaining_qty"`
	Freshness    string `json:"freshness"`
	ExpiresAt    string `json:"expires_at"`
	PriceCopecks int64  `json:"price_copecks"`
	Currency     string `json:"currency"`
}

// BatchCatalogReader is the read-model interface used by the storefront.
type BatchCatalogReader interface {
	FindAvailableByIngredient(ctx context.Context, storeID kernel.ID, ingredientID kernel.ID) ([]BatchCatalogRow, error)
	FindAllAvailable(ctx context.Context, storeID kernel.ID) ([]BatchCatalogRow, error)
}

// BatchCatalogRow is the raw data row returned from the read-model.
type BatchCatalogRow struct {
	BatchID      string
	IngredientID string
	StoreID      string
	RemainingQty int
	Freshness    string
	ExpiresAt    string
	PriceCopecks int64
	Currency     string
}

// GetCatalogCommand is the input for the catalog query.
type GetCatalogCommand struct {
	StoreID      string
	IngredientID string // optional filter
}

// GetCatalogHandler reads available batches for a given store.
type GetCatalogHandler struct {
	reader BatchCatalogReader
}

func NewGetCatalogHandler(reader BatchCatalogReader) *GetCatalogHandler {
	return &GetCatalogHandler{reader: reader}
}

func (h *GetCatalogHandler) Handle(ctx context.Context, cmd GetCatalogCommand) ([]CatalogItem, error) {
	storeID := kernel.ID(cmd.StoreID)

	var rows []BatchCatalogRow
	var err error

	if cmd.IngredientID != "" {
		rows, err = h.reader.FindAvailableByIngredient(ctx, storeID, kernel.ID(cmd.IngredientID))
	} else {
		rows, err = h.reader.FindAllAvailable(ctx, storeID)
	}
	if err != nil {
		return nil, err
	}

	items := make([]CatalogItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, CatalogItem{
			BatchID:      r.BatchID,
			IngredientID: r.IngredientID,
			StoreID:      r.StoreID,
			RemainingQty: r.RemainingQty,
			Freshness:    r.Freshness,
			ExpiresAt:    r.ExpiresAt,
			PriceCopecks: r.PriceCopecks,
			Currency:     r.Currency,
		})
	}
	return items, nil
}
