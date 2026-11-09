// Use database migration to write a models and mentioned all 2 way relationships in the models.
// Use json tags for json serialization and db tags for database mapping.
// Use pointer for nullable fields. Use json.RawMessage for JSONB fields.

package product

// import (
// 	"encoding/json"
// 	"time"

// 	"github.com/google/uuid"
// )

// // User represents the users table
// type User struct {
// 	ID          uuid.UUID `json:"id" db:"id"`
// 	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
// 	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
// 	Username    string    `json:"username" db:"username"`
// 	Email       string    `json:"email" db:"email"`
// 	FirstName   string    `json:"firstName" db:"first_name"`
// 	LastName    string    `json:"lastName" db:"last_name"`
// 	PhoneNumber string    `json:"phoneNumber" db:"phone_number"`

// 	// Relational field
// 	Products []Product `json:"products,omitempty" db:"-"`
// }

// type Size string

// const (
// 	SizeSmall  Size = "small"
// 	SizeMedium Size = "medium"
// 	SizeLarge  Size = "large"
// 	SizeXLarge Size = "xlarge"
// )

// // Product represents the products table
// type Product struct {
// 	ID        uuid.UUID `json:"id" db:"id"`
// 	CreatedAt time.Time `json:"createdAt" db:"created_at"`
// 	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
// 	Title     string    `json:"title" db:"title"`
// 	// Use a pointer for description because it is nullable in SQL
// 	Description   *string `json:"description,omitempty" db:"description"`
// 	Price         int     `json:"price" db:"price"`
// 	StockQuantity int     `json:"stockQuantity" db:"stock_quantity"`
// 	Size          Size    `json:"size" db:"size"`
// 	Status        string  `json:"status" db:"status"`
// 	// json.RawMessage safely holds Postgres JSONB data
// 	Metadata json.RawMessage `json:"metadata,omitempty" db:"metadata"`
// 	UserID   uuid.UUID       `json:"userId" db:"user_id"`

// 	// Relational field
// 	User *User `json:"user,omitempty" db:"-"`
// }
