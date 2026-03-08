package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/dbaratey/florist-core/internal/storefront/application"
)

// Handler wires storefront application use-cases to HTTP endpoints.
type Handler struct {
	catalog    *application.GetCatalogHandler
	placeOrder *application.PlaceOrderHandler
}

func NewHandler(catalog *application.GetCatalogHandler, placeOrder *application.PlaceOrderHandler) *Handler {
	return &Handler{catalog: catalog, placeOrder: placeOrder}
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
// Body: {"store_id": "...", "items": [{"product_id": "...", "qty": 2}]}
func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StoreID string `json:"store_id"`
		Items   []struct {
			ProductID string `json:"product_id"`
			Qty       int    `json:"qty"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.StoreID == "" {
		http.Error(w, `{"error":"store_id is required"}`, http.StatusBadRequest)
		return
	}

	cmd := application.PlaceOrderCommand{StoreID: req.StoreID}
	for _, it := range req.Items {
		cmd.Items = append(cmd.Items, application.PlaceOrderItem{
			ProductID: it.ProductID,
			Qty:       it.Qty,
		})
	}

	orderID, err := h.placeOrder.Handle(r.Context(), cmd)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"order_id": orderID.String(),
	})
}
