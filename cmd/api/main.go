package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := loadConfig()

	// TODO: инициализация зависимостей (вынесено за скобку по принципу DI)
	// db := postgres.Connect(cfg.DSN)
	// redis := redisclient.Connect(cfg.RedisAddr)
	// publisher := outbox.NewPublisher(db)
	// batchRepo := postgres.NewBatchRepository(db)
	// orderRepo := postgres.NewOrderRepository(db)
	//
	// inventoryHandlers := inventoryhttp.NewHandlers(
	//     inventory_app.NewReceiveBatchHandler(batchRepo, publisher),
	//     inventory_app.NewConsumeBatchHandler(batchRepo, publisher),
	// )
	// orderHandlers := orderinghttp.NewHandlers(
	//     ordering_app.NewConfirmOrderHandler(orderRepo, batchRepo, publisher),
	// )

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler)
	// Регистрация маршрутов (WIP)
	// inventoryHandlers.Register(mux)
	// orderHandlers.Register(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting HTTP server", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	// Грацефульный шатдаун
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
	slog.Info("server stopped")
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"florist-core"}`))
}

type Config struct {
	Port      string
	DSN       string
	RedisAddr string
	LogLevel  string
}

func loadConfig() Config {
	return Config{
		Port:      getEnv("PORT", "8080"),
		DSN:       getEnv("DATABASE_URL", "postgres://florist:florist@localhost:5432/floristdb?sslmode=disable"),
		RedisAddr: getEnv("REDIS_ADDR", "localhost:6379"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
