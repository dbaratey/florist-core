package application

import (
	"context"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// OrderRepository defines persistence operations for the Order aggregate.
type OrderRepository interface {
	GetByID(ctx context.Context, id kernel.OrderID) (*domain.Order, error)
	Save(ctx context.Context, order *domain.Order) error
}

// AvailabilityService checks whether a product can be fulfilled at a given store.
type AvailabilityService interface {
	CheckProduct(ctx context.Context, storeID kernel.StoreID, productID kernel.ProductID, qty int) (AvailabilityResult, error)
}

// AvailabilityStatus represents storefront availability state.
type AvailabilityStatus string

const (
	AvailableNow             AvailabilityStatus = "available_now"
	AvailableWithSubstitution AvailabilityStatus = "available_with_substitution"
	PreorderOnly             AvailabilityStatus = "preorder_only"
	Unavailable              AvailabilityStatus = "unavailable"
)

type AvailabilityResult struct {
	Status          AvailabilityStatus
	SubstitutionIDs []kernel.IngredientID
}

func (r AvailabilityResult) CanConfirm() bool {
	return r.Status == AvailableNow || r.Status == AvailableWithSubstitution
}

// ReservationService handles inventory reservation across batches.
type ReservationService interface {
	ReserveForOrder(ctx context.Context, orderID kernel.OrderID) error
	ReleaseForOrder(ctx context.Context, orderID kernel.OrderID) error
}

// UnitOfWork provides transactional boundary for application handlers.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}
