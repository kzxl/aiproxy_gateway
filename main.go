package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

func getCache(hashID string) (string, bool) {
	var responseBody string
	query := `SELECT response_body FROM prompt_cache WHERE hash_id = ?`
	err := db.QueryRow(query, hashID).Scan(&responseBody)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		log.Printf("DB Error: %v", err)
		return "", false
	}
	return responseBody, true
}

func saveCache(hashID string, responseBody string) {
	query := `INSERT INTO prompt_cache (hash_id, response_body, created_at) VALUES (?, ?, ?)`
	_, err := db.Exec(query, hashID, responseBody, time.Now().Unix())
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
	cachedResponse, found := getCache(hashID)
	if found {
		duration := time.Since(start).Milliseconds()
		fmt.Printf("[\033[32mCACHE HIT\033[0m] Hash: %s | Time: %dms | \033[32mSAVED 💰\033[0m\n", hashID[:8], duration)
		c.Data(http.StatusOK, "application/json", []byte(cachedResponse))
		return
	}

	// 4. Cache Miss -> Forward to OpenAI (Or MOCK if key is mock)
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
