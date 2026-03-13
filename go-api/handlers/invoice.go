package handlers

import (
	"allshop-api/models"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	defaultSellerName    = "Công ty TNHH AllShop Việt Nam"
	defaultSellerTaxCode = "0123456789"
	defaultSellerAddress = "123 Nguyễn Huệ, Quận 1, TP. Hồ Chí Minh"
	defaultTaxRate       = 0.08
)

func (h *Handler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	orderID := chi.URLParam(r, "id")

	// Parse optional request body (buyerTaxCode, taxRate)
	var req models.CreateInvoiceRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	taxRate := defaultTaxRate
	if req.TaxRate > 0 && req.TaxRate <= 1 {
		taxRate = req.TaxRate
	}

	// Verify order belongs to user and is in valid status
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

	if order.Status != "confirmed" && order.Status != "delivered" && order.Status != "shipping" {
		writeError(w, http.StatusBadRequest, "order must be confirmed, shipping or delivered to create invoice")
		return
	}

	// Check if invoice already exists for this order
	var existingCount int
	h.db.QueryRow(
		`SELECT COUNT(*) FROM invoices WHERE order_id = $1 AND status != 'cancelled'`, orderID,
	).Scan(&existingCount)
	if existingCount > 0 {
		writeError(w, http.StatusConflict, "an active invoice already exists for this order")
		return
	}

	// Get buyer info
	var buyerEmail string
	h.db.QueryRow(`SELECT email FROM users WHERE id = $1`, userID).Scan(&buyerEmail)

	// Get order items
	rows, err := h.db.Query(
		`SELECT product_id, name, price, quantity FROM order_items WHERE order_id = $1`, orderID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query order items")
		return
	}
	defer rows.Close()

	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		if err := rows.Scan(&item.ProductID, &item.Name, &item.Price, &item.Quantity); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan order item")
			return
		}
		items = append(items, item)
	}

	// Generate invoice number: HD-YYYYMMDD-XXXX
	now := time.Now()
	dateStr := now.Format("20060102")
	var seq int
	h.db.QueryRow(
		`SELECT COUNT(*) FROM invoices WHERE invoice_number LIKE $1`,
		fmt.Sprintf("HD-%s-%%", dateStr),
	).Scan(&seq)
	invoiceNumber := fmt.Sprintf("HD-%s-%04d", dateStr, seq+1)

	// Calculate amounts
	subtotal := order.Total
	taxAmount := roundMoney(subtotal * taxRate)
	totalAmount := subtotal + taxAmount

	// Create invoice in transaction
	tx, err := h.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	var invoice models.Invoice
	err = tx.QueryRow(
		`INSERT INTO invoices (
			invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, 'draft')
		RETURNING id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			issued_at, cancelled_at, created_at`,
		invoiceNumber, orderID, userID,
		defaultSellerName, defaultSellerTaxCode, defaultSellerAddress,
		order.Name, req.BuyerTaxCode, order.Address, buyerEmail,
		subtotal, taxRate, taxAmount, totalAmount,
	).Scan(
		&invoice.ID, &invoice.InvoiceNumber, &invoice.OrderID, &invoice.UserID,
		&invoice.SellerName, &invoice.SellerTaxCode, &invoice.SellerAddress,
		&invoice.BuyerName, &invoice.BuyerTaxCode, &invoice.BuyerAddress, &invoice.BuyerEmail,
		&invoice.Subtotal, &invoice.TaxRate, &invoice.TaxAmount, &invoice.TotalAmount, &invoice.Status,
		&invoice.IssuedAt, &invoice.CancelledAt, &invoice.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create invoice")
		return
	}

	// Create invoice items
	for _, item := range items {
		lineTotal := item.Price * float64(item.Quantity)
		lineTax := roundMoney(lineTotal * taxRate)
		lineAmount := lineTotal + lineTax

		var invItem models.InvoiceItem
		err = tx.QueryRow(
			`INSERT INTO invoice_items (
				invoice_id, product_id, product_name, unit, quantity,
				unit_price, tax_rate, tax_amount, total_amount
			) VALUES ($1, $2, $3, 'Cái', $4, $5, $6, $7, $8)
			RETURNING id, invoice_id, product_id, product_name, unit, quantity,
				unit_price, tax_rate, tax_amount, total_amount`,
			invoice.ID, item.ProductID, item.Name, item.Quantity,
			item.Price, taxRate, lineTax, lineAmount,
		).Scan(
			&invItem.ID, &invItem.InvoiceID, &invItem.ProductID, &invItem.ProductName,
			&invItem.Unit, &invItem.Quantity, &invItem.UnitPrice, &invItem.TaxRate,
			&invItem.TaxAmount, &invItem.TotalAmount,
		)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create invoice item")
			return
		}
		invoice.Items = append(invoice.Items, invItem)
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit invoice")
		return
	}

	if invoice.Items == nil {
		invoice.Items = []models.InvoiceItem{}
	}

	writeJSON(w, http.StatusCreated, invoice)
}

func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	rows, err := h.db.Query(
		`SELECT id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			issued_at, cancelled_at, created_at
		 FROM invoices WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query invoices")
		return
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var inv models.Invoice
		if err := rows.Scan(
			&inv.ID, &inv.InvoiceNumber, &inv.OrderID, &inv.UserID,
			&inv.SellerName, &inv.SellerTaxCode, &inv.SellerAddress,
			&inv.BuyerName, &inv.BuyerTaxCode, &inv.BuyerAddress, &inv.BuyerEmail,
			&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.TotalAmount, &inv.Status,
			&inv.IssuedAt, &inv.CancelledAt, &inv.CreatedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan invoice")
			return
		}
		inv.Items = []models.InvoiceItem{}
		invoices = append(invoices, inv)
	}

	if invoices == nil {
		invoices = []models.Invoice{}
	}

	writeJSON(w, http.StatusOK, invoices)
}

func (h *Handler) GetInvoice(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	invoiceID := chi.URLParam(r, "id")

	var inv models.Invoice
	err := h.db.QueryRow(
		`SELECT id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			issued_at, cancelled_at, created_at
		 FROM invoices WHERE id = $1 AND user_id = $2`, invoiceID, userID,
	).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.OrderID, &inv.UserID,
		&inv.SellerName, &inv.SellerTaxCode, &inv.SellerAddress,
		&inv.BuyerName, &inv.BuyerTaxCode, &inv.BuyerAddress, &inv.BuyerEmail,
		&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.TotalAmount, &inv.Status,
		&inv.IssuedAt, &inv.CancelledAt, &inv.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	// Get invoice items
	rows, err := h.db.Query(
		`SELECT id, invoice_id, product_id, product_name, unit, quantity,
			unit_price, tax_rate, tax_amount, total_amount
		 FROM invoice_items WHERE invoice_id = $1`, invoiceID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query invoice items")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var item models.InvoiceItem
		if err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.ProductID, &item.ProductName,
			&item.Unit, &item.Quantity, &item.UnitPrice, &item.TaxRate,
			&item.TaxAmount, &item.TotalAmount,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan invoice item")
			return
		}
		inv.Items = append(inv.Items, item)
	}

	if inv.Items == nil {
		inv.Items = []models.InvoiceItem{}
	}

	writeJSON(w, http.StatusOK, inv)
}

func (h *Handler) IssueInvoice(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	invoiceID := chi.URLParam(r, "id")

	var status string
	err := h.db.QueryRow(
		`SELECT status FROM invoices WHERE id = $1 AND user_id = $2`, invoiceID, userID,
	).Scan(&status)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	if status != "draft" {
		writeError(w, http.StatusBadRequest, "only draft invoices can be issued")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	now := time.Now()
	var inv models.Invoice
	err = tx.QueryRow(
		`UPDATE invoices SET status = 'issued', issued_at = $1
		 WHERE id = $2 AND user_id = $3
		 RETURNING id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			issued_at, cancelled_at, created_at`,
		now, invoiceID, userID,
	).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.OrderID, &inv.UserID,
		&inv.SellerName, &inv.SellerTaxCode, &inv.SellerAddress,
		&inv.BuyerName, &inv.BuyerTaxCode, &inv.BuyerAddress, &inv.BuyerEmail,
		&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.TotalAmount, &inv.Status,
		&inv.IssuedAt, &inv.CancelledAt, &inv.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue invoice")
		return
	}

	// Auto-create journal entry: bán hàng
	// Nợ 131 (Phải thu khách hàng) = totalAmount
	// Có 5111 (Doanh thu bán hàng hóa) = subtotal
	// Có 33311 (Thuế GTGT đầu ra) = taxAmount
	desc := fmt.Sprintf("Phát hành hóa đơn %s — %s", inv.InvoiceNumber, inv.BuyerName)
	_, err = createJournalEntry(tx, inv.ID, desc, []journalLineInput{
		{accountCode: "131", description: "Phải thu khách hàng", debit: inv.TotalAmount, credit: 0},
		{accountCode: "5111", description: "Doanh thu bán hàng hóa", debit: 0, credit: inv.Subtotal},
		{accountCode: "33311", description: "Thuế GTGT đầu ra", debit: 0, credit: inv.TaxAmount},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create journal entry")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	inv.Items = []models.InvoiceItem{}
	writeJSON(w, http.StatusOK, inv)
}

func (h *Handler) CancelInvoice(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	invoiceID := chi.URLParam(r, "id")

	var status string
	err := h.db.QueryRow(
		`SELECT status FROM invoices WHERE id = $1 AND user_id = $2`, invoiceID, userID,
	).Scan(&status)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	if status == "cancelled" {
		writeError(w, http.StatusBadRequest, "invoice is already cancelled")
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start transaction")
		return
	}
	defer tx.Rollback()

	now := time.Now()
	var inv models.Invoice
	err = tx.QueryRow(
		`UPDATE invoices SET status = 'cancelled', cancelled_at = $1
		 WHERE id = $2 AND user_id = $3
		 RETURNING id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			issued_at, cancelled_at, created_at`,
		now, invoiceID, userID,
	).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.OrderID, &inv.UserID,
		&inv.SellerName, &inv.SellerTaxCode, &inv.SellerAddress,
		&inv.BuyerName, &inv.BuyerTaxCode, &inv.BuyerAddress, &inv.BuyerEmail,
		&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.TotalAmount, &inv.Status,
		&inv.IssuedAt, &inv.CancelledAt, &inv.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel invoice")
		return
	}

	// If invoice was "issued", reverse its journal entry
	if status == "issued" {
		var originalEntryID string
		err := tx.QueryRow(
			`SELECT id FROM journal_entries
			 WHERE invoice_id = $1 AND status = 'posted'
			 ORDER BY created_at DESC LIMIT 1`, invoiceID,
		).Scan(&originalEntryID)
		if err == nil {
			desc := fmt.Sprintf("Hủy hóa đơn %s — ghi đảo", inv.InvoiceNumber)
			if _, err := createReversalEntry(tx, originalEntryID, inv.ID, desc); err != nil {
				writeError(w, http.StatusInternalServerError, "failed to reverse journal entry")
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to commit transaction")
		return
	}

	inv.Items = []models.InvoiceItem{}
	writeJSON(w, http.StatusOK, inv)
}

func roundMoney(v float64) float64 {
	return math.Round(v*100) / 100
}
