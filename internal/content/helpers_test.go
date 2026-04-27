package content

import (
	"testing"

	"github.com/larsartmann/dynamic-markdown-site/internal/domain"
	"github.com/larsartmann/dynamic-markdown-site/internal/test"
)

func TestShouldSkipDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		dir  string
		want bool
	}{
		{name: "node_modules", dir: "node_modules", want: true},
		{name: ".git", dir: ".git", want: true},
		{name: "vendor", dir: "vendor", want: true},
		{name: "dist", dir: "dist", want: true},
		{name: "build", dir: "build", want: true},
		{name: "tmp", dir: "tmp", want: true},
		{name: "temp", dir: "temp", want: true},
		{name: "normal dir", dir: "docs", want: false},
		{name: "src", dir: "src", want: false},
		{name: "case insensitive", dir: "Node_Modules", want: true},
		{name: "case insensitive git", dir: ".GIT", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := ShouldSkipDir(tt.dir)
			if got != tt.want {
				t.Errorf("shouldSkipDir(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestIsMarkdownFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		file string
		want bool
	}{
		{name: ".md extension", file: "readme.md", want: true},
		{name: ".markdown extension", file: "readme.markdown", want: true},
		{name: "uppercase .MD", file: "readme.MD", want: true},
		{name: "uppercase .Markdown", file: "readme.Markdown", want: true},
		{name: ".txt extension", file: "notes.txt", want: false},
		{name: "no extension", file: "README", want: false},
		{name: ".go extension", file: "main.go", want: false},
		{name: "nested path .md", file: "docs/guide.md", want: true},
		{name: ".md in filename not extension", file: "readme.md.bak", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := IsMarkdownFile(tt.file)
			if got != tt.want {
				t.Errorf("isMarkdownFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestFilterEmptyDirectories(t *testing.T) {
	t.Parallel()

	t.Run("directory with file is kept", func(t *testing.T) {
		t.Parallel()

		dir := test.NewTestDir(t, "/docs")
		file := test.NewTestFile(t, "/docs/guide.md", "content")
		dir.AddChild(file)

		result := filterEmptyDirectories(dir)

		if !result {
			t.Error("filterEmptyDirectories should return true for dir with files")
		}

		if len(dir.Children()) != 1 {
			t.Errorf("expected 1 child, got %d", len(dir.Children()))
		}
	})

	t.Run("empty directory is pruned from parent", func(t *testing.T) {
		t.Parallel()

		parent := test.NewTestDir(t, "/root")
		emptyDir := test.NewTestDir(t, "/root/empty")
		file := test.NewTestFile(t, "/root/readme.md", "content")

		parent.AddChild(emptyDir)
		parent.AddChild(file)

		result := filterEmptyDirectories(parent)

		if !result {
			t.Error("parent should have markdown after filtering")
		}

		if len(parent.Children()) != 1 {
			t.Errorf("empty subdir should be pruned, got %d children", len(parent.Children()))
		}
	})

	t.Run("nested empty directories are pruned", func(t *testing.T) {
		t.Parallel()

		root := test.NewTestDir(t, "/")
		level1 := test.NewTestDir(t, "/level1")
		level2 := test.NewTestDir(t, "/level1/level2")

		root.AddChild(level1)
		level1.AddChild(level2)

		filterEmptyDirectories(root)

		if len(root.Children()) != 0 {
			t.Errorf("all empty dirs should be pruned, got %d children", len(root.Children()))
		}
	})

	t.Run("nested dir with file keeps parent chain", func(t *testing.T) {
		t.Parallel()

		root := test.NewTestDir(t, "/")
		subdir := test.NewTestDir(t, "/docs")
		file := test.NewTestFile(t, "/docs/guide.md", "content")

		root.AddChild(subdir)
		subdir.AddChild(file)

		filterEmptyDirectories(root)

		if len(root.Children()) != 1 {
			t.Errorf("root should keep 1 subdir, got %d", len(root.Children()))
		}

		sub, ok := root.Children()[0].(*domain.DirectoryNode)
		if !ok {
			t.Fatal("root child should be a DirectoryNode")
		}

		if len(sub.Children()) != 1 {
			t.Errorf("subdir should keep 1 file, got %d", len(sub.Children()))
		}
	})
}
