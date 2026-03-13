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
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	return nil
}
