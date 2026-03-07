package domain

import "github.com/dbaratey/florist-core/internal/shared/kernel"

type BatchReceivedEvent struct {
	kernel.BaseEvent
	BatchID      kernel.BatchID
	IngredientID kernel.IngredientID
	StoreID      kernel.StoreID
	Qty          int
}

type BatchExpiredEvent struct {
	kernel.BaseEvent
	BatchID      kernel.BatchID
	IngredientID kernel.IngredientID
	StoreID      kernel.StoreID
	Wasted       int
}

type BatchWrittenOffEvent struct {
	kernel.BaseEvent
	BatchID kernel.BatchID
	Qty     int
	Reason  string
}

type BatchFreshnessChangedEvent struct {
	kernel.BaseEvent
	BatchID  kernel.BatchID
	NewState string
}
