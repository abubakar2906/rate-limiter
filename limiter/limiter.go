package limiter

import (
	"sync"
	"time"
)

// RateLimiter is our struct — it just holds data
type RateLimiter struct {
	mu       sync.Mutex    // the toilet lock
	requests []time.Time   // list of timestamps in the current window
	limit    int           // max requests allowed
	window   time.Duration // how long the window is
}

// New is Go's version of a constructor — returns a pointer to a fresh RateLimiter
func New(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		window: window,
	}
}

// Allow checks if a request can go through
// (r *RateLimiter) means: this method belongs to RateLimiter, and r is a pointer so changes stick
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()         // lock — one request at a time through here
	defer r.mu.Unlock() // unlock when this function exits, no matter what

	now := time.Now()
	windowStart := now.Add(-r.window) // e.g. if window is 1s, this is 1 second ago

	// Go through our timestamp list and keep only the ones inside the window
	var valid []time.Time
	for _, t := range r.requests {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	r.requests = valid // replace old list with cleaned-up list

	// If we're already at the limit, reject
	if len(r.requests) >= r.limit {
		return false
	}

	// Otherwise, record this request and allow it
	r.requests = append(r.requests, now)
	return true
}
