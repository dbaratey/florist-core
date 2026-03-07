# Architecture

## Overview

`florist-core` is a **modular monolith** backend built with **Domain-Driven Design (DDD)** principles. The system provides core functionality for independent flower shops and local marketplace aggregators.

## Architectural Principles

### 1. Modular Monolith

- **One deployable unit**: Single binary, one database
- **Isolated bounded contexts**: Modules communicate through application services and domain events
- **No direct package imports**: Contexts never import each other's internals
- **Clear module boundaries**: Each bounded context is self-contained

### 2. Domain-Driven Design (DDD)

#### Layered Architecture

Each bounded context follows a strict 4-layer architecture:

```
/internal/<context>/
  /domain          # Value objects, Entities, Aggregates, Domain events
  /application     # Use-case handlers (Commands/Queries)
  /infrastructure  # Repositories, External services, Persistence
  /http            # HTTP handlers, DTOs, REST API
```

**Layer Dependencies** (inner → outer):
- `domain` → (no dependencies)
- `application` → `domain`
- `infrastructure` → `domain`, `application`
- `http` → `application`

#### Key DDD Patterns

- **Aggregate Roots**: `Order`, `Batch`, `Recipe`
- **Value Objects**: `Money`, `ProductID`, etc.
- **Domain Events**: Published via Outbox pattern
- **Repository Pattern**: Abstractions in `application`, implementations in `infrastructure`
- **Application Handlers**: One handler per use-case (Command/Query separation)

### 3. Bounded Contexts

```
/internal/
  /inventory/      # Batch management (freshness, FEFO)
  /ordering/       # Order lifecycle, confirmations
  /production/     # Recipe management, substitutions (future)
  /catalog/        # Product catalog, ingredients (future)
  /merchant/       # Store settings (future)
  /shared/         # Cross-cutting concerns (postgres, outbox, events)
```

**Communication**:
- Synchronous: Through application service interfaces
- Asynchronous: Domain events via Outbox + Event Publisher

### 4. Transactional Outbox Pattern

Used for reliable event publishing:

1. Business logic + event stored in single DB transaction
2. Background worker polls `outbox` table
3. Events published to message broker (future: Redis, Kafka)
4. Ensures eventual consistency across contexts

## Directory Structure

```
florist-core/
├── cmd/
│   └── api/                  # Entry point, DI wiring
│       ├── main.go
│       └── config.go
├── internal/
│   ├── inventory/           # Bounded Context: Inventory
│   │   ├── domain/          # Batch, allocation logic
│   │   ├── application/     # ReceiveBatch, ConsumeBatch handlers
│   │   ├── infrastructure/  # postgres/ (BatchRepository)
│   │   └── http/            # REST handlers
│   ├── ordering/            # Bounded Context: Ordering
│   │   ├── domain/          # Order aggregate
│   │   ├── application/     # ConfirmOrder handler
│   │   ├── infrastructure/  # postgres/ (OrderRepository)
│   │   └── http/            # REST handlers
│   └── shared/              # Shared infrastructure
│       ├── infrastructure/
│       │   ├── postgres/    # Connection pool
│       │   └── outbox/      # Outbox publisher
│       └── domain/          # Shared value objects (future)
├── migrations/              # SQL schema migrations
├── docs/                    # Documentation
└── docker-compose.yml       # Local dev environment
```

## Key Invariants

### Inventory Context

- **Batch**: Cannot consume expired batch; qty never goes negative
- **FEFO (First Expired First Out)**: Allocation prioritizes expiring stock

### Ordering Context

- **Order**: Confirm only if all items available; cancel releases reserves

## Technology Choices

See [TECH_STACK.md](./TECH_STACK.md) for detailed technology decisions.

## Integration Points

### HTTP API

- **Router**: `chi` (v5)
- **Logging**: `slog` (structured)
- **Endpoints**: REST JSON

### Database

- **PostgreSQL**: Single database, multi-schema (one per context)
- **Transactions**: ACID guarantees within bounded context
- **Migrations**: SQL-first, version-controlled

### Future: Event-Driven Communication

- **Redis Streams** or **Kafka**: For async domain events
- **CQRS**: Read models for complex queries

## Design Decisions

### Why Modular Monolith?

- Simpler deployment and operations
- ACID transactions within bounded contexts
- Easy to split into microservices later if needed

### Why DDD?

- Clear separation of business logic from infrastructure
- Testable domain models
- Aligns with business language (Ubiquitous Language)

### Why Outbox Pattern?

- Guarantees at-least-once delivery of domain events
- Avoids dual-write problem (database + message broker)

## Development Workflow

1. **Domain first**: Model aggregates, value objects, invariants
2. **Application handlers**: Implement use-cases
3. **Infrastructure**: Wire repositories, external services
4. **HTTP layer**: Expose REST API
5. **Tests**: Unit tests for domain, integration tests for handlers

## Future Enhancements

- **Production context**: Recipe management, substitution logic
- **Catalog context**: Product/ingredient catalog
- **Merchant context**: Multi-tenant store settings
- **Event-driven communication**: Redis Streams/Kafka
- **CQRS read models**: For complex analytics
