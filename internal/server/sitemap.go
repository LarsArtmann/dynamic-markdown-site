package server

import (
	"encoding/xml"
	"net/http"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

type SitemapEntry struct {
	Loc        string    `xml:"loc"`
	LastMod    time.Time `xml:"lastmod"`
	ChangeFreq string    `xml:"changefreq,omitempty"`
	Priority   float64   `xml:"priority"`
}

type URLSet struct {
	XMLName xml.Name       `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	URLs    []SitemapEntry `xml:"url"`
}

func (s *Server) handleSitemapXML(w http.ResponseWriter, r *http.Request) {
	root, err := s.repo.Root()
	if err != nil {
		s.logger.Error("failed to get root for sitemap", "error", err)
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)

		return
	}

	baseURL := s.baseURL(r)
	entries := s.buildSitemapEntries(root, baseURL)

	urlset := URLSet{
		XMLName: xml.Name{Space: "http://www.sitemaps.org/schemas/sitemap/0.9", Local: "urlset"},
		URLs:    entries,
	}

	w.Header().Set(headerContentType, "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")

	data, err := xml.Marshal(urlset)
	if err != nil {
		s.logger.Error("failed to marshal sitemap", "error", err)
		http.Error(w, "Error generating sitemap", http.StatusInternalServerError)

		return
	}

	_, _ = w.Write(data)
}

func (s *Server) buildSitemapEntries(root *domain.DirectoryNode, baseURL string) []SitemapEntry {
	var entries []SitemapEntry
	s.collectEntries(root, baseURL, &entries)

	return entries
}

func (s *Server) collectEntries(node domain.ContentNode, baseURL string, entries *[]SitemapEntry) {
	path := node.Path()

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

	if dir, ok := node.(*domain.DirectoryNode); ok {
		for _, child := range dir.Children() {
			s.collectEntries(child, baseURL, entries)
		}
	}
}

func (s *Server) calculatePriority(path domain.URLPath) float64 {
	segments := path.Segments()
	depth := len(segments)

	switch depth {
	case 0:
		return 1.0
	case 1:
		return 0.8
	case 2:
		return 0.6
	default:
		return 0.4
	}
}

func (s *Server) calculateChangeFreq(node domain.ContentNode) string {
	if node.Kind() == domain.NodeKindDirectory {
		return "daily"
	}

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
