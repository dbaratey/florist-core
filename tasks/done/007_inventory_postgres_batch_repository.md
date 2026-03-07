# Task 007 — Postgres BatchRepository (infra layer)

## Status: DONE

## Goal
Implement the full Postgres persistence layer for the `Batch` aggregate so that
all application services and handlers can read/write inventory data.

## Acceptance Criteria
- [x] Migration `005_alter_inventory_batches.up.sql` — extends table with all Batch fields + indices
- [x] `domain.BatchRepository` interface updated with typed kernel IDs + `FindAvailableByIngredient` (FEFO)
- [x] `domain.RehydrateParams` / `domain.RehydrateBatch` added for aggregate reconstruction
- [x] `domain.ErrBatchNotFound` and `domain.ErrOptimisticLock` added
- [x] `Batch` accessors added: `ReceivedQty`, `PurchasePrice`, `ReceivedAt`, `IsWrittenOff`, `WriteOffReason`
- [x] `Batch` struct extended with `writtenOff` and `writeOffReason` fields
- [x] `postgres.BatchRepository` rewritten: upsert with optimistic lock, `FindActive`, `FindByStore`, `FindAvailableByIngredient` (FEFO)

## Commits
- `feat(db): migration 005 — extend inventory_batches with full Batch fields`
- `feat(inventory): update BatchRepository interface with typed IDs and FEFO method`
- `feat(inventory): add RehydrateBatch, ErrBatchNotFound, ErrOptimisticLock, accessors`
- `feat(infra/inventory): rewrite BatchRepository — upsert, FindActive, FindByStore, FEFO`
