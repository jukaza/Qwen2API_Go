---
name: token-compression-saver
description: Hướng dẫn Agent cách thức triển khai và tối ưu hóa bộ nén token RTK (Rich Tool Output) và cấu hình Caveman Mode trong dự án LiteRouter.
---

# Kỹ năng Nén Dữ Liệu & Tiết Kiệm Token (Compression Skill)

Kỹ năng này định nghĩa các quy tắc thiết lập và phát triển hệ thống nén token đầu vào và đầu ra tại thư mục `src/lib/compression/`. Mục tiêu là tiết kiệm chi phí sử dụng API từ 20% đến 90%.

## 1. Bộ nén RTK Tool Saver (Nén đầu vào)

Bộ nén **RTK (Rich Tool Output)** chuyên xử lý các dữ liệu đầu ra cồng kềnh, lặp đi lặp lại từ các công cụ phát triển, kiểm thử hoặc hệ thống kiểm soát phiên bản trước khi gửi lên LLM.

### Các loại dữ liệu cần nén:
*   **Git Diffs:** Loại bỏ các phần metadata của git không cần thiết, nén các dòng code không đổi xung quanh thay đổi chính.
*   **Test logs (Jest/Mocha/Go test):** Rút gọn các dòng log thành công, chỉ giữ lại chi tiết lỗi (stack traces) và kết quả tổng quan.
*   **Grep / Search results:** Gom nhóm kết quả trùng lặp, loại bỏ khoảng trắng dư thừa và giới hạn độ dài dòng.
*   **Terminal Output / Command logs:** Rút gọn các dòng tiến trình (progress bars), các ký tự đặc biệt ANSI điều khiển màu sắc của terminal.

---

## 2. Caveman Mode (Nén đầu ra)

**Caveman Mode** ép buộc mô hình LLM trả lời ngắn gọn, trực diện, loại bỏ toàn bộ các câu từ giao tiếp xã giao không mang lại giá trị kỹ thuật.

### Các mức độ nén (Caveman Intensities):
1.  **`lite` (An toàn):** Lược bỏ câu chào đầu/cuối ("Chắc chắn rồi!", "Tôi có thể giúp gì..."), giữ nguyên cấu trúc giải thích.
2.  **`standard` (Cân bằng):** Trả lời trực diện, chỉ dùng các câu đơn ngắn, ưu tiên hiển thị code trước và giải thích sau.
3.  **`aggressive` (Mạnh mẽ):** Ép LLM trả lời tối giản như một người tối cổ, chỉ đưa ra kết quả/code thô, lược bỏ tối đa các lời giải thích dạng văn bản.
4.  **`ultra` (Khôi phục ngữ cảnh):** Chỉ trả lời đúng từ khóa hoặc đoạn code thay đổi duy nhất, không giải thích.

---

## 3. Cơ chế hoạt động Stacked Mode (Nén kép)

Khi kích hoạt **Stacked Mode**, hệ thống sẽ chạy chuỗi nén:
`Dữ liệu đầu vào` &rarr; `Nén RTK` &rarr; `Áp dụng System Prompt Caveman` &rarr; `Gửi LLM`.

---

## 4. Kỷ luật Viết code an toàn (Fail-Safe)

*   **Quy tắc Fail-Safe (Không làm vỡ Request):** Bộ nén token phải hoạt động độc lập và an toàn tuyệt đối. Nếu quá trình nén gặp lỗi (lỗi phân tích cú pháp, tràn bộ nhớ, Regex chạy vô hạn), hệ thống phải bắt giữ ngoại lệ (`try-catch`), **trả về nguyên bản dữ liệu gốc** và ghi nhận log lỗi. Tuyệt đối không được làm crash request của người dùng.
*   **Bảo toàn Code:** Bộ nén RTK không được làm thay đổi cấu trúc mã nguồn hoặc dữ liệu logic quan trọng nằm trong payload, chỉ nén các log/text trang trí.
