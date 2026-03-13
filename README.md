# AllShop API

Backend API server cho ứng dụng bán hàng AllShop, xây dựng bằng **Go** + **PostgreSQL**.

## Khởi chạy

```bash
docker compose up --build -d
```

API chạy tại: `http://localhost:8000`

## API Endpoints

### Public

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| GET | `/api/products` | Danh sách sản phẩm (`?category=`, `?search=`) |
| GET | `/api/products/:id` | Chi tiết sản phẩm |
| GET | `/api/categories` | Danh mục sản phẩm |
| POST | `/api/auth/register` | Đăng ký (`name`, `email`, `password`) |
| POST | `/api/auth/login` | Đăng nhập (`email`, `password`) |

### Protected (Bearer Token)

| Method | Endpoint | Mô tả |
|--------|----------|-------|
| POST | `/api/auth/logout` | Đăng xuất |
| GET | `/api/users/profile` | Thông tin người dùng |
| PUT | `/api/users/profile` | Cập nhật hồ sơ |
| GET | `/api/cart` | Giỏ hàng |
| POST | `/api/cart/items` | Thêm vào giỏ |
| PUT | `/api/cart/items/:productId` | Cập nhật số lượng |
| DELETE | `/api/cart/items/:productId` | Xóa khỏi giỏ |
| POST | `/api/orders` | Tạo đơn hàng |
| GET | `/api/orders` | Danh sách đơn hàng |
| GET | `/api/orders/:id` | Chi tiết đơn hàng |
