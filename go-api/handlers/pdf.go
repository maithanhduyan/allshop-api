package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-pdf/fpdf"
)

// ExportInvoicePDF renders a Vietnamese-style PDF invoice and streams it to the client.
func (h *Handler) ExportInvoicePDF(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	invoiceID := chi.URLParam(r, "id")

	// Load invoice
	var inv struct {
		ID, InvoiceNumber, OrderID, UserID                        string
		SellerName, SellerTaxCode, SellerAddress                   string
		BuyerName, BuyerAddress, BuyerEmail                        string
		BuyerTaxCode                                               *string
		Subtotal, TaxRate, TaxAmount, TotalAmount                  float64
		Status                                                     string
		IssuedAt, CancelledAt                                      *string
		CreatedAt                                                  string
	}
	err := h.db.QueryRow(
		`SELECT id, invoice_number, order_id, user_id,
			seller_name, seller_tax_code, seller_address,
			buyer_name, COALESCE(buyer_tax_code, ''), buyer_address, buyer_email,
			subtotal, tax_rate, tax_amount, total_amount, status,
			TO_CHAR(COALESCE(issued_at, created_at), 'DD/MM/YYYY HH24:MI'),
			created_at
		 FROM invoices WHERE id = $1 AND user_id = $2`, invoiceID, userID,
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

	// Load invoice items
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
		 FROM invoice_items WHERE invoice_id = $1 ORDER BY id`, invoiceID,
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

	// Build PDF
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Use built-in fonts (Helvetica supports basic Latin; Vietnamese diacritics will
	// render as best-effort with the core fonts – for production you would embed a
	// TTF like Roboto-Regular, but that requires bundling font files in the Docker image).
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(190, 10, "HOA DON GIA TRI GIA TANG", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(190, 5, "(VAT INVOICE)", "", 1, "C", false, 0, "")
	pdf.Ln(3)

	// Invoice number & date
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(95, 6, fmt.Sprintf("So (No.): %s", inv.InvoiceNumber), "", 0, "L", false, 0, "")
	issuedStr := ""
	if inv.IssuedAt != nil {
		issuedStr = *inv.IssuedAt
	}
	pdf.CellFormat(95, 6, fmt.Sprintf("Ngay (Date): %s", issuedStr), "", 1, "R", false, 0, "")
	pdf.Ln(3)

	// Status
	statusLabel := map[string]string{"draft": "NHAP", "issued": "DA PHAT HANH", "cancelled": "DA HUY"}
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(190, 5, fmt.Sprintf("Trang thai: %s", statusLabel[inv.Status]), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Seller info
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(190, 6, "Don vi ban hang (Seller):", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(190, 5, fmt.Sprintf("  Ten (Name): %s", asciiFold(inv.SellerName)), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, fmt.Sprintf("  Ma so thue (Tax code): %s", inv.SellerTaxCode), "", 1, "L", false, 0, "")
	pdf.CellFormat(190, 5, fmt.Sprintf("  Dia chi (Address): %s", asciiFold(inv.SellerAddress)), "", 1, "L", false, 0, "")
	pdf.Ln(3)

	// Buyer info
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

	// Items table header
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(240, 240, 240)
	colW := []float64{10, 70, 20, 15, 25, 25, 25}
	headers := []string{"STT", "Ten hang hoa", "DVT", "SL", "Don gia", "Thue", "Thanh tien"}
	aligns := []string{"C", "L", "C", "C", "R", "R", "R"}
	for i, h := range headers {
		pdf.CellFormat(colW[i], 7, h, "1", 0, aligns[i], true, 0, "")
	}
	pdf.Ln(-1)

	// Items
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

	// Totals
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

	// Stream PDF
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, inv.InvoiceNumber))
	if err := pdf.Output(w); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate PDF")
	}
}

// fmtMoney formats a number with thousand separators (e.g. 1.234.567).
func fmtMoney(v float64) string {
	s := fmt.Sprintf("%.0f", v)
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	for i, ch := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(ch)
	}
	return b.String()
}

// asciiFold removes Vietnamese diacritics for safe rendering with built-in PDF fonts.
func asciiFold(s string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ả", "a", "ã", "a", "ạ", "a",
		"ă", "a", "ắ", "a", "ằ", "a", "ẳ", "a", "ẵ", "a", "ặ", "a",
		"â", "a", "ấ", "a", "ầ", "a", "ẩ", "a", "ẫ", "a", "ậ", "a",
		"Á", "A", "À", "A", "Ả", "A", "Ã", "A", "Ạ", "A",
		"Ă", "A", "Ắ", "A", "Ằ", "A", "Ẳ", "A", "Ẵ", "A", "Ặ", "A",
		"Â", "A", "Ấ", "A", "Ầ", "A", "Ẩ", "A", "Ẫ", "A", "Ậ", "A",
		"é", "e", "è", "e", "ẻ", "e", "ẽ", "e", "ẹ", "e",
		"ê", "e", "ế", "e", "ề", "e", "ể", "e", "ễ", "e", "ệ", "e",
		"É", "E", "È", "E", "Ẻ", "E", "Ẽ", "E", "Ẹ", "E",
		"Ê", "E", "Ế", "E", "Ề", "E", "Ể", "E", "Ễ", "E", "Ệ", "E",
		"í", "i", "ì", "i", "ỉ", "i", "ĩ", "i", "ị", "i",
		"Í", "I", "Ì", "I", "Ỉ", "I", "Ĩ", "I", "Ị", "I",
		"ó", "o", "ò", "o", "ỏ", "o", "õ", "o", "ọ", "o",
		"ô", "o", "ố", "o", "ồ", "o", "ổ", "o", "ỗ", "o", "ộ", "o",
		"ơ", "o", "ớ", "o", "ờ", "o", "ở", "o", "ỡ", "o", "ợ", "o",
		"Ó", "O", "Ò", "O", "Ỏ", "O", "Õ", "O", "Ọ", "O",
		"Ô", "O", "Ố", "O", "Ồ", "O", "Ổ", "O", "Ỗ", "O", "Ộ", "O",
		"Ơ", "O", "Ớ", "O", "Ờ", "O", "Ở", "O", "Ỡ", "O", "Ợ", "O",
		"ú", "u", "ù", "u", "ủ", "u", "ũ", "u", "ụ", "u",
		"ư", "u", "ứ", "u", "ừ", "u", "ử", "u", "ữ", "u", "ự", "u",
		"Ú", "U", "Ù", "U", "Ủ", "U", "Ũ", "U", "Ụ", "U",
		"Ư", "U", "Ứ", "U", "Ừ", "U", "Ử", "U", "Ữ", "U", "Ự", "U",
		"ý", "y", "ỳ", "y", "ỷ", "y", "ỹ", "y", "ỵ", "y",
		"Ý", "Y", "Ỳ", "Y", "Ỷ", "Y", "Ỹ", "Y", "Ỵ", "Y",
		"đ", "d", "Đ", "D",
	)
	return replacer.Replace(s)
}

// truncStr truncates a string to maxLen characters.
func truncStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
