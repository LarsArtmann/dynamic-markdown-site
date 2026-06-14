package server

import (
	"encoding/json"
	"net/http"

	httputil "github.com/larsartmann/httputil"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set(headerContentType, "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		return
	}
}

func newResponseRecorder(w http.ResponseWriter) *httputil.ResponseRecorder {
	return httputil.NewResponseRecorder(w)
}

func clientIP(r *http.Request) string {
	return httputil.ClientIP(r)
}

func chain(handler http.Handler, middlewares ...httputil.Middleware) http.Handler {
	return httputil.Chain(handler, middlewares...)
}
