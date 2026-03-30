package domain_test

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
)

func assertTitle(t *testing.T, gotTitle, wantTitle, constructor string) {
	t.Helper()
	if gotTitle != wantTitle {
		t.Errorf("%s().Title() = %q, want %q", constructor, gotTitle, wantTitle)
	}
}

func TestURLPath_NewURLPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"root path", "/", "/", false},
		{"simple path", "/docs", "/docs", false},
		{"nested path", "/docs/getting-started", "/docs/getting-started", false},
		{"empty path", "", "", true},
		{
			"directory traversal attempt",
			"/docs/../secret",
			"/secret",
			false,
		}, // filepath.Clean normalizes before check
		{"traversal in middle", "/a/../b", "/b", false},
		{"trailing slash", "/docs/", "/docs", false},
		{"path without leading slash", "docs", "/docs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := domain.NewURLPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewURLPath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)

				return
			}

			if !tt.wantErr && got.String() != tt.want {
				t.Errorf("NewURLPath(%q) = %q, want %q", tt.input, got.String(), tt.want)
			}
		})
	}
}

func TestURLPath_MustURLPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		wantPanic bool
	}{
		{"valid path", "/docs", false},
		{"nested path", "/a/b/c", false},
		{"empty path", "", true},
		{"traversal normalized", "/../secret", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil != tt.wantPanic {
					t.Errorf("MustURLPath(%q) panic = %v, wantPanic %v", tt.input, r, tt.wantPanic)
				}
			}()

			got := domain.MustURLPath(tt.input)
			if !tt.wantPanic {
				// Handle case where input doesn't have leading slash or gets normalized
				expected := tt.input
				if tt.input != "" && tt.input[0] != '/' {
					expected = "/" + tt.input
				}
				// filepath.Clean normalizes paths like /../secret to /secret
				// So we need to compare with the cleaned expected value
				expected = filepath.Clean(expected)
				if got.String() != expected {
					t.Errorf("MustURLPath(%q) = %q, want %q", tt.input, got.String(), expected)
				}
			}
		})
	}
}

func TestURLPath_Join(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		base    string
		segment string
		want    string
		wantErr bool
	}{
		{
			name:    "join path",
			base:    "/docs",
			segment: "getting-started",
			want:    "/docs/getting-started",
			wantErr: false,
		},
		{name: "empty segment", base: "/docs", segment: "", want: "/docs", wantErr: false},
		{name: "traversal rejected", base: "/", segment: "../secret", want: "", wantErr: true},
		{name: "root join", base: "/", segment: "docs", want: "/docs", wantErr: false},
		{name: "nested join", base: "/a/b", segment: "c", want: "/a/b/c", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base, _ := domain.NewURLPath(tt.base)

			got, err := base.Join(tt.segment)
			if (err != nil) != tt.wantErr {
				t.Errorf(
					"URLPath.Join(%q, %q) error = %v, wantErr %v",
					tt.base,
					tt.segment,
					err,
					tt.wantErr,
				)

				return
			}

			if !tt.wantErr && got.String() != tt.want {
				t.Errorf(
					"URLPath.Join(%q, %q) = %q, want %q",
					tt.base,
					tt.segment,
					got.String(),
					tt.want,
				)
			}
		})
	}
}

func TestURLPath_Parent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"/a/b/c", "/a/b"},
		{"/a/b", "/a"},
		{"/a", "/"},
		{"/", "/"},
		{"/docs/getting-started", "/docs"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			path, _ := domain.NewURLPath(tt.input)

			got := path.Parent()
			if got.String() != tt.want {
				t.Errorf("URLPath(%q).Parent() = %q, want %q", tt.input, got.String(), tt.want)
			}
		})
	}
}

func TestURLPath_Segments(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  []string
	}{
		{"/", []string{""}},
		{"/docs", []string{"docs"}},
		{"/a/b/c", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			path, _ := domain.NewURLPath(tt.input)

			got := path.Segments()
			if len(got) != len(tt.want) {
				t.Errorf(
					"URLPath(%q).Segments() len = %d, want %d",
					tt.input,
					len(got),
					len(tt.want),
				)

				return
			}

			for i, s := range got {
				if s != tt.want[i] {
					t.Errorf("URLPath(%q).Segments()[%d] = %q, want %q", tt.input, i, s, tt.want[i])
				}
			}
		})
	}
}

// formatQuoted returns a quoted string representation for test error messages.
func formatQuoted(v string) string { return fmt.Sprintf("%q", v) }

// formatBool returns a boolean representation for test error messages.
func formatBool(v bool) string { return strconv.FormatBool(v) }

// testURLPathMethod is a generic helper for testing URLPath methods that return a single value.
func testURLPathMethod[T comparable](t *testing.T, name string, tests []struct {
	input string
	want  T
}, method func(*domain.URLPath) T, formatGot, formatWant func(T) string,
) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				path, _ := domain.NewURLPath(tt.input)

				got := method(&path)
				if got != tt.want {
					t.Errorf(
						"URLPath(%q).%s() = %s, want %s",
						tt.input,
						name,
						formatGot(got),
						formatWant(tt.want),
					)
				}
			})
		}
	})
}

func TestURLPath_Filename(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"/docs", "docs"},
		{"/a/b/c.md", "c.md"},
		{"/", "/"}, // Root path's filename is "/" itself
	}

	testURLPathMethod(t, "Filename", tests,
		func(p *domain.URLPath) string { return p.Filename() },
		formatQuoted,
		formatQuoted,
	)
}

func TestURLPath_IsRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"/", true},
		{"", true},
		{"/docs", false},
		{"/a/b", false},
	}

	testURLPathMethod(t, "IsRoot", tests,
		func(p *domain.URLPath) bool { return p.IsRoot() },
		formatBool,
		formatBool,
	)
}

func TestDirectoryNode_NewDirectoryNode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		path      string
		title     string
		modified  time.Time
		wantTitle string
		wantErr   bool
	}{
		{"valid", "/docs", "Documentation", time.Now(), "Documentation", false},
		{"empty title defaults to path", "/docs", "", time.Now(), "docs", false},
		{"empty path", "", "Docs", time.Now(), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path, _ := domain.NewURLPath(tt.path)

			got, err := domain.NewDirectoryNode(path, tt.title, tt.modified)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDirectoryNode() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				assertTitle(t, got.Title(), tt.wantTitle, "NewDirectoryNode")
			}
		})
	}
}

func TestDirectoryNode_AddChild(t *testing.T) {
	root, _ := domain.NewDirectoryNode(domain.MustURLPath("/"), "Root", time.Now())

	// Add a file first
	file, _ := domain.NewFileNode(
		domain.MustURLPath("/file.md"),
		"File",
		[]byte("content"),
		time.Now(),
		100,
	)
	root.AddChild(file)

	// Add a directory
	dir, _ := domain.NewDirectoryNode(domain.MustURLPath("/subdir"), "SubDir", time.Now())
	root.AddChild(dir)

	children := root.Children()
	if len(children) != 2 {
		t.Errorf("Root has %d children, want 2", len(children))
	}

	// Directory should come before file
	if children[0].Kind() != domain.NodeKindDirectory {
		t.Errorf("First child should be directory, got %v", children[0].Kind())
	}

	if children[1].Kind() != domain.NodeKindFile {
		t.Errorf("Second child should be file, got %v", children[1].Kind())
	}
}

func TestFileNode_NewFileNode(t *testing.T) {
	t.Parallel()
	content := []byte("# Hello World")
	modified := time.Now()

	tests := []struct {
		name      string
		path      string
		title     string
		content   []byte
		modified  time.Time
		size      uint64
		wantTitle string
		wantErr   bool
	}{
		{"valid", "/docs/hello.md", "Hello", content, modified, 100, "Hello", false},
		{
			name:      "with extension",
			path:      "/docs/hello.md",
			title:     "Hello.md",
			content:   content,
			modified:  modified,
			size:      100,
			wantTitle: "Hello.md",
			wantErr:   false,
		},
		{"empty path", "", "Hello", content, modified, 100, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path, _ := domain.NewURLPath(tt.path)

			got, err := domain.NewFileNode(path, tt.title, tt.content, tt.modified, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFileNode() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if !tt.wantErr {
				assertTitle(t, got.Title(), tt.wantTitle, "NewFileNode")
			}
		})
	}
}

func TestFileNode_ReadingTime(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		content     string
		wantMinTime uint
	}{
		{"short content", "short", 1},
		{"about 200 words", generateWords(200), 1},
		{"about 400 words", generateWords(400), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			node, _ := domain.NewFileNode(
				domain.MustURLPath("/test.md"),
				"Test",
				[]byte(tt.content),
				time.Now(),
				uint64(len(tt.content)),
			)

			got := node.ReadingTime()
			if got < tt.wantMinTime {
				t.Errorf("ReadingTime() = %d, want at least %d", got, tt.wantMinTime)
			}
		})
	}
}

func TestFileNode_Metadata(t *testing.T) {
	t.Parallel()
	node, _ := domain.NewFileNode(
		domain.MustURLPath("/test.md"),
		"Test",
		[]byte("content"),
		time.Now(),
		100,
	)

	meta := domain.Frontmatter{
		Title:       "Actual Title",
		Description: "A description",
		Author:      "John Doe",
		Draft:       false,
	}
	node.SetMetadata(meta)

	if node.Metadata().Title != "Actual Title" {
		t.Errorf("Metadata().Title = %q, want %q", node.Metadata().Title, "Actual Title")
	}

	if node.Title() != "Actual Title" {
		t.Errorf(
			"Title() = %q, want %q (should update from metadata)",
			node.Title(),
			"Actual Title",
		)
	}
}

func TestContentTree_NewContentTree(t *testing.T) {
	t.Parallel()
	root, _ := domain.NewDirectoryNode(domain.MustURLPath("/"), "Root", time.Now())
	tree := domain.NewContentTree(root)

	if tree.Root() != root {
		t.Error("ContentTree.Root() should return the root node")
	}
}

func TestContentTree_Find(t *testing.T) {
	t.Parallel()
	root, _ := domain.NewDirectoryNode(domain.MustURLPath("/"), "Root", time.Now())

	// Add structure: /subdir/file.md
	subdir, _ := domain.NewDirectoryNode(domain.MustURLPath("/subdir"), "SubDir", time.Now())
	root.AddChild(subdir)

	file, _ := domain.NewFileNode(
		domain.MustURLPath("/subdir/file.md"),
		"File",
		[]byte("content"),
		time.Now(),
		100,
	)
	subdir.AddChild(file)

	tree := domain.NewContentTree(root)

	tests := []struct {
		path string
		find bool
	}{
		{"/", true},
		{"/subdir", true},
		{"/subdir/file.md", true},
		{"/nonexistent", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			path, _ := domain.NewURLPath(tt.path)

			_, found := tree.Find(path)
			if found != tt.find {
				t.Errorf("Find(%q) = %v, want %v", tt.path, found, tt.find)
			}
		})
	}
}

func TestBuildBreadcrumbs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		path     string
		wantLen  int
		wantLast bool
	}{
		{name: "root path", path: "/", wantLen: 1, wantLast: true},
		{"/docs", "/docs", 2, true},
		{"/a/b/c", "/a/b/c", 4, true}, // home + a + b + c
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path, _ := domain.NewURLPath(tt.path)
			crumbs := domain.BuildBreadcrumbs(path)

			if len(crumbs) != tt.wantLen {
				t.Errorf(
					"BuildBreadcrumbs(%q) returned %d crumbs, want %d",
					tt.path,
					len(crumbs),
					tt.wantLen,
				)
			}

			if len(crumbs) > 0 && crumbs[len(crumbs)-1].IsActive != tt.wantLast {
				t.Errorf(
					"Last breadcrumb IsActive = %v, want %v",
					crumbs[len(crumbs)-1].IsActive,
					tt.wantLast,
				)
			}

			// First crumb should always be Home
			if len(crumbs) > 0 && crumbs[0].Title != "Home" {
				t.Errorf("First crumb title = %q, want %q", crumbs[0].Title, "Home")
			}
		})
	}
}

func TestNodeKind_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		kind domain.NodeKind
		want string
	}{
		{domain.NodeKindDirectory, "directory"},
		{domain.NodeKindFile, "file"},
		{domain.NodeKind(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.kind.String(); got != tt.want {
				t.Errorf("NodeKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

// Helper function to generate test words.
func generateWords(n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = "word"
	}

	return joinWords(words)
}

func joinWords(words []string) string {
	var result strings.Builder

	for i, w := range words {
		if i > 0 {
			result.WriteString(" ")
		}

		result.WriteString(w)
	}

	return result.String()
}
