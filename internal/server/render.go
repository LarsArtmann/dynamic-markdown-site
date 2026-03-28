package server

import (
	"net/http"

	templ "github.com/a-h/templ"
	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/cache"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/templates"
)

func (s *Server) renderDirectory(c *gin.Context, dir *domain.DirectoryNode) {
	crumbs := domain.BuildBreadcrumbs(dir.Path())

	props := templates.LayoutProps{
		Title:       dir.Title(),
		Description: "Browse " + dir.Title(),
		Breadcrumbs: crumbs,
		ActivePath:  dir.Path(),
		ShowNav:     true,
		DevMode:     s.devMode,
	}

	dirProps := templates.DirectoryViewProps{
		Layout:    props,
		Directory: dir,
		HasReadme: false,
	}

	component := templates.DirectoryView(dirProps)
	s.renderTemplate(c, component, "directory")
}

func (s *Server) renderTemplate(c *gin.Context, component templ.Component, context string) {
	// Must explicitly set 200 OK - Gin may have set 404 for unmatched routes
	c.Status(http.StatusOK)

	err := component.Render(c.Request.Context(), c.Writer)
	if err != nil {
		s.logger.Error("failed to render template", "context", context, "error", err)
		s.handle500(c)
	}
}

func (s *Server) renderFile(c *gin.Context, file *domain.FileNode) {
	path := file.Path().String()

	// Check cache first
	if cached := s.cache.Get(path); cached != nil {
		file.SetHTML(cached.HTML)
		file.SetTOC(cached.TOC)
		file.SetMetadata(cached.Metadata)
	} else if file.HTML() == "" {
		result, err := s.renderer.Render(file.Content())
		if err != nil {
			s.logger.Error("failed to render markdown",
				"path", path,
				"error", err,
			)
			s.handle500(c)

			return
		}

		file.SetHTML(result.HTML)
		file.SetTOC(result.TOC)
		file.SetMetadata(result.Metadata)

		// Cache the rendered content
		s.cache.Set(path, cache.RenderedContent{
			HTML:     result.HTML,
			TOC:      result.TOC,
			Metadata: result.Metadata,
		})
	}

	crumbs := domain.BuildBreadcrumbs(file.Path())

	title := file.Title()
	if title == "" {
		title = file.Path().Filename()
	}

	description := file.Metadata().Description
	if description == "" {
		description = "Read " + title
	}

	props := templates.LayoutProps{
		Title:       title,
		Description: description,
		Breadcrumbs: crumbs,
		ActivePath:  file.Path(),
		ShowNav:     true,
	}

	fileProps := templates.FileViewProps{
		Layout: props,
		File:   file,
		TOC:    file.TOC(),
	}

	component := templates.FileView(fileProps)
	s.renderTemplate(c, component, "file")
}

func (s *Server) renderSearch(c *gin.Context, query string, results []content.SearchResult) {
	crumbs := domain.BuildBreadcrumbs(domain.MustURLPath("/search"))

	props := templates.LayoutProps{
		Title:       "Search",
		Description: "Search content",
		Breadcrumbs: crumbs,
		ActivePath:  domain.MustURLPath("/search"),
		ShowNav:     false,
	}

	searchProps := templates.SearchViewProps{
		Layout:  props,
		Query:   query,
		Results: results,
	}

	component := templates.SearchView(searchProps)
	s.renderTemplate(c, component, "search")
}
