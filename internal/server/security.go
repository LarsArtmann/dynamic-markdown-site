package server

import (
	"github.com/gin-gonic/gin"
)

// securityHeadersMiddleware adds security-related HTTP headers to all responses.
func securityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking attacks
		c.Header("X-Frame-Options", "DENY")

		// XSS protection (legacy browsers)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HTTPS only)
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:; font-src 'self' https://cdn.jsdelivr.net; connect-src 'self'")

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (disable unnecessary browser features)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}
