package application

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// ConfirmOrderCommand — команда подтверждения заказа.
type ConfirmOrderCommand struct {
	OrderID string // UUID строкой, парсится в handler
}

// ConfirmOrderHandler оркестрирует подтверждение заказа:
// 1. Проверить наличие товара (AvailabilityService)
// 2. Зарезервировать инвентарь (ReservationService)
// 3. Перевести заказ в Confirmed + сохранить + outbox в одной tx.
type ConfirmOrderHandler struct {
	orders       OrderRepository
	availability AvailabilityService
	reservations ReservationService
	txRunner     TxRunner
	publisher    kernel.EventPublisher
}

func NewConfirmOrderHandler(
	orders OrderRepository,
	availability AvailabilityService,
	reservations ReservationService,
	txRunner TxRunner,
	publisher kernel.EventPublisher,
) *ConfirmOrderHandler {
	return &ConfirmOrderHandler{
		orders:       orders,
		availability: availability,
		reservations: reservations,
		txRunner:     txRunner,
		publisher:    publisher,
	}
}

func (h *ConfirmOrderHandler) Handle(ctx context.Context, cmd ConfirmOrderCommand) error {
	// Чтение заказа вне транзакции (нечистая читальная операция).
	orderID, err := kernel.ParseID(cmd.OrderID)
	if err != nil {
		return fmt.Errorf("ConfirmOrderHandler: invalid order id: %w", err)
	}

	order, err := h.orders.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	// Guard: заказ должен быть в статусе Draft.
	if order.Status() != domain.OrderStatusDraft {
		return fmt.Errorf("ConfirmOrderHandler: %w", domain.ErrOrderNotDraft)
	}

	// Проверка доступности по каждой позиции (вне tx — чистое чтение).
	for _, item := range order.Items() {
		res, err := h.availability.CheckProduct(ctx, order.StoreID(), item.ProductID, item.Qty)
		if err != nil {
			return err
		}
		if !res.CanConfirm() {
			return domain.ErrInsufficientStock
		}
	}

	// Атомарная часть: резерв + UPDATE заказа + outbox в одной транзакции.
	return h.txRunner.RunTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		// 1. Резервируем инвентарь.
		if err := h.reservations.ReserveForOrder(ctx, order.ID()); err != nil {
			return err
		}

		// 2. Доменный переход в Confirmed.
		if err := order.Confirm(); err != nil {
			return err
		}

		// 3. Сохраняем заказ в транзакции.
		if err := h.orders.UpdateTx(ctx, tx, order); err != nil {
			return err
		}

		// 4. Записываем outbox-события в ту же tx.
		events := order.PullEvents()
		if len(events) > 0 {
			if err := h.publisher.PublishTx(ctx, tx, events...); err != nil {
				return err
			}
		}

		// 5. RunTx коммитит tx после возврата nil.
		return nil
	})
}
