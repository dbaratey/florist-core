# Соглашения по разработке

## Структура коммитов

Используем Conventional Commits:

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

### Типы коммитов

- `feat`: Новая функциональность
- `fix`: Исправление ошибок
- `refactor`: Рефакторинг без изменения функциональности
- `docs`: Изменения в документации
- `test`: Добавление или изменение тестов
- `build`: Изменения в системе сборки
- `deps`: Обновление зависимостей
- `chore`: Прочие изменения (конфигурация, скрипты)

### Scope примеры

- `ordering`: Модуль заказов
- `inventory`: Модуль инвентаря
- `catalog`: Модуль каталога
- `cmd`: Команды и точки входа
- `infra`: Инфраструктурный слой

## Стиль кода Go

### Именование

- **Пакеты**: короткие, lowercase, без подчеркиваний (`ordering`, `inventory`)
- **Интерфейсы**: существительное + "er" для поведения (`Repository`, `Publisher`)
- **Структуры**: CamelCase, описательные имена (`BatchRepository`, `OrderService`)
- **Методы**: CamelCase, начинаются с глагола (`SaveBatch`, `GetOrder`)
- **Переменные**: camelCase, краткие в коротких scope

### Организация кода

```go
// 1. Package declaration
package domain

// 2. Imports (сгруппированы: stdlib, external, internal)
import (
    "context"
    "time"

    "github.com/google/uuid"

    "github.com/dbaratey/florist-core/internal/shared"
)

// 3. Constants
const MaxBatchSize = 1000

// 4. Types
type Batch struct { ... }

// 5. Constructor functions
func NewBatch(...) (*Batch, error) { ... }

// 6. Methods
func (b *Batch) Method() { ... }
```

## DDD Patterns

### Aggregate Root

- Всегда имеет уникальный ID (UUID)
- Инкапсулирует бизнес-логику
- Генерирует domain events при изменениях
- Валидирует инварианты в конструкторе и методах

```go
type Order struct {
    id     uuid.UUID
    events []shared.DomainEvent
}

func (o *Order) AddItem(item OrderItem) error {
    // Валидация
    // Изменение состояния
    o.events = append(o.events, OrderItemAdded{...})
    return nil
}
```

### Value Objects

- Immutable
- Сравниваются по значению
- Валидируются в конструкторе

```go
type Money struct {
    amount   int64
    currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
    if amount < 0 {
        return Money{}, errors.New("negative amount")
    }
    return Money{amount: amount, currency: currency}, nil
}
```

### Repository

- Интерфейс в domain слое
- Реализация в infrastructure слое
- Методы с транзакциями: `SaveTx`, `UpdateTx`
- Методы без транзакций: `Save`, `Update`, `Get`, `List`

```go
// domain/repository.go
type OrderRepository interface {
    Save(ctx context.Context, order *Order) error
    SaveTx(ctx context.Context, tx Transaction, order *Order) error
    Get(ctx context.Context, id uuid.UUID) (*Order, error)
}

// infrastructure/postgres/order_repository.go
type OrderRepository struct {
    db *sql.DB
}
```

### Domain Events

- Immutable структуры
- Прошедшее время в названии (`OrderCreated`, `BatchReceived`)
- Содержат минимум данных для идентификации

```go
type OrderCreated struct {
    OrderID   uuid.UUID
    CustomerID uuid.UUID
    CreatedAt time.Time
}

func (e OrderCreated) EventType() string {
    return "order.created"
}
```

## Обработка ошибок

### Wrapping errors

```go
import "fmt"

if err != nil {
    return fmt.Errorf("failed to save order: %w", err)
}
```

### Domain errors

```go
var (
    ErrOrderNotFound = errors.New("order not found")
    ErrInvalidQuantity = errors.New("invalid quantity")
)
```

### HTTP errors

```go
if errors.Is(err, domain.ErrOrderNotFound) {
    http.Error(w, "Order not found", http.StatusNotFound)
    return
}
```

## Тестирование

### Unit Tests

- Файлы: `*_test.go`
- Функции: `Test<Function>`
- Table-driven tests для множественных сценариев

```go
func TestOrder_AddItem(t *testing.T) {
    tests := []struct{
        name string
        // ...
    }{
        {name: "valid item"},
        {name: "invalid quantity"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

### Integration Tests

- Требуют testcontainers или docker-compose
- Используют реальные БД и инфраструктуру
- Очищают состояние между тестами

## Логирование

### Использование slog

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

logger.Info("order created",
    slog.String("order_id", order.ID().String()),
    slog.String("customer_id", order.CustomerID().String()),
)

logger.Error("failed to save order",
    slog.String("order_id", order.ID().String()),
    slog.Any("error", err),
)
```

### Уровни логирования

- `Debug`: Детальная информация для отладки
- `Info`: Информационные сообщения (создание заказа, старт сервиса)
- `Warn`: Предупреждения (повторные попытки, deprecation)
- `Error`: Ошибки требующие внимания

## HTTP API

### Роутинг с Chi

```go
import "github.com/go-chi/chi/v5"

r := chi.NewRouter()
r.Post("/orders", handler.CreateOrder)
r.Get("/orders/{id}", handler.GetOrder)
r.Put("/orders/{id}", handler.UpdateOrder)
```

### Request/Response

```go
// DTOs в http слое
type CreateOrderRequest struct {
    CustomerID string `json:"customer_id"`
    Items []OrderItemDTO `json:"items"`
}

type OrderResponse struct {
    ID string `json:"id"`
    Status string `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}
```

## Database

### Миграции

- Используем `golang-migrate/migrate`
- Файлы: `migrations/000001_initial.up.sql`, `000001_initial.down.sql`
- Нумерация последовательная

### SQL Queries

- Используем `database/sql` или `pgx`
- Prepared statements для безопасности
- Контекст для таймаутов и отмен

```go
row := tx.QueryRowContext(ctx, 
    "SELECT id, status FROM orders WHERE id = $1",
    orderID,
)
```
