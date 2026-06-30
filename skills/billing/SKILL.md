---
name: billing-wallet-sqlite
description: Hướng dẫn Agent cách thức triển khai quản lý tài khoản ví người dùng, số dư USD, kiểm tra khóa API và thực hiện các giao dịch SQLite an toàn trong LiteRouter.
---

# Kỹ năng Quản lý Ví, Giao dịch SQLite & Bảo mật API Key (Billing Skill)

Kỹ năng này định nghĩa các quy tắc thiết lập cơ chế xác thực, quản lý ví người dùng và trừ tiền số dư tài khoản tại thư mục `src/lib/billing/` và các lớp truy vấn SQLite.

## 1. Bảo mật API Key & Xác thực

*   Mọi yêu cầu gửi tới LiteRouter (ngoại trừ trang cấu hình công khai hoặc tài nguyên tĩnh) đều phải được xác thực qua header `Authorization: Bearer <API_KEY>`.
*   API Key phải được đối chiếu trực tiếp với bảng `api_keys` hoặc `users` trong SQLite.
*   **Chống Timing Attack:** Sử dụng các hàm so sánh chuỗi an toàn (ví dụ: `crypto.timingSafeEqual`) khi xác thực API Key để ngăn chặn dò khóa bằng phân tích thời gian phản hồi.

---

## 2. Giao dịch SQLite An Toàn (Transactions)

Các thao tác liên quan đến tiền bạc và số dư tài khoản người dùng (`balance_usd`) là các tài nguyên nhạy cảm, dễ xảy ra lỗi đồng thời (race conditions) khi có nhiều request gửi tới cùng một lúc.

### Quy tắc cứng về SQLite Transactions:
*   **Sử dụng transaction duy nhất:** Thao tác kiểm tra số dư và trừ tiền bắt buộc phải được bọc trong một transaction đơn nhất thông qua `db.transaction()` của `better-sqlite3`.
*   **Cơ chế khóa ghi (Write Lock):** Trong SQLite, khi chạy transaction sửa đổi số dư, hãy đảm bảo rằng luồng ghi được độc chiếm để ngăn chặn trường hợp "Double Spending" (hai request trừ tiền song song dẫn đến số dư bị cập nhật sai lệch hoặc âm tiền không mong muốn).
*   *Đúng:*
    ```typescript
    const deductBalance = db.transaction((userId: string, amount: number) => {
      // 1. SELECT số dư hiện tại của userId kèm khóa ghi
      const user = db.prepare('SELECT balance_usd FROM users WHERE id = ?').get(userId);
      if (!user || user.balance_usd < amount) {
        throw new Error('INSUFFICIENT_BALANCE');
      }
      // 2. UPDATE số dư mới
      db.prepare('UPDATE users SET balance_usd = balance_usd - ? WHERE id = ?').run(amount, userId);
      return true;
    });
    ```
*   **Không bao giờ cập nhật số dư bằng các truy vấn rời rạc ngoài Transaction.**

---

## 3. Theo dõi & Tính phí Token thực tế

*   Sau khi proxy nhận kết quả trả về từ nhà cung cấp mô hình AI, LiteRouter phải trích xuất số lượng token tiêu thụ thực tế (`prompt_tokens`, `completion_tokens`).
*   Tính toán chi phí dựa trên bảng giá mô hình được lưu trữ trong SQLite (không lưu cứng trong code).
*   Thực hiện trừ tiền tương ứng vào tài khoản người dùng ngay lập tức trong luồng phản hồi (luồng stream cần được theo dõi bằng bộ đếm chunk để trừ tiền ngay sau khi kết thúc stream).
