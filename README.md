# 🤖 AI Proxy Gateway (The Cost Saver)

<div align="center">

![Golang](https://img.shields.io/badge/Golang-1.22-00ADD8?logo=go)
![SQLite](https://img.shields.io/badge/SQLite-3-003B57?logo=sqlite)
![Gin](https://img.shields.io/badge/Gin-1.9-blue)
![License: MIT](https://img.shields.io/badge/License-MIT-green)

A Blazing Fast Reverse Proxy for LLM APIs (OpenAI / Anthropic / Local models). **Caches identical prompts** to save you 100% API costs during testing and development. Drops response latency from 3000ms down to **< 5ms**.

*Read this in other languages: [English](#english) | [Tiếng Việt](#tiếng-việt)*

</div>

---

## <a name="english"></a>🇬🇧 English

### 💡 The Problem
During AI app development, you often send the *same prompt* hundreds of times to tune UI or fix backend bugs. 
Every time you hit `api.openai.com`, you are **losing money** and **waiting 2-5 seconds** for a response you already know.

### 🚀 The Solution: AI Proxy Gateway
Simply change your SDK's Base URL to `http://localhost:8080`.
This Gateway intercepts your `POST /v1/chat/completions`:
1. Hashes your entire JSON body.
2. If it's a **Cache Hit**, returns the exact previous response instantly (Cost: $0.00, Time: 1ms).
3. If it's a **Cache Miss**, forwards to OpenAI, returns the response, and silently caches it in a high-speed local SQLite database.
4. **Token Cost Tracker**: Automatically intercepts `usage.total_tokens` and calculates exact USD $ saved per request.
5. **Built-in HTML Dashboard**: Head to `http://localhost:8080/admin` to view a gorgeous UI tracking your lifetime savings!

### ⚡ Quick Start

**1. Build & Run**
```bash
go mod tidy
go build main.go

# Run the proxy on port 8080
./main
```

**2. Connect Your App**
Instead of calling OpenAI directly, point your app to `localhost:8080`. Your API Keys will be cleanly forwarded to the upstream server.

**Example using cURL:**
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-actual-openai-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "Why is the sky blue?"
      }
    ]
  }'
```

**3. Watch the Magic 🎩**
Terminal Output:
```text
[CACHE MISS] Hash: a8f4b2c1 | Forwarding to OpenAI...
(Request took 1,250ms)

# Run the exact same request again:
[CACHE HIT] Hash: a8f4b2c1 | Time: 2ms | SAVED 💰
(Request took 2ms)
```

---

## <a name="tiếng-việt"></a>🇻🇳 Tiếng Việt

### 💡 Vấn Đề
Khi phát triển ứng dụng AI, các lập trình viên thường xuyên phải gửi đi gửi lại *cùng một câu lệnh (Prompt)* hàng trăm lần để sửa lỗi giao diện hoặc logic. Mỗi lần bạn gọi thẳng tới `api.openai.com`, bạn **đang bị mất tiền oan** và **phải chờ 2-5 giây** để nhận lại một câu trả lời mà bạn đã biết trước.

### 🚀 Giải Pháp: AI Proxy Gateway
Chỉ cần đổi Base URL trong SDK của bạn thành `http://localhost:8080`.
Gateway này sẽ đánh chặn lệnh `POST /v1/chat/completions`:
1. Tạo mã băm (Hash) toàn bộ nội dung JSON body của bạn.
2. Nếu **Trúng Cache (Hit)**, trả về y hệt kết quả của lần trước ngay lập tức (Chi phí: $0.00, Thời gian: 1ms).
3. Nếu **Trượt Cache (Miss)**, thay mặt bạn gọi tới OpenAI, hứng kết quả trả về, và âm thầm lưu nó vào cơ sở dữ liệu SQLite siêu tốc cục bộ.
4. **Kế Toán Thông Minh (Token Tracker)**: Tự động phân tích trường `usage.total_tokens` và tính ra chuẩn xác số tiền Đô La ($) bạn vừa tiết kiệm được.
5. **Giao Diện Dashboard Xịn Xò**: Truy cập `http://localhost:8080/admin` để ngắm nhìn bảng thống kê xịn xò bao gồm "Total Request" và "Total USD Saved 💰".

### ⚡ Hướng Dẫn Nhanh

**1. Cài Đặt & Chạy**
```bash
go mod tidy
go build main.go

# Chạy proxy server ở cổng 8080
./main
```

**2. Kết Nối Ứng Dụng**
Thay vì gọi OpenAI trực tiếp, hãy trỏ ứng dụng của bạn tới `localhost:8080`. Các API Keys của bạn vẫn sẽ được âm thầm chuyển tiếp (forward) an toàn lên máy chủ.

**Ví dụ bằng cURL:**
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-actual-openai-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {
        "role": "user",
        "content": "Tại sao bầu trời màu xanh?"
      }
    ]
  }'
```

**3. Tận Hưởng Phép Màu 🎩**
Kết quả hiển thị trên Terminal:
```text
[CACHE MISS] Hash: a8f4b2c1 | Forwarding to OpenAI...
(Thời gian chờ 1,250ms)

# Chạy lại y hệt câu lệnh vừa xong:
[CACHE HIT] Hash: a8f4b2c1 | Time: 2ms | SAVED 💰
(Thời gian chờ chỉ 2ms - Không tốn 1 xu)
```

---

## 🛠 Tech Stack (Công Nghệ)
- **Go 1.22**: Kiến trúc Zero-allocation siêu nhẹ.
- **Gin-gonic**: HTTP routing hiệu năng cao.
- **SQLite (modernc.org)**: Serverless, lưu trữ lịch sử prompt dưới dạng tệp (`aiproxy.db`) hoàn toàn bằng Go thuần túy (không dính CGO).

## 📜 License
Giấy phép MIT. Sử dụng miễn phí 100% cho mọi dự án AI của bạn để tối ưu hóa hóa đơn API!
