package server

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// rateLimiter provides in-memory per-IP rate limiting using token buckets.
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

func newRateLimiter(maxRequests int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     rate.Every(window / time.Duration(maxRequests)),
		burst:    maxRequests,
	}
}

func (rl *rateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
	}

	return limiter
}

func (rl *rateLimiter) checkRateLimit(ip string) bool {
	return rl.getLimiter(ip).Allow()
}

// Stop is a no-op kept for API compatibility. Token bucket requires no cleanup.
func (rl *rateLimiter) Stop() {}
