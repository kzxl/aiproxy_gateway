package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

var db *sql.DB

// Cost per 1 Token (Input + Output blended roughly) for MVP
var ModelPricing = map[string]float64{
	"gpt-3.5-turbo": 0.0000015,
	"gpt-4o":        0.000005,
	"gpt-4-turbo":   0.00001,
	"claude-3-haiku-20240307": 0.00000125,
}

type OpenAIResponse struct {
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite", "aiproxy.db")
	if err != nil {
		log.Fatal(err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS prompt_cache (
		hash_id TEXT PRIMARY KEY,
		response_body TEXT,
		model TEXT,
		saved_usd REAL,
		created_at INTEGER
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("SQLite Database initialized: aiproxy.db")
}

func calculateHash(body []byte) string {
	hasher := sha256.New()
	hasher.Write(body)
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCache(hashID string) (string, float64, bool) {
	var responseBody string
	var savedUsd float64
	query := `SELECT response_body, saved_usd FROM prompt_cache WHERE hash_id = ?`
	err := db.QueryRow(query, hashID).Scan(&responseBody, &savedUsd)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", 0, false
		}
		log.Printf("DB Error: %v", err)
		return "", 0, false
	}
	return responseBody, savedUsd, true
}

func saveCache(hashID string, responseBody string) {
	// Parse usage tokens
	var oaiResp OpenAIResponse
	json.Unmarshal([]byte(responseBody), &oaiResp)

	costPerToken, exists := ModelPricing[oaiResp.Model]
	if !exists {
		costPerToken = 0.000002 // default fallback
	}
	savedUsd := float64(oaiResp.Usage.TotalTokens) * costPerToken

	query := `INSERT INTO prompt_cache (hash_id, response_body, model, saved_usd, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, hashID, responseBody, oaiResp.Model, savedUsd, time.Now().Unix())
	if err != nil {
		log.Printf("Failed to cache prompt: %v", err)
	}
}

func proxyToOpenAI(c *gin.Context) {
	// 1. Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot read body"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// 2. Generate Hash
	hashID := calculateHash(bodyBytes)

	// 3. Search Cache
	start := time.Now()
	cachedResponse, savedUsd, found := getCache(hashID)
	if found {
		duration := time.Since(start).Milliseconds()
		fmt.Printf("[\033[32mCACHE HIT\033[0m] Hash: %s | Time: %dms | \033[32mSAVED $%.6f 💰\033[0m\n", hashID[:8], duration, savedUsd)
		c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
		return
	}

	// 4. Cache Miss -> Forward to OpenAI
	fmt.Printf("[\033[33mCACHE MISS\033[0m] Hash: %s | Forwarding to OpenAI...\n", hashID[:8])
	
	authHeader := c.GetHeader("Authorization")
	if authHeader == "Bearer sk-mock" {
		time.Sleep(1200 * time.Millisecond) // Simulate OpenAI Latency
		mockResponse := `{"id":"chatcmpl-mock","object":"chat.completion","created":1700000000,"model":"gpt-3.5-turbo","choices":[{"index":0,"message":{"role":"assistant","content":"Because of Rayleigh scattering!"},"finish_reason":"stop"}],"usage":{"prompt_tokens":14,"completion_tokens":6,"total_tokens":20}}`
		saveCache(hashID, mockResponse)
		c.Data(http.StatusOK, "application/json", []byte(mockResponse))
		return
	}

	proxyReq, err := http.NewRequest(c.Request.Method, "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}

	// Copy Headers (especially Authorization)
	for key, values := range c.Request.Header {
		if strings.ToLower(key) != "host" {
			proxyReq.Header[key] = values
		}
	}

	client := &http.Client{Timeout: 60 * time.Second}
	proxyResp, err := client.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "OpenAI API Timeout or Error"})
		return
	}
	defer proxyResp.Body.Close()

	// 5. Read OpenAI Response
	respBodyBytes, err := io.ReadAll(proxyResp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read OpenAI response"})
		return
	}

	// 6. Save to Cache ONLY if 200 OK
	if proxyResp.StatusCode == http.StatusOK {
		saveCache(hashID, string(respBodyBytes))
		// Optional: parse JSON to extract Usage Total_Tokens to calculate exact $ savings
	}

	// 7. Copy Headers back to Client
	for key, values := range proxyResp.Header {
		c.Header(key, strings.Join(values, ", "))
	}
	c.Data(proxyResp.StatusCode, "application/json", respBodyBytes)
}

func main() {
	// Custom colorful ASCII banner
	fmt.Println("\033[36m")
	fmt.Println("    ___    ____   ____                       ")
	fmt.Println("   /   |  /  _/  / __ \\_________  _  __  __ ")
	fmt.Println("  / /| |  / /   / /_/ / ___/ __ \\| |/_/ / / /")
	fmt.Println(" / ___ |_/ /   / _, _/ /  / /_/ />  <  / /_/ / ")
	fmt.Println("/_/  |_/___/  /_/ |_/_/   \\____/_/|_|  \\__, /  ")
	fmt.Println("                                      /____/   ")
	fmt.Println("\033[0m")
	fmt.Println("🚀 AI Proxy Gateway: Listening on :8080")
	fmt.Println("🔗 Point your AI SDK base URL to http://localhost:8080")
	fmt.Println("-----------------------------------------------------")

	initDB()
	defer db.Close()

	// Disable Gin debug logging for cleaner terminal output
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.POST("/v1/chat/completions", proxyToOpenAI)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "OK", "uptime": time.Now().Unix()})
	})

	// Dashboard UI
	r.GET("/admin", serveDashboard)
	r.GET("/api/stats", serveStats)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func serveStats(c *gin.Context) {
	var totalRequests int
	var totalSavedUsd float64

	db.QueryRow(`SELECT count(*), COALESCE(sum(saved_usd), 0) FROM prompt_cache`).Scan(&totalRequests, &totalSavedUsd)

	c.JSON(200, gin.H{
		"total_cached_prompts": totalRequests,
		"total_saved_usd":      fmt.Sprintf("%.5f", totalSavedUsd),
	})
}

func serveDashboard(c *gin.Context) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>AI Proxy - Financial Tracker</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-900 text-white min-h-screen flex items-center justify-center font-sans">
    <div class="bg-gray-800 p-10 rounded-2xl shadow-2xl w-full max-w-lg text-center border border-gray-700">
        <h1 class="text-4xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-green-400 to-blue-500 mb-6">
            🤖 AI Cash Saver
        </h1>
        <p class="text-gray-400 mb-8">Tracking API cost savings via intelligent local hashing.</p>
        
        <div class="grid grid-cols-2 gap-6">
            <div class="bg-gray-700 p-6 rounded-xl">
                <p class="text-sm uppercase tracking-wide text-gray-400">Total Saves</p>
                <div id="req-count" class="text-4xl font-black text-white mt-2">0</div>
            </div>
            <div class="bg-gray-700 p-6 rounded-xl border border-green-500/30 relative overflow-hidden">
                <div class="absolute inset-0 bg-green-500/10 mix-blend-overlay"></div>
                <p class="text-sm uppercase tracking-wide text-green-400 font-bold relative z-10">USD Saved 💰</p>
                <div id="usd-count" class="text-4xl font-black text-green-400 mt-2 relative z-10">$0.00</div>
            </div>
        </div>

        <button onclick="fetchStats()" class="mt-8 px-6 py-3 bg-blue-600 hover:bg-blue-500 transition-colors rounded-lg font-bold w-full">
            Refresh Dashboard
        </button>
    </div>

    <script>
        async function fetchStats() {
            try {
                const res = await fetch('/api/stats');
                const data = await res.json();
                document.getElementById('req-count').innerText = data.total_cached_prompts;
                document.getElementById('usd-count').innerText = '$' + data.total_saved_usd;
            } catch (e) { console.error("API Error", e); }
        }
        fetchStats();
        setInterval(fetchStats, 5000);
    </script>
</body>
</html>`

	c.Data(http.StatusOK, "text/html", []byte(html))
}
