package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dbaratey/florist-core/internal/production/application"
	"github.com/dbaratey/florist-core/internal/production/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	createRecipe *application.CreateRecipeHandler
	calcCost     *application.CalcCostHandler
}

func NewHandler(createRecipe *application.CreateRecipeHandler, calcCost *application.CalcCostHandler) *Handler {
	return &Handler{createRecipe: createRecipe, calcCost: calcCost}
}

func (h *Handler) Register(r chi.Router) {
	r.Post("/api/v1/production/recipes", h.CreateRecipe)
	r.Get("/api/v1/production/recipes/{id}/cost", h.CalcCost)
}

type createRecipeRequest struct {
	Name        string `json:"name"`
	Ingredients []struct {
		MaterialID string `json:"material_id"`
		Quantity   int    `json:"quantity"`
	} `json:"ingredients"`
}

func (h *Handler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	var req createRecipeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ingredients := make([]domain.Ingredient, len(req.Ingredients))
	for i, ing := range req.Ingredients {
		ingredients[i] = domain.Ingredient{
			MaterialID: kernel.IngredientID(ing.MaterialID),
			Quantity:   ing.Quantity,
		}
	}

	id, err := h.createRecipe.Handle(r.Context(), application.CreateRecipeCommand{
		Name:        req.Name,
		Ingredients: ingredients,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": string(id)})
}

// GET /api/v1/production/recipes/{id}/cost?qty=1
func (h *Handler) CalcCost(w http.ResponseWriter, r *http.Request) {
	recipeID := chi.URLParam(r, "id")
	qty := 1
	if q := r.URL.Query().Get("qty"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			qty = n
		}
	}

	result, err := h.calcCost.Handle(r.Context(), application.CalcCostCommand{
		RecipeID: recipeID,
		Qty:      qty,
	})
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
