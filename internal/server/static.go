package server

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *Server) serveStaticFile(c *gin.Context) {
	filepath := c.Request.URL.Path
	relativePath := strings.TrimPrefix(filepath, "/static/")

	if strings.Contains(relativePath, "..") {
		s.handle404(c)

		return
	}

	fullPath := "./internal/static/" + relativePath

	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		s.handle404(c)

		return
	}

	contentType := getContentType(fullPath)
	if contentType != "" {
		c.Header("Content-Type", contentType)
	}

	c.File(fullPath)
}

func getContentType(path string) string {
	switch {
	case strings.HasSuffix(path, ".css"):
		return "text/css"
	case strings.HasSuffix(path, ".js"):
		return "application/javascript"
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(path, ".gif"):
		return "image/gif"
	case strings.HasSuffix(path, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(path, ".woff"):
		return "font/woff"
	default:
		return ""
	}
}
