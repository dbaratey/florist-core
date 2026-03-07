package http

import (
	"encoding/json"
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/dbaratey/florist-core/internal/inventory/application"
)

// Handlers wires inventory application use-cases to HTTP endpoints.
type Handlers struct {
	receiveBatch  *application.ReceiveBatchHandler
	consumeBatch  *application.ConsumeBatchHandler
	writeoffBatch *application.WriteoffBatchHandler
}

func NewHandlers(
	receiveBatch *application.ReceiveBatchHandler,
	consumeBatch *application.ConsumeBatchHandler,
	writeoffBatch *application.WriteoffBatchHandler,
) *Handlers {
	return &Handlers{
		receiveBatch:  receiveBatch,
		consumeBatch:  consumeBatch,
		writeoffBatch: writeoffBatch,
	}
}

// Register mounts all inventory routes on mux.
func (h *Handlers) Register(mux *chi.Mux) {
	mux.Post("/api/v1/inventory/batches", h.receiveBatchHandler)
	mux.Post("/api/v1/inventory/batches/consume", h.consumeBatchHandler)
	mux.Post("/api/v1/inventory/batches/writeoff", h.writeoffBatchHandler)
}

// POST /api/v1/inventory/batches
func (h *Handlers) receiveBatchHandler(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReceiveBatchCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if _, err := h.receiveBatch.Handle(r.Context(), cmd); err != nil {
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

// POST /api/v1/inventory/batches/writeoff
func (h *Handlers) writeoffBatchHandler(w http.ResponseWriter, r *http.Request) {
	var cmd application.WriteoffBatchCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.writeoffBatch.Handle(r.Context(), cmd); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
