package database

import (
	"database/sql"
	"log"
)

func Seed(db *sql.DB) error {
	// Seed categories
	categories := []struct {
		ID, Name, Emoji string
	}{
		{"fashion", "Thời trang", "👕"},
		{"electronics", "Điện máy", "📺"},
		{"tools", "Công cụ", "🔧"},
		{"computers", "Máy tính", "💻"},
		{"phones", "Điện thoại", "📱"},
		{"accessories", "Phụ kiện", "🎧"},
	}

	for _, c := range categories {
		_, err := db.Exec(
			`INSERT INTO categories (id, name, emoji) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING`,
			c.ID, c.Name, c.Emoji,
		)
		if err != nil {
			return err
		}
	}

	// Seed products only if table is empty
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		log.Println("Products already seeded, skipping")
		return nil
	}

	products := []struct {
		Name, Slug, Description string
		Price                   float64
		OriginalPrice           *float64
		Image, Category, Brand  string
		Rating                  float64
		ReviewCount, Stock      int
		Specifications          string
	}{
		{
			"Áo thun nam cổ tròn Premium Cotton", "ao-thun-nam-co-tron",
			"Áo thun nam chất liệu 100% cotton cao cấp, mềm mại, thoáng mát.",
			199000, ptr(350000.0),
			"https://picsum.photos/seed/fashion1/600/600", "fashion", "AllShop Basic",
			4.5, 128, 50,
			`{"Chất liệu": "100% Cotton", "Xuất xứ": "Việt Nam"}`,
		},
		{
			"iPhone 15 Pro Max 256GB", "iphone-15-pro-max",
			"iPhone 15 Pro Max chính hãng Apple.",
			29990000, ptr(34990000.0),
			"https://picsum.photos/seed/phone1/600/600", "phones", "Apple",
			4.8, 542, 25, `{}`,
		},
		{
			"Laptop Dell XPS 15 Core i7", "laptop-dell-xps-15",
			"Laptop Dell XPS 15 mỏng nhẹ, hiệu năng cao.",
			35990000, ptr(42990000.0),
			"https://picsum.photos/seed/laptop1/600/600", "computers", "Dell",
			4.7, 89, 15, `{}`,
		},
		{
			"Tai nghe Sony WH-1000XM5", "tai-nghe-sony-wh1000xm5",
			"Tai nghe chống ồn chủ động hàng đầu.",
			7490000, ptr(8990000.0),
			"https://picsum.photos/seed/accessory1/600/600", "accessories", "Sony",
			4.9, 312, 30, `{}`,
		},
		{
			"Máy giặt Samsung Inverter 9.5kg", "may-giat-samsung",
			"Máy giặt Samsung công nghệ Inverter tiết kiệm điện.",
			8990000, ptr(12990000.0),
			"https://picsum.photos/seed/elec1/600/600", "electronics", "Samsung",
			4.6, 67, 10, `{}`,
		},
		{
			"Bộ dụng cụ sửa chữa 120 món", "bo-dung-cu-sua-chua",
			"Bộ dụng cụ đa năng 120 món cho gia đình.",
			890000, ptr(1290000.0),
			"https://picsum.photos/seed/tool1/600/600", "tools", "Bosch",
			4.3, 45, 40, `{}`,
		},
		{
			"Quần jean nam Slim Fit", "quan-jean-nam-slim-fit",
			"Quần jean nam co giãn, form slim fit.",
			450000, ptr(650000.0),
			"https://picsum.photos/seed/fashion2/600/600", "fashion", "AllShop Basic",
			4.4, 203, 60, `{}`,
		},
		{
			"Samsung Galaxy S24 Ultra", "samsung-galaxy-s24-ultra",
			"Samsung Galaxy S24 Ultra flagship mới nhất.",
			27990000, ptr(33990000.0),
			"https://picsum.photos/seed/phone2/600/600", "phones", "Samsung",
			4.7, 389, 20, `{}`,
		},
	}

	for _, p := range products {
		_, err := db.Exec(
			`INSERT INTO products (name, slug, description, price, original_price, images, category, brand, rating, review_count, stock, specifications)
			 VALUES ($1, $2, $3, $4, $5, ARRAY[$6], $7, $8, $9, $10, $11, $12::jsonb)`,
			p.Name, p.Slug, p.Description, p.Price, p.OriginalPrice,
			p.Image, p.Category, p.Brand, p.Rating, p.ReviewCount, p.Stock, p.Specifications,
		)
		if err != nil {
			return err
		}
	}

	log.Println("Database seeded successfully")
	return nil
}

func ptr(f float64) *float64 {
	return &f
}
