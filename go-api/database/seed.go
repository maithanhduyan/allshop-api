package database

import (
	"allshop-api/storage"
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

func Seed(db *sql.DB, store *storage.Storage) error {
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
		// ── Thời trang (fashion) ──
		{
			"Áo Polo Nam Cotton Pique Classic", "ao-polo-nam-cotton-pique",
			"Áo polo nam chất liệu cotton pique cao cấp, form regular fit thoải mái. Cổ bẻ dệt kim 2 màu tinh tế, thích hợp mặc đi làm và dạo phố. Bo tay và gấu áo co giãn tốt, giữ form sau nhiều lần giặt.",
			299000, ptr(450000.0),
			"https://images.unsplash.com/photo-1521572163474-6864f9cf17ab?w=600&h=600&fit=crop", "fashion", "YODY",
			4.6, 1247, 120,
			`{"Chất liệu": "100% Cotton Pique", "Xuất xứ": "Việt Nam", "Kiểu dáng": "Regular Fit", "Bảo hành": "Đổi trả 30 ngày"}`,
		},
		{
			"Đầm Midi Hoa Nhí Vintage", "dam-midi-hoa-nhi-vintage",
			"Đầm midi họa tiết hoa nhí phong cách vintage thanh lịch. Chất vải voan lụa mềm mại, thoáng mát. Thiết kế cổ V nhẹ nhàng, tay phồng nữ tính, eo chun co giãn thoải mái.",
			389000, ptr(550000.0),
			"https://images.unsplash.com/photo-1572804013309-59a88b7e92f1?w=600&h=600&fit=crop", "fashion", "CANIFA",
			4.7, 832, 85,
			`{"Chất liệu": "Voan lụa", "Xuất xứ": "Việt Nam", "Kiểu dáng": "Midi A-line"}`,
		},
		{
			"Quần Jogger Nam Thể Thao", "quan-jogger-nam-the-thao",
			"Quần jogger nam chất nỉ da cá mềm, co giãn 4 chiều. Thiết kế lưng chun dây rút tiện lợi, bo gấu năng động. Phù hợp tập gym, chạy bộ và mặc thường ngày.",
			259000, ptr(399000.0),
			"https://images.unsplash.com/photo-1552902865-b72c031ac5ea?w=600&h=600&fit=crop", "fashion", "ROUTINE",
			4.5, 2156, 200,
			`{"Chất liệu": "95% Cotton, 5% Spandex", "Xuất xứ": "Việt Nam", "Kiểu dáng": "Jogger Fit"}`,
		},
		{
			"Áo Khoác Gió Unisex Siêu Nhẹ", "ao-khoac-gio-unisex",
			"Áo khoác gió 2 lớp siêu nhẹ, chống nước nhẹ và chắn gió hiệu quả. Gấp gọn bỏ túi tiện lợi, trọng lượng chỉ 180g. Phản quang an toàn khi di chuyển ban đêm.",
			349000, ptr(520000.0),
			"https://images.unsplash.com/photo-1591047139829-d91aecb6caea?w=600&h=600&fit=crop", "fashion", "ARISTINO",
			4.8, 3421, 150,
			`{"Chất liệu": "Polyester chống nước", "Trọng lượng": "180g", "Tính năng": "Chắn gió, chống nước nhẹ, phản quang"}`,
		},
		// ── Điện máy (electronics) ──
		{
			"Tivi Samsung Crystal UHD 55 inch 4K", "tivi-samsung-crystal-uhd-55",
			"Smart TV Samsung Crystal UHD 55 inch, độ phân giải 4K sắc nét. Công nghệ Crystal Processor 4K cho màu sắc chân thực sống động. Tích hợp Tizen OS, kho ứng dụng phong phú. Thiết kế viền siêu mỏng AirSlim sang trọng.",
			10990000, ptr(14990000.0),
			"https://images.unsplash.com/photo-1593359677879-a4bb92f829d1?w=600&h=600&fit=crop", "electronics", "Samsung",
			4.7, 1567, 25,
			`{"Kích thước": "55 inch", "Độ phân giải": "4K UHD (3840x2160)", "Hệ điều hành": "Tizen", "Kết nối": "WiFi, Bluetooth 5.2, HDMI x3"}`,
		},
		{
			"Máy Giặt Lồng Ngang LG Inverter 10kg", "may-giat-lg-inverter-10kg",
			"Máy giặt lồng ngang LG AI DD 10kg, nhận diện vải thông minh bằng AI. Công nghệ giặt hơi nước Steam+ diệt 99.9% vi khuẩn. Motor Direct Drive Inverter bền bỉ, vận hành êm ái. Tiết kiệm điện nước tối ưu.",
			12490000, ptr(16990000.0),
			"https://images.unsplash.com/photo-1626806787461-102c1bfaaea1?w=600&h=600&fit=crop", "electronics", "LG",
			4.6, 892, 15,
			`{"Khối lượng giặt": "10 kg", "Công nghệ": "AI DD, Steam+", "Motor": "Direct Drive Inverter", "Tiết kiệm": "Năng lượng A+++"}`,
		},
		{
			"Tủ Lạnh Panasonic Inverter 366L", "tu-lanh-panasonic-inverter-366l",
			"Tủ lạnh Panasonic 2 cánh ngăn đá trên 366 lít. Công nghệ Panorama AG+ kháng khuẩn khử mùi 99.9%. Inverter ECONAVI tiết kiệm điện thông minh. Ngăn rau quả giữ ẩm Fresh Safe.",
			11290000, ptr(15490000.0),
			"https://images.unsplash.com/photo-1571175443880-49e1d25b2bc5?w=600&h=600&fit=crop", "electronics", "Panasonic",
			4.5, 654, 12,
			`{"Dung tích": "366 lít", "Công nghệ": "Panorama AG+, ECONAVI", "Loại": "2 cánh ngăn đá trên", "Xếp hạng năng lượng": "5 sao"}`,
		},
		{
			"Điều Hòa Daikin Inverter 1.5HP", "dieu-hoa-daikin-inverter-15hp",
			"Máy lạnh Daikin Inverter 1.5HP làm lạnh nhanh, tiết kiệm điện đến 60%. Công nghệ Coanda làm lạnh đều khắp phòng. Tấm lọc bụi mịn PM2.5 bảo vệ sức khỏe. Vận hành siêu êm chỉ 19dB.",
			12990000, ptr(17990000.0),
			"https://images.unsplash.com/photo-1585338107529-13afc5f02586?w=600&h=600&fit=crop", "electronics", "Daikin",
			4.8, 2341, 20,
			`{"Công suất": "1.5 HP (~12000 BTU)", "Công nghệ": "Inverter, Coanda", "Lọc khí": "PM2.5", "Độ ồn": "19 dB"}`,
		},
		// ── Máy tính (computers) ──
		{
			"MacBook Air M3 15 inch 2024", "macbook-air-m3-15-inch",
			"MacBook Air chip Apple M3 màn hình 15.3 inch Liquid Retina. Hiệu năng mạnh mẽ với 8-core CPU, 10-core GPU. Pin lên đến 18 giờ sử dụng liên tục. Thiết kế siêu mỏng nhẹ chỉ 1.51kg, vỏ nhôm nguyên khối sang trọng.",
			32990000, ptr(37990000.0),
			"https://images.unsplash.com/photo-1517336714731-489689fd1ca8?w=600&h=600&fit=crop", "computers", "Apple",
			4.9, 1823, 18,
			`{"Chip": "Apple M3 (8-core CPU, 10-core GPU)", "RAM": "16GB Unified", "SSD": "512GB", "Màn hình": "15.3 inch Liquid Retina", "Pin": "18 giờ"}`,
		},
		{
			"Laptop ASUS ROG Strix G16 RTX 4060", "asus-rog-strix-g16",
			"Laptop gaming ASUS ROG Strix G16, Intel Core i7-13650HX mạnh mẽ. Card đồ họa NVIDIA RTX 4060 8GB chiến mọi tựa game. Màn hình 16 inch 165Hz mượt mà, tấm nền IPS. Hệ thống tản nhiệt MUX Switch tối ưu hiệu năng.",
			28990000, ptr(34990000.0),
			"https://images.unsplash.com/photo-1593642632559-0c6d3fc62b89?w=600&h=600&fit=crop", "computers", "ASUS",
			4.7, 967, 10,
			`{"CPU": "Intel Core i7-13650HX", "GPU": "NVIDIA RTX 4060 8GB", "RAM": "16GB DDR5", "SSD": "512GB NVMe", "Màn hình": "16 inch FHD+ 165Hz IPS"}`,
		},
		{
			"Màn Hình Dell UltraSharp 27 inch 4K", "man-hinh-dell-ultrasharp-27-4k",
			"Màn hình chuyên đồ họa Dell UltraSharp 27 inch, độ phân giải 4K. Phủ 99% sRGB và 98% DCI-P3, màu sắc chính xác ΔE<2. Tấm nền IPS góc nhìn 178° rộng. Hỗ trợ USB-C PD 90W sạc laptop trực tiếp.",
			12490000, ptr(15990000.0),
			"https://images.unsplash.com/photo-1527443224154-c4a3942d3acf?w=600&h=600&fit=crop", "computers", "Dell",
			4.8, 534, 22,
			`{"Kích thước": "27 inch", "Độ phân giải": "4K UHD (3840x2160)", "Tấm nền": "IPS, 99% sRGB, 98% DCI-P3", "Kết nối": "USB-C PD 90W, HDMI, DP"}`,
		},
		{
			"PC Gaming Intel i5-13400F / RTX 4060", "pc-gaming-i5-13400f-rtx4060",
			"Bộ máy tính gaming cấu hình cao, Intel Core i5-13400F kết hợp RTX 4060. RAM 16GB DDR5 bus 5600MHz xử lý mượt mà. SSD NVMe 500GB khởi động nhanh. Case kính cường lực RGB đẹp mắt.",
			18990000, ptr(22990000.0),
			"https://images.unsplash.com/photo-1587831990711-23ca6441447b?w=600&h=600&fit=crop", "computers", "ASUS",
			4.6, 423, 8,
			`{"CPU": "Intel Core i5-13400F", "GPU": "NVIDIA RTX 4060 8GB", "RAM": "16GB DDR5 5600MHz", "SSD": "500GB NVMe Gen4", "Case": "Kính cường lực RGB"}`,
		},
		// ── Điện thoại (phones) ──
		{
			"iPhone 16 Pro Max 256GB", "iphone-16-pro-max-256gb",
			"iPhone 16 Pro Max chip A18 Pro mạnh nhất. Màn hình Super Retina XDR 6.9 inch ProMotion 120Hz. Camera 48MP Tetraprism Zoom quang 5x. Khung viền Titanium siêu bền, pin dùng cả ngày dài.",
			33990000, ptr(36990000.0),
			"https://images.unsplash.com/photo-1695048133142-1a20484d2569?w=600&h=600&fit=crop", "phones", "Apple",
			4.9, 4521, 30,
			`{"Chip": "A18 Pro", "Màn hình": "6.9 inch Super Retina XDR 120Hz", "Camera": "48MP + 48MP + 12MP, Zoom 5x", "Pin": "4685 mAh", "Bộ nhớ": "256GB"}`,
		},
		{
			"Samsung Galaxy S25 Ultra 5G", "samsung-galaxy-s25-ultra",
			"Samsung Galaxy S25 Ultra flagship cao cấp với chip Snapdragon 8 Elite. Camera 200MP AI nâng cấp, zoom quang 5x sắc nét. Bút S Pen tích hợp, Galaxy AI thông minh. Khung viền Titanium, kính Gorilla Armor 2.",
			31990000, ptr(35990000.0),
			"https://images.unsplash.com/photo-1610945265064-0e34e5519bbf?w=600&h=600&fit=crop", "phones", "Samsung",
			4.8, 3287, 25,
			`{"Chip": "Snapdragon 8 Elite", "Màn hình": "6.9 inch Dynamic AMOLED 2X 120Hz", "Camera": "200MP + 50MP + 10MP + 12MP", "RAM": "12GB", "Bộ nhớ": "256GB"}`,
		},
		{
			"Xiaomi 15 Pro 5G 512GB", "xiaomi-15-pro-5g",
			"Xiaomi 15 Pro chip Snapdragon 8 Elite, hiệu năng vượt trội. Camera Leica 50MP cảm biến 1 inch chụp đẹp mọi điều kiện. Sạc nhanh 120W đầy pin trong 19 phút. Màn hình AMOLED 2K 120Hz sắc nét.",
			18990000, ptr(22990000.0),
			"https://images.unsplash.com/photo-1511707171634-5f897ff02aa9?w=600&h=600&fit=crop", "phones", "Xiaomi",
			4.7, 1876, 40,
			`{"Chip": "Snapdragon 8 Elite", "Màn hình": "6.73 inch AMOLED 2K 120Hz", "Camera": "Leica 50MP 1 inch sensor", "Sạc": "120W có dây, 50W không dây", "Bộ nhớ": "512GB"}`,
		},
		{
			"OPPO Find X8 Ultra", "oppo-find-x8-ultra",
			"OPPO Find X8 Ultra camera Hasselblad đỉnh cao nhiếp ảnh di động. Chip Dimensity 9400, hiệu năng flagship mạnh mẽ. Pin 5910mAh kèm sạc 80W. Chống nước IP69, thiết kế cao cấp sang trọng.",
			23990000, ptr(27990000.0),
			"https://images.unsplash.com/photo-1598327105666-5b89351aff97?w=600&h=600&fit=crop", "phones", "OPPO",
			4.6, 921, 18,
			`{"Chip": "Dimensity 9400", "Camera": "Hasselblad 50MP x4", "Pin": "5910 mAh, sạc 80W", "Chống nước": "IP69", "Bộ nhớ": "512GB"}`,
		},
		// ── Phụ kiện (accessories) ──
		{
			"Tai Nghe AirPods Pro 3 USB-C", "airpods-pro-3",
			"AirPods Pro 3 chống ồn chủ động ANC thế hệ mới. Âm thanh không gian cá nhân hóa Adaptive Audio. Chống nước IP54, pin 30 giờ với hộp sạc. Chip H3 xử lý âm thanh thông minh, kết nối nhanh với hệ sinh thái Apple.",
			6790000, ptr(7490000.0),
			"https://images.unsplash.com/photo-1600294037681-c80b4cb5b434?w=600&h=600&fit=crop", "accessories", "Apple",
			4.9, 5123, 50,
			`{"Chip": "Apple H3", "Chống ồn": "ANC thích ứng", "Chống nước": "IP54", "Pin": "6h (30h với case)", "Kết nối": "Bluetooth 5.3"}`,
		},
		{
			"Đồng Hồ Thông Minh Galaxy Watch 7", "galaxy-watch-7",
			"Samsung Galaxy Watch 7 theo dõi sức khỏe toàn diện: nhịp tim, SpO2, giấc ngủ, thành phần cơ thể BIA. Chip Exynos W1000 mới mạnh mẽ. Màn hình Super AMOLED sáng rõ ngoài trời. Chống nước 5ATM + IP68.",
			7490000, ptr(8990000.0),
			"https://images.unsplash.com/photo-1579586337278-3befd40fd17a?w=600&h=600&fit=crop", "accessories", "Samsung",
			4.7, 1654, 35,
			`{"Chip": "Exynos W1000", "Màn hình": "Super AMOLED 1.5 inch", "Sức khỏe": "Nhịp tim, SpO2, BIA, ECG", "Chống nước": "5ATM + IP68", "Pin": "40 giờ"}`,
		},
		{
			"Loa Bluetooth JBL Charge 5", "loa-jbl-charge-5",
			"Loa bluetooth JBL Charge 5 công suất 40W, âm bass mạnh mẽ. Pin 20 giờ phát nhạc liên tục. Chống nước chống bụi IP67, mang đi bơi thoải mái. Tính năng Powerbank sạc điện thoại khi cần.",
			3290000, ptr(4190000.0),
			"https://images.unsplash.com/photo-1608043152269-423dbba4e7e1?w=600&h=600&fit=crop", "accessories", "JBL",
			4.8, 2876, 45,
			`{"Công suất": "40W", "Pin": "20 giờ", "Chống nước": "IP67", "Kết nối": "Bluetooth 5.1", "Tính năng": "PartyBoost, Powerbank"}`,
		},
		{
			"Bàn Phím Cơ Logitech MX Mechanical", "ban-phim-logitech-mx-mechanical",
			"Bàn phím cơ không dây Logitech MX Mechanical, switch tactile yên tĩnh. Kết nối 3 thiết bị cùng lúc qua Bluetooth + USB receiver. Đèn nền thông minh tự điều chỉnh. Pin sạc USB-C dùng 15 ngày, tương thích Mac/Windows.",
			3490000, ptr(4290000.0),
			"https://images.unsplash.com/photo-1618384887929-16ec33fab9ef?w=600&h=600&fit=crop", "accessories", "Logitech",
			4.7, 1243, 30,
			`{"Switch": "Tactile Quiet", "Kết nối": "Bluetooth x3 + USB Receiver", "Pin": "15 ngày (có đèn)", "Đèn nền": "Smart Backlight", "Tương thích": "macOS, Windows, Linux"}`,
		},
		// ── Công cụ (tools) ──
		{
			"Máy Khoan Pin Bosch GSR 185-LI", "may-khoan-bosch-gsr-185",
			"Máy khoan vặn vít pin Bosch 18V chuyên nghiệp. Momen xoắn 50Nm mạnh mẽ, 2 cấp tốc độ. Mâm cặp 13mm tự khóa. Kèm 2 pin 2.0Ah + sạc nhanh, bảo hành chính hãng 12 tháng.",
			2890000, ptr(3690000.0),
			"https://images.unsplash.com/photo-1504148455328-c376907d081c?w=600&h=600&fit=crop", "tools", "Bosch",
			4.8, 1567, 25,
			`{"Điện áp": "18V", "Momen xoắn": "50 Nm", "Pin": "2 x 2.0Ah Li-ion", "Mâm cặp": "13mm tự khóa", "Bảo hành": "12 tháng"}`,
		},
		{
			"Bộ Dụng Cụ Sửa Chữa Đa Năng 150 Món", "bo-dung-cu-150-mon",
			"Bộ dụng cụ đa năng 150 món trong vali nhôm chắc chắn. Gồm cờ lê, tuýp, kìm, tua vít, búa, thước cuộn... đầy đủ cho mọi nhu cầu sửa chữa. Thép Chrome Vanadium bền bỉ, không gỉ sét.",
			1290000, ptr(1890000.0),
			"https://images.unsplash.com/photo-1581783898377-1c85bf937427?w=600&h=600&fit=crop", "tools", "Stanley",
			4.5, 876, 40,
			`{"Số món": "150 chi tiết", "Chất liệu": "Thép Chrome Vanadium", "Hộp đựng": "Vali nhôm", "Bảo hành": "24 tháng"}`,
		},
		{
			"Máy Hút Bụi Robot Ecovacs T30 Pro", "robot-hut-bui-ecovacs-t30",
			"Robot hút bụi lau nhà Ecovacs Deebot T30 Pro Omni. Lực hút 11000Pa siêu mạnh, lau quay 180 vòng/phút. Tự động giặt giẻ nước nóng 70°C, sấy khô. Bản đồ AI 3D LiDAR thông minh, tránh vật cản chính xác.",
			14990000, ptr(18990000.0),
			"https://images.unsplash.com/photo-1603618090561-412154b4bd1b?w=600&h=600&fit=crop", "tools", "Ecovacs",
			4.7, 2134, 15,
			`{"Lực hút": "11000 Pa", "Lau nhà": "Quay 180 vòng/phút", "Dock": "Tự giặt giẻ + sấy khô", "Điều hướng": "LiDAR + AI 3D", "Pin": "5200 mAh"}`,
		},
		{
			"Máy Lọc Không Khí Xiaomi 4 Pro", "may-loc-khi-xiaomi-4-pro",
			"Máy lọc không khí Xiaomi Smart Air Purifier 4 Pro, phòng đến 60m². Lọc bụi mịn PM2.5 hiệu quả 99.97% với màng lọc HEPA H13. Màn hình OLED hiển thị chất lượng không khí. Điều khiển qua app Mi Home, tương thích Google Home.",
			3990000, ptr(5290000.0),
			"https://images.unsplash.com/photo-1585771724684-38269d6639fd?w=600&h=600&fit=crop", "tools", "Xiaomi",
			4.6, 1432, 30,
			`{"Diện tích": "Đến 60 m²", "Màng lọc": "HEPA H13", "CADR": "500 m³/h", "Điều khiển": "App Mi Home, Google Home", "Độ ồn": "32.1 dB"}`,
		},
	}

	for i, p := range products {
		imageURL := p.Image

		// If MinIO is available, download and store images there
		if store != nil {
			objectName := storage.ObjectNameFromURL(p.Slug, p.Image)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			minioURL, err := store.DownloadFromURL(ctx, p.Image, objectName)
			cancel()
			if err != nil {
				log.Printf("Warning: failed to upload image for %s to MinIO: %v (using original URL)", p.Name, err)
			} else {
				imageURL = minioURL
				log.Printf("[%d/%d] Uploaded image for %s to MinIO", i+1, len(products), p.Name)
			}
		}

		_, err := db.Exec(
			`INSERT INTO products (name, slug, description, price, original_price, images, category, brand, rating, review_count, stock, specifications)
			 VALUES ($1, $2, $3, $4, $5, ARRAY[$6], $7, $8, $9, $10, $11, $12::jsonb)`,
			p.Name, p.Slug, p.Description, p.Price, p.OriginalPrice,
			imageURL, p.Category, p.Brand, p.Rating, p.ReviewCount, p.Stock, p.Specifications,
		)
		if err != nil {
			return fmt.Errorf("insert product %s: %w", p.Name, err)
		}
	}

	log.Println("Database seeded successfully")
	return nil
}

func ptr(f float64) *float64 {
	return &f
}
