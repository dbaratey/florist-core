# florist-core

Modular monolith backend for flower retail & local marketplace.
Built with **Go**, **DDD**, **CQRS-lite**, **PostgreSQL**, **Redis**.

## Overview

`florist-core` is a trade/ops core for independent flower shops and local aggregator storefronts.
It handles inventory (batches, freshness, FEFO), recipes/substitutions, order lifecycle and production assembly.
Shops get a SaaS backbone; buyers get a unified local storefront with real-time availability.

## Architecture

Modular monolith: one deployable, one database, isolated bounded contexts.
Modules communicate through application services and domain events — never through direct package imports of each other's internals.

```
/internal
  /shared/kernel      # Value objects: ID, Money, DomainEvent, EventPublisher
  /inventory/domain   # Batch aggregate: freshness, consumption, writeoff
  /ordering/domain    # Order aggregate: lifecycle, items, cancel
  /ordering/application  # ConfirmOrderHandler, repositories, UoW
  /production/domain  # Recipe, RecipeVersion, SubstitutionGroup (WIP)
  /catalog/domain     # Product, Ingredient (WIP)
  /storefront         # Public API query handlers (WIP)
  /merchant           # Store settings (WIP)
/cmd/api             # HTTP entrypoint (WIP)
/cmd/worker          # Background jobs: freshness recalc, expiry (WIP)
```

## Bounded Contexts

| Context | Aggregate Roots | Key Invariants |
|---|---|---|
| **Inventory** | `Batch` | Cannot consume expired batch; qty never goes negative |
| **Ordering** | `Order` | Confirm only if all items available; cancel releases reserves |
| **Production** | `Recipe`, `ProductionJob` | One active recipe version per product; assembly consumes by batch |
| **Catalog** | `Product`, `Ingredient` | Published product must have active recipe or manual-assembly flag |

## Domain Events

- `order.created` / `order.confirmed` / `order.cancelled` / `order.shipped`
- `inventory.batch_received` / `inventory.batch_expired` / `inventory.batch_written_off`
- `inventory.batch_freshness_changed`
- `production.job_created` / `production.completed`

## Key Use Cases (implemented)

- `ConfirmOrderHandler` — checks availability, reserves inventory, transitions order to `confirmed`, publishes events
- `Order.Cancel` — releases reserves, handles pre/post-production cancellation
- `Batch.Consume/Release/Writeoff/Expire` — full freshness lifecycle with domain events

## Stack

- **Backend**: Go 1.22, PostgreSQL (source of truth), Redis (read cache, sessions)
- **Frontend**: Vue.js SPA (storefront & merchant cabinet)
- **Architecture**: Modular monolith → ready to split to services if needed
- **License**: MIT

## Status

Early development. Domain layer and application handlers for `ordering` and `inventory` are in progress.
`production`, `catalog`, `storefront`, HTTP layer and Postgres infrastructure are next.
