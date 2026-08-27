package limiter

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := New(2, 50*time.Millisecond)

	if !rl.Allow() {
		t.Fatal("expected first request to be allowed")
	}
	if !rl.Allow() {
		t.Fatal("expected second request to be allowed")
	}
	if rl.Allow() {
		t.Fatal("expected third request to be rejected (limit is 2)")
	}

	// After the window elapses, requests should be allowed again.
	time.Sleep(60 * time.Millisecond)
	if !rl.Allow() {
		t.Fatal("expected request to be allowed after window elapsed")
	}
}

func TestMultiLimiter_AllowPerKey(t *testing.T) {
	ml := NewMultiLimiter(1, 50*time.Millisecond)

	if !ml.Allow("a") {
		t.Fatal("expected first request for key 'a' to be allowed")
	}
	if ml.Allow("a") {
		t.Fatal("expected second request for key 'a' to be rejected (limit is 1)")
	}
	// A different key has its own independent limit.
	if !ml.Allow("b") {
		t.Fatal("expected first request for key 'b' to be allowed")
	}

	if got := ml.Len(); got != 2 {
		t.Fatalf("expected 2 tracked keys, got %d", got)
	}
}
