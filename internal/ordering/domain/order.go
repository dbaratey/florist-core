package domain

import (
	"errors"
	"time"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// OrderStatus represents the lifecycle state of an Order.
type OrderStatus string

const (
	OrderStatusDraft       OrderStatus = "draft"
	OrderStatusConfirmed   OrderStatus = "confirmed"
	OrderStatusInProduction OrderStatus = "in_production"
	OrderStatusAssembled   OrderStatus = "assembled"
	OrderStatusShipped     OrderStatus = "shipped"
	OrderStatusCancelled   OrderStatus = "cancelled"
)

// Domain errors
var (
	ErrOrderNotDraft        = errors.New("order must be in draft status")
	ErrOrderNotConfirmed    = errors.New("order must be confirmed before production")
	ErrOrderAlreadyCancelled = errors.New("order is already cancelled")
	ErrOrderItemNotFound    = errors.New("order item not found")
	ErrEmptyOrder           = errors.New("order must have at least one item")
)

// OrderItem is a line item inside an Order.
type OrderItem struct {
	ID        kernel.ID
	ProductID kernel.ProductID
	Qty       int
	UnitPrice kernel.Money
}

func (i OrderItem) Total() kernel.Money {
	return i.UnitPrice.Multiply(int64(i.Qty))
}

// CustomerSnapshot stores customer info at the time of order creation.
type CustomerSnapshot struct {
	Name  string
	Phone string
	Email string
}

// Order is the aggregate root for the ordering bounded context.
type Order struct {
	id          kernel.OrderID
	storeID     kernel.StoreID
	customer    CustomerSnapshot
	items       []OrderItem
	status      OrderStatus
	version     int
	createdAt   time.Time
	confirmedAt *time.Time
	cancelledAt *time.Time

	events []kernel.DomainEvent
}

// NewOrder creates a new Order in draft status.
func NewOrder(storeID kernel.StoreID, customer CustomerSnapshot) *Order {
	now := time.Now().UTC()
	o := &Order{
		id:        kernel.NewOrderID(),
		storeID:   storeID,
		customer:  customer,
		status:    OrderStatusDraft,
		createdAt: now,
	}
	o.recordEvent(OrderCreatedEvent{BaseEvent: kernel.NewBaseEvent("order.created"), OrderID: o.id})
	return o
}

// Getters
func (o *Order) ID() kernel.OrderID       { return o.id }
func (o *Order) StoreID() kernel.StoreID  { return o.storeID }
func (o *Order) Status() OrderStatus      { return o.status }
func (o *Order) Items() []OrderItem       { return o.items }
func (o *Order) Version() int             { return o.version }
func (o *Order) Customer() CustomerSnapshot { return o.customer }

// AddItem adds a product line to the order. Only allowed in draft.
func (o *Order) AddItem(productID kernel.ProductID, qty int, unitPrice kernel.Money) error {
	if o.status != OrderStatusDraft {
		return ErrOrderNotDraft
	}
	o.items = append(o.items, OrderItem{
		ID:        kernel.NewID(),
		ProductID: productID,
		Qty:       qty,
		UnitPrice: unitPrice,
	})
	return nil
}

// RemoveItem removes a line item by product ID.
func (o *Order) RemoveItem(productID kernel.ProductID) error {
	if o.status != OrderStatusDraft {
		return ErrOrderNotDraft
	}
	for i, item := range o.items {
		if item.ProductID == productID {
			o.items = append(o.items[:i], o.items[i+1:]...)
			return nil
		}
	}
	return ErrOrderItemNotFound
}

// Confirm transitions the order from draft to confirmed.
func (o *Order) Confirm() error {
	if o.status != OrderStatusDraft {
		return ErrOrderNotDraft
	}
	if len(o.items) == 0 {
		return ErrEmptyOrder
	}
	now := time.Now().UTC()
	o.status = OrderStatusConfirmed
	o.confirmedAt = &now
	o.recordEvent(OrderConfirmedEvent{BaseEvent: kernel.NewBaseEvent("order.confirmed"), OrderID: o.id})
	return nil
}

// MarkInProduction transitions confirmed order to production.
func (o *Order) MarkInProduction() error {
	if o.status != OrderStatusConfirmed {
		return ErrOrderNotConfirmed
	}
	o.status = OrderStatusInProduction
	return nil
}

// MarkAssembled transitions the order after production is complete.
func (o *Order) MarkAssembled() error {
	if o.status != OrderStatusInProduction {
		return errors.New("order must be in production to mark as assembled")
	}
	o.status = OrderStatusAssembled
	return nil
}

// Ship marks the order as shipped/handed to customer.
func (o *Order) Ship() error {
	if o.status != OrderStatusAssembled {
		return errors.New("order must be assembled before shipping")
	}
	o.status = OrderStatusShipped
	return nil
}

// Cancel cancels the order. Allowed in draft or confirmed states.
func (o *Order) Cancel(reason string) error {
	if o.status == OrderStatusCancelled {
		return ErrOrderAlreadyCancelled
	}
	if o.status == OrderStatusShipped {
		return errors.New("cannot cancel a shipped order")
	}
	now := time.Now().UTC()
	o.status = OrderStatusCancelled
	o.cancelledAt = &now
	o.recordEvent(OrderCancelledEvent{
		BaseEvent: kernel.NewBaseEvent("order.cancelled"),
		OrderID:   o.id,
		Reason:    reason,
		PrevStatus: string(o.status),
	})
	return nil
}

// RequireDraft is a guard used by application layer.
func (o *Order) RequireDraft() error {
	if o.status != OrderStatusDraft {
		return ErrOrderNotDraft
	}
	return nil
}

// PopEvents returns and clears accumulated domain events.
func (o *Order) PopEvents() []kernel.DomainEvent {
	events := o.events
	o.events = nil
	return events
}

func (o *Order) recordEvent(e kernel.DomainEvent) {
	o.events = append(o.events, e)
}
