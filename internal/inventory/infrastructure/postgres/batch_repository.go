package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dbaratey/florist-core/internal/inventory/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BatchRepository — Postgres-реализация domain.BatchRepository.
type BatchRepository struct {
	pool *pgxpool.Pool
}

func NewBatchRepository(pool *pgxpool.Pool) *BatchRepository {
	return &BatchRepository{pool: pool}
}

// ——— batchRow — scan DTO ———
type batchRow struct {
	ID             string
	StoreID        string
	IngredientID   string
	ReceivedQty    int32
	RemainingQty   int32
	PurchasePrice  int64
	Currency       string
	ReceivedAt     time.Time
	ExpiresAt      time.Time
	Freshness      string
	WrittenOff     bool
	WriteOffReason *string
	Version        int
}

const batchSelectCols = `
	id::text, store_id::text, ingredient_id::text,
	received_qty, remaining_qty,
	purchase_price, currency,
	received_at, expires_at,
	freshness, written_off, write_off_reason, version`

func scanBatch(row pgx.Row) (*domain.Batch, error) {
	var r batchRow
	err := row.Scan(
		&r.ID, &r.StoreID, &r.IngredientID,
		&r.ReceivedQty, &r.RemainingQty,
		&r.PurchasePrice, &r.Currency,
		&r.ReceivedAt, &r.ExpiresAt,
		&r.Freshness, &r.WrittenOff, &r.WriteOffReason, &r.Version,
	)
	if err != nil {
		return nil, err
	}
	return rowToBatch(r), nil
}

func scanBatches(rows pgx.Rows) ([]*domain.Batch, error) {
	defer rows.Close()
	var result []*domain.Batch
	for rows.Next() {
		var r batchRow
		if err := rows.Scan(
			&r.ID, &r.StoreID, &r.IngredientID,
			&r.ReceivedQty, &r.RemainingQty,
			&r.PurchasePrice, &r.Currency,
			&r.ReceivedAt, &r.ExpiresAt,
			&r.Freshness, &r.WrittenOff, &r.WriteOffReason, &r.Version,
		); err != nil {
			return nil, err
		}
		result = append(result, rowToBatch(r))
	}
	return result, rows.Err()
}

func rowToBatch(r batchRow) *domain.Batch {
	id, _ := kernel.ParseID(r.ID)
	storeID, _ := kernel.ParseID(r.StoreID)
	ingredientID, _ := kernel.ParseID(r.IngredientID)
	return domain.RehydrateBatch(domain.RehydrateParams{
		ID:             kernel.BatchID{ID: id},
		StoreID:        kernel.StoreID{ID: storeID},
		IngredientID:   kernel.IngredientID{ID: ingredientID},
		ReceivedQty:    int(r.ReceivedQty),
		RemainingQty:   int(r.RemainingQty),
		PurchasePrice:  kernel.NewMoney(r.PurchasePrice, r.Currency),
		ReceivedAt:     r.ReceivedAt,
		ExpiresAt:      r.ExpiresAt,
		Freshness:      domain.FreshnessState(r.Freshness),
		WrittenOff:     r.WrittenOff,
		WriteOffReason: ptrStr(r.WriteOffReason),
		Version:        r.Version,
	})
}

func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nilableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ——— Save (upsert + optimistic lock) ———
func (r *BatchRepository) Save(ctx context.Context, b *domain.Batch) error {
	const q = `
		INSERT INTO inventory_batches
			(id, store_id, ingredient_id, received_qty, remaining_qty,
			 purchase_price, currency, received_at, expires_at,
			 freshness, written_off, write_off_reason, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			remaining_qty    = EXCLUDED.remaining_qty,
			freshness        = EXCLUDED.freshness,
			written_off      = EXCLUDED.written_off,
			write_off_reason = EXCLUDED.write_off_reason,
			version          = inventory_batches.version + 1
		WHERE inventory_batches.version = $14
	`
	tag, err := r.pool.Exec(ctx, q,
		b.ID().ID.String(), b.StoreID().ID.String(), b.IngredientID().ID.String(),
		b.ReceivedQty(), b.RemainingQty(),
		b.PurchasePrice().Amount, b.PurchasePrice().Currency,
		b.ReceivedAt(), b.ExpiresAt(),
		string(b.Freshness()),
		b.IsWrittenOff(),
		nilableStr(b.WriteOffReason()),
		b.Version()+1,
		b.Version(), // WHERE version = $14
	)
	if err != nil {
		return fmt.Errorf("BatchRepository.Save: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("BatchRepository.Save: %w", domain.ErrOptimisticLock)
	}
	return nil
}

// ——— FindByID ———
func (r *BatchRepository) FindByID(ctx context.Context, id kernel.BatchID) (*domain.Batch, error) {
	q := `SELECT ` + batchSelectCols + ` FROM inventory_batches WHERE id = $1`
	b, err := scanBatch(r.pool.QueryRow(ctx, q, id.ID.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("BatchRepository.FindByID %s: %w", id.ID.String(), domain.ErrBatchNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindByID: %w", err)
	}
	return b, nil
}

// ——— FindActive ———
func (r *BatchRepository) FindActive(ctx context.Context) ([]*domain.Batch, error) {
	q := `SELECT ` + batchSelectCols + `
		FROM inventory_batches
		WHERE freshness != 'expired' AND remaining_qty > 0`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindActive: %w", err)
	}
	result, err := scanBatches(rows)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindActive scan: %w", err)
	}
	return result, nil
}

// ——— FindByStore ———
func (r *BatchRepository) FindByStore(ctx context.Context, storeID kernel.StoreID) ([]*domain.Batch, error) {
	q := `SELECT ` + batchSelectCols + `
		FROM inventory_batches
		WHERE store_id = $1 AND freshness != 'expired' AND remaining_qty > 0
		ORDER BY expires_at ASC`
	rows, err := r.pool.Query(ctx, q, storeID.ID.String())
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindByStore: %w", err)
	}
	result, err := scanBatches(rows)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindByStore scan: %w", err)
	}
	return result, nil
}

// ——— FindAvailableByIngredient (FEFO) ———
func (r *BatchRepository) FindAvailableByIngredient(
	ctx context.Context,
	storeID kernel.StoreID,
	ingredientID kernel.IngredientID,
) ([]*domain.Batch, error) {
	q := `SELECT ` + batchSelectCols + `
		FROM inventory_batches
		WHERE store_id = $1
		  AND ingredient_id = $2
		  AND NOT written_off
		  AND freshness != 'expired'
		  AND remaining_qty > 0
		ORDER BY expires_at ASC`
	rows, err := r.pool.Query(ctx, q, storeID.ID.String(), ingredientID.ID.String())
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindAvailableByIngredient: %w", err)
	}
	result, err := scanBatches(rows)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindAvailableByIngredient scan: %w", err)
	}
	return result, nil
}
