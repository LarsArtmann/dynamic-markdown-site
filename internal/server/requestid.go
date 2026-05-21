package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDLength = 16
)

func generateRequestID() string {
	b := make([]byte, requestIDLength)
	if _, err := rand.Read(b); err != nil {
		return fallbackRequestID()
	}

	return hex.EncodeToString(b)
}

func fallbackRequestID() string {
	b := make([]byte, requestIDLength)
	for i := range b {
		b[i] = byte(i)
	}

	return hex.EncodeToString(b)
}

func requestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(RequestIDHeader)
			if requestID == "" {
				requestID = generateRequestID()
			}

			ctx := contextWithRequestID(r.Context(), requestID)
			r = r.WithContext(ctx)

			w.Header().Set(RequestIDHeader, requestID)

			next.ServeHTTP(w, r)
		})
	}
}

func getRequestIDFromContext(ctx context.Context) string {
	return requestIDFromContext(ctx)
}
