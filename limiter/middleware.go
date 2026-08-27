package limiter

import (
	"fmt"
	"net"
	"net/http"
)

// Limiter is anything with an Allow method
// MultiLimiter satisfies this. RedisLimiter will too.
// Neither of them has to declare it — Go figures it out automatically.
type Limiter interface {
	Allow(key string) bool
}

// Middleware is now a standalone function that works with any Limiter
func Middleware(l Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !l.Allow(ip) {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintln(w, "rate limit exceeded, slow down")
			return
		}

		next.ServeHTTP(w, r)
	})
}
