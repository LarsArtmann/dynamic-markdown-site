package server

import (
	"net/http"
	"strings"
)

// dashboardPathPrefix is the dev-only go-health-dashboard mount
// (healthDashboardRoutes in dashboard_dev.go). CSP is scoped to it so the
// rest of the site keeps the site policy unchanged.
const dashboardPathPrefix = "/debug/health-dashboard"

// siteCSP is the standing policy for every non-dashboard route.
const siteCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
	"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' https://cdn.jsdelivr.net; " +
	"connect-src 'self'"

// dashboardCSP adds 'unsafe-eval' to the script sources: the Datastar SDK
// the dashboard embeds (WithEmbeddedDatastarSDK, same-origin from the
// binary — no CDN entry needed) compiles its data-* expressions with the
// Function constructor, and without 'unsafe-eval' it throws
// Error: GenerateExpression on init and the SSE connection never opens
// (the page renders, then silently never updates). 'unsafe-inline' stays:
// the dashboard's inline scripts carry the construction-time nonce only,
// and this dev surface deliberately keeps the site's existing inline-script
// posture rather than introducing a nonce lane.
const dashboardCSP = "default-src 'self'; " +
	"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
	"style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; " +
	"img-src 'self' data: https:; " +
	"font-src 'self' https://cdn.jsdelivr.net; " +
	"connect-src 'self'"

func securityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			if strings.HasPrefix(r.URL.Path, dashboardPathPrefix) {
				w.Header().Set("Content-Security-Policy", dashboardCSP)
			} else {
				w.Header().Set("Content-Security-Policy", siteCSP)
			}
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			next.ServeHTTP(w, r)
		})
	}
}
