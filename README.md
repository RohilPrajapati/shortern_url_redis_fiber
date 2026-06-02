# 🔗 URL Shortener
 
A fast, lightweight URL shortening service built with **Go + Fiber** and **Redis**, featuring rate limiting, custom short codes, click analytics, and persistent storage.

---

## 🚀 Key Features

* **Dual-Database Isolation Engine**: Utilizes separate Redis database namespaces to isolate critical system operations from core business metrics (`DB 0` for Key-to-URL mappings, `DB 1` for API Rate-Limiting thresholds).
* **Rate Limiting**: Protects your system from brute-force abuses via client IP profiling with automated window sliding and dynamic tracking.
* **Dockerized Orchestration**: Instant local container initialization combining multi-stage Go compilation with localized Redis storage persistence.

---

## 🛠️ System Architecture Diagram

```text
                         +-------------------+
                         |  Go / Fiber API   |
                         +---------+---------+
                                   |
            +----------------------+----------------------+
            |                                             |
     POST /api/v1 (Shorten)                         GET /:url (Resolve)
            |                                             |
            v                                             v
   +-----------------+                           +-----------------+
   |   Redis DB 1    |                           |   Redis DB 0    |
   | (Check Quota)   |                           | (Fetch Long URL)|
   +-----------------+                           +-----------------+
            |                                             |
            v                                             v
   +-----------------+                           +-----------------+
   |   Redis DB 0    |                           |   Redis DB 1    |
   | (Store mapping) |                           | (Incr Analytics)|
   +-----------------+                           +-----------------+
```

## 📁 Project Directory Layout
```
shortern_url_redis_fiber/
├── .data/
│   └── dump.rdb          # Redis backup file
├── database/
│   └── database.go       # Configures connections to Redis
├── helpers/
│   └── helpers.go        # URL cleansing & validation utilities
├── routes/
│   ├── shorten.go        # Shorten Handler: Core business & rate limit logic
│   ├── resolve.go        # Resolve Handler: Key check & redirection
│   └── routes_test.go    # Test Suite: Validates routes
├── .env                  # Configuration variables environment file
├── docker-compose.yml    # Multicontainer container orchestrator
├── main.go               # Application entry point binding Fiber & routes
└── README.md             # Project roadmap and documentation
```

## ⚙️ Environment Configuration
```bash
DB_ADDR="db:6379"
DB_PASS=""
APP_PORT=":3000"
DOMAIN="localhost:3000"
API_QUOTA=10
REDIS_DB_RATE=1
REDIS_DB_URL=0
```

---
 
## 🏎️ Getting Started (Local Deployment)
 
### 1. Build and Run Containers
 
```bash
docker compose up --build
```
 
This downloads dependencies, builds the Go binary, configures the network, and binds the server to your configured port.
 
> Server available at: **http://localhost:3000**
 
### 2. Verify Database State via CLI
 
```bash
# Check remaining rate-limit quota for a Docker gateway IP:
docker compose exec db redis-cli -n 1 GET 172.21.0.1
 
# Check total clicks for a custom short link named "mygit":
docker compose exec db redis-cli -n 1 GET mygit_clicks
```
 
---
 
## 🔍 API Reference
 
### `POST /api/v1` — Shorten a URL
 
**Headers:** `Content-Type: application/json`
 
**Request Body:**
 
```json
{
  "url": "https://github.com/rohilprajapati",
  "short": "mygit",
  "expiry": 24
}
```
 
> Leave `"short"` blank to auto-generate a random 6-character identifier.
 
**Response `200 OK`:**
 
```json
{
  "url": "https://github.com/rohilprajapati",
  "short": "localhost:3000/mygit",
  "expiry": 24,
  "rate_limit": 9,
  "rate_limit_reset": 18000
}
```

**Error Responses:**
 
| Status | Condition | Response Body |
|--------|-----------|---------------|
| `400 Bad Request` | Request body is not valid JSON | `{"error": "Cannot parse JSON"}` |
| `400 Bad Request` | `url` field is not a valid URL | `{"error": "Invalid URL"}` |
| `403 Forbidden` | Custom short code is already taken | `{"error": "Your custom short is already use"}` |
| `429 Too Many Requests` | IP has exceeded the API quota; resets after the window | `{"error": "Rate limit exceeded", "rate_limit_reset": <minutes_remaining>}` |
| `400 Bad Request` | URL resolves to the shortener's own domain (loop prevention) | `{"error": "URL is not Safe"}` |
| `500 Internal Server Error` | Redis write to DB 0 failed | `{"error": "Something went wrong, please try again later"}` |

---
 
### `GET /:url` — Resolve and Redirect
 
**Example:** `GET /mygit`
 
Looks up the short code in DB 0 and issues a **301 Moved Permanently** redirect to the original URL. Also increments a global `counter` key in DB 1 on each successful resolution.
 
**Error Responses:**
 
| Status | Condition | Response Body |
|--------|-----------|---------------|
| `404 Not Found` | Short code does not exist or has expired | `{"error": "Short not found"}` |
| `500 Internal Server Error` | Redis read from DB 0 failed | `{"error": "Something went wrong, please try again later"}` |
 
---
 
## 💾 Storage & Persistence
 
Redis RDB snapshots persist in-memory data to disk asynchronously via OS-level `fork()`, writing to a `dump.rdb` file inside the `redis_data` volume.
 
```bash
# View the timestamp of the last successful snapshot:
docker compose exec db redis-cli LASTSAVE
 
# Trigger an immediate background snapshot:
docker compose exec db redis-cli BGSAVE
```
 
---
 
## 🧪 Running Tests
 
The test suite covers custom short code registration, token deduplication, rate-limit sliding windows, and expiry math. Tests automatically target **DB 14 & DB 15** to keep live data safe.
 
Ensure Docker services are running, then:
 
```bash
go test -v ./routes/...
```