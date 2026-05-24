package server

import (
	"context"
	"encoding/json"
	"net/http"

	httputil "github.com/larsartmann/httputil"
)

type contextKey string

const requestIDCtxKey contextKey = "request_id"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return
	}
}

type responseRecorder = httputil.ResponseRecorder

func newResponseRecorder(w http.ResponseWriter) *httputil.ResponseRecorder {
	return httputil.NewResponseRecorder(w)
}

func clientIP(r *http.Request) string {
	return httputil.ClientIP(r)
}

func chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	return httputil.Chain(handler, middlewares...)
}

func requestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDCtxKey).(string); ok {
		return id
	}

	return ""
}

func contextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDCtxKey, id)
}
