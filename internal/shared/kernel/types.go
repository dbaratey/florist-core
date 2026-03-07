package kernel

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ID — универсальный идентификатор сущности
type ID = uuid.UUID

func NewID() ID {
	return uuid.New()
}

func ParseID(s string) (ID, error) {
	return uuid.Parse(s)
}

// Money — value object для денег (копейки)
type Money struct {
	Amount   int64  // в копейках
	Currency string // ISO 4217: "RUB", "USD"
}

func NewMoney(amount int64, currency string) Money {
	return Money{Amount: amount, Currency: currency}
}

func (m Money) Add(other Money) (Money, error) {
	if m.Currency != other.Currency {
		return Money{}, errors.New("currency mismatch")
	}
	return Money{Amount: m.Amount + other.Amount, Currency: m.Currency}, nil
}

func (m Money) Multiply(factor int64) Money {
	return Money{Amount: m.Amount * factor, Currency: m.Currency}
}

func (m Money) IsZero() bool {
	return m.Amount == 0
}

func (m Money) IsNegative() bool {
	return m.Amount < 0
}

// DomainEvent — базовый интерфейс для всех доменных событий
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
	AggregateID() ID
}

// BaseEvent — встраиваемая база для событий
type BaseEvent struct {
	Name        string
	AggID       ID
	OccurredAtT time.Time
}

func NewBaseEvent(name string, aggID ID) BaseEvent {
	return BaseEvent{
		Name:        name,
		AggID:       aggID,
		OccurredAtT: time.Now().UTC(),
	}
}

func (e BaseEvent) EventName() string    { return e.Name }
func (e BaseEvent) OccurredAt() time.Time { return e.OccurredAtT }
func (e BaseEvent) AggregateID() ID       { return e.AggID }

// EventPublisher — публикует доменные события (реализуется в инфраструктуре)
type EventPublisher interface {
	Publish(events ...DomainEvent) error
}

// Aggregate — встраиваемая база агрегата с накоплением событий
type Aggregate struct {
	events []DomainEvent
}

func (a *Aggregate) RecordEvent(e DomainEvent) {
	a.events = append(a.events, e)
}

func (a *Aggregate) PullEvents() []DomainEvent {
	evts := a.events
	a.events = nil
	return evts
}

// Quantity — количество товара
type Quantity struct {
	Value int32
	Unit  string // "шт", "кг", "л"
}

func NewQuantity(value int32, unit string) Quantity {
	return Quantity{Value: value, Unit: unit}
}

func (q Quantity) Add(other Quantity) (Quantity, error) {
	if q.Unit != other.Unit {
		return Quantity{}, errors.New("unit mismatch")
	}
	return Quantity{Value: q.Value + other.Value, Unit: q.Unit}, nil
}

func (q Quantity) Sub(other Quantity) (Quantity, error) {
	if q.Unit != other.Unit {
		return Quantity{}, errors.New("unit mismatch")
	}
	result := q.Value - other.Value
	if result < 0 {
		return Quantity{}, errors.New("quantity cannot be negative")
	}
	return Quantity{Value: result, Unit: q.Unit}, nil
}

func (q Quantity) IsZero() bool { return q.Value == 0 }
