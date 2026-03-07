# --- Стадия сборки ---
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходник
COPY . .

# Собираем с оптимизацией для прода:
# CGO_ENABLED=0 — статическая бинарника
# -ldflags "-w -s" — убираем дебаг символы
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/florist-api \
    ./cmd/api

# --- Финальный образ ---
FROM gcr.io/distroless/static-debian12

WORKDIR /app

# Копируем только бинарник
и (опционально) миграции в образ
COPY --from=builder /app/florist-api .

# Не root
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/florist-api"]
