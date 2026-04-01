package server

import (
	"sync"
	"time"

	"github.com/samber/lo"
)

// rateLimiter provides in-memory rate limiting by client IP.
type rateLimiter struct {
	mu          sync.RWMutex
	requests    map[string][]time.Time
	maxRequests int
	window      time.Duration
	cleanupTick time.Duration
	stopCleanup chan struct{}
	stopOnce    sync.Once
}

func newRateLimiter(maxRequests int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests:    make(map[string][]time.Time),
		maxRequests: maxRequests,
		window:      window,
		cleanupTick: window,
		stopCleanup: make(chan struct{}),
	}

	go rl.cleanup()

	return rl
}

// filterValidTimestamps removes expired timestamps from a slice.
func (rl *rateLimiter) filterValidTimestamps(timestamps []time.Time, now time.Time) []time.Time {
	return lo.Filter(timestamps, func(ts time.Time, _ int) bool {
		return now.Sub(ts) < rl.window
	})
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()

			for ip, timestamps := range rl.requests {
				valid := rl.filterValidTimestamps(timestamps, now)
				if len(valid) == 0 {
					delete(rl.requests, ip)
				} else {
					rl.requests[ip] = valid
				}
			}
			rl.mu.Unlock()
		case <-rl.stopCleanup:
			return
		}
	}
}

// Stop stops the cleanup goroutine. Safe to call multiple times.
func (rl *rateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stopCleanup) })
}

func (rl *rateLimiter) checkRateLimit(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	timestamps := rl.requests[ip]
	valid := rl.filterValidTimestamps(timestamps, now)

	if len(valid) >= rl.maxRequests {
		return false
	}

	valid = append(valid, now)
	rl.requests[ip] = valid

	return true
}
