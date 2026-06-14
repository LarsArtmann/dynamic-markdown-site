package server

import (
	"fmt"
	"net/http"
)

func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")

	// s.baseURL(r) returns scheme://host, where scheme is a hard-coded
	// constant and host comes from the trusted request. Not user-controlled.
	//nolint:gosec // G107: baseURL is constructed from validated scheme + host
	_, _ = fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", s.baseURL(r))
}

func (s *Server) baseURL(r *http.Request) string {
	scheme := schemeHTTP
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == schemeHTTPS {
		scheme = schemeHTTPS
	}

	return scheme + "://" + r.Host
}
