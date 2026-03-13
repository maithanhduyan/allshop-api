package main

import (
	"allshop-api/cache"
	"allshop-api/config"
	"allshop-api/database"
	"allshop-api/handlers"
	"allshop-api/middleware"
	"allshop-api/storage"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize Redis cache
	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Printf("Warning: Redis not available, running without cache: %v", err)
	} else {
		defer redisCache.Close()
	}

	// Initialize MinIO storage
	minioStorage, err := storage.New(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioPublicURL,
	)
	if err != nil {
		log.Printf("Warning: MinIO not available, running without object storage: %v", err)
	}

	// Seed database (with MinIO image migration)
	if err := database.Seed(db, minioStorage); err != nil {
		log.Printf("Seed warning: %v", err)
	}

	h := handlers.New(db, cfg.JWTSecret, redisCache, minioStorage)

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://127.0.0.1:3000", "http://127.0.0.1:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Swagger UI — OpenAPI docs at /docs/
	specData, err := os.ReadFile("docs/openapi.yaml")
	if err != nil {
		log.Printf("Warning: could not load openapi.yaml: %v", err)
		specData = []byte("openapi: '3.0.3'\ninfo:\n  title: AllShop API\n  version: '1.0.0'\npaths: {}")
	}
	docsHandler := handlers.SwaggerUI(specData)
	r.Get("/docs", docsHandler)
	r.Get("/docs/*", docsHandler)

	r.Route("/api", func(r chi.Router) {
		// Public routes
		r.Get("/products", h.ListProducts)
		r.Get("/products/{id}", h.GetProduct)
		r.Get("/categories", h.ListCategories)

		// Image proxy (serves images from MinIO through the API)
		r.Get("/images/*", h.ServeImage)

		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)

		// Public invoice access (by UUID public key, only issued invoices)
		r.Get("/p/invoices/{publicKey}", h.GetPublicInvoice)
		r.Get("/p/invoices/{publicKey}/pdf", h.ExportPublicInvoicePDF)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))

			r.Post("/auth/logout", h.Logout)

			r.Get("/users/profile", h.GetProfile)
			r.Put("/users/profile", h.UpdateProfile)

			r.Get("/cart", h.GetCart)
			r.Post("/cart/items", h.AddCartItem)
			r.Put("/cart/items/{productId}", h.UpdateCartItem)
			r.Delete("/cart/items/{productId}", h.RemoveCartItem)

			r.Post("/orders", h.CreateOrder)
			r.Get("/orders", h.ListOrders)
			r.Get("/orders/{id}", h.GetOrder)

			// Invoice routes
			r.Post("/orders/{id}/invoice", h.CreateInvoice)
			r.Get("/invoices", h.ListInvoices)
			r.Get("/invoices/{id}", h.GetInvoice)
			r.Put("/invoices/{id}/issue", h.IssueInvoice)
			r.Put("/invoices/{id}/cancel", h.CancelInvoice)

			// Accounting routes
			r.Get("/accounts", h.ListAccounts)
			r.Get("/journal-entries", h.ListJournalEntries)
			r.Get("/journal-entries/{id}", h.GetJournalEntry)
			r.Get("/trial-balance", h.GetTrialBalance)

			// PDF export
			r.Get("/invoices/{id}/pdf", h.ExportInvoicePDF)

			// Reports & dashboard
			r.Get("/reports/revenue", h.RevenueReport)
			r.Get("/reports/tax", h.TaxReport)
			r.Get("/reports/account-balance", h.AccountBalanceReport)
			r.Get("/reports/dashboard", h.DashboardSummary)
		})
	})

	log.Printf("AllShop API server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
