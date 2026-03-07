package application

import (
	"context"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// ConfirmOrderCmd is the command input for confirming an order.
type ConfirmOrderCmd struct {
	OrderID kernel.OrderID
}

// ConfirmOrderHandler orchestrates order confirmation:
// checks availability, reserves inventory, then confirms the order.
type ConfirmOrderHandler struct {
	orders       OrderRepository
	availability AvailabilityService
	reservations ReservationService
	uow          UnitOfWork
	publisher    kernel.EventPublisher
}

func NewConfirmOrderHandler(
	orders OrderRepository,
	availability AvailabilityService,
	reservations ReservationService,
	uow UnitOfWork,
	publisher kernel.EventPublisher,
) *ConfirmOrderHandler {
	return &ConfirmOrderHandler{
		orders:       orders,
		availability: availability,
		reservations: reservations,
		uow:          uow,
		publisher:    publisher,
	}
}

func (h *ConfirmOrderHandler) Handle(ctx context.Context, cmd ConfirmOrderCmd) error {
	return h.uow.Do(ctx, func(ctx context.Context) error {
		order, err := h.orders.GetByID(ctx, cmd.OrderID)
		if err != nil {
			return err
		}

		// Guard: order must be in draft
		if err := order.RequireDraft(); err != nil {
			return err
		}

		// Check availability for each item
		for _, item := range order.Items() {
			res, err := h.availability.CheckProduct(ctx, order.StoreID(), item.ProductID, item.Qty)
			if err != nil {
				return err
			}
			if !res.CanConfirm() {
				return domain.ErrInsufficientStock
			}
		}

		// Reserve inventory for the order
		if err := h.reservations.ReserveForOrder(ctx, order.ID()); err != nil {
			return err
		}

		// Transition order to confirmed
		if err := order.Confirm(); err != nil {
			return err
		}

		// Persist
		if err := h.orders.Save(ctx, order); err != nil {
			return err
		}

		// Publish domain events
		if h.publisher != nil {
			h.publisher.Publish(order.PopEvents()...)
		}

		return nil
	})
}
