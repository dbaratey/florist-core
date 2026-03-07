# Task 010: Ordering Module

## Status: DONE

## Goal
Implement the Ordering bounded context: Order aggregate, ConfirmOrder use-case, HTTP handler, and Postgres repository.

## Acceptance Criteria
- [x] `Order` aggregate root with line items and status lifecycle
- [x] Domain events: `OrderConfirmedEvent`
- [x] `OrderRepository` interface + Postgres implementation
- [x] `ConfirmOrderHandler` (atomic: debit inventory batches + save order + publish event)
- [x] `BatchRepository` interface in ordering/application for cross-module reads
- [x] HTTP handler `POST /api/v1/orders/confirm` wired in main.go
- [x] TxRunner used for atomic order confirmation

## Files
- `internal/ordering/domain/order.go` — Order aggregate
- `internal/ordering/domain/batch.go` — batch allocation logic
- `internal/ordering/domain/events.go` — domain events
- `internal/ordering/application/confirm_order.go` — use-case handler
- `internal/ordering/application/repository.go` — repository interfaces
- `internal/ordering/infrastructure/postgres/` — Postgres repo
- `internal/ordering/http/` — HTTP handlers
- `cmd/api/main.go` — wired with orderRepo, batchRepo, publisher
