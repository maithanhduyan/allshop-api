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
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}
