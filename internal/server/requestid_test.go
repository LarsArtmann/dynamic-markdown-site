package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(RequestIDHeader))
}

func TestRequestIDMiddleware_UsesExistingHeader(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	existingID := "existing-request-id-12345"
	req.Header.Set(RequestIDHeader, existingID)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, existingID, w.Header().Get(RequestIDHeader))
}

func TestRequestIDMiddleware_StoresInContext(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(requestIDMiddleware())

	var contextID string
	router.GET("/test", func(c *gin.Context) {
		contextID = getRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, contextID)
	assert.Equal(t, w.Header().Get(RequestIDHeader), contextID)
}

func TestGetRequestID_NotSet(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	// Note: NOT using requestIDMiddleware

	var contextID string
	router.GET("/test", func(c *gin.Context) {
		contextID = getRequestID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, contextID)
}

func TestGenerateRequestID_Length(t *testing.T) {
	t.Parallel()

	id := generateRequestID()

	// Should be 32 hex characters (16 bytes * 2)
	require.Len(t, id, 32)

	// Should be valid hex
	for _, c := range id {
		assert.True(t, isHex(c), "character %c is not valid hex", c)
	}
}

func TestGenerateRequestID_Uniqueness(t *testing.T) {
	t.Parallel()

	ids := make(map[string]bool)
	for range 100 {
		id := generateRequestID()
		assert.False(t, ids[id], "duplicate ID generated: %s", id)
		ids[id] = true
	}
}

func isHex(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

func TestGetRequestIDFromContext_WithID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(requestIDMiddleware())

	var extractedID string
	router.GET("/test", func(c *gin.Context) {
		extractedID = getRequestIDFromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	// Note: Gin context values don't propagate to request.Context() automatically
	// This tests the fallback behavior
	assert.Empty(t, extractedID)
}

func TestGetRequestIDFromContext_WithoutID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := getRequestIDFromContext(ctx)
	assert.Empty(t, id)
}

func TestRequestIDMiddleware_Chain(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	executionOrder := []string{}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		executionOrder = append(executionOrder, "before-middleware")
		c.Next()
		executionOrder = append(executionOrder, "after-middleware")
	})
	router.Use(requestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		executionOrder = append(executionOrder, "handler")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{
		"before-middleware",
		"handler",
		"after-middleware",
	}, executionOrder)
}
