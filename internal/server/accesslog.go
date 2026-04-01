package server

import (
	"time"

	"github.com/gin-gonic/gin"
)

// accessLogMiddleware logs each request with method, path, status, duration, and request ID.
func (s *Server) accessLogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		s.logger.Info("request",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration", duration,
			"request_id", getRequestID(c),
			"client_ip", c.ClientIP(),
		)
	}
}
