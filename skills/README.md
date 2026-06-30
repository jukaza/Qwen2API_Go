# Bộ Kỹ Năng Agent dành cho LiteRouter (Vietnamese Core Skills)

Tài liệu này chứa danh sách các hướng dẫn nghiệp vụ cốt lõi dành cho AI Agent khi lập trình và phát triển hệ thống LiteRouter. Mọi hoạt động sửa đổi, bổ sung code hoặc nâng cấp tính năng liên quan đến các mô-đun này phải tuân thủ nghiêm ngặt các quy định trong từng tệp kỹ năng tương ứng.

## Danh sách các Kỹ năng Nghiệp vụ Cốt lõi (Core Skills)

| Kỹ năng | Đường dẫn tệp | Mô tả nhiệm vụ |
| --- | --- | --- |
| **Định tuyến & Combo Fallback** | [skills/router/SKILL.md](router/SKILL.md) | Quản lý cấu hình định tuyến API, thiết lập danh sách fallback (chuyển đổi dự phòng), tự động tính điểm combo và proxy yêu cầu OpenAI ↔ Anthropic. |
| **Nén dữ liệu & Token Saving** | [skills/compression/SKILL.md](compression/SKILL.md) | Triển khai nén token RTK cho đầu ra terminal/test/git và Caveman Mode cho câu trả lời ngắn gọn, tối ưu chi phí token đầu vào/đầu ra. |
| **Quản lý ví & Giao dịch SQLite** | [skills/billing/SKILL.md](billing/SKILL.md) | Quản lý tài khoản người dùng, xác thực API Key, thực hiện trừ tiền số dư (`balance_usd`) qua SQLite transaction an toàn chống race-condition. |

---
