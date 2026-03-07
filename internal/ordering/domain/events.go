package domain

import "github.com/dbaratey/florist-core/internal/shared/kernel"

type OrderCreatedEvent struct {
	kernel.BaseEvent
	OrderID kernel.OrderID
	StoreID kernel.StoreID
}

type OrderConfirmedEvent struct {
	kernel.BaseEvent
	OrderID kernel.OrderID
}

type OrderCancelledEvent struct {
	kernel.BaseEvent
	OrderID    kernel.OrderID
	Reason     string
	PrevStatus string
}

type OrderShippedEvent struct {
	kernel.BaseEvent
	OrderID kernel.OrderID
}
