package domain

import (
	"errors"
	"time"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// FreshnessState represents the quality state of a batch.
type FreshnessState string

const (
	FreshnessFresh    FreshnessState = "fresh"
	FreshnessAging    FreshnessState = "aging"
	FreshnessCritical FreshnessState = "critical"
	FreshnessExpired  FreshnessState = "expired"
)

var (
	ErrBatchExpired         = errors.New("batch is expired and cannot be consumed")
	ErrInsufficientBatch    = errors.New("insufficient remaining quantity in batch")
	ErrBatchAlreadyExpired  = errors.New("batch is already expired")
)

// Batch is the aggregate root for a received shipment of a flower/material.
type Batch struct {
	id            kernel.BatchID
	storeID       kernel.StoreID
	ingredientID  kernel.IngredientID
	receivedQty   int
	remainingQty  int
	purchasePrice kernel.Money
	receivedAt    time.Time
	expiresAt     time.Time
	freshness     FreshnessState
	version       int
		writtenOff     bool
	writeOffReason string

	events []kernel.DomainEvent
}

func NewBatch(
	storeID kernel.StoreID,
	ingredientID kernel.IngredientID,
	qty int,
	price kernel.Money,
	receivedAt time.Time,
	expiresAt time.Time,
) *Batch {
	b := &Batch{
		id:            kernel.NewBatchID(),
		storeID:       storeID,
		ingredientID:  ingredientID,
		receivedQty:   qty,
		remainingQty:  qty,
		purchasePrice: price,
		receivedAt:    receivedAt,
		expiresAt:     expiresAt,
		freshness:     FreshnessFresh,
	}
	b.recordEvent(BatchReceivedEvent{
		BaseEvent:    kernel.NewBaseEvent("inventory.batch_received"),
		BatchID:      b.id,
		IngredientID: b.ingredientID,
		StoreID:      b.storeID,
		Qty:          qty,
	})
	return b
}

func (b *Batch) ID() kernel.BatchID               { return b.id }
func (b *Batch) StoreID() kernel.StoreID           { return b.storeID }
func (b *Batch) IngredientID() kernel.IngredientID { return b.ingredientID }
func (b *Batch) RemainingQty() int                 { return b.remainingQty }
func (b *Batch) Freshness() FreshnessState         { return b.freshness }
func (b *Batch) ExpiresAt() time.Time              { return b.expiresAt }
func (b *Batch) Version() int                      { return b.version }
func (b *Batch) IsUsable() bool                    { return b.freshness != FreshnessExpired }

// Consume reduces remaining quantity. Used during order confirmation or production.
func (b *Batch) Consume(qty int) error {
	if b.freshness == FreshnessExpired {
		return ErrBatchExpired
	}
	if qty > b.remainingQty {
		return ErrInsufficientBatch
	}
	b.remainingQty -= qty
		b.recordEvent(BatchConsumedEvent{
		BaseEvent:    kernel.NewBaseEvent("inventory.batch_consumed"),
		BatchID:      b.id,
		IngredientID: b.ingredientID,
		StoreID:      b.storeID,
		QtyConsumed:  qty,
		RemainingQty: b.remainingQty,
	})
	return nil
}

// Release restores quantity (e.g., on order cancellation).
func (b *Batch) Release(qty int) {
	b.remainingQty += qty
}

// MarkAging transitions freshness from fresh to aging.
func (b *Batch) MarkAging() {
	if b.freshness == FreshnessFresh {
		b.freshness = FreshnessAging
		b.recordEvent(BatchFreshnessChangedEvent{
			BaseEvent: kernel.NewBaseEvent("inventory.batch_freshness_changed"),
			BatchID:   b.id,
			NewState:  string(FreshnessAging),
		})
	}
}

// MarkCritical transitions freshness to critical.
func (b *Batch) MarkCritical() {
	if b.freshness == FreshnessAging || b.freshness == FreshnessFresh {
		b.freshness = FreshnessCritical
	}
}

// Expire marks the batch as expired, preventing further consumption.
func (b *Batch) Expire() error {
	if b.freshness == FreshnessExpired {
		return ErrBatchAlreadyExpired
	}
	b.freshness = FreshnessExpired
	b.recordEvent(BatchExpiredEvent{
		BaseEvent:    kernel.NewBaseEvent("inventory.batch_expired"),
		BatchID:      b.id,
		IngredientID: b.ingredientID,
		StoreID:      b.storeID,
		Wasted:       b.remainingQty,
	})
	return nil
}

// Writeoff manually removes a quantity from an usable batch (spoilage).
func (b *Batch) Writeoff(qty int, reason string) error {
	if b.freshness == FreshnessExpired {
		return ErrBatchExpired
	}
	if qty > b.remainingQty {
		return ErrInsufficientBatch
	}
	b.remainingQty -= qty
	b.recordEvent(BatchWrittenOffEvent{
		BaseEvent: kernel.NewBaseEvent("inventory.batch_written_off"),
		BatchID:   b.id,
		Qty:       qty,
		Reason:    reason,
	})
	return nil
}

func (b *Batch) PopEvents() []kernel.DomainEvent {
	events := b.events
	b.events = nil
	return events
}

func (b *Batch) recordEvent(e kernel.DomainEvent) {
	b.events = append(b.events, e)
}

// ErrBatchNotFound is returned when a batch cannot be found in the repository.
var ErrBatchNotFound = errors.New("batch not found")

// ErrOptimisticLock is returned when an optimistic lock conflict is detected.
var ErrOptimisticLock = errors.New("optimistic lock conflict: batch was modified concurrently")

// RehydrateParams holds all fields needed to reconstruct a Batch from storage.
type RehydrateParams struct {
	ID             kernel.BatchID
	StoreID        kernel.StoreID
	IngredientID   kernel.IngredientID
	ReceivedQty    int
	RemainingQty   int
	PurchasePrice  kernel.Money
	ReceivedAt     time.Time
	ExpiresAt      time.Time
	Freshness      FreshnessState
	WrittenOff     bool
	WriteOffReason string
	Version        int
}

// RehydrateBatch reconstructs a Batch aggregate from persisted state (no events emitted).
func RehydrateBatch(p RehydrateParams) *Batch {
	return &Batch{
		id:             p.ID,
		storeID:        p.StoreID,
		ingredientID:   p.IngredientID,
		receivedQty:    p.ReceivedQty,
		remainingQty:   p.RemainingQty,
		purchasePrice:  p.PurchasePrice,
		receivedAt:     p.ReceivedAt,
		expiresAt:      p.ExpiresAt,
		freshness:      p.Freshness,
		writtenOff:     p.WrittenOff,
		writeOffReason: p.WriteOffReason,
		version:        p.Version,
	}
}

// Additional accessors needed by infrastructure layer.
func (b *Batch) ReceivedQty() int        { return b.receivedQty }
func (b *Batch) PurchasePrice() kernel.Money { return b.purchasePrice }
func (b *Batch) ReceivedAt() time.Time   { return b.receivedAt }
func (b *Batch) IsWrittenOff() bool      { return b.writtenOff }
func (b *Batch) WriteOffReason() string  { return b.writeOffReason }
