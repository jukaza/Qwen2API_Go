---
name: router-fallback-mitm
description: Hướng dẫn Agent triển khai logic định tuyến, combo fallback (chuyển đổi dự phòng) và dịch định dạng API (MITM) trong dự án LiteRouter.
---

# Kỹ năng Định tuyến, Combo Fallback & Dịch định dạng API (Router Skill)

Kỹ năng này định nghĩa các quy tắc và cấu trúc triển khai logic định tuyến API, xử lý lỗi tự động, và dịch chuyển định dạng (OpenAI ↔ Anthropic ↔ Gemini) tại thư mục `src/lib/routing/` và `src/mitm/`.

## 1. Định nghĩa Combo & Chiến lược Định tuyến

**Combo** là một nhóm các nhà cung cấp/mô hình (targets) được gộp chung dưới một ID duy nhất (ví dụ: `my-fast-combo`). Khi người dùng gọi tới Endpoint với ID mô hình này, LiteRouter sẽ tự động áp dụng chiến lược định tuyến để phân phối tải và fallback.

### Các chiến lược định tuyến cốt lõi:
1.  **`priority` (Thứ tự ưu tiên):** Luôn thử target đầu tiên. Nếu gặp lỗi (Rate Limit, Server Error, Quota Exceeded), tự động chuyển sang target tiếp theo trong danh sách.
2.  **`weighted` (Trọng số):** Phân phối ngẫu nhiên theo tỷ lệ phần trăm trọng số được cấu hình.
3.  **`auto` (Tự động tính điểm):** Chấm điểm target dựa trên 9 yếu tố (chi phí, độ trễ p50, giới hạn quota, tình trạng circuit breaker, v.v.) để chọn ra mô hình tối ưu nhất tại thời điểm gửi request.
4.  **`round-robin` (Vòng tròn):** Luân phiên gọi tuần tự các target trong danh sách.
5.  **`cost-optimized` (Tối ưu chi phí):** Dựa trên số lượng token ước tính để chọn nhà cung cấp rẻ nhất.

---

## 2. Cơ chế Fallback an toàn (Circuit Breakers)

Mỗi target trong combo phải được bảo vệ bởi một **Circuit Breaker** (Bộ ngắt mạch) để tránh làm nghẽn hệ thống khi nhà cung cấp gặp sự cố liên tục.
*   **Trạng thái `CLOSED`:** Hoạt động bình thường.
*   **Trạng thái `OPEN`:** Tạm thời chặn target (không gửi request tới target này) trong một khoảng thời gian nhất định (ví dụ: 30 giây) sau khi target gặp lỗi liên tiếp N lần.
*   **Trạng thái `HALF_OPEN`:** Gửi một lượng nhỏ request thử nghiệm để kiểm tra xem nhà cung cấp đã khôi phục chưa. Nếu thành công, đưa về `CLOSED`, nếu thất bại tiếp tục chuyển về `OPEN`.

---

## 3. Dịch định dạng API ở lớp MITM (`src/mitm/`)

LiteRouter hoạt động như một proxy trung gian, nhận yêu cầu tương thích với OpenAI `/v1/chat/completions` và chuyển đổi sang định dạng của các nhà cung cấp khác nếu cần:

### Quy tắc dịch Anthropic (OpenAI ↔ Anthropic):
*   **Request:**
    *   Chuyển đổi `messages` của OpenAI (chứa role `system` ở đầu hoặc bất kỳ đâu) thành `system` prompt tham số riêng của Anthropic, và gộp/lọc các role `user`/`assistant` xen kẽ chuẩn chỉ.
    *   Bản đồ hóa tham số: `max_tokens` &rarr; `max_tokens`, `temperature` &rarr; `temperature`, `stream` &rarr; `stream`.
*   **Response:**
    *   Bọc đầu ra `content[0].text` hoặc stream chunks của Anthropic trở lại định dạng OpenAI (`choices[0].delta.content` hoặc `choices[0].message.content`).

### Quy tắc dịch Gemini (OpenAI ↔ Gemini):
*   Tương tự, chuyển đổi `messages` thành cấu trúc `contents` với các `parts`, trích xuất `systemInstruction` và bọc kết quả trả về đúng chuẩn OpenAI.

---

## 4. Quy ước viết code (TypeScript & Error Handling)

*   **Cấm bypass type check:** Tất cả các hàm chuyển đổi định dạng hoặc kiểm tra target phải định nghĩa kiểu dữ liệu Zod hoặc TypeScript rõ ràng, cấm dùng `any`.
*   **Fail-safe:** Khi quá trình dịch định dạng hoặc định tuyến gặp lỗi ngoài ý muốn, phải có cơ chế trả về lỗi chuẩn `502 Bad Gateway` hoặc fallback sang mô hình mặc định cấu hình sẵn, không được làm sập luồng request của client.
*   **Transaction SQLite:** Việc ghi nhận nhật ký cuộc gọi và cập nhật trạng thái lỗi target phải được lưu trực tiếp vào cơ sở dữ liệu SQLite để làm dữ liệu tính điểm.
