//go:build dev

package server

import (
	"net/http"
	"slices"
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

// TestEndpointsListingCarriesLiveReloadInDevBuilds pins the build-tag seam
// from the listing side: a dev build running a dev-mode server registers
// GET /api/live-reload through the registry, so the listing carries the row.
// Release builds compile the route out; endpoints_prod_test.go pins that the
// row is gone there.
func TestEndpointsListingCarriesLiveReloadInDevBuilds(t *testing.T) {
	t.Parallel()

	srv := newTestServerWithDevMode(t, content.NewInMemoryRepository(), true)
	rec := requestEndpoints(t, newTestHandler(srv), http.MethodGet, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /endpoints: status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	want := append(siteRouteTable(), "GET /api/live-reload")
	slices.Sort(want)

	if got := listedPatterns(decodeListing(t, rec.Body.String())); !slices.Equal(got, want) {
		t.Errorf("listing patterns = %v, want %v", got, want)
	}
}
