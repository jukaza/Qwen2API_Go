---
name: qwen-data-storage-concurrency
description: Hướng dẫn Agent cách thức triển khai lưu trữ thông tin cấu hình, tài khoản và metrics qua File JSON phẳng hoặc Redis, đảm bảo an toàn đa luồng (thread-safe) trong Qwen2API_Go.
---

# Kỹ năng Lưu trữ Dữ liệu & An toàn Đa luồng (Storage Skill)

Kỹ năng này định nghĩa các quy tắc lưu trữ thông tin cấu hình hệ thống, danh sách tài khoản Qwen, phiên hội thoại, và số liệu metrics thông qua File JSON cục bộ hoặc Redis DB, đồng thời quản lý an toàn luồng (thread-safety) trong Go backend tại thư mục `internal/storage/`.

## 1. Các chế độ lưu trữ dữ liệu (DATA_SAVE_MODE)

Hệ thống hỗ trợ 2 chế độ lưu trữ linh hoạt thông qua biến môi trường `DATA_SAVE_MODE`:

### Chế độ `file` (Lưu file JSON phẳng):
*   Dữ liệu tài khoản (`accounts.json`), dữ liệu phiên hội thoại và cấu hình runtime được ghi trực tiếp vào các file JSON trong thư mục `data/`.
*   **Quy trình ghi file an toàn (Safe-Write):** Để tránh hỏng dữ liệu (file corruption) khi ghi file bị gián đoạn giữa chừng do crash hoặc mất nguồn điện, hệ thống phải thực hiện ghi dữ liệu mới vào một file tạm thời (ví dụ: `accounts.json.tmp`), sau đó thực hiện lệnh đổi tên file atomic (`os.Rename`) để thay thế file cũ.

### Chế độ `redis` (Lưu cơ sở dữ liệu Redis):
*   Kết nối tới Redis server thông qua thư viện `github.com/redis/go-redis/v9`.
*   Lưu trữ danh sách tài khoản, thông tin đăng nhập, token, metrics dưới dạng Redis Hashes hoặc các key-value tương ứng.
*   Thiết lập cơ chế tự động kết nối lại và xử lý lỗi kết nối Redis an toàn.

---

## 2. Đồng bộ luồng & An toàn Đa luồng (Thread-Safety)

Trong Golang, server HTTP phục vụ hàng ngàn request đồng thời (concurrently). Mọi thao tác truy cập vào bộ nhớ đệm (cache) hoặc store lưu trữ tài khoản/metrics phải đảm bảo thread-safe:
*   **Sử dụng `sync.RWMutex`:** Sử dụng Read-Write Mutex khi quản lý store ở chế độ `file`. Cho phép nhiều request đọc trạng thái tài khoản song song (`RLock`/`RUnlock`), nhưng độc chiếm lock khi cập nhật trạng thái hoặc ghi dữ liệu mới (`Lock`/`Unlock`).
*   **Hạn chế giữ Lock lâu:** Không bao giờ giữ Write Lock (`Lock()`) trong khi đang thực hiện các tác vụ tốn thời gian như gọi API mạng Qwen hoặc thực hiện thao tác I/O ổ đĩa chậm. Chỉ Lock khi cập nhật nhanh biến trong RAM, sau đó Unlock ngay lập tức.
*   **Các lệnh atomic trong Redis:** Khi hoạt động ở chế độ `redis`, tận dụng các lệnh atomic của Redis (như `HSET`, `HGET`, `INCR`...) hoặc Redis Transactions/Pipelines để đảm bảo tính nhất quán dữ liệu mà không cần dùng Mutex cục bộ của Go.

---

## 3. Xác thực & Bảo mật API Key

*   Mọi request đến API Gateway của Qwen2API_Go phải được xác thực qua API Key được thiết lập trong biến môi trường `API_KEY` (hoặc `ADMIN_KEY` đối với các API quản trị).
*   API Key được tách biệt và đối chiếu thông qua lớp quản lý xác thực `internal/auth/`.
*   Đảm bảo các khóa API được đối chiếu an toàn, log thông tin lỗi auth mà không hiển thị công khai toàn bộ API Key (sử dụng hàm mask che bớt ký tự nhạy cảm).
