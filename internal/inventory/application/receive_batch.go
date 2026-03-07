package application

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/dbaratey/florist-core/internal/inventory/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// TxRunner открывает pgx-транзакцию и передаёт её в fn.
// Реализуется в infrastructure-слое.
type TxRunner interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
}

// BatchRepository — порт для работы с хранилищем партий.
type BatchRepository interface {
	Save(ctx context.Context, b *domain.Batch) error
	FindByID(ctx context.Context, id kernel.ID) (*domain.Batch, error)
	FindAvailableByIngredient(ctx context.Context, storeID kernel.ID, ingredientID kernel.ID) ([]*domain.Batch, error)
	Update(ctx context.Context, b *domain.Batch) error
	// Tx-варианты: записывают в уже открытую транзакцию (без своего коммита).
	SaveTx(ctx context.Context, tx pgx.Tx, b *domain.Batch) error
	UpdateTx(ctx context.Context, tx pgx.Tx, b *domain.Batch) error
}

// --- ReceiveBatch ---

type ReceiveBatchCommand struct {
	StoreID      kernel.ID
	IngredientID kernel.ID
	Qty          int
	PriceKopecks int64
	Currency     string
	ReceivedAt   time.Time
	ExpiresAt    time.Time
}

type ReceiveBatchHandler struct {
	repo      BatchRepository
	txRunner  TxRunner
	publisher kernel.EventPublisher
}

func NewReceiveBatchHandler(repo BatchRepository, txRunner TxRunner, publisher kernel.EventPublisher) *ReceiveBatchHandler {
	return &ReceiveBatchHandler{repo: repo, txRunner: txRunner, publisher: publisher}
}

// Handle: сохраняет партию + outbox атомарно.
func (h *ReceiveBatchHandler) Handle(ctx context.Context, cmd ReceiveBatchCommand) (kernel.ID, error) {
	price := kernel.NewMoney(cmd.PriceKopecks, cmd.Currency)
	batch := domain.NewBatch(
		cmd.StoreID,
		cmd.IngredientID,
		cmd.Qty,
		price,
		cmd.ReceivedAt,
		cmd.ExpiresAt,
	)

	var batchID kernel.ID
	err := h.txRunner.RunTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := h.repo.SaveTx(ctx, tx, batch); err != nil {
			return err
		}
		events := batch.PullEvents()
		if len(events) > 0 {
			if err := h.publisher.PublishTx(ctx, tx, events...); err != nil {
				return err
			}
		}
		batchID = batch.ID()
		return nil
	})
	if err != nil {
		return kernel.ID{}, err
	}
	return batchID, nil
}

// --- ConsumeBatch ---

type ConsumeBatchCommand struct {
	BatchID kernel.ID
	Qty     int
}

type ConsumeBatchHandler struct {
	repo      BatchRepository
	txRunner  TxRunner
	publisher kernel.EventPublisher
}

func NewConsumeBatchHandler(repo BatchRepository, txRunner TxRunner, publisher kernel.EventPublisher) *ConsumeBatchHandler {
	return &ConsumeBatchHandler{repo: repo, txRunner: txRunner, publisher: publisher}
}

func (h *ConsumeBatchHandler) Handle(ctx context.Context, cmd ConsumeBatchCommand) error {
	batch, err := h.repo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return err
	}
	if err := batch.Consume(cmd.Qty); err != nil {
		return err
	}
	return h.txRunner.RunTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := h.repo.UpdateTx(ctx, tx, batch); err != nil {
			return err
		}
		events := batch.PullEvents()
		if len(events) > 0 {
			return h.publisher.PublishTx(ctx, tx, events...)
		}
		return nil
	})
}

// --- WriteOffBatch ---

type WriteOffBatchCommand struct {
	BatchID kernel.ID
	Reason  string
}

type WriteOffBatchHandler struct {
	repo      BatchRepository
	txRunner  TxRunner
	publisher kernel.EventPublisher
}

func NewWriteOffBatchHandler(repo BatchRepository, txRunner TxRunner, publisher kernel.EventPublisher) *WriteOffBatchHandler {
	return &WriteOffBatchHandler{repo: repo, txRunner: txRunner, publisher: publisher}
}

func (h *WriteOffBatchHandler) Handle(ctx context.Context, cmd WriteOffBatchCommand) error {
	batch, err := h.repo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return err
	}
	if err := batch.WriteOff(cmd.Reason); err != nil {
		return err
	}
	return h.txRunner.RunTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := h.repo.UpdateTx(ctx, tx, batch); err != nil {
			return err
		}
		events := batch.PullEvents()
		if len(events) > 0 {
			return h.publisher.PublishTx(ctx, tx, events...)
		}
		return nil
	})
}
