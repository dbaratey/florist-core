package kernel

import "time"

// DomainEvent is the base interface for all domain events.
type DomainEvent interface {
	EventName() string
	OccurredAt() time.Time
}

// BaseEvent provides common fields for all domain events.
type BaseEvent struct {
	name       string
	occurredAt time.Time
}

func NewBaseEvent(name string) BaseEvent {
	return BaseEvent{name: name, occurredAt: time.Now().UTC()}
}

func (e BaseEvent) EventName() string    { return e.name }
func (e BaseEvent) OccurredAt() time.Time { return e.occurredAt }

// EventPublisher dispatches domain events after aggregate operations.
type EventPublisher interface {
	Publish(events ...DomainEvent)
}
