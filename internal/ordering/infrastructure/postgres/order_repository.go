package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrderRepository — Postgres-реализация порта ordering/application.OrderRepository
type OrderRepository struct {
	pool *pgxpool.Pool
}

func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

// ——— внутренние DTO ———

type orderRow struct {
	ID         [16]byte
	StoreID    [16]byte
	CustomerID [16]byte
	Status     string
	TotalPrice int64
	Currency   string
	Notes      *string
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type orderItemRow struct {
	ID           [16]byte
	OrderID      [16]byte
	ProductID    [16]byte
	IngredientID [16]byte
	Qty          int32
	Unit         string
	UnitPrice    int64
	Currency     string
	BatchID      *[16]byte
	Reserved     bool
}

// ——— Save ———

func (r *OrderRepository) Save(ctx context.Context, o *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("OrderRepository.Save begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const qOrder = `
		INSERT INTO orders
			(id, store_id, customer_id, status, total_price, currency, notes, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err = tx.Exec(ctx, qOrder,
		o.ID(), o.StoreID(), o.CustomerID(),
		string(o.Status()),
		o.TotalPrice().Amount, o.TotalPrice().Currency,
		nilableStr(o.Notes()),
		o.Version(),
	)
	if err != nil {
		return fmt.Errorf("OrderRepository.Save insert order: %w", err)
	}

	if err := r.upsertItems(ctx, tx, o); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ——— FindByID ———

func (r *OrderRepository) FindByID(ctx context.Context, id kernel.ID) (*domain.Order, error) {
	const q = `
		SELECT id, store_id, customer_id, status, total_price, currency,
			   notes, version, created_at, updated_at
		FROM orders WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, q, id)
	var rw orderRow
	err := row.Scan(
		&rw.ID, &rw.StoreID, &rw.CustomerID, &rw.Status,
		&rw.TotalPrice, &rw.Currency, &rw.Notes,
		&rw.Version, &rw.CreatedAt, &rw.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("OrderRepository.FindByID %s: %w", id, domain.ErrOrderNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("OrderRepository.FindByID: %w", err)
	}

	items, err := r.findItems(ctx, kernel.ID(rw.ID))
	if err != nil {
		return nil, err
	}

	return domain.RehydrateOrder(domain.RehydrateOrderParams{
		ID:         kernel.ID(rw.ID),
		StoreID:    kernel.ID(rw.StoreID),
		CustomerID: kernel.ID(rw.CustomerID),
		Status:     domain.OrderStatus(rw.Status),
		TotalPrice: kernel.NewMoney(rw.TotalPrice, rw.Currency),
		Notes:      ptrStr(rw.Notes),
		Items:      items,
		Version:    rw.Version,
	}), nil
}

// ——— Update (оптимистичная блокировка) ———

func (r *OrderRepository) Update(ctx context.Context, o *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("OrderRepository.Update begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `
		UPDATE orders SET
			status      = $1,
			total_price = $2,
			notes       = $3,
			version     = version + 1,
			updated_at  = NOW()
		WHERE id = $4 AND version = $5
	`
	tag, err := tx.Exec(ctx, q,
		string(o.Status()),
		o.TotalPrice().Amount,
		nilableStr(o.Notes()),
		o.ID(),
		o.Version(),
	)
	if err != nil {
		return fmt.Errorf("OrderRepository.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("OrderRepository.Update: %w", domain.ErrOptimisticLock)
	}

	if err := r.upsertItems(ctx, tx, o); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ——— вспомогательные ———

func (r *OrderRepository) findItems(ctx context.Context, orderID kernel.ID) ([]domain.OrderItem, error) {
	const q = `
		SELECT id, order_id, product_id, ingredient_id,
			   qty, unit, unit_price, currency, batch_id, reserved
		FROM order_items WHERE order_id = $1
	`
	rows, err := r.pool.Query(ctx, q, orderID)
	if err != nil {
		return nil, fmt.Errorf("OrderRepository.findItems: %w", err)
	}
	defer rows.Close()

	var items []domain.OrderItem
	for rows.Next() {
		var rw orderItemRow
		if err := rows.Scan(
			&rw.ID, &rw.OrderID, &rw.ProductID, &rw.IngredientID,
			&rw.Qty, &rw.Unit, &rw.UnitPrice, &rw.Currency,
			&rw.BatchID, &rw.Reserved,
		); err != nil {
			return nil, fmt.Errorf("OrderRepository.findItems scan: %w", err)
		}
		var batchID kernel.ID
		if rw.BatchID != nil {
			batchID = kernel.ID(*rw.BatchID)
		}
		items = append(items, domain.OrderItem{
			ID:           kernel.ID(rw.ID),
			ProductID:    kernel.ID(rw.ProductID),
			IngredientID: kernel.ID(rw.IngredientID),
			Qty:          int(rw.Qty),
			Unit:         rw.Unit,
			UnitPrice:    kernel.NewMoney(rw.UnitPrice, rw.Currency),
			BatchID:      batchID,
			Reserved:     rw.Reserved,
		})
	}
	return items, rows.Err()
}

func (r *OrderRepository) upsertItems(ctx context.Context, tx pgx.Tx, o *domain.Order) error {
	const q = `
		INSERT INTO order_items
			(id, order_id, product_id, ingredient_id, qty, unit, unit_price, currency, batch_id, reserved)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			batch_id = EXCLUDED.batch_id,
			reserved = EXCLUDED.reserved
	`
	for _, item := range o.Items() {
		var batchID interface{} = nil
		if item.BatchID != (kernel.ID{}) {
			batchID = item.BatchID
		}
		_, err := tx.Exec(ctx, q,
			item.ID, o.ID(),
			item.ProductID, item.IngredientID,
			item.Qty, item.Unit,
			item.UnitPrice.Amount, item.UnitPrice.Currency,
			batchID, item.Reserved,
		)
		if err != nil {
			return fmt.Errorf("OrderRepository.upsertItems: %w", err)
		}
	}
	return nil
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

	
	// ——— SaveTx — сохраняет заказ В ТРАНЗАКЦИЮ (не создаёт свою) ———
func (r *OrderRepository) SaveTx(ctx context.Context, tx pgx.Tx, o *domain.Order) error {
	const qOrder = `
		INSERT INTO orders
		(id, store_id, customer_id, status, total_price, currency, notes, version)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`
	_, err := tx.Exec(ctx, qOrder,
		o.ID(), o.StoreID(), o.CustomerID(),
		string(o.Status()),
		o.TotalPrice().Amount, o.TotalPrice().Currency,
		nilableStr(o.Notes()),
		o.Version(),
	)
	if err != nil {
		return fmt.Errorf("OrderRepository.SaveTx: %w", err)
	}
	return r.upsertItems(ctx, tx, o)
}

// ——— UpdateTx — обновляет заказ В ТРАНЗАКЦИЮ (не создаёт свою) ———
func (r *OrderRepository) UpdateTx(ctx context.Context, tx pgx.Tx, o *domain.Order) error {
	const q = `
		UPDATE orders SET
			status = $1,
			total_price = $2,
			notes = $3,
			version = version + 1,
			updated_at = NOW()
		WHERE id = $4 AND version = $5
	`
	tag, err := tx.Exec(ctx, q,
		string(o.Status()),
		o.TotalPrice().Amount,
		nilableStr(o.Notes()),
		o.ID(),
		o.Version(),
	)
	if err != nil {
		return fmt.Errorf("OrderRepository.UpdateTx: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("OrderRepository.UpdateTx: %w", domain.ErrOptimisticLock)
	}
	return r.upsertItems(ctx, tx, o)
}

// ——— RunTx — реализует TxRunner.RunTx для application-слоя ———
func (r *OrderRepository) RunTx(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("OrderRepository.RunTx begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}return &s
}
