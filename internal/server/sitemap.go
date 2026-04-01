package server

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

// SitemapEntry represents a single entry in the sitemap.
type SitemapEntry struct {
	Loc        string    `xml:"loc"`
	LastMod    time.Time `xml:"lastmod"`
	ChangeFreq string    `xml:"changefreq,omitempty"`
	Priority   float64   `xml:"priority"`
}

// URLSet is the root element of a sitemap.xml file.
type URLSet struct {
	XMLName xml.Name       `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []SitemapEntry `xml:"url"`
}

func (s *Server) handleSitemapXML(c *gin.Context) {
	root, err := s.repo.Root()
	if err != nil {
		s.logger.Error("failed to get root for sitemap", "error", err)
		c.String(http.StatusInternalServerError, "Error generating sitemap")
		return
	}

	baseURL := s.baseURL(c)
	entries := s.buildSitemapEntries(root, baseURL)

	urlset := URLSet{URLs: entries}

	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.Header("Cache-Control", "public, max-age=3600")

	c.XML(http.StatusOK, urlset)
}

func (s *Server) buildSitemapEntries(root *domain.DirectoryNode, baseURL string) []SitemapEntry {
	var entries []SitemapEntry
	s.collectEntries(root, baseURL, &entries)
	return entries
}

func (s *Server) collectEntries(node domain.ContentNode, baseURL string, entries *[]SitemapEntry) {
	path := node.Path()

	// Skip root path - we want actual content paths
	if !path.IsRoot() {
		loc := baseURL + path.String()
		priority := s.calculatePriority(path)
		changeFreq := s.calculateChangeFreq(node)

		*entries = append(*entries, SitemapEntry{
			Loc:        loc,
			LastMod:    node.Modified(),
			ChangeFreq: changeFreq,
			Priority:   priority,
		})
	}

	// Recursively collect from directories
	if dir, ok := node.(*domain.DirectoryNode); ok {
		for _, child := range dir.Children() {
			s.collectEntries(child, baseURL, entries)
		}
	}
}

func (s *Server) calculatePriority(path domain.URLPath) float64 {
	segments := path.Segments()
	depth := len(segments)

	// Higher priority for shallower paths
	switch depth {
	case 0:
		return 1.0 // Root
	case 1:
		return 0.8 // Top-level pages
	case 2:
		return 0.6 // Second-level
	default:
		return 0.4 // Deeper pages
	}
}

func (s *Server) calculateChangeFreq(node domain.ContentNode) string {
	if node.Kind() == domain.NodeKindDirectory {
		return "daily" // Directories change when children are added/removed
	}

	// Files change based on age
	age := time.Since(node.Modified())
	switch {
	case age < 24*time.Hour:
		return "hourly"
	case age < 7*24*time.Hour:
		return "daily"
	case age < 30*24*time.Hour:
		return "weekly"
	case age < 365*24*time.Hour:
		return "monthly"
	default:
		return "yearly"
	}
}
