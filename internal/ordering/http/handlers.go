package http

import (
	"encoding/json"
	"net/http"

	"github.com/dbaratey/florist-core/internal/ordering/application"
)

// Handlers wires ordering application use-cases to HTTP endpoints.
type Handlers struct {
	confirmOrder *application.ConfirmOrderHandler
}

func NewHandlers(confirmOrder *application.ConfirmOrderHandler) *Handlers {
	return &Handlers{confirmOrder: confirmOrder}
}

// Register mounts all ordering routes on mux.
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/orders", h.createOrder)
	mux.HandleFunc("POST /api/v1/orders/{id}/confirm", h.confirmOrder)
}

// POST /api/v1/orders
// Stub: returns 501 — full create-order flow will be added in step 2.
func (h *Handlers) createOrder(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	_, _ = w.Write([]byte(`{"error":"not implemented yet"}`))
}

// POST /api/v1/orders/{id}/confirm
func (h *Handlers) confirmOrder(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")
	if orderID == "" {
		http.Error(w, `{"error":"missing order id"}`, http.StatusBadRequest)
		return
	}

	var cmd application.ConfirmOrderCommand
	if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	cmd.OrderID = orderID

	if err := h.confirmOrder.Handle(r.Context(), cmd); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
