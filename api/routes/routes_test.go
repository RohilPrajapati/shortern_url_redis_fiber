package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/rohilprajapati/shortern_url_redis_fiber/database"
)

func setupTestApp() *fiber.App {
	app := fiber.New()
	app.Post("/api/v1", ShortenURL)
	return app
}

func clearRedisDBs() {
	dbURL, _ := strconv.Atoi(os.Getenv("REDIS_DB_URL"))
	dbRate, _ := strconv.Atoi(os.Getenv("REDIS_DB_RATE"))

	r0 := database.CreateClient(dbURL)
	defer r0.Close()
	r0.FlushDB(database.Ctx)

	r2 := database.CreateClient(dbRate)
	defer r2.Close()
	r2.FlushDB(database.Ctx)
}

func TestShortenURL(t *testing.T) {
	os.Setenv("REDIS_DB_URL", "14")
	os.Setenv("REDIS_DB_RATE", "15")
	os.Setenv("API_QUOTA", "2")
	os.Setenv("DOMAIN", "localhost:3000")

	app := setupTestApp()
	// 1. Check for custom_short (PASSING)

	t.Run("check for custom_short", func(t *testing.T) {
		clearRedisDBs()

		reqBody := request{
			URL:         "https://github.com/rohilprajapati",
			CustomShort: "mygit",
			Expiry:      24,
		}
		jsonReq, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1", bytes.NewBuffer(jsonReq))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "127.0.0.1:1234"

		resp, _ := app.Test(req, -1)

		var body response
		_ = json.NewDecoder(resp.Body).Decode(&body)

		expectedShort := "localhost:3000/mygit"
		if body.CustomShort != expectedShort {
			t.Errorf("Expected custom short %q, got %q", expectedShort, body.CustomShort)
		}
	})

	// 2. Check for repeat_custom_short

	t.Run("check for repeat_custom_short", func(t *testing.T) {
		clearRedisDBs()

		reqBody := request{
			URL:         "https://google.com",
			CustomShort: "unique",
			Expiry:      24,
		}
		jsonReq, _ := json.Marshal(reqBody)

		req1 := httptest.NewRequest("POST", "/api/v1", bytes.NewBuffer(jsonReq))
		req1.Header.Set("Content-Type", "application/json")
		req1.RemoteAddr = "127.0.0.1:1234"
		_, _ = app.Test(req1, -1)

		req2 := httptest.NewRequest("POST", "/api/v1", bytes.NewBuffer(jsonReq))
		req2.Header.Set("Content-Type", "application/json")
		req2.RemoteAddr = "127.0.0.1:1234"

		resp, _ := app.Test(req2, -1)

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Expected status 403 Forbidden, got %d", resp.StatusCode)
		}

		bodyBytes, _ := io.ReadAll(resp.Body)
		if !bytes.Contains(bodyBytes, []byte("already use")) {
			t.Errorf("Expected duplication message, got: %s", string(bodyBytes))
		}
	})

	// 3. Check rate limit

	t.Run("check rate limit", func(t *testing.T) {
		clearRedisDBs()
		os.Setenv("API_QUOTA", "1")

		reqBody := request{URL: "https://google.com"}
		jsonReq, _ := json.Marshal(reqBody)

		req1 := httptest.NewRequest("POST", "/api/v1", bytes.NewBuffer(jsonReq))
		req1.Header.Set("Content-Type", "application/json")
		req1.RemoteAddr = "192.168.1.50:5555"
		resp1, _ := app.Test(req1, -1)

		var res1 response
		_ = json.NewDecoder(resp1.Body).Decode(&res1)
		if res1.XRateRemaining != 0 {
			t.Errorf("Expected remaining rate limits to hit 0, got %d", res1.XRateRemaining)
		}

		req2 := httptest.NewRequest("POST", "/api/v1", bytes.NewBuffer(jsonReq))
		req2.Header.Set("Content-Type", "application/json")
		req2.RemoteAddr = "192.168.1.50:5555"
		resp2, _ := app.Test(req2, -1)

		if resp2.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("Expected status 503, got %d", resp2.StatusCode)
		}
	})

	// 4. Check rate limit reset
	t.Run("check rate limit reset", func(t *testing.T) {
		clearRedisDBs()
		os.Setenv("API_QUOTA", "10") // Set standard quota context

		reqBody := request{URL: "https://golang.org"}
		jsonReq, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1", bytes.NewBuffer(jsonReq))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "10.0.0.1:9999"

		resp, _ := app.Test(req, -1)

		var bodyMap map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&bodyMap)


		resetVal, exists := bodyMap["rate_limit_reset"]
		if !exists {
			t.Fatalf("Response JSON missing expected 'rate_limit_reset'. Got body: %v", bodyMap)
		}

		minutesLeft := int(resetVal.(float64))
		if minutesLeft < 0 {
			t.Errorf("Expected valid reset time duration metric, got: %d", minutesLeft)
		}
	})
}
