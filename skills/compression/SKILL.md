---
name: qwen-api-format-stream
description: Hướng dẫn Agent triển khai và tối ưu hóa logic dịch định dạng API giữa OpenAI/Anthropic và Qwen Chat upstream, xử lý Server-Sent Events (SSE) và nén ngữ cảnh hội thoại.
---

# Kỹ năng Dịch định dạng API & Xử lý Stream SSE (API translation Skill)

Kỹ năng này định nghĩa các quy tắc chuyển đổi định dạng request/response tương thích OpenAI và Anthropic sang định dạng Qwen Chat Web API và ngược lại, đồng thời định cấu hình luồng Server-Sent Events (SSE) mượt mà tại thư mục `internal/openai/` và `internal/qwen/`.

## 1. Dịch định dạng Request và Model Mapping

Hệ thống phải ánh xạ chính xác các tham số từ các API chuẩn (OpenAI `/v1/chat/completions`, Anthropic `/v1/messages`) sang định dạng API Web Qwen:
*   **Model Mapping:** Ánh xạ các model ID yêu cầu từ client (ví dụ: `gpt-4o`, `claude-3-5-sonnet`, `qwen-max`) sang ID model thực tế hoặc hành vi xử lý của Qwen Web Client (ví dụ: sử dụng chế độ tìm kiếm mạng `search-info-mode`, bật/tắt `OutThink` suy nghĩ sâu).
*   **System Prompt:** Qwen Web API hỗ trợ truyền chỉ dẫn hệ thống hoặc cấu hình tiền hội thoại. Hệ thống phải trích xuất system message từ danh sách `messages` và thiết lập cấu hình tương ứng cho Qwen.
*   **Hội thoại nhiều lượt (Multi-turn conversations):** Chuyển đổi mảng tin nhắn `messages` (gồm các role `user`, `assistant`, `system`) sang định dạng chuỗi lịch sử hội thoại chuẩn của Qwen Web API.

---

## 2. Xử lý Luồng Dữ Liệu Stream (SSE)

Khi client yêu cầu phản hồi dạng luồng (`stream: true`), hệ thống phải xử lý SSE cực kỳ chuẩn xác:
*   **Đọc và Parse SSE từ Qwen:** Qwen Web API trả về các block SSE dạng `data: {...}` chứa văn bản sinh ra theo thời gian thực (real-time). Go backend phải lắng nghe luồng response, giải mã JSON và trích xuất lượng ký tự mới (delta).
*   **Chuyển đổi sang chunk OpenAI/Anthropic:** Bọc lượng ký tự mới vào định dạng chunk dữ liệu của OpenAI hoặc Anthropic rồi ghi trực tiếp xuống HTTP Response Writer.
*   **Ký tự đặc biệt & Line endings:** Xử lý chính xác các ký tự đặc biệt, dấu xuống dòng (`\n`), khoảng trắng để đảm bảo nội dung hiển thị trên client không bị lỗi định dạng.
*   **Kết thúc luồng:** Gửi chunk kết thúc luồng (`data: [DONE]`) và đóng kết nối HTTP đúng cách.

---

## 3. Quản lý Lịch sử Hội thoại & Nén Ngữ cảnh

*   **Giới hạn token lịch sử:** Vì Qwen Chat Web có thể bị quá tải context hoặc gặp lỗi khi lịch sử trò chuyện quá dài, Agent cần triển khai cơ chế giới hạn hoặc rút ngắn các tin nhắn cũ hơn trong lịch sử hội thoại (ví dụ: chỉ giữ lại N tin nhắn gần nhất hoặc tóm tắt nội dung cũ) để tiết kiệm token và đảm bảo độ ổn định của API.
*   **Tối ưu hóa Prompt và System Overrides:** Hỗ trợ cấu hình Prompts ghi đè (Qwen Web2 Control Prompt) để điều khiển hành vi trả lời của Qwen, giúp kết quả đầu ra tự nhiên và tương thích tốt nhất với yêu cầu của client.

---

## 4. Kỷ luật An toàn luồng dữ liệu (Fail-Safe)

*   **Không ngắt luồng đột ngột:** Nếu luồng xử lý stream gặp lỗi giữa chừng (ví dụ: mất kết nối upstream tạm thời), hệ thống phải cố gắng gửi chunk kết thúc an toàn kèm lý do lỗi hoặc log lỗi, không để kết nối của client bị treo vô hạn.
*   **Giải phóng tài nguyên:** Luôn giải phóng kết nối mạng (`resp.Body.Close()`) và dọn dẹp các session trong Go `defer` block để tránh rò rỉ bộ nhớ (memory leak) hoặc file descriptor.
