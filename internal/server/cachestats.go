package server

import (
	"net/http"
)

// handleCacheStats returns cache hit/miss counters in JSON form.
//
// This endpoint is intentionally unauthenticated because the data is aggregate
// and non-sensitive. Operators that want to lock it down can front it with a
// reverse proxy or move it behind the same auth as the metrics endpoint.
func (s *Server) handleCacheStats(w http.ResponseWriter, _ *http.Request) {
	stats := s.cache.Stats()
	writeJSON(w, http.StatusOK, map[string]any{
		jsonKeyStatus:  jsonStatusSuccess,
		"size":         s.cache.EstimatedSize(),
		"hits":         stats.Hits,
		"misses":       stats.Misses,
		"evictions":    stats.Evictions,
		"loadSuccess":  stats.LoadSuccesses,
		"loadFailures": stats.LoadFailures,
		"loadDuration": stats.TotalLoadTime.String(),
		"hitRatio":     stats.HitRatio(),
		"requests":     stats.Requests(),
	})
}
