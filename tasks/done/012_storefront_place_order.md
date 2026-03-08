# Task 012: Storefront PlaceOrder Endpoint

## Status: DONE

## Goal
Realize `POST /api/v1/storefront/orders` — витрина принимает заказ, делегирует
создание черновика + подтверждение в ordering module.

## Acceptance Criteria
- [x] `PlaceOrderCommand` / `PlaceOrderItem` structs
- [x] `PlaceOrderHandler` создаёт `domain.NewOrder` → сохраняет через `SaveTx` → вызывает `ConfirmOrderHandler`
- [x] HTTP handler `PlaceOrder` декодирует JSON, вызывает handler, возвращает `{order_id}` 201
- [x] `Handler` struct содержит `placeOrder *application.PlaceOrderHandler`
- [x] Маршрут зарегистрирован `r.Post("/orders", h.PlaceOrder)`

## Files
- `internal/storefront/application/place_order.go`
- `internal/storefront/http/handlers.go` (updated)

## Commits
- `feat(storefront/app): add PlaceOrderHandler`
- `feat(storefront/http): implement PlaceOrder endpoint`
