# Task 011: Storefront Catalog API

## Status: DONE

## Goal
Implement the open Storefront API allowing external clients (vitrina/marketplace) to browse available inventory and place orders.

## Acceptance Criteria
- [x] `GET /api/v1/storefront/catalog?store_id=...&ingredient_id=...` returns available batches as JSON
- [x] Filters: non-expired, non-written-off, remaining_qty > 0
- [x] Response includes: batch_id, ingredient_id, store_id, remaining_qty, freshness, expires_at, price
- [x] `BatchCatalogReader` interface defined in storefront/application
- [x] Postgres `CatalogReader` implementation with parameterized SQL query
- [x] `NewHandler` accepts `GetCatalogHandler` (replaces empty stub)
- [x] `main.go` wires `storefrontpg.NewCatalogReader` + `storefrontapp.NewGetCatalogHandler`

## Files
- `internal/storefront/application/catalog.go` — GetCatalogHandler + interfaces
- `internal/storefront/infrastructure/postgres/catalog_reader.go` — SQL read-model
- `internal/storefront/http/handlers.go` — real HTTP handler replacing stub
- `cmd/api/main.go` — wiring

## Commits
- `feat(storefront): add GetCatalogHandler application service`
- `feat(storefront): add CatalogReader postgres implementation`
- `feat(storefront): wire GetCatalogHandler into storefront HTTP handler`
- `feat(storefront): wire CatalogReader + GetCatalogHandler into main.go`

## Next
- Task 012: Storefront PlaceOrder endpoint (delegates to ConfirmOrderHandler)
- Task 013: Production — cost calculation (Автоматический расчет себестоимости)
