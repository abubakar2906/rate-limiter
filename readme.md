# Distributed Rate Limiter

A production-grade HTTP rate limiter in Go with a sliding window algorithm. Supports both in-memory (single server) and Redis-backed (distributed, multi-server) operation.

Built as a Go learning project — going from zero Go experience to a working distributed system.

---

## Features

- **Sliding window algorithm** — fairer than fixed windows, no boundary bursts
- **Per-key limiting** — each IP (or any key) gets its own independent window
- **Two backends** — swap between in-memory and Redis with one line
- **HTTP middleware** — drop into any `net/http` server
- **Atomic Redis operations** — Lua scripting prevents race conditions under concurrent load
- **Background cleanup** — idle keys are automatically evicted (in-memory mode)
- **Fail-open** — if Redis goes down, requests are allowed rather than taking your app down

---

## Project Structure

```
rate-limiter/
├── go.mod
├── main.go
└── limiter/
    ├── limiter.go      # core sliding window logic (single key)
    ├── multi.go        # multi-key in-memory limiter + background cleanup
    ├── middleware.go   # net/http middleware + Limiter interface
    └── redis.go        # Redis-backed distributed limiter
```

---

## Prerequisites

- Go 1.25+ (see `go.mod`)
- A Redis instance (for distributed mode)
  - [Upstash](https://upstash.com) — free cloud Redis, no installation needed
  - Or Docker: `docker run -d -p 6379:6379 redis`

---

## Getting Started

**1 — Clone and install dependencies**

```bash
git clone https://github.com/abubakar2906/rate-limiter
cd rate-limiter
go mod tidy
```

**2 — Run in-memory mode (no Redis needed)**

In `main.go`, use `NewMultiLimiter`:

```go
rl := limiter.NewMultiLimiter(5, 10*time.Second)
```

```bash
go run .
```

**3 — Run in distributed mode (Redis)**

In `main.go`, use `NewRedisLimiter`:

```go
rl := limiter.NewRedisLimiter("rediss://default:password@host.upstash.io:6379", 5, 10*time.Second)
```

```bash
go run .
```

---

## Usage

### In-Memory (Single Server)

```go
import (
    "net/http"
    "time"
    "rate-limiter/limiter"
)

func main() {
    // 5 requests per 10 seconds per IP
    ml := limiter.NewMultiLimiter(5, 10*time.Second)

    // Start background goroutine — sweeps every minute, evicts keys idle for 5 minutes
    ml.StartCleanup(1*time.Minute, 5*time.Minute)
    defer ml.Stop()

    myHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("hello!"))
    })

    http.Handle("/", limiter.Middleware(ml, myHandler))
    http.ListenAndServe(":8080", nil)
}
```

### Distributed (Redis)

```go
// Drop-in replacement for NewMultiLimiter — same Middleware works with both
rl := limiter.NewRedisLimiter("rediss://default:password@host.upstash.io:6379", 5, 10*time.Second)

http.Handle("/", limiter.Middleware(rl, myHandler))
```

### Custom Key (e.g. User ID instead of IP)

The `Limiter` interface has one method — `Allow(key string) bool`. You can call it with any string as the key:

```go
// In your own handler or middleware
userID := r.Header.Get("X-User-ID")
if !myLimiter.Allow(userID) {
    http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
    return
}
```

---

## How It Works

### Sliding Window Algorithm

Every request is stored as a timestamp. On each new request, timestamps older than the window are evicted, and the remaining count is checked against the limit. The window "slides" with every request — unlike a fixed window, there are no boundary bursts.

```
Timeline:  ──────────────────────────────────────────▶
Requests:        ● ● ●     ●       ●
Window:                   [──── 1s ────]   ← slides with every request
                                       ▲ now
```

### In-Memory Mode

Each key gets its own `RateLimiter` (a slice of timestamps protected by a `sync.Mutex`). A `sync.RWMutex` protects the key map — concurrent reads don't block each other, only writes do. A background goroutine sweeps idle keys on a configurable interval.

### Redis Mode

Each key maps to a Redis sorted set where every member is a request timestamp (score = milliseconds). On each request, a Lua script atomically:

1. Removes all entries older than the window start (`ZREMRANGEBYSCORE`)
2. Counts what's left (`ZCARD`)
3. If under the limit, records the new request (`ZADD`) and sets a TTL (`PEXPIRE`)

Running this as a Lua script means it executes atomically inside Redis — no race conditions between the count check and the write, even under heavy concurrent load.

### The Limiter Interface

Both backends satisfy the same interface:

```go
type Limiter interface {
    Allow(key string) bool
}
```

The middleware accepts any `Limiter` — swap backends without touching your handler code.

---

## Configuration Reference

### `NewMultiLimiter(limit int, window time.Duration)`

| Parameter | Description | Example |
|---|---|---|
| `limit` | Max requests allowed per window | `5` |
| `window` | Length of the sliding window | `10 * time.Second` |

### `StartCleanup(interval, maxIdle time.Duration)`

| Parameter | Description | Example |
|---|---|---|
| `interval` | How often to run the sweep | `1 * time.Minute` |
| `maxIdle` | How long a key can go unused before eviction | `5 * time.Minute` |

### `NewRedisLimiter(redisURL string, limit int, window time.Duration)`

| Parameter | Description | Example |
|---|---|---|
| `redisURL` | Full Redis connection URL | `"rediss://default:pass@host:6379"` |
| `limit` | Max requests allowed per window | `5` |
| `window` | Length of the sliding window | `10 * time.Second` |

---

## Testing

Fire 7 requests at a server limited to 5 per 10 seconds:

**Mac / Linux:**
```bash
for i in $(seq 1 7); do curl -s http://localhost:8080/; done
```

**Windows (PowerShell):**
```powershell
for ($i = 1; $i -le 7; $i++) { curl -s http://localhost:8080/ }
```

**Expected output:**
```
hello! request allowed.
hello! request allowed.
hello! request allowed.
hello! request allowed.
hello! request allowed.
rate limit exceeded, slow down
rate limit exceeded, slow down
```

---

## Design Decisions

**Why sliding window over fixed window?**
Fixed windows allow burst traffic at window boundaries — a client can send N requests just before the window resets and N more just after, effectively doubling the allowed rate. Sliding windows prevent this entirely.

**Why Lua for Redis?**
Three separate Redis calls (remove → count → add) have a race window between the count read and the write. Two concurrent requests could both read count = 4 under a limit of 5, both add themselves, and both get through — breaking the limit. A Lua script runs atomically: nothing else executes inside Redis between the count and the add.

**Why fail-open on Redis errors?**
If Redis becomes unavailable, fail-closed (blocking all requests) takes your application down with it. Fail-open keeps the app running while you fix Redis — a better tradeoff for most production scenarios. Swap `return true` to `return false` in `redis.go` if your use case requires the opposite.

**Why `RWMutex` in MultiLimiter?**
Most operations on the key map are reads (looking up whether a key exists). A plain `Mutex` would serialise all reads, making them wait in line unnecessarily. `RWMutex` allows concurrent reads — only writes (adding new keys) require exclusive access.

---

## What's Next

This project is part of a two-project Go learning series. The next project is a **Messaging System** — a pub/sub broker using Go channels, goroutines, and fan-out patterns. Everything built here (interfaces, goroutines, channels, select) carries straight over.

---

## License

MIT
