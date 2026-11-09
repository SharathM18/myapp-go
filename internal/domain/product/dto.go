package product

// import (
// 	"encoding/json"

// 	"github.com/go-playground/validator/v10"
// 	"github.com/google/uuid"
// )

// // Shared, thread-safe validator instance
// var validate = validator.New()

// type CreateProductPayload struct {
// 	Title         string          `json:"title" validate:"required,min=3,max=255"`
// 	Description   *string         `json:"description,omitempty" validate:"omitempty,max=1000"`
// 	Price         int             `json:"price" validate:"required,gt=0"`
// 	StockQuantity int             `json:"stockQuantity" validate:"gte=0"`
// 	Size          string          `json:"size" validate:"required,oneof=small medium large xlarge"`
// 	Status        *string         `json:"status,omitempty" validate:"omitempty,oneof=active inactive archived"`
// 	Metadata      json.RawMessage `json:"metadata,omitempty"`
// }

// func (p *CreateProductPayload) Validate() error {
// 	return validate.Struct(p)
// }

// // ------------------------------------------------------------
// type UpdateProductPayload struct {
// 	Title         *string         `json:"title,omitempty" validate:"omitempty,min=3,max=255"`
// 	Description   *string         `json:"description,omitempty" validate:"omitempty,max=1000"`
// 	Price         *int            `json:"price,omitempty" validate:"omitempty,gt=0"`
// 	StockQuantity *int            `json:"stockQuantity,omitempty" validate:"omitempty,gte=0"`
// 	Size          *string         `json:"size,omitempty" validate:"omitempty,oneof=small medium large xlarge"`
// 	Status        *string         `json:"status,omitempty" validate:"omitempty,oneof=active inactive archived"`
// 	Metadata      json.RawMessage `json:"metadata,omitempty"`
// }

// func (p *UpdateProductPayload) Validate() error {
// 	return validate.Struct(p)
// }

// // ------------------------------------------------------------
// type DeleteProductPayload struct {
// 	ID uuid.UUID `param:"id" validate:"required,uuid"`
// }

// func (p *DeleteProductPayload) Validate() error {
// 	return validate.Struct(p)
// }

// // ------------------------------------------------------------
// type GetProductByIDPayload struct {
// 	ID uuid.UUID `param:"id" validate:"required,uuid"`
// }

// func (p *GetProductByIDPayload) Validate() error {
// 	return validate.Struct(p)
// }

// // ------------------------------------------------------------
// type GetProductByQuery struct {
// 	// Pagination & Sorting
// 	Page  *int    `query:"page" validate:"omitempty,gte=1"`
// 	Limit *int    `query:"limit" validate:"omitempty,gte=1,lte=100"`
// 	Sort  *string `query:"sort" validate:"omitempty,oneof=created_at updated_at title price status"`
// 	Order *string `query:"order" validate:"omitempty,oneof=asc desc"`

// 	//  Filters
// 	Status   *string `query:"status" validate:"omitempty,oneof=active inactive archived"`
// 	MinPrice *int    `query:"minPrice" validate:"omitempty,gte=0"`
// 	MaxPrice *int    `query:"maxPrice" validate:"omitempty,gte=0"`
// 	Size     *string `query:"size" validate:"omitempty,oneof=small medium large xlarge"`

// 	// Extensions
// 	InStockOnly *bool   `query:"inStockOnly" validate:"omitempty"`
// 	Search      *string `query:"search" validate:"omitempty,min=1,max=100"`
// }

// func (q *GetProductByQuery) Validate() error {

// 	if err := validate.Struct(q); err != nil {
// 		return err
// 	}

// 	// Set defaults for pagination
// 	if q.Page == nil {
// 		defaultPage := 1
// 		q.Page = &defaultPage
// 	}
// 	if q.Limit == nil {
// 		defaultLimit := 20
// 		q.Limit = &defaultLimit
// 	}
// 	if q.Sort == nil {
// 		defaultSort := "created_at"
// 		q.Sort = &defaultSort
// 	}
// 	if q.Order == nil {
// 		defaultOrder := "desc"
// 		q.Order = &defaultOrder
// 	}

// 	return nil
// }
