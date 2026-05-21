package server

import (
	"embed"
	"net/http"
	"strings"

	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

//go:embed all:static
var staticFS embed.FS

func (s *Server) serveStaticFile(w http.ResponseWriter, r *http.Request) {
	filepath := r.URL.Path
	relativePath := strings.TrimPrefix(filepath, "/static/")

	if strings.Contains(relativePath, "..") {
		s.handle404(w, r)

		return
	}

	fullPath := "static/" + relativePath

	data, err := staticFS.ReadFile(fullPath)
	if err != nil {
		s.handle404(w, r)

		return
	}

	contentType := staticContentType(relativePath)
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func staticContentType(path string) string {
	ct := content.GetContentType(path)

	if ct != "" && ct != "application/octet-stream" {
		return ct
	}

	return ""
}
