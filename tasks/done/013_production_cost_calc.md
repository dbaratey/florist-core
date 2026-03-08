# Task 013: Production Cost Calculation

## Status: DONE

## Goal
Расчёт себестоимости букета по рецепту с детализацией по материалам.

## Acceptance Criteria
- [x] `MaterialPriceReader` интерфейс в `application` (порт к инвентарю)
- [x] `CalcCostCommand`, `CostLine`, `CalcCostResult` structs
- [x] `CalcCostHandler.Handle` загружает рецепт → спрашивает цену каждого ингредиента → возвращает `CalcCostResult`
- [x] `RecipeRepository.FindByID` добавлен в интерфейс
- [x] `Recipe.Ingredients()` getter добавлен в domain
- [x] HTTP `GET /api/v1/production/recipes/{id}/cost?qty=N` endpoint
- [x] `Handler` struct содержит `calcCost *application.CalcCostHandler`

## Files
- `internal/production/application/calc_cost.go` (new)
- `internal/production/application/handlers.go` (updated: FindByID)
- `internal/production/domain/recipe.go` (updated: getters)
- `internal/production/http/handlers.go` (updated: CalcCost endpoint)

## Commits
- `feat(production/app): add CalcCostHandler and MaterialPriceReader`
- `feat(production/app): add FindByID to RecipeRepository interface`
- `feat(production/domain): add Ingredients/ID/Name/Version getters to Recipe`
- `feat(production/http): add CalcCost endpoint GET /recipes/{id}/cost`
