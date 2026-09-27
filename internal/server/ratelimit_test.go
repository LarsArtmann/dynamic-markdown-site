package server

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func newBurstOnlyLimiter(burst int) *rateLimiter {
	return newRateLimiter(burst, time.Hour)
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Parallel()

	rl := newBurstOnlyLimiter(3)
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

	rl := newBurstOnlyLimiter(2)
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

	rl := newBurstOnlyLimiter(100)
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

func TestRateLimiter_EvictsIdleVisitors(t *testing.T) {
	t.Parallel()

	rl := newRateLimiterWithSweepInterval(1, 20*time.Millisecond, time.Hour)
	defer rl.Stop()

	rl.mu.Lock()
	rl.visitors["10.0.0.1"] = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst), lastSeen: time.Now()}
	rl.visitors["10.0.0.2"] = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst), lastSeen: time.Now().Add(-rl.ttl - time.Second)}
	rl.mu.Unlock()

	rl.evictIdle(time.Now())

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if _, exists := rl.visitors["10.0.0.1"]; !exists {
		t.Error("recently seen visitor should survive eviction")
	}

	if _, exists := rl.visitors["10.0.0.2"]; exists {
		t.Error("stale visitor should be evicted")
	}
}

func TestRateLimiter_LastSeenRefreshed(t *testing.T) {
	t.Parallel()

	rl := newRateLimiterWithSweepInterval(1, 20*time.Millisecond, time.Hour)
	defer rl.Stop()

	rl.checkRateLimit("10.1.1.1")
	rl.mu.Lock()
	firstSeen := rl.visitors["10.1.1.1"].lastSeen
	rl.mu.Unlock()

	rl.checkRateLimit("10.1.1.1")
	rl.mu.Lock()
	secondSeen := rl.visitors["10.1.1.1"].lastSeen
	rl.mu.Unlock()

	if !secondSeen.After(firstSeen) {
		t.Error("lastSeen should refresh on every request")
	}
}

func TestRateLimiter_SweepGoroutineEvicts(t *testing.T) {
	t.Parallel()

	rl := newRateLimiterWithSweepInterval(1, 20*time.Millisecond, 5*time.Millisecond)
	defer rl.Stop()

	rl.checkRateLimit("10.2.0.1")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		rl.mu.Lock()
		size := len(rl.visitors)
		rl.mu.Unlock()

		if size == 0 {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Error("sweep goroutine did not evict the idle visitor within 2s")
}

func TestRateLimiter_StopIsIdempotent(t *testing.T) {
	t.Parallel()

	rl := newRateLimiterWithSweepInterval(1, time.Hour, time.Minute)
	rl.checkRateLimit("10.3.0.1")
	rl.Stop()
	rl.Stop()
}

func TestRateLimiter_ConcurrentWithSweep(t *testing.T) {
	t.Parallel()

	rl := newRateLimiterWithSweepInterval(100, time.Hour, time.Millisecond)
	defer rl.Stop()

	var wg sync.WaitGroup

	for i := range 100 {
		wg.Go(func() {
			rl.checkRateLimit(fmt.Sprintf("10.4.%d.%d", i/10, i%10))
		})
	}

	wg.Wait()
}

func TestRateLimiter_RefillsTokens(t *testing.T) {
	t.Parallel()

	rl := newRateLimiter(2, 100*time.Millisecond)
	defer rl.Stop()

	if !rl.checkRateLimit("10.5.0.1") || !rl.checkRateLimit("10.5.0.1") {
		t.Fatal("bucket should start full: two requests allowed")
	}

	if rl.checkRateLimit("10.5.0.1") {
		t.Fatal("empty bucket should deny the next request")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if rl.checkRateLimit("10.5.0.1") {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Error("token refill did not admit a request within 2s")
}
