# Bộ Kỹ Năng Agent dành cho Qwen2API_Go (Vietnamese Core Skills)

Tài liệu này chứa danh sách các hướng dẫn nghiệp vụ cốt lõi dành cho AI Agent khi lập trình và phát triển hệ thống Qwen2API_Go. Mọi hoạt động sửa đổi, bổ sung code hoặc nâng cấp tính năng liên quan đến các mô-đun này phải tuân thủ nghiêm ngặt các quy định trong từng tệp kỹ năng tương ứng.

## Danh sách các Kỹ năng Nghiệp vụ Cốt lõi (Core Skills)

| Kỹ năng | Đường dẫn tệp | Mô tả nhiệm vụ |
| --- | --- | --- |
| **Quản lý Tài khoản & Xoay vòng** | [skills/router/SKILL.md](router/SKILL.md) | Quản lý vòng đời tài khoản Qwen, đăng nhập tự động, tự động làm mới session (refresh), xử lý lỗi 429/风控 (risk control), xoay vòng tài khoản (rotation) và duy trì pool khỏe mạnh. |
| **Dịch định dạng API & Stream** | [skills/compression/SKILL.md](compression/SKILL.md) | Chuyển đổi định dạng API request/response tương thích OpenAI / Anthropic sang Qwen Web API, xử lý định dạng luồng Stream (SSE), tối ưu lịch sử chat. |
| **Lưu trữ dữ liệu & Đồng bộ luồng** | [skills/billing/SKILL.md](billing/SKILL.md) | Quản lý lưu trữ cấu hình, tài khoản và metrics qua File JSON hoặc Redis dựa trên `DATA_SAVE_MODE`, đảm bảo an toàn đa luồng (thread-safe/atomic). |

---
