package server

import (
	"fmt"
	"net/http"

	"github.com/larsartmann/httputil/endpoints"
)

// registeredHandler is what Server.Handler returns: the fully configured site
// handler plus the registry its routes were recorded in. handlerRoutes lets
// WithHealthProbes read that registry, so the probe mux in front records its
// own routes in the same GET /endpoints listing instead of shadowing it with
// a second, partial one.
type registeredHandler struct {
	http.Handler
	routes *endpoints.Registry
}

// handlerRoutes returns the route registry of a site handler built by
// Server.Handler, or a fresh registry for any other handler: nothing serves
// that registry's listing, but the probe routes still go through the seam.
func handlerRoutes(next http.Handler) *endpoints.Registry {
	registered, ok := next.(*registeredHandler)
	if !ok {
		return endpoints.NewRegistry()
	}

	return registered.routes
}

// mount registers handler on mux under pattern and records the route for the
// GET /endpoints listing in the same declaration (httputil/endpoints): the
// only way a route joins this server, so the listing cannot drift from what
// is served. http.Handler values (sub-muxes, the listing itself) go through
// mustMount(routes.Handle(...)) the same way. A pattern ServeMux rejects was a
// boot-time panic before the registry seam; mustMount keeps it one.
func mount(routes *endpoints.Registry, mux *http.ServeMux, pattern string, handler http.HandlerFunc, summary string) {
	mustMount(routes.HandleFunc(mux, pattern, handler, summary))
}

// record mirrors a route that a library registers on the mux by itself into
// the listing. The library call stays the served registration (go-health's
// RegisterRoutes owns its unstamped-version guard), the registry records only
// what it mounts, and v0.1.0 has no record-only call: so the row is mounted
// on a throwaway mux that is never served, with a stand-in handler. An empty
// pattern means the library mounted nothing (same rule as RegisterRoutes).
func record(routes *endpoints.Registry, pattern string, summary string) {
	if pattern == "" {
		return
	}

	mustMount(routes.Handle(http.NewServeMux(), pattern, http.NotFoundHandler(), summary))
}

// mustMount makes a rejected route declaration boot-fatal: a server that
// cannot mount or record every route must never start serving a listing that
// omits one.
func mustMount(err error) {
	if err != nil {
		panic(fmt.Errorf("server routes: %w", err))
	}
}
