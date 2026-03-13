package handlers

import (
	"allshop-api/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

const (
	productListTTL = 5 * time.Minute
	productTTL     = 10 * time.Minute
	categoryTTL    = 30 * time.Minute
)

func (h *Handler) ListProducts(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	// Try cache first
	if h.cache != nil {
		cacheKey := fmt.Sprintf("products:list:%s:%s", category, search)
		if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write([]byte(cached))
			return
		} else if err != redis.Nil {
			log.Printf("cache get error: %v", err)
		}
	}

	query := `SELECT id, name, slug, description, price, original_price, images, category, brand, rating, review_count, stock, specifications, created_at FROM products WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if category != "" {
		query += ` AND category = $` + strconv.Itoa(argIdx)
		args = append(args, category)
		argIdx++
	}

	if search != "" {
		query += ` AND (LOWER(name) LIKE $` + strconv.Itoa(argIdx) + ` OR LOWER(description) LIKE $` + strconv.Itoa(argIdx) + `)`
		args = append(args, "%"+strings.ToLower(search)+"%")
		argIdx++
	}

	query += ` ORDER BY created_at DESC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query products")
		return
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		var specs []byte
		err := rows.Scan(
			&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.OriginalPrice,
			pq.Array(&p.Images), &p.Category, &p.Brand, &p.Rating, &p.ReviewCount,
			&p.Stock, &specs, &p.CreatedAt,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan product")
			return
		}
		if len(specs) > 0 {
			json.Unmarshal(specs, &p.Specifications)
		}
		products = append(products, p)
	}

	response := models.ProductListResponse{
		Products: products,
		Total:    len(products),
	}

	// Cache the response
	if h.cache != nil {
		cacheKey := fmt.Sprintf("products:list:%s:%s", category, search)
		if data, err := json.Marshal(response); err == nil {
			if err := h.cache.Set(r.Context(), cacheKey, string(data), productListTTL); err != nil {
				log.Printf("cache set error: %v", err)
			}
		}
	}

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Try cache first
	if h.cache != nil {
		cacheKey := fmt.Sprintf("products:%s", id)
		if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write([]byte(cached))
			return
		} else if err != redis.Nil {
			log.Printf("cache get error: %v", err)
		}
	}

	var p models.Product
	var specs []byte
	err := h.db.QueryRow(
		`SELECT id, name, slug, description, price, original_price, images, category, brand, rating, review_count, stock, specifications, created_at
		 FROM products WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.Name, &p.Slug, &p.Description, &p.Price, &p.OriginalPrice,
		pq.Array(&p.Images), &p.Category, &p.Brand, &p.Rating, &p.ReviewCount,
		&p.Stock, &specs, &p.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if len(specs) > 0 {
		json.Unmarshal(specs, &p.Specifications)
	}

	// Cache the response
	if h.cache != nil {
		cacheKey := fmt.Sprintf("products:%s", id)
		if data, err := json.Marshal(p); err == nil {
			if err := h.cache.Set(r.Context(), cacheKey, string(data), productTTL); err != nil {
				log.Printf("cache set error: %v", err)
			}
		}
	}

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	// Try cache first
	if h.cache != nil {
		cacheKey := "categories:list"
		if cached, err := h.cache.Get(r.Context(), cacheKey); err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write([]byte(cached))
			return
		} else if err != redis.Nil {
			log.Printf("cache get error: %v", err)
		}
	}

	rows, err := h.db.Query(`SELECT id, name, emoji FROM categories ORDER BY id`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query categories")
		return
	}
	defer rows.Close()

	categories := []models.Category{}
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Emoji); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan category")
			return
		}
		categories = append(categories, c)
	}

	// Cache the response
	if h.cache != nil {
		cacheKey := "categories:list"
		if data, err := json.Marshal(categories); err == nil {
			if err := h.cache.Set(r.Context(), cacheKey, string(data), categoryTTL); err != nil {
				log.Printf("cache set error: %v", err)
			}
		}
	}

	w.Header().Set("X-Cache", "MISS")
	writeJSON(w, http.StatusOK, categories)
}
