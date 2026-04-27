package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHandleRobotsTxt(t *testing.T) {
	t.Parallel()

	repo := &FailingRepository{}
	srv := newTestServer(t, repo)

	tests := []struct {
		name            string
		host            string
		forwardedProto  string
		wantStatus      int
		wantContains    []string
		wantCacheCtrl   string
		wantContentType string
	}{
		{
			name:       "basic http request",
			host:       "example.com",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"User-agent: *",
				"Allow: /",
				"Sitemap: http://example.com/sitemap.xml",
			},
			wantCacheCtrl:   "public, max-age=86400",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name:           "https via X-Forwarded-Proto",
			host:           "example.com",
			forwardedProto: "https",
			wantStatus:     http.StatusOK,
			wantContains: []string{
				"Sitemap: https://example.com/sitemap.xml",
			},
			wantCacheCtrl:   "public, max-age=86400",
			wantContentType: "text/plain; charset=utf-8",
		},
		{
			name:       "custom host with port",
			host:       "localhost:8080",
			wantStatus: http.StatusOK,
			wantContains: []string{
				"Sitemap: http://localhost:8080/sitemap.xml",
			},
			wantCacheCtrl:   "public, max-age=86400",
			wantContentType: "text/plain; charset=utf-8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			srv.RegisterRoutes(router)

			req := httptest.NewRequestWithContext(
				context.Background(),
				http.MethodGet,
				"/robots.txt",
				nil,
			)

			req.Host = tt.host
			if tt.forwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.forwardedProto)
			}

			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			body := rec.Body.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("body should contain %q, got: %s", want, body)
				}
			}

			if got := rec.Header().Get("Cache-Control"); got != tt.wantCacheCtrl {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheCtrl)
			}

			if got := rec.Header().Get("Content-Type"); got != tt.wantContentType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantContentType)
			}
		})
	}
}
