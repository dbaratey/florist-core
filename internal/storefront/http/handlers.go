package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Register(r chi.Router) {
	r.Route("/api/v1/storefront", func(r chi.Router) {
		r.Get("/catalog", h.GetCatalog)
		r.Post("/orders", h.PlaceOrder)
	})
}

func (h *Handler) GetCatalog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": []interface{}{},
	})
}

func (h *Handler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
