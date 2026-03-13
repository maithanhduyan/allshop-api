package database

import "database/sql"

func Migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			phone VARCHAR(50),
			avatar TEXT,
			password VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS categories (
			id VARCHAR(50) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			emoji VARCHAR(10) NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS products (
			id SERIAL PRIMARY KEY,
			name VARCHAR(500) NOT NULL,
			slug VARCHAR(500) UNIQUE NOT NULL,
			description TEXT,
			price NUMERIC(15,2) NOT NULL,
			original_price NUMERIC(15,2),
			images TEXT[] DEFAULT '{}',
			category VARCHAR(50) REFERENCES categories(id),
			brand VARCHAR(255),
			rating NUMERIC(3,2) DEFAULT 0,
			review_count INT DEFAULT 0,
			stock INT DEFAULT 0,
			specifications JSONB DEFAULT '{}',
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS cart_items (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id) ON DELETE CASCADE,
			product_id INT REFERENCES products(id) ON DELETE CASCADE,
			quantity INT NOT NULL DEFAULT 1,
			UNIQUE(user_id, product_id)
		)`,

		`CREATE TABLE IF NOT EXISTS orders (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id),
			total NUMERIC(15,2) NOT NULL,
			status VARCHAR(50) DEFAULT 'pending',
			name VARCHAR(255) NOT NULL,
			phone VARCHAR(50) NOT NULL,
			address TEXT NOT NULL,
			note TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS order_items (
			id SERIAL PRIMARY KEY,
			order_id INT REFERENCES orders(id) ON DELETE CASCADE,
			product_id INT REFERENCES products(id),
			name VARCHAR(500) NOT NULL,
			image TEXT,
			price NUMERIC(15,2) NOT NULL,
			quantity INT NOT NULL
		)`,

		`CREATE TABLE IF NOT EXISTS invoices (
			id SERIAL PRIMARY KEY,
			public_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
			invoice_number VARCHAR(50) UNIQUE NOT NULL,
			order_id INT REFERENCES orders(id),
			user_id INT REFERENCES users(id),
			seller_name VARCHAR(255) NOT NULL,
			seller_tax_code VARCHAR(20) NOT NULL,
			seller_address TEXT NOT NULL,
			buyer_name VARCHAR(255) NOT NULL,
			buyer_tax_code VARCHAR(20),
			buyer_address TEXT NOT NULL,
			buyer_email VARCHAR(255),
			subtotal NUMERIC(15,2) NOT NULL,
			tax_rate NUMERIC(5,4) NOT NULL DEFAULT 0.08,
			tax_amount NUMERIC(15,2) NOT NULL,
			total_amount NUMERIC(15,2) NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'draft',
			issued_at TIMESTAMPTZ,
			cancelled_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Add public_key to existing invoices (idempotent)
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='invoices' AND column_name='public_key') THEN
				ALTER TABLE invoices ADD COLUMN public_key UUID UNIQUE NOT NULL DEFAULT gen_random_uuid();
			END IF;
		END $$`,

		`CREATE TABLE IF NOT EXISTS invoice_items (
			id SERIAL PRIMARY KEY,
			invoice_id INT NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
			product_id INT REFERENCES products(id),
			product_name VARCHAR(500) NOT NULL,
			unit VARCHAR(50) NOT NULL DEFAULT 'Cái',
			quantity INT NOT NULL,
			unit_price NUMERIC(15,2) NOT NULL,
			tax_rate NUMERIC(5,4) NOT NULL DEFAULT 0.08,
			tax_amount NUMERIC(15,2) NOT NULL,
			total_amount NUMERIC(15,2) NOT NULL
		)`,

		`CREATE INDEX IF NOT EXISTS idx_invoices_user_id ON invoices(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_order_id ON invoices(order_id)`,
		`CREATE INDEX IF NOT EXISTS idx_invoices_status ON invoices(status)`,

		// ── Accounting ──
		`CREATE TABLE IF NOT EXISTS accounts (
			id SERIAL PRIMARY KEY,
			code VARCHAR(20) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			type VARCHAR(20) NOT NULL,
			parent_code VARCHAR(20),
			level INT NOT NULL DEFAULT 1,
			is_active BOOLEAN DEFAULT true
		)`,

		`CREATE TABLE IF NOT EXISTS journal_entries (
			id SERIAL PRIMARY KEY,
			entry_number VARCHAR(50) UNIQUE NOT NULL,
			invoice_id INT REFERENCES invoices(id),
			description TEXT NOT NULL,
			entry_date DATE NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'posted',
			reversed_by INT REFERENCES journal_entries(id),
			reverses INT REFERENCES journal_entries(id),
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS journal_lines (
			id SERIAL PRIMARY KEY,
			journal_entry_id INT NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
			account_code VARCHAR(20) NOT NULL REFERENCES accounts(code),
			description TEXT,
			debit NUMERIC(15,2) NOT NULL DEFAULT 0,
			credit NUMERIC(15,2) NOT NULL DEFAULT 0,
			CHECK (debit >= 0 AND credit >= 0)
		)`,

		`CREATE INDEX IF NOT EXISTS idx_journal_entries_invoice_id ON journal_entries(invoice_id)`,
		`CREATE INDEX IF NOT EXISTS idx_journal_entries_entry_date ON journal_entries(entry_date)`,
		`CREATE INDEX IF NOT EXISTS idx_journal_lines_account_code ON journal_lines(account_code)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}
