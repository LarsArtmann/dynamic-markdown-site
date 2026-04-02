package server

import (
	"context"
	"net/http"

	templ "github.com/a-h/templ"
	"github.com/cockroachdb/errors"
	"github.com/gin-gonic/gin"
	"github.com/larsartmann/dynamic-markdown-site/internal/content"
	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/templates"
)

func (s *Server) renderDirectory(c *gin.Context, dir *domain.DirectoryNode) {
	crumbs := domain.BuildBreadcrumbs(dir.Path())

	hasReadme := false
	for _, child := range dir.Children() {
		if child.Kind() == domain.NodeKindFile && child.Path().Filename() == "README.md" {
			hasReadme = true

			break
		}
	}

	props := templates.LayoutProps{
		Title:       dir.Title(),
		Description: "Browse " + dir.Title(),
		Breadcrumbs: crumbs,
		ActivePath:  dir.Path(),
		ShowNav:     true,
		DevMode:     s.devMode,
		SiteName:    s.siteName,
	}

	dirProps := templates.DirectoryViewProps{
		Layout:    props,
		Directory: dir,
		HasReadme: hasReadme,
	}

	component := templates.DirectoryView(dirProps)
	s.renderTemplate(c, component, "directory")
}

func (s *Server) renderTemplate(c *gin.Context, component templ.Component, context string) {
	// Must explicitly set 200 OK - Gin may have set 404 for unmatched routes
	c.Status(http.StatusOK)
	s.renderComponent(c, component, http.StatusOK, context)
}

func (s *Server) renderFile(c *gin.Context, file *domain.FileNode) {
	path := file.Path().String()

	// Get or render content
	renderedContent := s.getOrRenderContent(c.Request.Context(), path, file)

	// Create immutable RenderedFile combining FileNode with rendered content
	renderedFile := domain.NewRenderedFileWithContent(file, renderedContent)

	crumbs := domain.BuildBreadcrumbs(file.Path())

	title := renderedFile.Title()
	if title == "" {
		title = file.Path().Filename()
	}

	description := renderedFile.Metadata().Description
	if description == "" {
		description = "Read " + title
	}

	props := templates.LayoutProps{
		Title:       title,
		Description: description,
		Breadcrumbs: crumbs,
		ActivePath:  file.Path(),
		ShowNav:     true,
		HasMermaid:  renderedFile.HasMermaid(),
		SiteName:    s.siteName,
	}

	fileProps := templates.FileViewProps{
		Layout:       props,
		RenderedFile: renderedFile,
	}

	component := templates.FileView(fileProps)
	s.renderTemplate(c, component, "file")
}

// getOrRenderContent returns cached content or renders and caches it.
func (s *Server) getOrRenderContent(
	ctx context.Context, path string, file *domain.FileNode,
) domain.RenderedContent {
	result, err := s.cache.GetOrCompute(ctx, path, func() (domain.RenderedContent, error) {
		res, renderErr := s.renderer.Render(file.Content())
		if renderErr != nil {
			s.logger.Error("failed to render markdown", "path", path, "error", renderErr)

			return domain.RenderedContent{}, errors.Wrapf(renderErr, "render markdown at %s", path)
		}

		return res, nil
	})
	if err != nil {
		s.logger.Error("cache get failed for path", "path", path, "error", err)

		return domain.RenderedContent{}
	}

	return *result
}

func (s *Server) renderSearch(c *gin.Context, query string, results []content.SearchResult) {
	crumbs := domain.BuildBreadcrumbs(domain.MustURLPath("/search"))

	props := templates.LayoutProps{
		Title:       "Search",
		Description: "Search content",
		Breadcrumbs: crumbs,
		ActivePath:  domain.MustURLPath("/search"),
		ShowNav:     false,
		SiteName:    s.siteName,
	}

	searchProps := templates.SearchViewProps{
		Layout:  props,
		Query:   query,
		Results: results,
	}

	component := templates.SearchView(searchProps)
	s.renderTemplate(c, component, "search")
}
