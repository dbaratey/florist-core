package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	storefrontapp "github.com/dbaratey/florist-core/internal/storefront/application"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// CatalogReader implements storefront/application.BatchCatalogReader using postgres.
type CatalogReader struct {
	pool *pgxpool.Pool
}

func NewCatalogReader(pool *pgxpool.Pool) *CatalogReader {
	return &CatalogReader{pool: pool}
}

// FindAllAvailable returns all non-expired, non-written-off batches for a store.
func (r *CatalogReader) FindAllAvailable(ctx context.Context, storeID kernel.ID) ([]storefrontapp.BatchCatalogRow, error) {
	query := `
		SELECT
			id,
			ingredient_id,
			store_id,
			remaining_qty,
			freshness,
			to_char(expires_at, 'YYYY-MM-DD') AS expires_at,
			purchase_price_copecks,
			purchase_price_currency
		FROM inventory_batches
		WHERE store_id = $1
		  AND written_off = false
		  AND freshness != 'expired'
		  AND remaining_qty > 0
		ORDER BY expires_at ASC
	`
	return r.scanRows(ctx, query, storeID)
}

// FindAvailableByIngredient returns available batches filtered by ingredient.
func (r *CatalogReader) FindAvailableByIngredient(ctx context.Context, storeID kernel.ID, ingredientID kernel.ID) ([]storefrontapp.BatchCatalogRow, error) {
	query := `
		SELECT
			id,
			ingredient_id,
			store_id,
			remaining_qty,
			freshness,
			to_char(expires_at, 'YYYY-MM-DD') AS expires_at,
			purchase_price_copecks,
			purchase_price_currency
		FROM inventory_batches
		WHERE store_id = $1
		  AND ingredient_id = $2
		  AND written_off = false
		  AND freshness != 'expired'
		  AND remaining_qty > 0
		ORDER BY expires_at ASC
	`
	return r.scanRows(ctx, query, storeID, ingredientID)
}

func (r *CatalogReader) scanRows(ctx context.Context, query string, args ...any) ([]storefrontapp.BatchCatalogRow, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("catalog query: %w", err)
	}
	defer rows.Close()

	var result []storefrontapp.BatchCatalogRow
	for rows.Next() {
		var row storefrontapp.BatchCatalogRow
		if err := rows.Scan(
			&row.BatchID,
			&row.IngredientID,
			&row.StoreID,
			&row.RemainingQty,
			&row.Freshness,
			&row.ExpiresAt,
			&row.PriceCopecks,
			&row.Currency,
		); err != nil {
			return nil, fmt.Errorf("catalog scan: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}
