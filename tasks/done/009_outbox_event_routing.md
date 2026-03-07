# Task 009: Outbox Event Routing to RecalcFreshnessHandler

## Status: DONE

## Goal
Route inventory domain events from the outbox worker to the appropriate in-process handlers.

## Acceptance Criteria
- [x] `inventory.batch_consumed` triggers `RecalcFreshnessHandler.Handle(ctx)`
- [x] `inventory.batch_expired` triggers `RecalcFreshnessHandler.Handle(ctx)`
- [x] `inventory.batch_written_off` triggers `RecalcFreshnessHandler.Handle(ctx)`
- [x] Unknown events are logged (no-op, no error)
- [x] Errors from handler are logged and returned to the worker for retry

## Files
- `cmd/api/main.go` — switch/case dispatcher in outbox worker callback

## Commits
- `feat(inventory): route outbox events to RecalcFreshnessHandler`
