package server

import (
	"net/http"
	"time"

	httputil "github.com/larsartmann/httputil"
)

func (s *Server) accessLogMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := newResponseRecorder(w)

			next.ServeHTTP(rec, r)

			duration := time.Since(start)

			s.logger.Info(
				"request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.Status(),
				"duration", duration,
				"request_id", httputil.RequestIDFromContext(r.Context()),
				"client_ip", clientIP(r),
			)
		})
	}
}
