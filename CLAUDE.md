# Quy ước Lập trình dành cho AI Agent (CLAUDE.md)

Tài liệu này đóng vai trò như cẩm nang hướng dẫn vận hành trực tiếp dành cho các AI Agent (như Antigravity, Cursor, Claude Code, Cline) khi phát triển dự án **Qwen2API_Go**. Hãy tuân thủ nghiêm ngặt các quy tắc dưới đây để đảm bảo chất lượng, cấu trúc mã nguồn và tiết kiệm token ngữ cảnh.

---

## 0. Quy tắc Giao tiếp & Khởi đầu phản hồi (Bắt buộc)

*   **Dòng khai báo nhận thức chống ảo giác:** Ở đầu **MỌI** câu trả lời, Agent bắt buộc phải in ra một dòng khai báo trạng thái có dạng:
    `[Qwen2API-Agent] <Chào người dùng> | Trạng thái: <Mô tả ngắn gọn nhận thức hiện tại của bạn về yêu cầu / tệp tin đang mở>`
    Ví dụ: `[Qwen2API-Agent] Chào bạn! | Trạng thái: Đang phân tích yêu cầu tích hợp logic ví và tệp internal/storage/account.go đang mở.`
    *Quy tắc này giúp người dùng phát hiện ngay lập tức nếu Agent bắt đầu bị mất ngữ cảnh hoặc ảo giác.*
*   **Kỷ luật làm rõ yêu cầu trước khi lập kế hoạch (Plan):** CẤM TUYỆT ĐỐI tự ý đề xuất Kế hoạch triển khai (Implementation Plan) hoặc viết mã nguồn khi yêu cầu của người dùng còn mơ hồ, thiếu chi tiết hoặc chưa được làm rõ 100%. Agent phải chủ động đặt câu hỏi làm rõ (hỏi đi hỏi lại user) để thống nhất chi tiết trước, sau khi người dùng xác nhận đầy đủ mọi chi tiết của yêu cầu mới được phép lên plan hoặc thực thi.

---

## 1. Tổng quan & Công nghệ sử dụng

Qwen2API_Go là bộ proxy chuyển đổi Qwen Chat thành API tương thích OpenAI / Anthropic, tích hợp quản lý tài khoản pool, quản lý tệp tin, sinh ảnh/video và một giao diện panel điều khiển (dashboard). Hệ thống gồm **2 lớp tách biệt (hybrid stack)**: một **Backend (BE) viết bằng Golang** đảm nhận toàn bộ logic proxy/account pool/billing, và một **Frontend (FE) viết bằng Next.js** đóng vai trò Panel điều khiển (dashboard).

### Tech Stack — Backend (BE / Golang)
*   **Ngôn ngữ:** Go 1.25.6+ (module `qwen2api`)
*   **Web Server:** `net/http` standard library (sử dụng `http.NewServeMux()` định tuyến, không dùng Gin)
*   **Database/Storage:** Lưu trữ local file (JSON) hoặc Redis tùy thuộc vào cấu hình `DATA_SAVE_MODE` (`file` hoặc `redis`)
*   **Thư viện hỗ trợ:** `github.com/redis/go-redis/v9` (khi dùng chế độ Redis), `github.com/aliyun/aliyun-oss-go-sdk` (upload OSS)
*   **Mã nguồn:** `cmd/qwen2api/main.go` (điểm khởi chạy) + thư mục `internal/` (account, admin, auth, cleanup, config, logging, metrics, openai, qwen, server, storage)
*   **Cổng API:** `http://127.0.0.1:3000` (hoặc cổng cấu hình qua `SERVICE_PORT`)

### Tech Stack — Frontend (FE / Next.js)
*   **Runtime:** Node.js (v20+ hoặc v22+ LTS)
*   **Framework:** Next.js 16.2.4 (App Router) + React 19.2.4
*   **CSS & Styling:** Tailwind CSS v4 + HeroUI (`@heroui/react` v3.0.3)
*   **TypeScript:** v5+ (Strict Mode)
*   **Mã nguồn:** thư mục `public/` (chứa toàn bộ dự án Next.js)
*   **Cổng Panel (khi chạy dev):** `http://localhost:3000` (được proxy qua server Go hoặc chạy độc lập qua Next dev)
*   **Biên dịch tĩnh (Static Export):** Build ra thư mục `public/out`, được Go backend phục vụ trực tiếp qua route `/`

---

## 2. Các lệnh vận hành (Commands)

AI Agent **phải** sử dụng đúng bộ lệnh theo từng lớp khi kiểm tra hoặc biên dịch dự án.

### Backend (Golang) — chạy tại thư mục gốc
*   **Tải/đồng bộ dependency:** `go mod tidy`
*   **Khởi chạy dev:** `go run cmd/qwen2api/main.go`
*   **Biên dịch (Build):** `go build -o qwen2api cmd/qwen2api/main.go` (hoặc `go build -o qwen2api.exe cmd/qwen2api/main.go` trên Windows)
*   **Kiểm tra tĩnh (Vet):** `go vet ./...`
*   **Định dạng mã (Format):** `gofmt -w .` (hoặc `go fmt ./...`)

### Frontend (Next.js) — chạy tại thư mục `public/`
*   **Cài đặt thư viện:** `npm install` (phải chạy bên trong thư mục `public/`)
*   **Khởi chạy dev server:** `npm run dev` (phải chạy bên trong thư mục `public/`)
*   **Biên dịch dự án (Build & Export):** `npm run build` (phải chạy bên trong thư mục `public/`, tạo ra thư mục `public/out`)
*   **Kiểm tra lỗi cú pháp (Lint):** `npm run lint` (phải chạy bên trong thư mục `public/`)

---

## 3. Tiêu chuẩn viết mã nguồn (Code Style & Patterns)

### Golang nghiêm ngặt — áp dụng cho `cmd/qwen2api/main.go` & `internal/` (Backend)
*   **Định dạng chuẩn:** Mọi tệp `.go` phải vượt qua `gofmt`. Không tự bịa style thụt đầu dòng riêng.
*   **Xử lý lỗi tường minh:** Luôn kiểm tra `err` ngay sau lệnh trả về lỗi; bọc ngữ cảnh bằng `fmt.Errorf("...: %w", err)` để giữ chuỗi lỗi (error wrapping). Cấm bỏ qua lỗi bằng `_` trừ khi thực sự vô hại và có chú thích lý do.
*   **Không panic trong luồng request:** Trả về mã HTTP phù hợp thay vì `panic`. Cơ chế hoạt động của proxy/account pool phải fail-safe.
*   **Quản lý dependency:** Cấm thêm package Go mới khi chưa được người dùng đồng ý. Sau khi thêm phải chạy `go mod tidy`.
*   **Context-aware:** Các lời gọi mạng/ngược dòng (upstream Qwen) phải nhận và tôn trọng `context.Context` để hủy/timeout đúng cách.

### TypeScript nghiêm ngặt — áp dụng cho `public/` (Frontend)
*   **Cấm sử dụng `any`:** Tất cả các tham số, biến và kiểu trả về của hàm phải được khai báo rõ ràng.
*   **Cấm tắt cảnh báo linter:** Không sử dụng `@ts-ignore` hoặc `@ts-nocheck` trừ trường hợp được yêu cầu cụ thể để giải quyết lỗi của thư viện bên thứ ba.

### Quy tắc Database & Storage
*   **DATA_SAVE_MODE:** Hỗ trợ `file` (JSON phẳng ở thư mục `data/`) và `redis` (thông qua `go-redis`).
*   **Đồng bộ dữ liệu:** Các thao tác thay đổi trạng thái tài khoản, cấu hình hoặc metrics phải đảm bảo an toàn luồng (thread-safe) sử dụng Mutex hoặc cơ chế atomic của Redis.

### Phân tách lớp nghiệp vụ (Separation of Concerns)

**Backend (Golang) — `internal/`:**
*   **`internal/server/`:** Tầng định tuyến HTTP — sử dụng `http.NewServeMux()`, cấu hình middleware xác thực và CORS, phục vụ thư mục tĩnh `public/out`.
*   **`internal/openai/`:** Xử lý các request tương thích OpenAI và Anthropic API, định dạng lại luồng Stream SSE.
*   **`internal/qwen/`:** API client kết nối trực tiếp đến Qwen chat upstream.
*   **`internal/account/` & `internal/storage/`:** Quản lý danh sách tài khoản, đăng nhập tự động, xoay vòng tài khoản (rotation) và lưu trữ trạng thái.

**Frontend (Next.js) — `public/`:**
*   Phát triển giao diện React sạch sẽ, gọn gàng trong thư mục `public/`.
*   Không trộn lẫn logic giao diện và logic của Backend.

---

## 4. Giới hạn & Quy tắc cấm (Boundaries)

*   **Giới hạn phạm vi:** Chỉ được làm việc trên thư mục gốc và các thư mục con `cmd/`, `internal/`, `public/`.
*   **Không tự ý cài đặt thư viện:** Cấm cài đặt thêm thư viện NPM (phía FE) **hoặc** thêm Go module mới (phía BE) khi chưa có sự đồng ý của người dùng.
*   **Quản lý thư mục tạm/rác:** Mọi tệp tin tạm thời, script test (như Python, PowerShell, bash) được tạo ra trong quá trình phát triển bắt buộc phải nằm trong thư mục `trash/` ở thư mục gốc (hoặc tự động xóa sạch sau khi làm xong).

---

## 5. Bắt buộc áp dụng Bộ Kỹ Năng Agent (Taste Skills & Code Skills)

Mọi hoạt động phát triển, thiết kế UI, viết mã nguồn hoặc tái thiết kế (redesign) của Agent **bắt buộc** phải tuân thủ nghiêm ngặt các hướng dẫn và bộ quy tắc nằm trong thư mục `.agents/skills/` và `skills/`.

### 5.1 Kỹ năng Giao diện & Thiết kế (Frontend & Taste Skills)
*   **design-taste-frontend** (`.agents/skills/design-taste-frontend/SKILL.md`): Áp dụng 100% khi xây dựng hoặc cập nhật UI.
*   **full-output-enforcement** (`.agents/skills/full-output-enforcement/SKILL.md`): Cấm tuyệt đối viết tắt, viết code giữ chỗ kiểu `// ...` hay `// TODO`.
*   **image-to-code** (`.agents/skills/image-to-code/SKILL.md`): Phác thảo/sinh mockup bằng `generate_image` trước khi triển khai UI phức tạp.
*   **industrial-brutalist-ui** & **minimalist-ui**: Phù hợp với giao diện tối giản của panel quản trị.

### 5.2 Kỹ năng Nghiệp vụ Cốt lõi (Core Skills)
*   **router/SKILL.md** (`skills/router/SKILL.md`): Tham chiếu về định tuyến, combo fallback và dịch định dạng.
*   **compression/SKILL.md** (`skills/compression/SKILL.md`): Hướng dẫn nén token đầu vào/đầu ra.
*   **billing/SKILL.md** (`skills/billing/SKILL.md`): Tham chiếu quản lý tài khoản và giao dịch an toàn.

---

## 6. Tiêu chuẩn hoàn thành công việc (Definition of Done)

Trước khi bàn giao bất kỳ tính năng nào, AI Agent phải tự kiểm tra và thực hiện:
1.  **Backend (nếu chạm `internal/` hoặc `cmd/`):** Biên dịch sạch (`go build -o qwen2api cmd/qwen2api/main.go`), không lỗi `go vet ./...`, đã chạy `gofmt -w .`.
2.  **Frontend (nếu chạm `public/`):** Không lỗi kiểu (`npm run typecheck` hoặc build thành công `npm run build` không lỗi linter).
3.  Lỗi hệ thống được xử lý tường minh, bọc lỗi chi tiết và trả về mã lỗi HTTP OpenAI-compatible.
