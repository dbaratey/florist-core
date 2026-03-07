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

	"github.com/go-chi/chi/v5"
	"github.com/dbaratey/florist-core/internal/inventory/application"
	inventoryhttp "github.com/dbaratey/florist-core/internal/inventory/http"
	inventorypg "github.com/dbaratey/florist-core/internal/inventory/infrastructure/postgres"
	orderingapp "github.com/dbaratey/florist-core/internal/ordering/application"
	orderinghttp "github.com/dbaratey/florist-core/internal/ordering/http"
	orderingpg "github.com/dbaratey/florist-core/internal/ordering/infrastructure/postgres"
	productionapp "github.com/dbaratey/florist-core/internal/production/application"
	productionhttp "github.com/dbaratey/florist-core/internal/production/http"
	productionpg "github.com/dbaratey/florist-core/internal/production/infrastructure/postgres"

	"github.com/dbaratey/florist-core/internal/shared/infrastructure/outbox"
	"github.com/dbaratey/florist-core/internal/shared/infrastructure/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := loadConfig()

	// --- Infrastructure ---
	pool, err := postgres.NewPool(context.Background(), cfg.DSN)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	publisher := outbox.NewPublisher(pool)
	batchRepo := inventorypg.NewBatchRepository(pool)
	orderRepo := orderingpg.NewOrderRepository(pool)
	recipeRepo := productionpg.NewRecipeRepository(pool)

	// --- Application handlers ---
	inventoryHandlers := inventoryhttp.NewHandlers(
		application.NewReceiveBatchHandler(batchRepo, publisher),
		application.NewConsumeBatchHandler(batchRepo, publisher),
	)

	orderHandlers := orderinghttp.NewHandlers(
		orderingapp.NewConfirmOrderHandler(orderRepo, batchRepo, publisher),
	)

	productionHandlers := productionhttp.NewHandler(
		productionapp.NewCreateRecipeHandler(recipeRepo),
	)

	// --- HTTP mux ---
	mux := chi.NewRouter()
	mux.Get("/healthz", healthzHandler)

	inventoryHandlers.Register(mux)
	orderHandlers.Register(mux)
	productionHandlers.Register(mux)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
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

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
