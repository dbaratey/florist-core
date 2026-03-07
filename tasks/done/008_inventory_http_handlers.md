# Task 008: Inventory HTTP Handlers

## Status: DONE

## Goal
Expose ReceiveBatch, ConsumeBatch, WriteOffBatch use-cases as HTTP endpoints.

## Acceptance Criteria
- [x] `POST /api/v1/inventory/batches` — receive a new batch
- [x] `POST /api/v1/inventory/batches/consume` — consume from batch (atomic via TxRunner)
- [x] `POST /api/v1/inventory/batches/writeoff` — write off a batch
- [x] All handlers decode JSON body, call application handler, return 2xx or error JSON
- [x] Handlers registered in `internal/inventory/http/handlers.go`
- [x] `main.go` wires TxRunner + all three application handlers

## Files
- `internal/inventory/http/handlers.go` — HTTP layer
- `internal/inventory/application/receive_batch.go` — all three app handlers
- `cmd/api/main.go` — wiring (TxRunner, WriteOffBatch added)

## Commits
- `feat(inventory): wire TxRunner + WriteOffBatch handler into main.go`
