package application

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/dbaratey/florist-core/internal/ordering/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// TxRunner открывает pgx-транзакцию и передаёт её в fn.
// Реализуется в infrastructure-слое (pgxpool.Pool).
type TxRunner interface {
	RunTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error
}

// OrderRepository — порт для работы с агрегатом Order.
type OrderRepository interface {
	FindByID(ctx context.Context, id kernel.ID) (*domain.Order, error)
	Save(ctx context.Context, o *domain.Order) error
	Update(ctx context.Context, o *domain.Order) error
	// Tx-варианты: записывают в уже открытую транзакцию (без своего коммита).
	SaveTx(ctx context.Context, tx pgx.Tx, o *domain.Order) error
	UpdateTx(ctx context.Context, tx pgx.Tx, o *domain.Order) error
}

// AvailabilityService проверяет доступность продукта в магазине.
type AvailabilityService interface {
	CheckProduct(ctx context.Context, storeID kernel.ID, productID kernel.ID, qty int) (AvailabilityResult, error)
}

// AvailabilityStatus представляет статус витрины.
type AvailabilityStatus string

const (
	AvailableNow             AvailabilityStatus = "available_now"
	AvailableWithSubstitution AvailabilityStatus = "available_with_substitution"
	PreorderOnly             AvailabilityStatus = "preorder_only"
	Unavailable              AvailabilityStatus = "unavailable"
)

type AvailabilityResult struct {
	Status          AvailabilityStatus
	SubstitutionIDs []kernel.ID
}

func (r AvailabilityResult) CanConfirm() bool {
	return r.Status == AvailableNow || r.Status == AvailableWithSubstitution
}

// ReservationService резервирует инвентарь для заказа.
type ReservationService interface {
	ReserveForOrder(ctx context.Context, orderID kernel.ID) error
	ReleaseForOrder(ctx context.Context, orderID kernel.ID) error
}
