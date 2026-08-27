package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"rate-limiter/limiter"
)

func main() {
	// Swap this line to go back to in-memory: limiter.NewMultiLimiter(5, 10*time.Second)
	// Set REDIS_URL to your Upstash (or other) connection string, e.g.
	// redis://default:password@host.upstash.io:6379
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("REDIS_URL environment variable is not set")
	}
	rl := limiter.NewRedisLimiter(redisURL, 5, 10*time.Second)
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello! request allowed.")
	})

	// Middleware now takes any Limiter — RedisLimiter or MultiLimiter both work
	http.Handle("/", limiter.Middleware(rl, helloHandler))

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
