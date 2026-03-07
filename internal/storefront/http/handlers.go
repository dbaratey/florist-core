package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dbaratey/florist-core/internal/storefront/application"
)

// Handler wires storefront application use-cases to HTTP endpoints.
type Handler struct {
	catalog *application.GetCatalogHandler
}

func NewHandler(catalog *application.GetCatalogHandler) *Handler {
	return &Handler{catalog: catalog}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/api/v1/storefront", func(r chi.Router) {
		r.Get("/catalog", h.GetCatalog)
		r.Post("/orders", h.PlaceOrder)
	})
}

// GET /api/v1/storefront/catalog?store_id=...&ingredient_id=...
func (h *Handler) GetCatalog(w http.ResponseWriter, r *http.Request) {
	storeID := r.URL.Query().Get("store_id")
	if storeID == "" {
		http.Error(w, `{"error":"store_id is required"}`, http.StatusBadRequest)
		return
	}

	cmd := application.GetCatalogCommand{
		StoreID:      storeID,
		IngredientID: r.URL.Query().Get("ingredient_id"),
	}

	items, err := h.catalog.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"items": items,
	})
}

// POST /api/v1/storefront/orders
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	// TODO(011): implement storefront PlaceOrder via ConfirmOrderHandler
	w.WriteHeader(http.StatusNotImplemented)
}
