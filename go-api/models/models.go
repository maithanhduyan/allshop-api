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

type Invoice struct {
	ID            string        `json:"id"`
	PublicKey     string        `json:"publicKey"`
	InvoiceNumber string        `json:"invoiceNumber"`
	OrderID       string        `json:"orderId"`
	UserID        string        `json:"userId"`
	SellerName    string        `json:"sellerName"`
	SellerTaxCode string        `json:"sellerTaxCode"`
	SellerAddress string        `json:"sellerAddress"`
	BuyerName     string        `json:"buyerName"`
	BuyerTaxCode  *string       `json:"buyerTaxCode,omitempty"`
	BuyerAddress  string        `json:"buyerAddress"`
	BuyerEmail    string        `json:"buyerEmail"`
	Subtotal      float64       `json:"subtotal"`
	TaxRate       float64       `json:"taxRate"`
	TaxAmount     float64       `json:"taxAmount"`
	TotalAmount   float64       `json:"totalAmount"`
	Status        string        `json:"status"`
	Items         []InvoiceItem `json:"items"`
	IssuedAt      *time.Time    `json:"issuedAt,omitempty"`
	CancelledAt   *time.Time    `json:"cancelledAt,omitempty"`
	CreatedAt     time.Time     `json:"createdAt"`
}

type InvoiceItem struct {
	ID          string  `json:"id"`
	InvoiceID   string  `json:"invoiceId"`
	ProductID   string  `json:"productId"`
	ProductName string  `json:"productName"`
	Unit        string  `json:"unit"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unitPrice"`
	TaxRate     float64 `json:"taxRate"`
	TaxAmount   float64 `json:"taxAmount"`
	TotalAmount float64 `json:"totalAmount"`
}

type CreateInvoiceRequest struct {
	BuyerTaxCode *string `json:"buyerTaxCode,omitempty"`
	TaxRate      float64 `json:"taxRate"`
}

type CancelInvoiceRequest struct {
	Reason string `json:"reason"`
}

// ── Accounting ──

type Account struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Type       string `json:"type"` // asset, liability, equity, revenue, expense
	ParentCode string `json:"parentCode,omitempty"`
	Level      int    `json:"level"`
	IsActive   bool   `json:"isActive"`
}

type JournalEntry struct {
	ID          string        `json:"id"`
	EntryNumber string        `json:"entryNumber"`
	InvoiceID   *string       `json:"invoiceId,omitempty"`
	Description string        `json:"description"`
	EntryDate   string        `json:"entryDate"`
	Status      string        `json:"status"` // posted, reversed
	ReversedBy  *string       `json:"reversedBy,omitempty"`
	Reverses    *string       `json:"reverses,omitempty"`
	Lines       []JournalLine `json:"lines"`
	CreatedAt   time.Time     `json:"createdAt"`
}

type JournalLine struct {
	ID             string  `json:"id"`
	JournalEntryID string  `json:"journalEntryId"`
	AccountCode    string  `json:"accountCode"`
	AccountName    string  `json:"accountName,omitempty"`
	Description    string  `json:"description"`
	Debit          float64 `json:"debit"`
	Credit         float64 `json:"credit"`
}

type ProductListResponse struct {
	Products []Product `json:"products"`
	Total    int       `json:"total"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
