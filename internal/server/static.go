package server

import (
	"embed"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
)

//go:embed all:static
var staticFS embed.FS

func (s *Server) serveStaticFile(c *gin.Context) {
	filepath := c.Request.URL.Path
	relativePath := strings.TrimPrefix(filepath, "/static/")

	if strings.Contains(relativePath, "..") {
		s.handle404(c)

		return
	}

	fullPath := "static/" + relativePath

	data, err := staticFS.ReadFile(fullPath)
	if err != nil {
		s.handle404(c)

		return
	}

	contentType := staticContentType(relativePath)
	if contentType != "" {
		c.Header("Content-Type", contentType)
	}

	c.Data(http.StatusOK, contentType, data)
}

// staticContentType returns the content type for static assets.
// Falls back to the shared content.GetContentType for known MIME types.
func staticContentType(path string) string {
	ct := content.GetContentType(path)

	if ct != "" && ct != "application/octet-stream" {
		return ct
	}

	return ""
}
