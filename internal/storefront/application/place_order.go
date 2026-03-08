package application

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	ordapp "github.com/dbaratey/florist-core/internal/ordering/application"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// PlaceOrderCommand — команда размещения заказа через витрину.
type PlaceOrderCommand struct {
	StoreID  string
	Items    []PlaceOrderItem
}

type PlaceOrderItem struct {
	ProductID string
	Qty       int
}

// PlaceOrderHandler создаёт черновик заказа + вызывает ConfirmOrderHandler.
type PlaceOrderHandler struct {
	orders   ordapp.OrderRepository
	txRunner ordapp.TxRunner
	confirm  *ordapp.ConfirmOrderHandler
}

func NewPlaceOrderHandler(
	orders ordapp.OrderRepository,
	txRunner ordapp.TxRunner,
	confirm *ordapp.ConfirmOrderHandler,
) *PlaceOrderHandler {
	return &PlaceOrderHandler{
		orders:   orders,
		txRunner: txRunner,
		confirm:  confirm,
	}
}

// Handle создаёт заказ-черновик и сразу подтверждает его.
func (h *PlaceOrderHandler) Handle(ctx context.Context, cmd PlaceOrderCommand) (kernel.ID, error) {
	storeID, err := kernel.ParseID(cmd.StoreID)
	if err != nil {
		return kernel.ID{}, fmt.Errorf("PlaceOrderHandler: invalid store id: %w", err)
	}

	// 1. Собираем позиции заказа.
	var items []domain.OrderItem
	for _, it := range cmd.Items {
		pid, err := kernel.ParseID(it.ProductID)
		if err != nil {
			return kernel.ID{}, fmt.Errorf("PlaceOrderHandler: invalid product id: %w", err)
		}
		items = append(items, domain.OrderItem{
			ProductID: pid,
			Qty:       it.Qty,
		})
	}

	// 2. Создаём заказ в статусе Draft.
	order := domain.NewOrder(storeID, items)

	var orderID kernel.ID
	err = h.txRunner.RunTx(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := h.orders.SaveTx(ctx, tx, order); err != nil {
			return err
		}
		orderID = order.ID()
		return nil
	})
	if err != nil {
		return kernel.ID{}, fmt.Errorf("PlaceOrderHandler: save draft: %w", err)
	}

	// 3. Подтверждаем заказ (проверка наличия + резервирование + outbox).
	confirmCmd := ordapp.ConfirmOrderCommand{OrderID: orderID.String()}
	if err := h.confirm.Handle(ctx, confirmCmd); err != nil {
		return kernel.ID{}, fmt.Errorf("PlaceOrderHandler: confirm: %w", err)
	}

	return orderID, nil
}
