package limiter

import (
	"sync"
	"time"
)

type MultiLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*RateLimiter
	lastSeen map[string]time.Time // tracks when each key was last used
	done     chan struct{}         // the stop signal channel
	limit    int
	window   time.Duration
}

func NewMultiLimiter(limit int, window time.Duration) *MultiLimiter {
	return &MultiLimiter{
		limiters: make(map[string]*RateLimiter),
		lastSeen: make(map[string]time.Time),
		done:     make(chan struct{}),
		limit:    limit,
		window:   window,
	}
}

func (m *MultiLimiter) Allow(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	rl, exists := m.limiters[key]
	if !exists {
		rl = New(m.limit, m.window)
		m.limiters[key] = rl
	}

	// update the last time this key was seen
	m.lastSeen[key] = time.Now()

	return rl.Allow()
}

// Len returns how many keys are currently in the limiter — useful for testing
func (m *MultiLimiter) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.limiters)
}

// StartCleanup launches a background goroutine that removes idle keys
// interval — how often to run the sweep
// maxIdle  — how long a key can go unused before being deleted
func (m *MultiLimiter) StartCleanup(interval, maxIdle time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				m.cleanup(maxIdle)
			case <-m.done:
				ticker.Stop()
				return // exits the goroutine
			}
		}
	}()
}

// Stop signals the background goroutine to exit
func (m *MultiLimiter) Stop() {
	close(m.done)
}

// cleanup sweeps through all keys and removes ones idle longer than maxIdle
func (m *MultiLimiter) cleanup(maxIdle time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for key, lastSeen := range m.lastSeen {
		if now.Sub(lastSeen) > maxIdle {
			delete(m.limiters, key)
			delete(m.lastSeen, key)
		}
	}
}