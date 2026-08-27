package limiter

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisLimiter struct {
	client *redis.Client
	limit  int
	window time.Duration
	script *redis.Script // the Lua script, pre-loaded
}

func NewRedisLimiter(redisURL string, limit int, window time.Duration) *RedisLimiter {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		panic("invalid redis URL: " + err.Error())
	}

	// Fix for Windows TLS verification issues
	opt.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	client := redis.NewClient(opt)

	script := redis.NewScript(`
		local key      = KEYS[1]
		local now_ms   = tonumber(ARGV[1])
		local now_ns   = ARGV[2]
		local window   = tonumber(ARGV[3])
		local limit    = tonumber(ARGV[4])

		redis.call('ZREMRANGEBYSCORE', key, 0, now_ms - window)

		local count = redis.call('ZCARD', key)

		if count < limit then
			redis.call('ZADD', key, now_ms, now_ns)
			redis.call('PEXPIRE', key, window)
			return 1
		end

		return 0
	`)

	return &RedisLimiter{
		client: client,
		limit:  limit,
		window: window,
		script: script,
	}
}

func (r *RedisLimiter) Allow(key string) bool {
	ctx := context.Background()

	nowMs := time.Now().UnixMilli() // milliseconds — used as the score
	nowNs := time.Now().UnixNano()  // nanoseconds — used as the unique member
	windowMs := r.window.Milliseconds()

	// Run the Lua script on Redis
	// []string{key} — the KEYS array
	// nowMs, nowNs, windowMs, r.limit — the ARGV array
	result, err := r.script.Run(ctx, r.client, []string{key}, nowMs, nowNs, windowMs, r.limit).Int()
	if err != nil {
		fmt.Println("Redis error:", err) // fail-open: log and allow the request through
		return true
	}

	return result == 1
}
