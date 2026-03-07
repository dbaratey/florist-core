package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dbaratey/florist-core/internal/production/application"
	"github.com/dbaratey/florist-core/internal/production/domain"
)

type RecipeRepository struct {
	pool *pgxpool.Pool
}

func NewRecipeRepository(pool *pgxpool.Pool) *RecipeRepository {
	return &RecipeRepository{pool: pool}
}

func (r *RecipeRepository) Save(ctx context.Context, recipe *domain.Recipe) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("RecipeRepository.Save begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	const qRecipe = `
		INSERT INTO recipes (id, name, version, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			version = EXCLUDED.version,
			updated_at = NOW()
	`

	_, err = tx.Exec(ctx, qRecipe, recipe.ID(), recipe.Name(), recipe.Version())
	if err != nil {
		return fmt.Errorf("RecipeRepository.Save insert recipe: %w", err)
	}

	_, err = tx.Exec(ctx, "DELETE FROM recipe_ingredients WHERE recipe_id = $1", recipe.ID())
	if err != nil {
		return fmt.Errorf("RecipeRepository.Save delete ingredients: %w", err)
	}

	const qIng = `INSERT INTO recipe_ingredients (recipe_id, material_id, quantity) VALUES ($1, $2, $3)`
	for _, ing := range recipe.Ingredients() {
		_, err = tx.Exec(ctx, qIng, recipe.ID(), ing.MaterialID, ing.Quantity)
		if err != nil {
			return fmt.Errorf("RecipeRepository.Save insert ingredient: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("RecipeRepository.Save commit: %w", err)
	}

	return nil
}
