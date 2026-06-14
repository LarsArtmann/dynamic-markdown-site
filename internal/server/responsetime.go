package server

import (
	"net/http"
	"strconv"
	"time"
)

const headerResponseTime = "X-Response-Time"

// responseTimeMiddleware records the duration of every request and exposes it via
// the X-Response-Time response header. This is intentionally lightweight —
// richer metrics (Prometheus exposition, percentiles) live in a separate package
// to keep the middleware dependency-free.
func (s *Server) responseTimeMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			next.ServeHTTP(w, r)

			elapsed := time.Since(start)
			w.Header().Set(headerResponseTime, strconv.FormatInt(elapsed.Microseconds(), 10)+"us")
		})
	}
}
