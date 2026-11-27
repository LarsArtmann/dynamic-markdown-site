package server

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/httputil/endpoints"
)

// siteRouteTable is the site mux's complete route table in listing form
// (ServeMux pattern syntax) for a server outside dev mode. The dev-mode-only
// GET /api/live-reload row is pinned per build variant by
// endpoints_dev_test.go and endpoints_prod_test.go.
func siteRouteTable() []string {
	return []string{
		"GET /",
		"GET /{$}",
		"GET /cache/stats",
		"GET /endpoints",
		"GET /metrics",
		"GET /refresh",
		"POST /refresh",
		"GET /robots.txt",
		"GET /search",
		"GET /sitemap.xml",
		"GET /static/",
	}
}

// probeRouteTable is the route table WithHealthProbes mounts in front of the
// site handler. The two dashboard rows are registered in every build; release
// builds answer them with 404.
func probeRouteTable() []string {
	return []string{
		"/",
		"GET /debug/health-dashboard",
		"GET /debug/health-dashboard/",
		"GET /health",
		"GET /healthz",
		"GET /readyz",
		"GET /startupz",
		"GET /version",
	}
}

// newProbedHandler composes the handler the way main does: the probe mux in
// front of the site handler, over a healthy probe.
func newProbedHandler(t *testing.T) http.Handler {
	t.Helper()

	repo := content.NewInMemoryRepository()
	srv := newTestServer(t, repo)

	return WithHealthProbes(newTestHandler(srv), newHealthyTestProbe(t, repo))
}

// requestEndpoints sends a request for /endpoints through handler and returns
// the recorded response. The identity encoding keeps the body readable: the
// compression middleware gzips responses above 512 bytes otherwise.
func requestEndpoints(t *testing.T, handler http.Handler, method, accept string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, "/endpoints", nil)
	req.Header.Set("Accept-Encoding", "identity")
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

// decodeListing returns the routes a JSON listing body carries.
func decodeListing(t *testing.T, body string) []endpoints.Route {
	t.Helper()

	var listing endpoints.Response
	if err := json.Unmarshal([]byte(body), &listing); err != nil {
		t.Fatalf("decode listing: %v (body=%s)", err, body)
	}

	return listing.Endpoints
}

// listedPatterns returns the patterns of routes, sorted.
func listedPatterns(routes []endpoints.Route) []string {
	patterns := make([]string, 0, len(routes))
	for _, route := range routes {
		patterns = append(patterns, route.Pattern)
	}

	slices.Sort(patterns)

	return patterns
}

// hasVary reports whether the Vary header lists token. The compression
// middleware adds its own Accept-Encoding entry, so the listing's Accept
// entry must be looked up token by token.
func hasVary(header http.Header, token string) bool {
	for _, value := range header.Values("Vary") {
		for candidate := range strings.SplitSeq(value, ",") {
			if strings.EqualFold(strings.TrimSpace(candidate), token) {
				return true
			}
		}
	}

	return false
}

// TestEndpointsListingCoversEveryRoute pins the property the registry exists
// for: GET /endpoints serves exactly the routes the process mounts, on the
// probe mux as well as the site mux. Every route is declared through mount
// (or recorded by record), so a mismatch means a route bypassed the seam.
func TestEndpointsListingCoversEveryRoute(t *testing.T) {
	t.Parallel()

	rec := requestEndpoints(t, newProbedHandler(t), http.MethodGet, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /endpoints: status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("GET /endpoints: Content-Type = %q, want application/json", ct)
	}

	if !hasVary(rec.Header(), "Accept") {
		t.Errorf("GET /endpoints: Vary = %v, want it to list Accept", rec.Header().Values("Vary"))
	}

	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("GET /endpoints: Cache-Control = %q, want no-store", cc)
	}

	// The listing rides the site middleware chain like every site route.
	if frame := rec.Header().Get("X-Frame-Options"); frame != "DENY" {
		t.Errorf("GET /endpoints: X-Frame-Options = %q, want DENY", frame)
	}

	listed := decodeListing(t, rec.Body.String())
	want := append(siteRouteTable(), probeRouteTable()...)
	slices.Sort(want)

	patterns := listedPatterns(listed)

	for _, pattern := range want {
		if !slices.Contains(patterns, pattern) {
			t.Errorf("route %q is mounted but MISSING from the listing", pattern)
		}
	}

	for _, pattern := range patterns {
		if !slices.Contains(want, pattern) {
			t.Errorf("listing advertises %q, which is not a mounted route", pattern)
		}
	}

	for _, route := range listed {
		if route.Summary == "" {
			t.Errorf("listing row %q carries no summary", route.Pattern)
		}
	}
}

// TestEndpointsListingWithoutProbeMux pins the site handler on its own (the
// nil-probe wiring of WithHealthProbes): its listing carries the site routes
// and nothing from the probe mux.
func TestEndpointsListingWithoutProbeMux(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t, content.NewInMemoryRepository())
	rec := requestEndpoints(t, newTestHandler(srv), http.MethodGet, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /endpoints: status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	want := siteRouteTable()
	slices.Sort(want)

	if got := listedPatterns(decodeListing(t, rec.Body.String())); !slices.Equal(got, want) {
		t.Errorf("listing patterns = %v, want %v", got, want)
	}
}

// TestEndpointsListingNegotiatesHTML pins the browser representation: an
// Accept header asking for HTML gets the HTML table instead of JSON.
func TestEndpointsListingNegotiatesHTML(t *testing.T) {
	t.Parallel()

	rec := requestEndpoints(t, newProbedHandler(t), http.MethodGet, "text/html,application/xhtml+xml")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /endpoints (html): status = %d, want 200", rec.Code)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("GET /endpoints (html): Content-Type = %q, want text/html; charset=utf-8", ct)
	}

	body := rec.Body.String()
	for _, want := range []string{
		"<table",
		`<th scope="col">Pattern</th>`,
		"GET /search",
		"GET /healthz",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("HTML listing lacks %q; body: %.400s", want, body)
		}
	}
}

// TestEndpointsListingHEAD pins HEAD: same status and headers as GET, no body.
// It needs a real server because only net/http drops the body of a HEAD
// response; httptest.ResponseRecorder would keep what the handler writes.
func TestEndpointsListingHEAD(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(newProbedHandler(t))
	t.Cleanup(server.Close)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodHead, server.URL+"/endpoints", nil)
	if err != nil {
		t.Fatalf("new HEAD /endpoints request: %v", err)
	}

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("HEAD /endpoints: %v", err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read HEAD /endpoints body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("HEAD /endpoints: status = %d, want 200", resp.StatusCode)
	}

	if len(body) != 0 {
		t.Errorf("HEAD /endpoints: body = %q, want empty", body)
	}

	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("HEAD /endpoints: Content-Type = %q, want application/json", ct)
	}
}

// TestEndpointsListingRejectsNonGET pins that the listing is never served to
// a non-GET/HEAD request. Both muxes register GET-only patterns for their
// routes, so ServeMux itself answers 405 with an Allow header: the probe
// mux's "/" catch-all forwards the request to the site handler, whose mux has
// no pattern for the method. The listing handler's own 405 branch is never
// reached on this wiring.
func TestEndpointsListingRejectsNonGET(t *testing.T) {
	t.Parallel()

	handler := newProbedHandler(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()

			rec := requestEndpoints(t, handler, method, "")

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("%s /endpoints: status = %d, want 405", method, rec.Code)
			}

			if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
				t.Errorf("%s /endpoints: Allow = %q, want GET, HEAD", method, allow)
			}

			if strings.Contains(rec.Body.String(), `"endpoints"`) {
				t.Errorf("%s /endpoints served the listing; body: %.200s", method, rec.Body.String())
			}
		})
	}
}

// TestRecordedProbeRoutesAreServed pins that the rows record() mirrors from
// go-health's RegisterRoutes are real routes: each one answers 200 through
// the probe mux. A renamed go-health pattern would otherwise leave the
// listing advertising a route that no longer exists.
func TestRecordedProbeRoutesAreServed(t *testing.T) {
	t.Parallel()

	handler := newProbedHandler(t)

	for _, path := range []string{"/healthz", "/readyz", "/startupz", "/version"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			rec := executeRequest(handler, path)
			if rec.Code != http.StatusOK {
				t.Errorf("GET %s: status = %d, want 200 (it is listed as a served route)", path, rec.Code)
			}
		})
	}
}

// rawMuxRegistration matches a direct registration on a ServeMux value
// (mux.Handle, dashboardMux.HandleFunc, ...). ServeMux cannot enumerate its
// patterns, so such a route would be served but missing from the listing.
// The registry calls (mount, record, mustMount(routes.Handle(...))) do not
// match, and neither does go-health's own RegisterRoutes, whose rows are
// mirrored by record.
var rawMuxRegistration = regexp.MustCompile(`\b(?:[A-Za-z_]\w*\.)*(?:m|mux|\w*[mM]ux)\.(?:Handle|HandleFunc)\(`)

// TestRoutesGoThroughTheRegistry is the static guard behind the listing: no
// non-test source of this package may register on a ServeMux outside the
// registry seam in routes.go. Both build variants are scanned because the
// dev and prod files sit side by side on disk.
func TestRoutesGoThroughTheRegistry(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		source, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}

		for lineNo, line := range strings.Split(string(source), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") {
				continue
			}

			if rawMuxRegistration.MatchString(trimmed) {
				t.Errorf(
					"%s:%d registers directly on the mux (%s); route it through mount in routes.go "+
						"so the /endpoints listing cannot drift",
					name,
					lineNo+1,
					trimmed,
				)
			}
		}
	}
}
