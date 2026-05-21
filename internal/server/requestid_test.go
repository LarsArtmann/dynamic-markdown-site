package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRequest() (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	return w, req
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	t.Parallel()

	handler := requestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(RequestIDHeader))
}

func TestRequestIDMiddleware_UsesExistingHeader(t *testing.T) {
	t.Parallel()

	handler := requestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w, req := newTestRequest()
	existingID := "existing-request-id-12345"
	req.Header.Set(RequestIDHeader, existingID)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, existingID, w.Header().Get(RequestIDHeader))
}

func TestRequestIDMiddleware_StoresInContext(t *testing.T) {
	t.Parallel()

	var contextID string

	handler := requestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = requestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, contextID)
	assert.Equal(t, w.Header().Get(RequestIDHeader), contextID)
}

func TestGetRequestID_NotSet(t *testing.T) {
	t.Parallel()

	var contextID string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID = requestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, contextID)
}

func TestGenerateRequestID_Length(t *testing.T) {
	t.Parallel()

	id := generateRequestID()

	require.Len(t, id, 32)

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

	var extractedID string

	handler := requestIDMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		extractedID = getRequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, extractedID)
}

func TestGetRequestIDFromContext_WithoutID(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	id := getRequestIDFromContext(ctx)
	assert.Empty(t, id)
}

func TestRequestIDMiddleware_Chain(t *testing.T) {
	t.Parallel()

	executionOrder := []string{}

	handler := chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "handler")
			w.WriteHeader(http.StatusOK)
		}),
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, "before-middleware")
				next.ServeHTTP(w, r)
				executionOrder = append(executionOrder, "after-middleware")
			})
		},
		requestIDMiddleware(),
	)

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{
		"before-middleware",
		"handler",
		"after-middleware",
	}, executionOrder)
}
