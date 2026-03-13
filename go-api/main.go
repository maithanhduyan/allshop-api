package main

import (
	"allshop-api/config"
	"allshop-api/database"
	"allshop-api/handlers"
	"allshop-api/middleware"
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

	if err := database.Seed(db); err != nil {
		log.Printf("Seed warning: %v", err)
	}

	h := handlers.New(db, cfg.JWTSecret)

	r := chi.NewRouter()

	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
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

		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)

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
		})
	})

	log.Printf("AllShop API server starting on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
