//go:build !dev

package server

import (
	"net/http"
	"slices"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

// TestEndpointsListingOmitsLiveReloadInReleaseBuilds pins the build-tag seam
// from the listing side: release builds compile GET /api/live-reload out, so
// even a dev-mode server must not advertise it. The dev build carries the row
// (endpoints_dev_test.go).
func TestEndpointsListingOmitsLiveReloadInReleaseBuilds(t *testing.T) {
	t.Parallel()

	srv := newTestServerWithDevMode(t, content.NewInMemoryRepository(), true)
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
