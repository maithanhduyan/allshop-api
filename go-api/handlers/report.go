package handlers

import (
	"net/http"
	"time"
)

// ── Revenue Report ──

func (h *Handler) RevenueReport(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	// Daily revenue from issued invoices
	rows, err := h.db.Query(
		`SELECT DATE(issued_at) AS day,
			COUNT(*) AS invoice_count,
			COALESCE(SUM(subtotal), 0) AS subtotal,
			COALESCE(SUM(tax_amount), 0) AS tax,
			COALESCE(SUM(total_amount), 0) AS total
		 FROM invoices
		 WHERE status = 'issued'
		   AND DATE(issued_at) >= $1 AND DATE(issued_at) <= $2
		 GROUP BY DATE(issued_at)
		 ORDER BY day`, from, to,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query revenue")
		return
	}
	defer rows.Close()

	type DayRow struct {
		Date         string  `json:"date"`
		InvoiceCount int     `json:"invoiceCount"`
		Subtotal     float64 `json:"subtotal"`
		Tax          float64 `json:"tax"`
		Total        float64 `json:"total"`
	}

	var days []DayRow
	var grandSubtotal, grandTax, grandTotal float64
	var grandCount int

	for rows.Next() {
		var d DayRow
		var day time.Time
		if err := rows.Scan(&day, &d.InvoiceCount, &d.Subtotal, &d.Tax, &d.Total); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan revenue")
			return
		}
		d.Date = day.Format("2006-01-02")
		grandSubtotal += d.Subtotal
		grandTax += d.Tax
		grandTotal += d.Total
		grandCount += d.InvoiceCount
		days = append(days, d)
	}
	if days == nil {
		days = []DayRow{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"from":         from,
		"to":           to,
		"days":         days,
		"totalInvoices": grandCount,
		"totalSubtotal": grandSubtotal,
		"totalTax":      grandTax,
		"totalRevenue":  grandTotal,
	})
}

// ── Tax Report ──

func (h *Handler) TaxReport(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().AddDate(0, -1, 0).Format("2006-01-02")
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	// Output VAT (đầu ra) – from issued invoices
	var outputCount int
	var outputTax float64
	h.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(tax_amount), 0)
		 FROM invoices
		 WHERE status = 'issued'
		   AND DATE(issued_at) >= $1 AND DATE(issued_at) <= $2`, from, to,
	).Scan(&outputCount, &outputTax)

	// Input VAT (đầu vào) – from journal entries posting to 1331 (would be populated by purchase invoices)
	var inputTax float64
	h.db.QueryRow(
		`SELECT COALESCE(SUM(jl.debit), 0)
		 FROM journal_lines jl
		 JOIN journal_entries je ON je.id = jl.journal_entry_id
		 WHERE jl.account_code = '1331'
		   AND je.status = 'posted'
		   AND je.entry_date >= $1 AND je.entry_date <= $2`, from, to,
	).Scan(&inputTax)

	// Cancelled invoice tax (reversed)
	var cancelledTax float64
	h.db.QueryRow(
		`SELECT COALESCE(SUM(tax_amount), 0)
		 FROM invoices
		 WHERE status = 'cancelled' AND issued_at IS NOT NULL
		   AND DATE(cancelled_at) >= $1 AND DATE(cancelled_at) <= $2`, from, to,
	).Scan(&cancelledTax)

	netTax := outputTax - inputTax - cancelledTax

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"from":          from,
		"to":            to,
		"outputTax":     outputTax,
		"outputCount":   outputCount,
		"inputTax":      inputTax,
		"cancelledTax":  cancelledTax,
		"netTaxPayable": netTax,
	})
}

// ── Account Balance Report ──

func (h *Handler) AccountBalanceReport(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = "2000-01-01"
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	rows, err := h.db.Query(
		`SELECT a.code, a.name, a.type, a.level,
			COALESCE(SUM(CASE WHEN je.entry_date < $1 THEN jl.debit ELSE 0 END), 0) AS opening_debit,
			COALESCE(SUM(CASE WHEN je.entry_date < $1 THEN jl.credit ELSE 0 END), 0) AS opening_credit,
			COALESCE(SUM(CASE WHEN je.entry_date >= $1 AND je.entry_date <= $2 THEN jl.debit ELSE 0 END), 0) AS period_debit,
			COALESCE(SUM(CASE WHEN je.entry_date >= $1 AND je.entry_date <= $2 THEN jl.credit ELSE 0 END), 0) AS period_credit
		 FROM accounts a
		 LEFT JOIN journal_lines jl ON jl.account_code = a.code
		 LEFT JOIN journal_entries je ON je.id = jl.journal_entry_id AND je.status = 'posted'
		 WHERE a.is_active = true
		 GROUP BY a.code, a.name, a.type, a.level
		 HAVING COALESCE(SUM(jl.debit), 0) != 0 OR COALESCE(SUM(jl.credit), 0) != 0
		 ORDER BY a.code`, from, to,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query account balances")
		return
	}
	defer rows.Close()

	type BalanceRow struct {
		AccountCode    string  `json:"accountCode"`
		AccountName    string  `json:"accountName"`
		AccountType    string  `json:"accountType"`
		Level          int     `json:"level"`
		OpeningDebit   float64 `json:"openingDebit"`
		OpeningCredit  float64 `json:"openingCredit"`
		OpeningBalance float64 `json:"openingBalance"`
		PeriodDebit    float64 `json:"periodDebit"`
		PeriodCredit   float64 `json:"periodCredit"`
		ClosingBalance float64 `json:"closingBalance"`
	}

	var result []BalanceRow
	for rows.Next() {
		var row BalanceRow
		if err := rows.Scan(&row.AccountCode, &row.AccountName, &row.AccountType, &row.Level,
			&row.OpeningDebit, &row.OpeningCredit, &row.PeriodDebit, &row.PeriodCredit); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan balance")
			return
		}
		row.OpeningBalance = row.OpeningDebit - row.OpeningCredit
		row.ClosingBalance = row.OpeningBalance + row.PeriodDebit - row.PeriodCredit
		result = append(result, row)
	}
	if result == nil {
		result = []BalanceRow{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"from":     from,
		"to":       to,
		"accounts": result,
	})
}

// ── Dashboard Summary ──

func (h *Handler) DashboardSummary(w http.ResponseWriter, r *http.Request) {
	// Invoices summary
	var totalInvoices, issuedInvoices, cancelledInvoices int
	var issuedRevenue, issuedTax float64
	h.db.QueryRow(`SELECT COUNT(*) FROM invoices`).Scan(&totalInvoices)
	h.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(total_amount), 0), COALESCE(SUM(tax_amount), 0)
		 FROM invoices WHERE status = 'issued'`,
	).Scan(&issuedInvoices, &issuedRevenue, &issuedTax)
	h.db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE status = 'cancelled'`).Scan(&cancelledInvoices)

	// Journal entries summary
	var totalEntries, postedEntries, reversedEntries int
	h.db.QueryRow(`SELECT COUNT(*) FROM journal_entries`).Scan(&totalEntries)
	h.db.QueryRow(`SELECT COUNT(*) FROM journal_entries WHERE status = 'posted'`).Scan(&postedEntries)
	h.db.QueryRow(`SELECT COUNT(*) FROM journal_entries WHERE status = 'reversed'`).Scan(&reversedEntries)

	// Total debit/credit from posted entries
	var totalDebit, totalCredit float64
	h.db.QueryRow(
		`SELECT COALESCE(SUM(jl.debit), 0), COALESCE(SUM(jl.credit), 0)
		 FROM journal_lines jl
		 JOIN journal_entries je ON je.id = jl.journal_entry_id
		 WHERE je.status = 'posted'`,
	).Scan(&totalDebit, &totalCredit)

	// Orders summary
	var totalOrders int
	var ordersRevenue float64
	h.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(total), 0)
		 FROM orders WHERE status != 'cancelled'`,
	).Scan(&totalOrders, &ordersRevenue)

	// Account count
	var totalAccounts int
	h.db.QueryRow(`SELECT COUNT(*) FROM accounts WHERE is_active = true`).Scan(&totalAccounts)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"invoices": map[string]interface{}{
			"total":     totalInvoices,
			"issued":    issuedInvoices,
			"cancelled": cancelledInvoices,
			"revenue":   issuedRevenue,
			"tax":       issuedTax,
		},
		"journalEntries": map[string]interface{}{
			"total":    totalEntries,
			"posted":   postedEntries,
			"reversed": reversedEntries,
		},
		"accounting": map[string]interface{}{
			"totalDebit":    totalDebit,
			"totalCredit":   totalCredit,
			"isBalanced":    totalDebit == totalCredit,
			"totalAccounts": totalAccounts,
		},
		"orders": map[string]interface{}{
			"total":   totalOrders,
			"revenue": ordersRevenue,
		},
	})
}
