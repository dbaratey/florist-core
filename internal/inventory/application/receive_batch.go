package application

import (
	"context"
	"time"

	"github.com/dbaratey/florist-core/internal/inventory/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// BatchRepository — порт для работы с хранилищем партий
type BatchRepository interface {
	Save(ctx context.Context, b *domain.Batch) error
	FindByID(ctx context.Context, id kernel.ID) (*domain.Batch, error)
	FindAvailableByIngredient(ctx context.Context, storeID kernel.ID, ingredientID kernel.ID) ([]*domain.Batch, error)
	Update(ctx context.Context, b *domain.Batch) error
}

// ReceiveBatchCommand — команда приёмки новой партии товара
type ReceiveBatchCommand struct {
	StoreID      kernel.ID
	IngredientID kernel.ID
	Qty          int
	PriceKopecks int64
	Currency     string
	ReceivedAt   time.Time
	ExpiresAt    time.Time
}

// ReceiveBatchHandler — use case: принять партию товара на склад
type ReceiveBatchHandler struct {
	repo      BatchRepository
	publisher kernel.EventPublisher
}

func NewReceiveBatchHandler(repo BatchRepository, publisher kernel.EventPublisher) *ReceiveBatchHandler {
	return &ReceiveBatchHandler{repo: repo, publisher: publisher}
}

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

	if err := h.repo.Save(ctx, batch); err != nil {
		return kernel.ID{}, err
	}

	events := batch.PullEvents()
	if len(events) > 0 {
		if err := h.publisher.Publish(events...); err != nil {
			return kernel.ID{}, err
		}
	}

	return batch.ID(), nil
}

// ConsumeBatchCommand — команда списания из партии (FEFO)
type ConsumeBatchCommand struct {
	BatchID kernel.ID
	Qty     int
}

type ConsumeBatchHandler struct {
	repo      BatchRepository
	publisher kernel.EventPublisher
}

func NewConsumeBatchHandler(repo BatchRepository, publisher kernel.EventPublisher) *ConsumeBatchHandler {
	return &ConsumeBatchHandler{repo: repo, publisher: publisher}
}

func (h *ConsumeBatchHandler) Handle(ctx context.Context, cmd ConsumeBatchCommand) error {
	batch, err := h.repo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return err
	}

	if err := batch.Consume(cmd.Qty); err != nil {
		return err
	}

	if err := h.repo.Update(ctx, batch); err != nil {
		return err
	}

	events := batch.PullEvents()
	if len(events) > 0 {
		return h.publisher.Publish(events...)
	}
	return nil
}

// ExpireBatchesCommand — помечает просроченные партии
type ExpireBatchesHandler struct {
	repo      BatchRepository
	publisher kernel.EventPublisher
}

func NewExpireBatchesHandler(repo BatchRepository, publisher kernel.EventPublisher) *ExpireBatchesHandler {
	return &ExpireBatchesHandler{repo: repo, publisher: publisher}
}

// WrittenOffBatchCommand — принудительное списание (порча, повреждение)
type WriteOffBatchCommand struct {
	BatchID kernel.ID
	Reason  string
}

type WriteOffBatchHandler struct {
	repo      BatchRepository
	publisher kernel.EventPublisher
}

func NewWriteOffBatchHandler(repo BatchRepository, publisher kernel.EventPublisher) *WriteOffBatchHandler {
	return &WriteOffBatchHandler{repo: repo, publisher: publisher}
}

func (h *WriteOffBatchHandler) Handle(ctx context.Context, cmd WriteOffBatchCommand) error {
	batch, err := h.repo.FindByID(ctx, cmd.BatchID)
	if err != nil {
		return err
	}

	if err := batch.WriteOff(cmd.Reason); err != nil {
		return err
	}

	if err := h.repo.Update(ctx, batch); err != nil {
		return err
	}

	events := batch.PullEvents()
	if len(events) > 0 {
		return h.publisher.Publish(events...)
	}
	return nil
}
