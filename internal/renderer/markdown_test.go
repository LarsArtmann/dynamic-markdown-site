package renderer

import (
	"strings"
	"testing"
)

func TestNewGoldmarkRenderer(t *testing.T) {
	renderer := NewGoldmarkRenderer()
	if renderer == nil {
		t.Fatal("NewGoldmarkRenderer() returned nil")
	}

	if renderer.md == nil {
		t.Error("NewGoldmarkRenderer() renderer.md is nil")
	}
}

func TestRenderBasicMarkdown(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	tests := []struct {
		name           string
		input          string
		shouldContain  string
		shouldNotMatch string
	}{
		{
			name:          "simple paragraph",
			input:         "Hello World",
			shouldContain: "<p>",
		},
		{
			name:          "heading",
			input:         "# Title",
			shouldContain: "<h1",
		},
		{
			name:          "h2 heading",
			input:         "## Section",
			shouldContain: "<h2",
		},
		{
			name:          "bold text",
			input:         "**bold**",
			shouldContain: "<strong>",
		},
		{
			name:          "italic text",
			input:         "*italic*",
			shouldContain: "<em>",
		},
		{
			name:          "code inline",
			input:         "`code`",
			shouldContain: "<code>",
		},
		{
			name:          "link",
			input:         "[Example](https://example.com)",
			shouldContain: "<a href",
		},
		{
			name:          "unordered list",
			input:         "- item 1\n- item 2",
			shouldContain: "<ul>",
		},
		{
			name:          "ordered list",
			input:         "1. first\n2. second",
			shouldContain: "<ol>",
		},
		{
			name:          "code block",
			input:         "```\ncode\n```",
			shouldContain: "<pre>",
		},
		{
			name:          "block quote",
			input:         "> quote",
			shouldContain: "<blockquote>",
		},
		{
			name:          "table",
			input:         "| A | B |\n|---|---|\n| 1 | 2 |",
			shouldContain: "<table>",
		},
		{
			name:          "strikethrough",
			input:         "~~deleted~~",
			shouldContain: "<del>",
		},
		{
			name:          "task list checked",
			input:         "- [x] done",
			shouldContain: "checked",
		},
		{
			name:          "task list unchecked",
			input:         "- [ ] todo",
			shouldContain: "<input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := renderer.Render([]byte(tt.input))
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}

			if !strings.Contains(string(result.HTML), tt.shouldContain) {
				t.Errorf("Render() HTML should contain %q, got: %s", tt.shouldContain, result.HTML)
			}
		})
	}
}

func TestRenderWithFrontmatter(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `---
title: Test Document
description: A test document
author: Test Author
tags:
  - tag1
  - tag2
draft: true
---

# Content

Body text here.`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if result.Metadata.Title != "Test Document" {
		t.Errorf("Metadata.Title = %q, want %q", result.Metadata.Title, "Test Document")
	}

	if result.Metadata.Description != "A test document" {
		t.Errorf(
			"Metadata.Description = %q, want %q",
			result.Metadata.Description,
			"A test document",
		)
	}

	if result.Metadata.Author != "Test Author" {
		t.Errorf("Metadata.Author = %q, want %q", result.Metadata.Author, "Test Author")
	}

	if len(result.Metadata.Tags) != 2 {
		t.Errorf("Metadata.Tags length = %d, want 2", len(result.Metadata.Tags))
	}

	if result.Metadata.Draft != true {
		t.Error("Metadata.Draft should be true")
	}
}

func TestRenderFrontmatterMissing(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `# No Frontmatter

Just content.`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if result.Metadata.Title != "" {
		t.Errorf("Metadata.Title should be empty, got %q", result.Metadata.Title)
	}

	if result.Metadata.Description != "" {
		t.Errorf("Metadata.Description should be empty, got %q", result.Metadata.Description)
	}

	if result.Metadata.Author != "" {
		t.Errorf("Metadata.Author should be empty, got %q", result.Metadata.Author)
	}

	if len(result.Metadata.Tags) != 0 {
		t.Errorf("Metadata.Tags should be empty, got %v", result.Metadata.Tags)
	}
}

func TestRenderTOC(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `# Main Title

## Section 1

Content here.

### Subsection 1.1

More content.

## Section 2

### Subsection 2.1

#### Deep nesting

## Section 3
`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Should have 3 main sections (h2 level, h1 is skipped)
	if len(result.TOC) != 3 {
		t.Errorf("TOC length = %d, want 3", len(result.TOC))
	}

	// Check first section
	if result.TOC[0].Title != "Section 1" {
		t.Errorf("TOC[0].Title = %q, want %q", result.TOC[0].Title, "Section 1")
	}

	if result.TOC[0].Level != 2 {
		t.Errorf("TOC[0].Level = %d, want 2", result.TOC[0].Level)
	}

	if result.TOC[0].Anchor == "" {
		t.Error("TOC[0].Anchor should not be empty")
	}

	// Check nested items in Section 1
	if len(result.TOC[0].Children) != 1 {
		t.Errorf("TOC[0].Children length = %d, want 1", len(result.TOC[0].Children))
	} else {
		if result.TOC[0].Children[0].Title != "Subsection 1.1" {
			t.Errorf(
				"TOC[0].Children[0].Title = %q, want %q",
				result.TOC[0].Children[0].Title,
				"Subsection 1.1",
			)
		}

		if result.TOC[0].Children[0].Level != 3 {
			t.Errorf("TOC[0].Children[0].Level = %d, want 3", result.TOC[0].Children[0].Level)
		}
	}

	// Check Section 2 has nested subsection with deep nesting
	if len(result.TOC[1].Children) != 1 {
		t.Errorf("TOC[1].Children length = %d, want 1", len(result.TOC[1].Children))
	} else if len(result.TOC[1].Children[0].Children) != 1 {
		t.Errorf(
			"TOC[1].Children[0].Children length = %d, want 1 (deep nesting)",
			len(result.TOC[1].Children[0].Children),
		)
	}
}

func TestRenderTOCEmpty(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `Just a paragraph with no headings.`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if len(result.TOC) != 0 {
		t.Errorf("TOC length = %d, want 0 (no headings)", len(result.TOC))
	}
}

func TestGenerateAnchorID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple Title", "simple-title"},
		{"Title With_Underscore", "title-with-underscore"},
		{"Title-With-Dashes", "title-with-dashes"},
		{"UPPERCASE TITLE", "uppercase-title"},
		{"Title 123", "title-123"},
		{"Title!@#$%^&*()", "title"},
		{"  Spaces  ", "--spaces--"},
		{"Title-With_Multiple___Dashes", "title-with-multiple---dashes"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := generateAnchorID(tt.input)
			if result != tt.expected {
				t.Errorf("generateAnchorID(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractHeadingText(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	// Parse a simple heading to get the AST node
	input := "## Test Heading\n\nContent."

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Verify the TOC extracted the correct text
	if len(result.TOC) != 1 {
		t.Fatalf("Expected 1 TOC item, got %d", len(result.TOC))
	}

	if result.TOC[0].Title != "Test Heading" {
		t.Errorf("TOC[0].Title = %q, want %q", result.TOC[0].Title, "Test Heading")
	}
}

func TestRenderResultStructure(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `---
title: Test
---

## Section

Content.`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Verify all fields are populated
	if result.HTML == "" {
		t.Error("RenderResult.HTML should not be empty")
	}

	if len(result.TOC) == 0 {
		t.Error("RenderResult.TOC should not be empty")
	}

	if result.Metadata.Title != "Test" {
		t.Errorf("RenderResult.Metadata.Title = %q, want %q", result.Metadata.Title, "Test")
	}
}

func TestRenderEmptyInput(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	result, err := renderer.Render([]byte(""))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if result.HTML != "" {
		t.Errorf("Render() HTML should be empty for empty input, got: %s", result.HTML)
	}
}

func TestRenderCodeBlockWithLanguage(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := "```go\npackage main\n\nfunc main() {}\n```"

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Should contain syntax-highlighted code
	html := string(result.HTML)
	if !strings.Contains(html, "<pre") {
		t.Error("Should contain <pre> tag")
	}

	if !strings.Contains(html, "<code") {
		t.Error("Should contain <code> tag")
	}
}

func TestRenderTable(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 25 |`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	html := string(result.HTML)
	if !strings.Contains(html, "<table>") {
		t.Error("Should contain <table> tag")
	}

	if !strings.Contains(html, "<thead>") {
		t.Error("Should contain <thead> tag")
	}

	if !strings.Contains(html, "<tbody>") {
		t.Error("Should contain <tbody> tag")
	}
}

// SimpleRenderer tests

func TestNewSimpleRenderer(t *testing.T) {
	renderer := NewSimpleRenderer()
	if renderer == nil {
		t.Fatal("NewSimpleRenderer() returned nil")
	}

	if renderer.md == nil {
		t.Error("NewSimpleRenderer() renderer.md is nil")
	}
}

func TestSimpleRendererRender(t *testing.T) {
	renderer := NewSimpleRenderer()

	input := "# Title\n\nParagraph with **bold**."

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("SimpleRenderer.Render() error: %v", err)
	}

	html := string(result)
	if !strings.Contains(html, "<h1") {
		t.Error("Should contain <h1> tag")
	}

	if !strings.Contains(html, "<strong>") {
		t.Error("Should contain <strong> tag")
	}
}

// assertSimpleRendererContains renders markdown and asserts the result contains expected string.
func assertSimpleRendererContains(t *testing.T, input, expected, errMsg string) {
	t.Helper()

	renderer := NewSimpleRenderer()

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("SimpleRenderer.Render() error: %v", err)
	}

	if !strings.Contains(string(result), expected) {
		t.Error(errMsg)
	}
}

func TestSimpleRendererTable(t *testing.T) {
	assertSimpleRendererContains(
		t,
		"| A | B |\n|---|---|\n| 1 | 2 |",
		"<table>",
		"SimpleRenderer should support tables",
	)
}

func TestSimpleRendererStrikethrough(t *testing.T) {
	assertSimpleRendererContains(
		t,
		"~~strikethrough~~",
		"<del>",
		"SimpleRenderer should support strikethrough",
	)
}

// Integration tests

func TestRenderComplexDocument(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `---
title: Complex Document
description: A complex test document
author: Test Suite
tags:
  - integration
  - test
draft: false
---

# Complex Document

## Introduction

This document tests *multiple* **features**.

### Lists

- Item 1
- Item 2
  - Nested item

### Code

` + "```go" + `
func main() {
    println("Hello")
}
` + "```" + `

### Table

| A | B |
|---|---|
| 1 | 2 |

### Quote

> This is a quote.

### Links

[Example](https://example.com)

---

## Conclusion

The end.
`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Check metadata
	if result.Metadata.Title != "Complex Document" {
		t.Errorf("Title = %q, want %q", result.Metadata.Title, "Complex Document")
	}

	// Check TOC
	if len(result.TOC) < 2 {
		t.Errorf("Expected at least 2 TOC items, got %d", len(result.TOC))
	}

	// Check HTML content
	html := string(result.HTML)

	expectedElements := []string{
		"<h1", "<h2", "<h3",
		"<ul>", "<li>",
		"<pre", "<code",
		"<table>",
		"<blockquote>",
		"<a href",
		"<hr",
		"<em>", "<strong>",
	}
	for _, elem := range expectedElements {
		if !strings.Contains(html, elem) {
			t.Errorf("HTML should contain %q", elem)
		}
	}
}

// Domain type verification

func TestRenderResultImplementsDomainTypes(t *testing.T) {
	renderer := NewGoldmarkRenderer()

	input := `---
title: Test
tags:
  - one
  - two
---

## Section
`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Verify domain.Frontmatter fields
	_ = result.Metadata

	// Verify domain.TOCItem fields
	if len(result.TOC) > 0 {
		item := result.TOC[0]
		_ = item.Title
		_ = item.Level
		_ = item.Anchor
		_ = item.Children
	}
}
