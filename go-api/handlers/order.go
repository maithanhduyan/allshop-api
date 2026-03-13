package handlers

import (
	"allshop-api/models"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Phone == "" || req.Address == "" {
		writeError(w, http.StatusBadRequest, "name, phone and address are required")
		return
	}

	// Get cart items
	rows, err := h.db.Query(
		`SELECT ci.product_id, p.name, p.images[1], p.price, ci.quantity
		 FROM cart_items ci
		 JOIN products p ON p.id = ci.product_id
		 WHERE ci.user_id = $1`, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query cart")
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	var total float64
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Image, &item.Price, &item.Quantity); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan cart item")
			return
		}
		total += item.Price * float64(item.Quantity)
		items = append(items, item)
	}

	if len(items) == 0 {
		writeError(w, http.StatusBadRequest, "cart is empty")
		return
	}

	// Create order in a transaction
	tx, err := h.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	var order models.Order
	err = tx.QueryRow(
		`INSERT INTO orders (user_id, total, name, phone, address, note)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, total, status, name, phone, address, note, created_at`,
		userID, total, req.Name, req.Phone, req.Address, req.Note,
	).Scan(&order.ID, &order.UserID, &order.Total, &order.Status,
		&order.Name, &order.Phone, &order.Address, &order.Note, &order.CreatedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create order")
		return
	}

	for _, item := range items {
		_, err := tx.Exec(
			`INSERT INTO order_items (order_id, product_id, name, image, price, quantity)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			order.ID, item.ProductID, item.Name, item.Image, item.Price, item.Quantity,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create order items")
			return
		}

		// Decrease stock
		_, err = tx.Exec(
			`UPDATE products SET stock = stock - $1 WHERE id = $2`,
			item.Quantity, item.ProductID,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update stock")
			return
		}
	}

	// Clear cart
	_, err = tx.Exec(`DELETE FROM cart_items WHERE user_id = $1`, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to clear cart")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit order")
		return
	}

	order.Items = items
	writeJSON(w, http.StatusCreated, order)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	rows, err := h.db.Query(
		`SELECT id, user_id, total, status, name, phone, address, note, created_at
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query orders")
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(&o.ID, &o.UserID, &o.Total, &o.Status,
			&o.Name, &o.Phone, &o.Address, &o.Note, &o.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan order")
			return
		}
		orders = append(orders, o)
	}

	if orders == nil {
		orders = []models.Order{}
	}

	writeJSON(w, http.StatusOK, orders)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	orderID := chi.URLParam(r, "id")

	var order models.Order
	err := h.db.QueryRow(
		`SELECT id, user_id, total, status, name, phone, address, note, created_at
		 FROM orders WHERE id = $1 AND user_id = $2`, orderID, userID,
	).Scan(&order.ID, &order.UserID, &order.Total, &order.Status,
		&order.Name, &order.Phone, &order.Address, &order.Note, &order.CreatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "order not found")
		return
	}

	// Get order items
	rows, err := h.db.Query(
		`SELECT product_id, name, image, price, quantity FROM order_items WHERE order_id = $1`,
		orderID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query order items")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Image, &item.Price, &item.Quantity); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan order item")
			return
		}
		order.Items = append(order.Items, item)
	}

	if order.Items == nil {
		order.Items = []models.OrderItem{}
	}

	writeJSON(w, http.StatusOK, order)
}
