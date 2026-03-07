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

// BatchRepository — Postgres-реализация порта inventory/application.BatchRepository
type BatchRepository struct {
	pool *pgxpool.Pool
}

func NewBatchRepository(pool *pgxpool.Pool) *BatchRepository {
	return &BatchRepository{pool: pool}
}

// ——— batchRow — внутренний DTO для scan ———

type batchRow struct {
	ID             [16]byte
	StoreID        [16]byte
	IngredientID   [16]byte
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

func rowToBatch(r batchRow) *domain.Batch {
	return domain.RehydrateBatch(domain.RehydrateParams{
		ID:             kernel.ID(r.ID),
		StoreID:        kernel.ID(r.StoreID),
		IngredientID:   kernel.ID(r.IngredientID),
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

// ——— Save ———

func (r *BatchRepository) Save(ctx context.Context, b *domain.Batch) error {
	const q = `
		INSERT INTO inventory_batches
			(id, store_id, ingredient_id, received_qty, remaining_qty,
			 purchase_price, currency, received_at, expires_at, freshness, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, q,
		b.ID(), b.StoreID(), b.IngredientID(),
		b.ReceivedQty(), b.RemainingQty(),
		b.PurchasePrice().Amount, b.PurchasePrice().Currency,
		b.ReceivedAt(), b.ExpiresAt(),
		string(b.Freshness()),
		b.Version(),
	)
	if err != nil {
		return fmt.Errorf("BatchRepository.Save: %w", err)
	}
	return nil
}

// ——— FindByID ———

func (r *BatchRepository) FindByID(ctx context.Context, id kernel.ID) (*domain.Batch, error) {
	const q = `
		SELECT id, store_id, ingredient_id, received_qty, remaining_qty,
			   purchase_price, currency, received_at, expires_at,
			   freshness, written_off, write_off_reason, version
		FROM inventory_batches
		WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	var rw batchRow
	err := row.Scan(
		&rw.ID, &rw.StoreID, &rw.IngredientID,
		&rw.ReceivedQty, &rw.RemainingQty,
		&rw.PurchasePrice, &rw.Currency,
		&rw.ReceivedAt, &rw.ExpiresAt,
		&rw.Freshness, &rw.WrittenOff, &rw.WriteOffReason, &rw.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("BatchRepository.FindByID %s: %w", id, domain.ErrBatchNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindByID: %w", err)
	}
	return rowToBatch(rw), nil
}

// ——— FindAvailableByIngredient (FEFO — First Expired, First Out) ———

func (r *BatchRepository) FindAvailableByIngredient(
	ctx context.Context,
	storeID kernel.ID,
	ingredientID kernel.ID,
) ([]*domain.Batch, error) {
	const q = `
		SELECT id, store_id, ingredient_id, received_qty, remaining_qty,
			   purchase_price, currency, received_at, expires_at,
			   freshness, written_off, write_off_reason, version
		FROM inventory_batches
		WHERE store_id = $1
		  AND ingredient_id = $2
		  AND NOT written_off
		  AND freshness != 'expired'
		  AND remaining_qty > 0
		ORDER BY expires_at ASC
	`
	rows, err := r.pool.Query(ctx, q, storeID, ingredientID)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.FindAvailableByIngredient: %w", err)
	}
	defer rows.Close()

	var result []*domain.Batch
	for rows.Next() {
		var rw batchRow
		if err := rows.Scan(
			&rw.ID, &rw.StoreID, &rw.IngredientID,
			&rw.ReceivedQty, &rw.RemainingQty,
			&rw.PurchasePrice, &rw.Currency,
			&rw.ReceivedAt, &rw.ExpiresAt,
			&rw.Freshness, &rw.WrittenOff, &rw.WriteOffReason, &rw.Version,
		); err != nil {
			return nil, fmt.Errorf("BatchRepository.FindAvailableByIngredient scan: %w", err)
		}
		result = append(result, rowToBatch(rw))
	}
	return result, rows.Err()
}

// ——— Update (оптимистичная блокировка через version) ———

func (r *BatchRepository) Update(ctx context.Context, b *domain.Batch) error {
	const q = `
		UPDATE inventory_batches SET
			remaining_qty    = $1,
			freshness        = $2,
			written_off      = $3,
			write_off_reason = $4,
			version          = version + 1,
			updated_at       = NOW()
		WHERE id = $5 AND version = $6
	`
	tag, err := r.pool.Exec(ctx, q,
		b.RemainingQty(),
		string(b.Freshness()),
		b.IsWrittenOff(),
		nilableStr(b.WriteOffReason()),
		b.ID(),
		b.Version(),
	)
	if err != nil {
		return fmt.Errorf("BatchRepository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("BatchRepository.Update: %w", domain.ErrOptimisticLock)
	}
	return nil
}

func nilableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
