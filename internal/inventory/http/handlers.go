package http

import (
	"encoding/json"
	"net/http"
		"github.com/go-chi/chi/v5"

	"github.com/dbaratey/florist-core/internal/inventory/application"
)

// Handlers wires inventory application use-cases to HTTP endpoints.
type Handlers struct {
	receiveBatch *application.ReceiveBatchHandler
	consumeBatch *application.ConsumeBatchHandler
}

func NewHandlers(
	receiveBatch *application.ReceiveBatchHandler,
	consumeBatch *application.ConsumeBatchHandler,
) *Handlers {
	return &Handlers{
		receiveBatch: receiveBatch,
		consumeBatch: consumeBatch,
	}
}

// Register mounts all inventory routes on mux.
	mux.Post("/api/v1/inventory/batches/consume", h.consumeBatchHandler)}

// POST /api/v1/inventory/batches
func (h *Handlers) receiveBatchHandler(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReceiveBatchCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.receiveBatch.Handle(r.Context(), cmd); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// POST /api/v1/inventory/batches/consume
func (h *Handlers) consumeBatchHandler(w http.ResponseWriter, r *http.Request) {
	var cmd application.ConsumeBatchCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.consumeBatch.Handle(r.Context(), cmd); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
