package handlers

import (
	"allshop-api/models"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-pdf/fpdf"
)

// GetPublicInvoice returns invoice data by its public UUID key (no auth required).
// Only issued invoices are accessible publicly.
func (h *Handler) GetPublicInvoice(w http.ResponseWriter, r *http.Request) {
	publicKey := chi.URLParam(r, "publicKey")

	var inv models.Invoice
	err := h.db.QueryRow(
		`SELECT id, public_key, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, buyer_tax_code, buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			issued_at, cancelled_at, created_at
		 FROM invoices WHERE public_key = $1 AND status = 'issued'`, publicKey,
	).Scan(
		&inv.ID, &inv.PublicKey, &inv.InvoiceNumber, &inv.OrderID, &inv.UserID,
		&inv.SellerName, &inv.SellerTaxCode, &inv.SellerAddress,
		&inv.BuyerName, &inv.BuyerTaxCode, &inv.BuyerAddress, &inv.BuyerEmail,
		&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.TotalAmount, &inv.Status,
		&inv.IssuedAt, &inv.CancelledAt, &inv.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	rows, err := h.db.Query(
		`SELECT id, invoice_id, product_id, product_name, unit, quantity,
			unit_price, tax_rate, tax_amount, total_amount
		 FROM invoice_items WHERE invoice_id = $1`, inv.ID,
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

// ExportPublicInvoicePDF generates a PDF for a publicly-accessible issued invoice.
func (h *Handler) ExportPublicInvoicePDF(w http.ResponseWriter, r *http.Request) {
	publicKey := chi.URLParam(r, "publicKey")

	var inv struct {
		ID, InvoiceNumber, OrderID, UserID                       string
		SellerName, SellerTaxCode, SellerAddress                  string
		BuyerName, BuyerAddress, BuyerEmail                       string
		BuyerTaxCode                                              *string
		Subtotal, TaxRate, TaxAmount, TotalAmount                 float64
		Status                                                    string
		IssuedAt                                                  *string
		CreatedAt                                                 string
	}
	err := h.db.QueryRow(
		`SELECT id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, COALESCE(buyer_tax_code, ''), buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			TO_CHAR(COALESCE(issued_at, created_at), 'DD/MM/YYYY HH24:MI'),
			created_at
		 FROM invoices WHERE public_key = $1 AND status = 'issued'`, publicKey,
	).Scan(
		&inv.ID, &inv.InvoiceNumber, &inv.OrderID, &inv.UserID,
		&inv.SellerName, &inv.SellerTaxCode, &inv.SellerAddress,
		&inv.BuyerName, &inv.BuyerTaxCode, &inv.BuyerAddress, &inv.BuyerEmail,
		&inv.Subtotal, &inv.TaxRate, &inv.TaxAmount, &inv.TotalAmount, &inv.Status,
		&inv.IssuedAt, &inv.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}

	type pdfItem struct {
		ProductName string
		Unit        string
		Quantity    int
		UnitPrice   float64
		TaxAmount   float64
		TotalAmount float64
	}
	rows, err := h.db.Query(
		`SELECT product_name, unit, quantity, unit_price, tax_amount, total_amount
		 FROM invoice_items WHERE invoice_id = $1 ORDER BY id`, inv.ID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load items")
		return
	}
	defer rows.Close()

	var items []pdfItem
	for rows.Next() {
		var it pdfItem
		if err := rows.Scan(&it.ProductName, &it.Unit, &it.Quantity, &it.UnitPrice, &it.TaxAmount, &it.TotalAmount); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan item")
			return
		}
		items = append(items, it)
	}

	// Build PDF (same layout as ExportInvoicePDF in pdf.go)
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(190, 10, "HOA DON GIA TRI GIA TANG", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(190, 5, "(VAT INVOICE)", "", 1, "C", false, 0, "")
	pdf.Ln(3)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(95, 6, fmt.Sprintf("So (No.): %s", inv.InvoiceNumber), "", 0, "L", false, 0, "")
	issuedStr := ""
	if inv.IssuedAt != nil {
		issuedStr = *inv.IssuedAt
	}
	pdf.CellFormat(95, 6, fmt.Sprintf("Ngay (Date): %s", issuedStr), "", 1, "R", false, 0, "")
	pdf.Ln(3)

	statusLabel := map[string]string{"draft": "NHAP", "issued": "DA PHAT HANH", "cancelled": "DA HUY"}
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(190, 5, fmt.Sprintf("Trang thai: %s", statusLabel[inv.Status]), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(190, 6, "Don vi ban hang (Seller):", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(190, 5, fmt.Sprintf("  Ten (Name): %s", asciiFold(inv.SellerName)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, fmt.Sprintf("  Ma so thue (Tax code): %s", inv.SellerTaxCode), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, fmt.Sprintf("  Dia chi (Address): %s", asciiFold(inv.SellerAddress)), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(190, 6, "Don vi mua hang (Buyer):", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(190, 5, fmt.Sprintf("  Ten (Name): %s", asciiFold(inv.BuyerName)), "", 1, "L", false, 0, "")
	buyerTax := ""
	if inv.BuyerTaxCode != nil && *inv.BuyerTaxCode != "" {
		buyerTax = *inv.BuyerTaxCode
	}
	if buyerTax != "" {
		pdf.CellFormat(190, 5, fmt.Sprintf("  Ma so thue (Tax code): %s", buyerTax), "", 1, "L", false, 0, "")
	}
	pdf.CellFormat(190, 5, fmt.Sprintf("  Dia chi (Address): %s", asciiFold(inv.BuyerAddress)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, fmt.Sprintf("  Email: %s", inv.BuyerEmail), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	colW := []float64{10, 70, 20, 15, 25, 25, 25}
	headers := []string{"STT", "Ten hang hoa", "DVT", "SL", "Don gia", "Thue", "Thanh tien"}
	aligns := []string{"C", "L", "C", "C", "R", "R", "R"}
	for i, hdr := range headers {
		pdf.CellFormat(colW[i], 7, hdr, "1", 0, aligns[i], true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 9)
	for idx, it := range items {
		pdf.CellFormat(colW[0], 6, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[1], 6, truncStr(asciiFold(it.ProductName), 38), "1", 0, "L", false, 0, "")
		pdf.CellFormat(colW[2], 6, it.Unit, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[3], 6, fmt.Sprintf("%d", it.Quantity), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colW[4], 6, fmtMoney(it.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colW[5], 6, fmtMoney(it.TaxAmount), "1", 0, "R", false, 0, "")
		pdf.CellFormat(colW[6], 6, fmtMoney(it.TotalAmount), "1", 0, "R", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(3)
	pdf.SetFont("Helvetica", "", 10)
	totalLabelW := float64(140)
	totalValW := float64(50)
	pdf.CellFormat(totalLabelW, 6, "Cong tien hang (Subtotal):", "", 0, "R", false, 0, "")
	pdf.CellFormat(totalValW, 6, fmtMoney(inv.Subtotal), "", 1, "R", false, 0, "")

	pdf.CellFormat(totalLabelW, 6, fmt.Sprintf("Thue GTGT (VAT %d%%):", int(inv.TaxRate*100)), "", 0, "R", false, 0, "")
	pdf.CellFormat(totalValW, 6, fmtMoney(inv.TaxAmount), "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(totalLabelW, 8, "Tong cong (Total):", "", 0, "R", false, 0, "")
	pdf.CellFormat(totalValW, 8, fmtMoney(inv.TotalAmount)+" VND", "", 1, "R", false, 0, "")

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s.pdf"`, inv.InvoiceNumber))
	if err := pdf.Output(w); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate PDF")
	}
}
