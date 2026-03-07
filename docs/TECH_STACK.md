# Technology Stack

## Overview

This document describes the technology choices for `florist-core` and the rationale behind each decision.

## Backend

### Language & Runtime

- **Go 1.24+**
  - **Why**: Simplicity, excellent concurrency, fast compilation, strong standard library
  - **Benefits**: Single binary deployment, low memory footprint, great tooling
  - **Trade-offs**: Less expressive than some languages, but enforces clarity

### HTTP Framework

- **chi v5** (`github.com/go-chi/chi/v5`)
  - **Why**: Lightweight, idiomatic Go, composable middleware, context-based routing
  - **Alternatives considered**: Gin (too much magic), Echo (similar to chi but less idiomatic)
  - **Benefits**: Minimal dependencies, RESTful routing, excellent performance

### Database

- **PostgreSQL 15+**
  - **Why**: ACID transactions, robust SQL support, JSON columns, excellent performance
  - **Use cases**: 
    - Transactional data (orders, batches)
    - Event outbox table
    - Future: JSONB for flexible product attributes
  - **Connection pool**: `pgx/v5` (native PostgreSQL driver)

### Database Migrations

- **SQL-first migrations** (plain `.sql` files in `/migrations`)
  - **Why**: Direct control, no ORM abstraction, easy to review
  - **Tool**: Manual execution for now; future: `golang-migrate` or `goose`
  - **Convention**: Version-prefixed files (e.g., `001_create_inventory.up.sql`)

### Logging

- **slog** (Go standard library, Go 1.21+)
  - **Why**: Structured logging, standard library, zero dependencies
  - **Format**: JSON for production, text for development
  - **Levels**: INFO (default), ERROR, DEBUG (via config)

### Testing

- **testing** (Go standard library)
  - **Unit tests**: Domain logic, use-case handlers
  - **Integration tests**: HTTP handlers, repository implementations (with test DB)
  - **Test doubles**: Interfaces for repositories, no mocking frameworks

## Infrastructure

### Containerization

- **Docker** + **docker-compose**
  - **Local development**: PostgreSQL, Redis (future)
  - **Dockerfile**: Multi-stage build (builder + distroless final image)

### Deployment (Future)

- **Target**: Docker + Kubernetes or simple VPS with systemd
- **Config**: Environment variables via `.env` or K8s secrets
- **Health checks**: `/healthz` endpoint

## Data Patterns

### Outbox Pattern

- **Implementation**: Custom (`internal/shared/infrastructure/outbox`)
- **Storage**: PostgreSQL table (`outbox`)
- **Polling**: Background goroutine in main process
- **Future**: Replace with dedicated service or message broker

### Repository Pattern

- **Interface**: Defined in `application` layer
- **Implementation**: `infrastructure/postgres`
- **Transactions**: Explicit `*sql.Tx` passed to repository methods (`SaveTx`, `UpdateTx`)

## Frontend (Future)

### SaaS Admin Panel

- **VueJS 3** + **Vite**
  - **Why**: Reactive, component-based, excellent DX
  - **State**: Pinia (Vue 3 store)
  - **HTTP**: Axios or Fetch API

### Public Storefront

- **VueJS 3** (same stack as admin)
  - **Routing**: Vue Router
  - **UI**: Tailwind CSS or Vuetify

## Message Broker (Planned)

### Redis Streams or Kafka

- **Use case**: Async domain event publishing
- **Why Redis**: Simpler for MVP, built-in caching
- **Why Kafka**: Better for high-throughput, persistent logs
- **Decision**: Start with Redis, migrate to Kafka if needed

## Caching (Planned)

- **Redis**
  - **Use cases**: 
    - Product catalog reads
    - Session storage (future)
    - Rate limiting (future)
  - **Library**: `go-redis/redis/v9`

## Monitoring & Observability (Future)

### Logging

- **Current**: `slog` to stdout
- **Future**: Centralized logging (e.g., Loki, ELK stack)

### Metrics

- **Planned**: Prometheus + Grafana
  - **Metrics**: HTTP request latency, DB query duration, outbox lag

### Tracing

- **Planned**: OpenTelemetry (OTEL)
  - **Integration**: Trace HTTP requests across contexts

## Development Tools

### Code Quality

- **Linter**: `golangci-lint` (aggregates multiple linters)
- **Formatter**: `gofmt` (standard), `goimports` (auto-import)
- **Pre-commit hooks**: Run tests, linters before commit

### Local Development

- **Environment**: `docker-compose up` for PostgreSQL + Redis
- **Hot reload**: `air` (optional, for rapid iteration)
- **IDE**: VS Code with Go extension, or GoLand

## Dependencies

See `go.mod` for full list. Key dependencies:

```
github.com/go-chi/chi/v5       v5.2.0   # HTTP router
github.com/jackc/pgx/v5        v5.7.2   # PostgreSQL driver
```

## Security

### Current

- **SQL injection**: Prepared statements via `pgx`
- **HTTPS**: Reverse proxy (nginx/Caddy) in production
- **Secrets**: Environment variables, never committed

### Future

- **Authentication**: JWT tokens
- **Authorization**: Role-based access control (RBAC)
- **Rate limiting**: Redis-based

## Performance Considerations

### Database

- **Indexes**: Strategic indexes on foreign keys, query filters
- **Connection pool**: Tuned for concurrent load (`pgx` pool settings)
- **Transactions**: Kept short, only when needed

### HTTP

- **Timeouts**: Read/Write/Idle timeouts configured
- **Graceful shutdown**: 15-second timeout for in-flight requests

### Concurrency

- **Goroutines**: Used for background tasks (outbox polling)
- **Context**: Propagated for cancellation, timeouts

## Trade-offs & Future Improvements

### Why not use an ORM?

- **Current**: Raw SQL for transparency and control
- **Trade-off**: More boilerplate, but easier to optimize
- **Future**: Consider `sqlc` (generates type-safe Go from SQL)

### Why not microservices?

- **Current**: Modular monolith for simplicity
- **Trade-off**: Easier operations, but less independent scaling
- **Future**: Split into services if traffic/team size demands

### Why not gRPC?

- **Current**: REST/JSON for simplicity and broad client support
- **Future**: gRPC for inter-service communication if we split into microservices

## Version Requirements

- **Go**: 1.24+
- **PostgreSQL**: 15+
- **Docker**: 20.10+
- **Node.js** (frontend): 20+ (LTS)

## References

- [Go documentation](https://go.dev/doc/)
- [chi router](https://github.com/go-chi/chi)
- [pgx PostgreSQL driver](https://github.com/jackc/pgx)
- [PostgreSQL best practices](https://wiki.postgresql.org/wiki/Don%27t_Do_This)
