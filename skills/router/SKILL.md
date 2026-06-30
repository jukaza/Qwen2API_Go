---
name: qwen-account-pool-rotation
description: Hướng dẫn Agent triển khai logic quản lý danh sách tài khoản Qwen Chat, tự động đăng nhập, làm mới token, xoay vòng tài khoản (rotation) và xử lý lỗi/wind control trong dự án Qwen2API_Go.
---

# Kỹ năng Quản lý Tài khoản, Xoay vòng & Xử lý lỗi (Account Pool Skill)

Kỹ năng này định nghĩa các quy tắc và cấu trúc triển khai quản lý danh sách tài khoản Qwen Chat upstream, tự động hóa luồng đăng nhập lấy token, xoay vòng tài khoản khi xử lý request chat, và đối phó với cơ chế风控 (risk control) của Qwen tại thư mục `internal/account/` và `internal/storage/`.

## 1. Quản lý Tài khoản Qwen (Account Pool)

Mỗi tài khoản Qwen trong hệ thống được lưu trữ với các thông tin:
*   `Email`, `Password`: Thông tin đăng nhập gốc.
*   `AccessToken`, `RefreshToken`: Token phiên làm việc hiện tại thu được từ Qwen.
*   `Status`: Trạng thái hoạt động (`healthy`, `rate_limited`, `unauthorized`, `captcha_required`, `banned`).
*   `LastUsedTime`: Thời điểm cuối cùng tài khoản được sử dụng để phân phối tải đều.
*   `Concurrency`: Số lượng request đang chạy đồng thời trên tài khoản này để không vượt quá giới hạn cho phép.

### Nguyên tắc xoay vòng (Rotation):
*   Khi nhận được request chat, hệ thống phải chọn tài khoản từ pool dựa trên các tiêu chí: trạng thái `healthy`, thời gian sử dụng cuối cách xa nhất (Least Recently Used), và số lượng request đồng thời (`Concurrency`) chưa vượt giới hạn.
*   Duy trì cơ chế khóa luồng ngắn (Mutex lock) khi lấy và cập nhật thông tin tài khoản từ pool để tránh race conditions giữa các request song song.

---

## 2. Đăng nhập tự động & Làm mới Session (Auth Lifecycle)

*   **Tự động Đăng nhập:** Khi khởi chạy hệ thống hoặc khi tài khoản chưa có token, hệ thống phải tự động thực hiện luồng đăng nhập (email/password) thông qua Qwen API Client để lấy `accessToken`.
*   **Refresh Token:** Trước khi thực hiện request, nếu phát hiện token sắp hết hạn (dựa trên thời gian hết hạn hoặc nhận lỗi `401 Unauthorized` từ upstream), hệ thống phải tự động thực hiện luồng refresh session để lấy token mới mà không cần can thiệp thủ công.
*   **Fail-safe khi đăng nhập thất bại:** Nếu một tài khoản đăng nhập sai thông tin liên tiếp (ví dụ: mật khẩu sai), hãy đánh dấu trạng thái tài khoản là `unauthorized` hoặc `banned` để tránh gửi request đăng nhập liên tục dẫn đến bị khóa IP/tài khoản.

---

## 3. Xử lý Lỗi Rate Limit (429) & Phong Khống (Risk Control)

Qwen áp dụng các biện pháp wind control (phòng chống bot, captcha) rất nghiêm ngặt. Khi nhận kết quả từ upstream, hệ thống cần nhận diện đúng loại lỗi:
1.  **Lỗi Rate Limit (429):**
    *   Tự động cập nhật trạng thái tài khoản sang `rate_limited` kèm thời gian hết hạn chặn (cooldown duration).
    *   Kích hoạt cơ chế xoay vòng tài khoản (fallback) để chọn tài khoản thay thế khác trong pool khỏe mạnh để xử lý tiếp request hiện tại của người dùng.
2.  **Lỗi Captcha / Verification:**
    *   Đánh dấu trạng thái tài khoản cần giải captcha (`captcha_required`).
    *   Chuyển request sang tài khoản khác.
3.  **Lỗi Bị Khóa Tài Khoản (Banned):**
    *   Đánh dấu trạng thái tài khoản là `banned`.
    *   Log cảnh báo mức `WARN` hoặc `ERROR` để người quản trị biết và thay thế tài khoản.

---

## 4. Kỷ luật Fail-Safe ở API Gateway

*   Nếu **TẤT CẢ** tài khoản trong pool đều gặp lỗi hoặc bị giới hạn, hệ thống không được crash mà phải trả về một phản hồi lỗi HTTP chuẩn OpenAI (ví dụ: mã `429 Too Many Requests` hoặc `503 Service Unavailable`) kèm JSON body mô tả chi tiết lỗi thân thiện với API Client.
