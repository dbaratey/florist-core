package kernel

import "github.com/google/uuid"

type ID struct{ val string }

func NewID() ID                  { return ID{val: uuid.NewString()} }
func IDFromString(s string) (ID, error) {
	if _, err := uuid.Parse(s); err != nil {
		return ID{}, err
	}
	return ID{val: s}, nil
}
func (id ID) String() string { return id.val }
func (id ID) IsZero() bool   { return id.val == "" }

type StoreID      struct{ ID }
type ProductID    struct{ ID }
type IngredientID struct{ ID }
type BatchID      struct{ ID }
type OrderID      struct{ ID }
type RecipeID     struct{ ID }
type JobID        struct{ ID }

func NewStoreID() StoreID           { return StoreID{NewID()} }
func NewProductID() ProductID       { return ProductID{NewID()} }
func NewIngredientID() IngredientID { return IngredientID{NewID()} }
func NewBatchID() BatchID           { return BatchID{NewID()} }
func NewOrderID() OrderID           { return OrderID{NewID()} }
func NewRecipeID() RecipeID         { return RecipeID{NewID()} }
func NewJobID() JobID               { return JobID{NewID()} }
