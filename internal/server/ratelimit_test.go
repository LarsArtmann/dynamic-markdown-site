package server

import (
	"sync"
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(3, time.Second)
	defer rl.Stop()

	// First request should be allowed
	if !rl.checkRateLimit("192.168.1.1") {
		t.Error("first request should be allowed")
	}

	// Second request should be allowed
	if !rl.checkRateLimit("192.168.1.1") {
		t.Error("second request should be allowed")
	}

	// Third request should be allowed
	if !rl.checkRateLimit("192.168.1.1") {
		t.Error("third request should be allowed")
	}

	// Fourth request should be denied (over limit)
	if rl.checkRateLimit("192.168.1.1") {
		t.Error("fourth request should be denied")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(2, time.Second)
	defer rl.Stop()

	// IP1 should be allowed twice
	if !rl.checkRateLimit("192.168.1.1") {
		t.Error("IP1 first request should be allowed")
	}

	if !rl.checkRateLimit("192.168.1.1") {
		t.Error("IP1 second request should be allowed")
	}

	// IP2 should have its own limit
	if !rl.checkRateLimit("192.168.1.2") {
		t.Error("IP2 first request should be allowed")
	}

	if !rl.checkRateLimit("192.168.1.2") {
		t.Error("IP2 second request should be allowed")
	}

	// Both IPs should now be at their limit
	if rl.checkRateLimit("192.168.1.1") {
		t.Error("IP1 over limit should be denied")
	}

	if rl.checkRateLimit("192.168.1.2") {
		t.Error("IP2 over limit should be denied")
	}
}

func TestRateLimiter_Concurrent(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(100, time.Second)
	defer rl.Stop()

	var wg sync.WaitGroup

	allowed := make(chan bool, 200)

	for range 200 {
		wg.Go(func() {
			allowed <- rl.checkRateLimit("192.168.1.1")
		})
	}

	wg.Wait()
	close(allowed)

	allowedCount := 0

	for a := range allowed {
		if a {
			allowedCount++
		}
	}

	// Should have exactly 100 allowed requests
	if allowedCount != 100 {
		t.Errorf("expected 100 allowed requests, got %d", allowedCount)
	}
}
