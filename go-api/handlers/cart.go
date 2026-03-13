package handlers

import (
	"allshop-api/models"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	rows, err := h.db.Query(
		`SELECT ci.product_id, p.name, p.images[1], p.price, ci.quantity, p.stock
		 FROM cart_items ci
		 JOIN products p ON p.id = ci.product_id
		 WHERE ci.user_id = $1`, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query cart")
		return
	}
	defer rows.Close()

	items := []models.CartItem{}
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Image, &item.Price, &item.Quantity, &item.Stock); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan cart item")
			return
		}
		items = append(items, item)
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) AddCartItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req models.AddCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ProductID == "" || req.Quantity < 1 {
		writeError(w, http.StatusBadRequest, "productId and quantity (>=1) are required")
		return
	}

	// Verify product exists and has stock
	var stock int
	err := h.db.QueryRow(`SELECT stock FROM products WHERE id = $1`, req.ProductID).Scan(&stock)
	if err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if req.Quantity > stock {
		writeError(w, http.StatusBadRequest, "requested quantity exceeds stock")
		return
	}

	_, err = h.db.Exec(
		`INSERT INTO cart_items (user_id, product_id, quantity) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, product_id) DO UPDATE SET quantity = cart_items.quantity + $3`,
		userID, req.ProductID, req.Quantity,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add cart item")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "item added to cart"})
}

func (h *Handler) UpdateCartItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	productID := chi.URLParam(r, "productId")

	var req models.UpdateCartItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Quantity < 1 {
		writeError(w, http.StatusBadRequest, "quantity must be at least 1")
		return
	}

	result, err := h.db.Exec(
		`UPDATE cart_items SET quantity = $1 WHERE user_id = $2 AND product_id = $3`,
		req.Quantity, userID, productID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update cart item")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "cart item not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "cart item updated"})
}

func (h *Handler) RemoveCartItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	productID := chi.URLParam(r, "productId")

	result, err := h.db.Exec(
		`DELETE FROM cart_items WHERE user_id = $1 AND product_id = $2`,
		userID, productID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to remove cart item")
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		writeError(w, http.StatusNotFound, "cart item not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "cart item removed"})
}
