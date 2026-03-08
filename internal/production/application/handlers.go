package application

import (
	"context"
	"github.com/dbaratey/florist-core/internal/production/domain"
	"github.com/dbaratey/florist-core/internal/shared/kernel"
)

type RecipeRepository interface {
	Save(ctx context.Context, recipe *domain.Recipe) error
	FindByID(ctx context.Context, id kernel.RecipeID) (*domain.Recipe, error)
}

type CreateRecipeCommand struct {
	Name        string
	Ingredients []domain.Ingredient
}

type CreateRecipeHandler struct {
	repo RecipeRepository
}

func NewCreateRecipeHandler(repo RecipeRepository) *CreateRecipeHandler {
	return &CreateRecipeHandler{repo: repo}
}

func (h *CreateRecipeHandler) Handle(ctx context.Context, cmd CreateRecipeCommand) (kernel.RecipeID, error) {
	id := kernel.NewRecipeID()
	recipe, err := domain.NewRecipe(id, cmd.Name, cmd.Ingredients)
	if err != nil {
		return kernel.RecipeID{}, err
	}

	if err := h.repo.Save(ctx, recipe); err != nil {
		return kernel.RecipeID{}, err
	}

	return id, nil
}
