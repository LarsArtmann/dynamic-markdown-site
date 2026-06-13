package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httputil "github.com/larsartmann/httputil"
	"github.com/stretchr/testify/assert"
)

const requestIDHeader = "X-Request-ID"

func newTestRequest() (*httptest.ResponseRecorder, *http.Request) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "/test", nil)

	return w, req
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	t.Parallel()

	handler := httputil.RequestID(httputil.DefaultRequestIDConfig())(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get(requestIDHeader))
}

func TestRequestIDMiddleware_UsesExistingHeader(t *testing.T) {
	t.Parallel()

	handler := httputil.RequestID(httputil.DefaultRequestIDConfig())(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	w, req := newTestRequest()
	existingID := "existing-request-id-12345"
	req.Header.Set(requestIDHeader, existingID)
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, existingID, w.Header().Get(requestIDHeader))
}

func TestRequestIDMiddleware_StoresInContext(t *testing.T) {
	t.Parallel()

	var contextID string

	handler := httputil.RequestID(httputil.DefaultRequestIDConfig())(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			contextID = httputil.RequestIDFromContext(r.Context())
		}),
	)

	w, req := newTestRequest()
	handler.ServeHTTP(w, req)

	assert.NotEmpty(t, contextID)
	assert.Equal(t, w.Header().Get(requestIDHeader), contextID)
}

func TestRequestIDFromContext_NotSet(t *testing.T) {
	t.Parallel()

	id := httputil.RequestIDFromContext(context.Background())
	assert.Empty(t, id)
}

func TestRequestIDMiddleware_ChainOrder(t *testing.T) {
	t.Parallel()

	executionOrder := []string{}

	handler := chain(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
		httputil.RequestID(httputil.DefaultRequestIDConfig()),
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
