# Task 006 — BatchConsumedEvent + RecalcFreshnessHandler

## Status: DONE

## Goal
Emit `BatchConsumedEvent` when inventory is consumed, and introduce
`RecalcFreshnessHandler` to periodically update freshness state for all active batches.

## Acceptance Criteria
- [x] `BatchConsumedEvent` struct defined in `internal/inventory/domain/events.go`
- [x] `Batch.Consume()` emits `BatchConsumedEvent` after reducing `remainingQty`
- [x] `BatchRepository` domain interface created (`repository.go`) with `Save`, `FindByID`, `FindActive`, `FindByStore`
- [x] `RecalcFreshnessHandler` created in `internal/inventory/application/`
- [x] `kernel.AgingThreshold` (48h) and `kernel.CriticalThreshold` (24h) constants added

## Commits
- `feat(inventory): add BatchConsumedEvent to domain events`
- `feat(inventory): emit BatchConsumedEvent on Consume`
- `feat(inventory): add RecalcFreshnessHandler application service`
- `feat(kernel): add AgingThreshold and CriticalThreshold constants`
- `feat(inventory): add BatchRepository domain interface`
