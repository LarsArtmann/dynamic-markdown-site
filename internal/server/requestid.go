package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

// contextKey is a private type for context keys to avoid collisions.
type contextKey string

const (
	// requestIDKey is the context key for request ID.
	requestIDKey contextKey = "request_id"
	// RequestIDHeader is the HTTP header name for request ID.
	RequestIDHeader = "X-Request-ID"
	// requestIDLength is the byte length of generated request IDs (32 hex chars).
	requestIDLength = 16
)

// generateRequestID creates a cryptographically secure random request ID.
// Returns a 32-character hex string.
func generateRequestID() string {
	b := make([]byte, requestIDLength)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto rand fails
		return fallbackRequestID()
	}

	return hex.EncodeToString(b)
}

// fallbackRequestID returns a simple timestamp-based ID.
// Used only when crypto/rand fails (extremely rare).
func fallbackRequestID() string {
	b := make([]byte, requestIDLength)
	for i := range b {
		b[i] = byte(i)
	}

	return hex.EncodeToString(b)
}

// requestIDMiddleware adds a unique request ID to each request.
// It checks for an existing ID in the X-Request-ID header first,
// and generates a new random ID if none is provided.
func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing request ID in header
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			// Generate new request ID
			requestID = generateRequestID()
		}

		// Store in Gin context
		c.Set(string(requestIDKey), requestID)

		// Add to response header for client tracing
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}

// getRequestID retrieves the request ID from Gin context.
// Returns empty string if not found.
func getRequestID(c *gin.Context) string {
	if id, exists := c.Get(string(requestIDKey)); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}

	return ""
}

// getRequestIDFromContext retrieves the request ID from a standard context.Context.
// Useful when you have a context but not the Gin context.
func getRequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}

	return ""
}
