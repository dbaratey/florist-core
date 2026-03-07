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
	storefronthttp "github.com/dbaratey/florist-core/internal/storefront/http"
	sharedpg "github.com/dbaratey/florist-core/internal/shared/postgres"
	sharedinfrapg "github.com/dbaratey/florist-core/internal/shared/infrastructure/postgres"
	"github.com/dbaratey/florist-core/internal/shared/infrastructure/redis"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg := loadConfig()

	// --- Infrastructure ---
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := sharedinfrapg.NewPool(ctx, cfg.DSN)
	if err != nil {
		slog.Error("failed to connect to postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	redisClient := redis.MustNewClient(ctx, cfg.RedisAddr)
	defer redisClient.Close()

	// EventPublisher: writes domain events into outbox_events table
	publisher := sharedpg.NewOutboxEventPublisher(pool)

	// TxRunner: wraps pgx transactions
	txRunner := sharedinfrapg.NewTxRunner(pool)

	// Repositories
	batchRepo := inventorypg.NewBatchRepository(pool)
	orderRepo := orderingpg.NewOrderRepository(pool)
	recipeRepo := productionpg.NewRecipeRepository(pool)

	// RecalcFreshnessHandler: triggered by inventory events
	recalcFreshness := application.NewRecalcFreshnessHandler(batchRepo)

	// OutboxWorker: polls outbox_events and dispatches to in-process handlers
	outboxWorker := sharedpg.NewOutboxWorker(
		pool,
		func(ctx context.Context, eventType string, payload []byte) error {
			switch eventType {
			case "inventory.batch_consumed",
				"inventory.batch_expired",
				"inventory.batch_written_off":
				if err := recalcFreshness.Handle(ctx); err != nil {
					slog.Error("recalc freshness failed", "event_type", eventType, "err", err)
					return err
				}
			default:
				slog.Info("outbox event dispatched (no handler)", "event_type", eventType)
			}
			return nil
		},
		2*time.Second,
		5,
		logger,
	)
	go outboxWorker.Run(ctx)
	slog.Info("outbox worker started")

	// --- Application handlers ---
	inventoryHandlers := inventoryhttp.NewHandlers(
		application.NewReceiveBatchHandler(batchRepo, txRunner, publisher),
		application.NewConsumeBatchHandler(batchRepo, txRunner, publisher),
		application.NewWriteOffBatchHandler(batchRepo, txRunner, publisher),
	)

	orderHandlers := orderinghttp.NewHandlers(
		orderingapp.NewConfirmOrderHandler(orderRepo, batchRepo, publisher),
	)

	productionHandlers := productionhttp.NewHandler(
		productionapp.NewCreateRecipeHandler(recipeRepo),
	)

	storefrontHandlers := storefronthttp.NewHandler()

	// --- HTTP mux ---
	mux := chi.NewRouter()
	mux.Get("/healthz", healthzHandler)
	inventoryHandlers.Register(mux)
	orderHandlers.Register(mux)
	productionHandlers.Register(mux)
	storefrontHandlers.Register(mux)

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

	<-ctx.Done()
	slog.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("server stopped")
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
