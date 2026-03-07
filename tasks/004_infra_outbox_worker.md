# Task 004 — Infra: Outbox Worker

## Status
Done

## Goal
Implement transactional outbox pattern for reliable domain event delivery.
Worker polls `outbox_events` table and publishes pending events.

## Acceptance Criteria
- [x] Migration: `outbox_events` table with status, payload, aggregate info
- [x] `OutboxWorker` struct polls every N seconds (configurable)
- [x] Events marked `processing` before dispatch, `done` after
- [x] Failed events incremented `attempts`, max retries configurable
- [x] Worker starts as goroutine in `cmd/api/main.go`

## Files
- `migrations/004_outbox_events.up.sql`
- `internal/shared/postgres/outbox_worker.go`

## Notes
Critical infra for decoupling inventory, ordering and future notification modules.
