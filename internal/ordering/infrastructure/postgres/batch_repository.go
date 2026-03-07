package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BatchRepository — Postgres-реализация репозитория партий для ordering/application.BatchRepository
type BatchRepository struct {
	pool *pgxpool.Pool
}

func NewBatchRepository(pool *pgxpool.Pool) *BatchRepository {
	return &BatchRepository{pool: pool}
}

// ========== DTO ==========

type batchRow struct {
	ID          []byte
	SKU         string
	Qty         int
	ETA         sql.NullTime
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type batchAllocationRow struct {
	BatchID   []byte
	OrderID   []byte
	Qty       int
	CreatedAt time.Time
}

// ========== Conn ==========

func (r *BatchRepository) Save(ctx context.Context, b *domain.Batch) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("BatchRepository.Save begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const qBatch = `
		INSERT INTO batches
			(id, sku, qty, eta, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`
	now := time.Now()
	_, err = tx.Exec(ctx, qBatch,
		b.ID(), b.SKU, b.Qty(), nullableTime(b.ETA()), b.Version(), now, now,
	)
	if err != nil {
		return fmt.Errorf("BatchRepository.Save insert batch: %w", err)
	}

	if err := r.saveAllocations(ctx, tx, b); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *BatchRepository) SaveTx(ctx context.Context, tx pgx.Tx, b *domain.Batch) error {
	const qBatch = `
		INSERT INTO batches
			(id, sku, qty, eta, version, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
	`
	now := time.Now()
	_, err := tx.Exec(ctx, qBatch,
		b.ID(), b.SKU, b.Qty(), nullableTime(b.ETA()), b.Version(), now, now,
	)
	if err != nil {
		return fmt.Errorf("BatchRepository.SaveTx insert batch: %w", err)
	}

	return r.saveAllocations(ctx, tx, b)
}

func (r *BatchRepository) Update(ctx context.Context, b *domain.Batch) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("BatchRepository.Update begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := r.UpdateTx(ctx, tx, b); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *BatchRepository) UpdateTx(ctx context.Context, tx pgx.Tx, b *domain.Batch) error {
	const qBatch = `
		UPDATE batches SET
			sku = $2,
			qty = $3,
			eta = $4,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $1 AND version = $5
	`
	tag, err := tx.Exec(ctx, qBatch,
		b.ID(), b.SKU, b.Qty(), nullableTime(b.ETA()), b.Version(),
	)
	if err != nil {
		return fmt.Errorf("BatchRepository.UpdateTx: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("BatchRepository.UpdateTx: %w", domain.ErrOptimisticLock)
	}

	// Удаляем старые аллокации и сохраняем новые
	const qDel = `DELETE FROM batch_allocations WHERE batch_id = $1`
	if _, err := tx.Exec(ctx, qDel, b.ID()); err != nil {
		return fmt.Errorf("BatchRepository.UpdateTx delete allocations: %w", err)
	}

	return r.saveAllocations(ctx, tx, b)
}

func (r *BatchRepository) GetByID(ctx context.Context, id domain.BatchID) (*domain.Batch, error) {
	const qBatch = `
		SELECT id, sku, qty, eta, version, created_at, updated_at
		FROM batches
		WHERE id = $1
	`
	var row batchRow
	err := r.pool.QueryRow(ctx, qBatch, id).Scan(
		&row.ID, &row.SKU, &row.Qty, &row.ETA, &row.Version, &row.CreatedAt, &row.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.GetByID: %w", err)
	}

	const qAlloc = `
		SELECT batch_id, order_id, qty, created_at
		FROM batch_allocations
		WHERE batch_id = $1
	`
	rows, err := r.pool.Query(ctx, qAlloc, id)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.GetByID allocations: %w", err)
	}
	defer rows.Close()

	var allocs []batchAllocationRow
	for rows.Next() {
		var a batchAllocationRow
		if err := rows.Scan(&a.BatchID, &a.OrderID, &a.Qty, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("BatchRepository.GetByID scan alloc: %w", err)
		}
		allocs = append(allocs, a)
	}

	return r.rowToDomain(row, allocs)
}

func (r *BatchRepository) GetBySKU(ctx context.Context, sku string) ([]*domain.Batch, error) {
	const qBatch = `
		SELECT id, sku, qty, eta, version, created_at, updated_at
		FROM batches
		WHERE sku = $1
		ORDER BY eta NULLS LAST, created_at
	`
	rows, err := r.pool.Query(ctx, qBatch, sku)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.GetBySKU: %w", err)
	}
	defer rows.Close()

	var batchRows []batchRow
	for rows.Next() {
		var row batchRow
		if err := rows.Scan(&row.ID, &row.SKU, &row.Qty, &row.ETA, &row.Version, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, fmt.Errorf("BatchRepository.GetBySKU scan: %w", err)
		}
		batchRows = append(batchRows, row)
	}

	// Загружаем все аллокации одним запросом
	if len(batchRows) == 0 {
		return nil, nil
	}

	var batchIDs [][]byte
	for _, row := range batchRows {
		batchIDs = append(batchIDs, row.ID)
	}

	const qAlloc = `
		SELECT batch_id, order_id, qty, created_at
		FROM batch_allocations
		WHERE batch_id = ANY($1)
	`
	allocRows, err := r.pool.Query(ctx, qAlloc, batchIDs)
	if err != nil {
		return nil, fmt.Errorf("BatchRepository.GetBySKU allocations: %w", err)
	}
	defer allocRows.Close()

	// Группируем аллокации по batch_id
	allocsByBatch := make(map[string][]batchAllocationRow)
	for allocRows.Next() {
		var a batchAllocationRow
		if err := allocRows.Scan(&a.BatchID, &a.OrderID, &a.Qty, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("BatchRepository.GetBySKU scan alloc: %w", err)
		}
		key := string(a.BatchID)
		allocsByBatch[key] = append(allocsByBatch[key], a)
	}

	// Конвертируем в domain
	var batches []*domain.Batch
	for _, row := range batchRows {
		allocs := allocsByBatch[string(row.ID)]
		b, err := r.rowToDomain(row, allocs)
		if err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}

	return batches, nil
}

func (r *BatchRepository) RunTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("BatchRepository.RunTx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := fn(ctx, tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ========== Helpers ==========

func (r *BatchRepository) saveAllocations(ctx context.Context, tx pgx.Tx, b *domain.Batch) error {
	if len(b.Allocations()) == 0 {
		return nil
	}

	const qAlloc = `
		INSERT INTO batch_allocations
			(batch_id, order_id, qty, created_at)
		VALUES ($1,$2,$3,$4)
	`
	now := time.Now()
	for orderID, qty := range b.Allocations() {
		_, err := tx.Exec(ctx, qAlloc, b.ID(), []byte(orderID), qty, now)
		if err != nil {
			return fmt.Errorf("BatchRepository.saveAllocations: %w", err)
		}
	}
	return nil
}

func (r *BatchRepository) rowToDomain(row batchRow, allocs []batchAllocationRow) (*domain.Batch, error) {
	var eta *time.Time
	if row.ETA.Valid {
		eta = &row.ETA.Time
	}

	allocMap := make(map[domain.OrderID]int)
	for _, a := range allocs {
		allocMap[domain.OrderID(a.OrderID)] = a.Qty
	}

	return domain.ReconstructBatch(
		domain.BatchID(row.ID),
		row.SKU,
		row.Qty,
		eta,
		allocMap,
		row.Version,
	), nil
}

func nullableTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
