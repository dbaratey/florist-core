# Task 005 — Infra: EventPublisher (Outbox-backed)

## Status
Done

## Goal
Подключить `kernel.EventPublisher` к outbox-таблице и wire-up воркера в `main.go`.
Обеспечить надёжную доставку доменных событий между модулями.

## Acceptance Criteria
- [x] `kernel.EventPublisher` интерфейс уже существовал в `events.go`
- [x] `OutboxEventPublisher` реализует `Publish(...DomainEvent)` через INSERT в `outbox_events`
- [x] `PublishTx` — транзакционная версия для application services
- [x] `AggregateEvent` опциональный интерфейс для metadata (aggregate_type, aggregate_id)
- [x] `OutboxWorker` запускается как горутина в `main.go` (интервал 2s, max 5 попыток)
- [x] `main.go` использует `sharedpg.NewOutboxEventPublisher` вместо устаревшего `outbox.NewPublisher`

## Files
- `internal/shared/postgres/event_publisher.go` — реализация паблишера
- `cmd/api/main.go` — wire-up воркера и паблишера

## Notes
- TODO(006): роутинг событий в обработчики модулей (inventory freshness recalc при BatchConsumed)
- EventHandler в OutboxWorker сейчас логирует событие — заглушка до следующего шага
