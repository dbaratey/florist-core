package domain

import (
	"errors"
	"time"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// BatchID — уникальный идентификатор партии.
type BatchID []byte

func NewBatchID() BatchID {
	return BatchID(kernel.NewID())
}

// OrderID — уникальный идентификатор заказа.
type OrderID []byte

// Domain errors
var (
	ErrOptimisticLock = errors.New("optimistic lock version mismatch")
	ErrInsufficientQty = errors.New("insufficient quantity available in batch")
	ErrOrderNotAllocated = errors.New("order is not allocated to this batch")
)

// Batch — агрегат партии товара (SKU) для модели аллокации.
// Поддерживает выделение заказов в определенном количестве.
type Batch struct {
	id BatchID
	SKU string // товарный код
	qty int // доступное количество
	eta *time.Time // ожидаемое время прибытия (ETA)
	allocations map[OrderID]int // выделенные количества по заказам
	version int // для оптимистичной блокировки
}

// NewBatch создает новую партию товара.
func NewBatch(sku string, qty int, eta *time.Time) *Batch {
	return &Batch{
		id: NewBatchID(),
		SKU: sku,
		qty: qty,
		eta: eta,
		allocations: make(map[OrderID]int),
		version: 0,
	}
}

// ReconstructBatch восстанавливает агрегат из хранилища.
func ReconstructBatch(
	id BatchID,
	sku string,
	qty int,
	eta *time.Time,
	allocations map[OrderID]int,
	version int,
) *Batch {
	if allocations == nil {
		allocations = make(map[OrderID]int)
	}
	return &Batch{
		id: id,
		SKU: sku,
		qty: qty,
		eta: eta,
		allocations: allocations,
		version: version,
	}
}

// Getters
func (b *Batch) ID() BatchID { return b.id }
func (b *Batch) Qty() int { return b.qty }
func (b *Batch) ETA() *time.Time { return b.eta }
func (b *Batch) Version() int { return b.version }
func (b *Batch) Allocations() map[OrderID]int {
	// Возвращаем копию для инкапсуляции
	copy := make(map[OrderID]int, len(b.allocations))
	for k, v := range b.allocations {
		copy[k] = v
	}
	return copy
}

// AvailableQty возвращает свободное (невыделенное) количество.
func (b *Batch) AvailableQty() int {
	allocated := 0
	for _, qty := range b.allocations {
		allocated += qty
	}
	return b.qty - allocated
}

// CanAllocate проверяет, можно ли выделить нужное количество.
func (b *Batch) CanAllocate(qty int) bool {
	return b.AvailableQty() >= qty
}

// Allocate выделяет количество для заказа.
func (b *Batch) Allocate(orderID OrderID, qty int) error {
	if !b.CanAllocate(qty) {
		return ErrInsufficientQty
	}
	b.allocations[orderID] = qty
	return nil
}

// Deallocate освобождает выделенное количество для заказа.
func (b *Batch) Deallocate(orderID OrderID) error {
	if _, exists := b.allocations[orderID]; !exists {
		return ErrOrderNotAllocated
	}
	delete(b.allocations, orderID)
	return nil
}
