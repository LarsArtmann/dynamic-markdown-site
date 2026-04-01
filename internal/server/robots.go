package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleRobotsTxt(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=86400")
	c.String(http.StatusOK, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", s.baseURL(c))
}

func (s *Server) baseURL(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}

	return scheme + "://" + c.Request.Host
}
