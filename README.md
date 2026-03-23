# 🤖 AI Proxy Gateway (The Cost Saver)

<div align="center">

![Golang](https://img.shields.io/badge/Golang-1.22-00ADD8?logo=go)
![SQLite](https://img.shields.io/badge/SQLite-3-003B57?logo=sqlite)
![Gin](https://img.shields.io/badge/Gin-1.9-blue)
![License: MIT](https://img.shields.io/badge/License-MIT-green)

A Blazing Fast Reverse Proxy for LLM APIs (OpenAI / Anthropic / Local models). **Caches identical prompts** to save you 100% API costs during testing and development. Drops response latency from 3000ms down to **< 5ms**.

</div>

---

## 💡 The Problem
During AI app development, you often send the *same prompt* hundreds of times to tune UI or fix backend bugs. 
Every time you hit `api.openai.com`, you are **losing money** and **waiting 2-5 seconds** for a response you already know.

## 🚀 The Solution: AI Proxy Gateway
Simply change your SDK's Base URL to `http://localhost:8080`.
This Gateway intercepts your `POST /v1/chat/completions`:
1. Hashes your entire JSON body.
2. If it's a **Cache Hit**, returns the exact previous response instantly (Cost: $0.00, Time: 1ms).
3. If it's a **Cache Miss**, forwards to OpenAI, returns the response, and silently caches it in a high-speed local SQLite database.

---

## ⚡ Quick Start

### 1. Build & Run
```bash
go mod tidy
go build main.go

# Run the proxy on port 8080
./main
```

### 2. Connect Your App
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

### 3. Watch the Magic 🎩
Terminal Output:
```text
[CACHE MISS] Hash: a8f4b2c1 | Forwarding to OpenAI...
(Request took 1,250ms)

# Run the exact same request again:
[CACHE HIT] Hash: a8f4b2c1 | Time: 2ms | SAVED 💰
(Request took 2ms)
```

---

## 🛠 Tech Stack
- **Go 1.22**: Zero-allocation architecture.
- **Gin-gonic**: High-performance HTTP routing.
- **SQLite (modernc.org)**: Serverless, file-based prompt database (`aiproxy.db`) with zero CGO dependencies.

## 📜 License
MIT License. Free to use for your AI projects to save big on bills!
