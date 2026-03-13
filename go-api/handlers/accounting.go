package handlers

import (
	"allshop-api/models"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// ── Account endpoints ──

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT id, code, name, type, COALESCE(parent_code, ''), level, is_active
		 FROM accounts ORDER BY code`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query accounts")
		return
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.Code, &a.Name, &a.Type, &a.ParentCode, &a.Level, &a.IsActive); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan account")
			return
		}
		accounts = append(accounts, a)
	}
	if accounts == nil {
		accounts = []models.Account{}
	}
	writeJSON(w, http.StatusOK, accounts)
}

// ── Journal Entry endpoints ──

func (h *Handler) ListJournalEntries(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT id, entry_number, invoice_id, description, entry_date,
			status, reversed_by, reverses, created_at
		 FROM journal_entries ORDER BY created_at DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query journal entries")
		return
	}
	defer rows.Close()

	var entries []models.JournalEntry
	for rows.Next() {
		var e models.JournalEntry
		var entryDate time.Time
		if err := rows.Scan(
			&e.ID, &e.EntryNumber, &e.InvoiceID, &e.Description, &entryDate,
			&e.Status, &e.ReversedBy, &e.Reverses, &e.CreatedAt,
		); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan journal entry")
			return
		}
		e.EntryDate = entryDate.Format("2006-01-02")
		e.Lines = []models.JournalLine{}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []models.JournalEntry{}
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) GetJournalEntry(w http.ResponseWriter, r *http.Request) {
	entryID := chi.URLParam(r, "id")

	var e models.JournalEntry
	var entryDate time.Time
	err := h.db.QueryRow(
		`SELECT id, entry_number, invoice_id, description, entry_date,
			status, reversed_by, reverses, created_at
		 FROM journal_entries WHERE id = $1`, entryID,
	).Scan(
		&e.ID, &e.EntryNumber, &e.InvoiceID, &e.Description, &entryDate,
		&e.Status, &e.ReversedBy, &e.Reverses, &e.CreatedAt,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "journal entry not found")
		return
	}
	e.EntryDate = entryDate.Format("2006-01-02")

	rows, err := h.db.Query(
		`SELECT jl.id, jl.journal_entry_id, jl.account_code, a.name,
			COALESCE(jl.description, ''), jl.debit, jl.credit
		 FROM journal_lines jl
		 JOIN accounts a ON a.code = jl.account_code
		 WHERE jl.journal_entry_id = $1
		 ORDER BY jl.id`, entryID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query journal lines")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var l models.JournalLine
		if err := rows.Scan(&l.ID, &l.JournalEntryID, &l.AccountCode, &l.AccountName,
			&l.Description, &l.Debit, &l.Credit); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan journal line")
			return
		}
		e.Lines = append(e.Lines, l)
	}
	if e.Lines == nil {
		e.Lines = []models.JournalLine{}
	}

	writeJSON(w, http.StatusOK, e)
}

func (h *Handler) GetTrialBalance(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(
		`SELECT a.code, a.name, a.type,
			COALESCE(SUM(jl.debit), 0) AS total_debit,
			COALESCE(SUM(jl.credit), 0) AS total_credit
		 FROM accounts a
		 LEFT JOIN journal_lines jl ON jl.account_code = a.code
		 LEFT JOIN journal_entries je ON je.id = jl.journal_entry_id AND je.status = 'posted'
		 WHERE a.is_active = true
		 GROUP BY a.code, a.name, a.type
		 HAVING COALESCE(SUM(jl.debit), 0) != 0 OR COALESCE(SUM(jl.credit), 0) != 0
		 ORDER BY a.code`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query trial balance")
		return
	}
	defer rows.Close()

	type TrialBalanceRow struct {
		AccountCode string  `json:"accountCode"`
		AccountName string  `json:"accountName"`
		AccountType string  `json:"accountType"`
		TotalDebit  float64 `json:"totalDebit"`
		TotalCredit float64 `json:"totalCredit"`
		Balance     float64 `json:"balance"`
	}

	var result []TrialBalanceRow
	for rows.Next() {
		var row TrialBalanceRow
		if err := rows.Scan(&row.AccountCode, &row.AccountName, &row.AccountType,
			&row.TotalDebit, &row.TotalCredit); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to scan trial balance")
			return
		}
		row.Balance = row.TotalDebit - row.TotalCredit
		result = append(result, row)
	}
	if result == nil {
		result = []TrialBalanceRow{}
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Auto journal entry helpers ──

// createJournalEntry creates a journal entry with lines inside an existing transaction.
func createJournalEntry(tx *sql.Tx, invoiceID string, description string, lines []journalLineInput) (string, error) {
	now := time.Now()
	dateStr := now.Format("20060102")

	var seq int
	tx.QueryRow(
		`SELECT COUNT(*) FROM journal_entries WHERE entry_number LIKE $1`,
		fmt.Sprintf("JE-%s-%%", dateStr),
	).Scan(&seq)
	entryNumber := fmt.Sprintf("JE-%s-%04d", dateStr, seq+1)

	var entryID string
	err := tx.QueryRow(
		`INSERT INTO journal_entries (entry_number, invoice_id, description, entry_date, status)
		 VALUES ($1, $2, $3, $4, 'posted')
		 RETURNING id`,
		entryNumber, invoiceID, description, now,
	).Scan(&entryID)
	if err != nil {
		return "", fmt.Errorf("insert journal entry: %w", err)
	}

	for _, l := range lines {
		_, err := tx.Exec(
			`INSERT INTO journal_lines (journal_entry_id, account_code, description, debit, credit)
			 VALUES ($1, $2, $3, $4, $5)`,
			entryID, l.accountCode, l.description, l.debit, l.credit,
		)
		if err != nil {
			return "", fmt.Errorf("insert journal line %s: %w", l.accountCode, err)
		}
	}

	return entryID, nil
}

// createReversalEntry creates a reversal journal entry for a previously posted entry.
func createReversalEntry(tx *sql.Tx, originalEntryID, invoiceID string, description string) (string, error) {
	// Get original lines
	rows, err := tx.Query(
		`SELECT account_code, description, debit, credit
		 FROM journal_lines WHERE journal_entry_id = $1`, originalEntryID,
	)
	if err != nil {
		return "", fmt.Errorf("query original lines: %w", err)
	}
	defer rows.Close()

	var reversedLines []journalLineInput
	for rows.Next() {
		var l journalLineInput
		if err := rows.Scan(&l.accountCode, &l.description, &l.debit, &l.credit); err != nil {
			return "", fmt.Errorf("scan original line: %w", err)
		}
		// Swap debit and credit for reversal
		reversedLines = append(reversedLines, journalLineInput{
			accountCode: l.accountCode,
			description: l.description,
			debit:       l.credit,
			credit:      l.debit,
		})
	}

	reversalID, err := createJournalEntry(tx, invoiceID, description, reversedLines)
	if err != nil {
		return "", err
	}

	// Link reversal: original.reversed_by = reversal, reversal.reverses = original
	if _, err := tx.Exec(`UPDATE journal_entries SET reversed_by = $1 WHERE id = $2`, reversalID, originalEntryID); err != nil {
		return "", fmt.Errorf("update reversed_by: %w", err)
	}
	if _, err := tx.Exec(`UPDATE journal_entries SET reverses = $1 WHERE id = $2`, originalEntryID, reversalID); err != nil {
		return "", fmt.Errorf("update reverses: %w", err)
	}
	// Mark original as reversed
	if _, err := tx.Exec(`UPDATE journal_entries SET status = 'reversed' WHERE id = $1`, originalEntryID); err != nil {
		return "", fmt.Errorf("update original status: %w", err)
	}

	return reversalID, nil
}

type journalLineInput struct {
	accountCode string
	description string
	debit       float64
	credit      float64
}
