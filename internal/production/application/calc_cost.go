package application

import (
	"context"
	"fmt"

	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

// MaterialPriceReader запрашивает цену материала из инвентаря.
// Реализуется в infrastructure (например, через gRPC или прямой SQL-запрос).
type MaterialPriceReader interface {
	// GetPrice возвращает цену за единицу в копейках (инт для простоты).
	GetPrice(ctx context.Context, materialID kernel.IngredientID) (int64, error)
}

// CalcCostCommand — запрос себестоимости рецепта.
type CalcCostCommand struct {
	RecipeID string
	// Qty — количество букетов (по умолчанию 1).
	Qty int
}

// CostLine — строка себестоимости по одному материалу.
type CostLine struct {
	MaterialID string
	Qty        int
	UnitPrice  int64 // копейки
	Total      int64 // Qty * UnitPrice
}

// CalcCostResult — итог расчёта себестоимости.
type CalcCostResult struct {
	RecipeID  string
	Qty       int
	Lines     []CostLine
	TotalCost int64 // сумма всех Lines.Total * Qty
}

// CalcCostHandler рассчитывает себестоимость букета по рецепту:
// загружает рецепт → по каждому ингредиенту спрашивает цену → возвращает детализацию.
type CalcCostHandler struct {
	recipes RecipeRepository
	prices  MaterialPriceReader
}

func NewCalcCostHandler(recipes RecipeRepository, prices MaterialPriceReader) *CalcCostHandler {
	return &CalcCostHandler{recipes: recipes, prices: prices}
}

func (h *CalcCostHandler) Handle(ctx context.Context, cmd CalcCostCommand) (CalcCostResult, error) {
	recipeID, err := kernel.ParseRecipeID(cmd.RecipeID)
	if err != nil {
		return CalcCostResult{}, fmt.Errorf("CalcCostHandler: invalid recipe id: %w", err)
	}

	qty := cmd.Qty
	if qty <= 0 {
		qty = 1
	}

	recipe, err := h.recipes.FindByID(ctx, recipeID)
	if err != nil {
		return CalcCostResult{}, fmt.Errorf("CalcCostHandler: load recipe: %w", err)
	}

	result := CalcCostResult{
		RecipeID: cmd.RecipeID,
		Qty:      qty,
	}

	var total int64
	for _, ing := range recipe.Ingredients() {
		price, err := h.prices.GetPrice(ctx, ing.MaterialID)
		if err != nil {
			return CalcCostResult{}, fmt.Errorf("CalcCostHandler: get price for %s: %w", ing.MaterialID, err)
		}
		lineTotal := price * int64(ing.Quantity)
		result.Lines = append(result.Lines, CostLine{
			MaterialID: ing.MaterialID.String(),
			Qty:        ing.Quantity,
			UnitPrice:  price,
			Total:      lineTotal,
		})
		total += lineTotal
	}
	result.TotalCost = total * int64(qty)

	return result, nil
}
