package models

import "time"

type Product struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Slug           string            `json:"slug"`
	Description    string            `json:"description"`
	Price          float64           `json:"price"`
	OriginalPrice  *float64          `json:"originalPrice,omitempty"`
	Images         []string          `json:"images"`
	Category       string            `json:"category"`
	Brand          string            `json:"brand"`
	Rating         float64           `json:"rating"`
	ReviewCount    int               `json:"reviewCount"`
	Stock          int               `json:"stock"`
	Specifications map[string]string `json:"specifications,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
}

type Category struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Emoji string `json:"emoji"`
}

type User struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone,omitempty"`
	Avatar    *string `json:"avatar,omitempty"`
	Password  string  `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

type CartItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Image     string  `json:"image"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
	Stock     int     `json:"stock"`
}

type Order struct {
	ID         string      `json:"id"`
	UserID     string      `json:"userId"`
	Items      []OrderItem `json:"items"`
	Total      float64     `json:"total"`
	Status     string      `json:"status"`
	Name       string      `json:"name"`
	Phone      string      `json:"phone"`
	Address    string      `json:"address"`
	Note       string      `json:"note,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}

type OrderItem struct {
	ProductID string  `json:"productId"`
	Name      string  `json:"name"`
	Image     string  `json:"image"`
	Price     float64 `json:"price"`
	Quantity  int     `json:"quantity"`
}

// Request/Response types

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type AddCartItemRequest struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type UpdateCartItemRequest struct {
	Quantity int `json:"quantity"`
}

type CreateOrderRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	Note    string `json:"note,omitempty"`
}

type UpdateProfileRequest struct {
	Name  string  `json:"name"`
	Phone *string `json:"phone,omitempty"`
}

type ProductListResponse struct {
	Products []Product `json:"products"`
	Total    int       `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
