package domain

import (
	"errors"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

var (
	ErrEmptyRecipeName = errors.New("recipe name cannot be empty")
	ErrNoIngredients   = errors.New("recipe must have at least one ingredient")
)

// Ingredient represents a component of a bouquet or composition
type Ingredient struct {
	MaterialID kernel.IngredientID
	Quantity   int
}

// Recipe is the aggregate root for a flower composition (bouquet, etc.)
type Recipe struct {
	id          kernel.RecipeID
	name        string
	ingredients []Ingredient
	version     int

	events []kernel.DomainEvent
}

func NewRecipe(id kernel.RecipeID, name string, ingredients []Ingredient) (*Recipe, error) {
	if name == "" {
		return nil, ErrEmptyRecipeName
	}
	if len(ingredients) == 0 {
		return nil, ErrNoIngredients
	}

	r := &Recipe{
		id:          id,
		name:        name,
		ingredients: ingredients,
	}

	return r, nil
}
