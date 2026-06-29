package main

import (
	"fmt"
	"net/http"
	"time"

	"rate-limiter/limiter"
)

func main() {
	// Swap this line to go back to in-memory: limiter.NewMultiLimiter(5, 10*time.Second)
	// paste your Upstash URL here
	rl := limiter.NewRedisLimiter("redis://default:gQAAAAAAAiIyAAIgcDJjYjkzZmYyZjZjNDQ0YTIyYjU3NDMyMzI1YTJlYWE2MA@charmed-prawn-139826.upstash.io:6379", 5, 10*time.Second)
	helloHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello! request allowed.")
	})

	// Middleware now takes any Limiter — RedisLimiter or MultiLimiter both work
	http.Handle("/", limiter.Middleware(rl, helloHandler))

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}
