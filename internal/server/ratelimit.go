package server

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	// defaultSweepInterval is how often the eviction sweep runs.
	defaultSweepInterval = time.Minute
)

// rateLimiter provides in-memory per-IP rate limiting using token buckets.
// Each visitor keeps its last-seen timestamp and a sweep goroutine evicts
// entries idle beyond the TTL (3× the window), so the visitor map cannot
// grow without bound under sustained traffic from many IPs.
type rateLimiter struct {
	mu            sync.Mutex
	visitors      map[string]*visitor
	rate          rate.Limit
	burst         int
	ttl           time.Duration
	sweepInterval time.Duration
	stop          chan struct{}
	stopOnce      sync.Once
	done          chan struct{}
}

// visitor pairs a client's limiter with its last-seen time for eviction.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// newRateLimiter allows maxRequests per window per IP. The bucket starts
// full (burst = maxRequests) and refills at window/maxRequests tokens per
// second, so a client may spend the whole quota up front and then trickle
// at the steady rate; the deterministic burst-only tests rely on a window
// long enough that no refill can occur mid-test.
func newRateLimiter(maxRequests int, window time.Duration) *rateLimiter {
	return newRateLimiterWithSweepInterval(maxRequests, window, defaultSweepInterval)
}

// newRateLimiterWithSweepInterval exists for tests that need a fast eviction
// sweep; production uses the one-minute default.
func newRateLimiterWithSweepInterval(maxRequests int, window, sweepInterval time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors:      make(map[string]*visitor),
		rate:          rate.Every(window / time.Duration(maxRequests)),
		burst:         maxRequests,
		ttl:           3 * window,
		sweepInterval: sweepInterval,
		stop:          make(chan struct{}),
		done:          make(chan struct{}),
	}
	go rl.sweep()

	return rl
}

func (rl *rateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{limiter: rate.NewLimiter(rl.rate, rl.burst)}
		rl.visitors[ip] = v
	}
	v.lastSeen = time.Now()

	return v.limiter
}

func (rl *rateLimiter) checkRateLimit(ip string) bool {
	return rl.getLimiter(ip).Allow()
}

// sweep evicts visitors idle beyond the TTL until Stop is called.
func (rl *rateLimiter) sweep() {
	defer close(rl.done)

	ticker := time.NewTicker(rl.sweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case now := <-ticker.C:
			rl.evictIdle(now)
		}
	}
}

func (rl *rateLimiter) evictIdle(now time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, v := range rl.visitors {
		if now.Sub(v.lastSeen) > rl.ttl {
			delete(rl.visitors, ip)
		}
	}
}

// Stop terminates the sweep goroutine and waits for its exit. Calling it
// more than once is safe.
func (rl *rateLimiter) Stop() {
	rl.stopOnce.Do(func() { close(rl.stop) })
	<-rl.done
}
