package renderer

import (
	"strings"
	"testing"
)

// assertHTMLContains is a helper to assert rendered HTML contains expected content.
func assertHTMLContains(t *testing.T, html, expected, msg string) {
	t.Helper()

	if !strings.Contains(html, expected) {
		t.Errorf("%s: expected HTML to contain %q, got: %s", msg, expected, html)
	}
}

func TestNewGoldmarkRenderer(t *testing.T) {
	t.Parallel()

	renderer := NewGoldmarkRenderer()
	if renderer == nil {
		t.Fatal("NewGoldmarkRenderer() returned nil")
	}

	if renderer.md == nil {
		t.Error("NewGoldmarkRenderer() renderer.md is nil")
	}
}

func TestRenderBasicMarkdown(t *testing.T) {
	t.Parallel()

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
			t.Parallel()

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

func TestRenderResultStructure(t *testing.T) {
	t.Parallel()

	result := renderMust(t, NewGoldmarkRenderer(), `---
title: Test
---

## Section

Content.`)

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

func TestRenderCodeBlockWithLanguage(t *testing.T) {
	t.Parallel()

	renderer := NewGoldmarkRenderer()

	input := "```go\npackage main\n\nfunc main() {}\n```"

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	// Should contain syntax-highlighted code
	html := string(result.HTML)
	assertHTMLContains(t, html, "<pre", "code block")
	assertHTMLContains(t, html, "<code", "code element")
}

func TestRenderTable(t *testing.T) {
	t.Parallel()

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
	assertHTMLContains(t, html, "<table>", "table")
	assertHTMLContains(t, html, "<thead>", "table head")
	assertHTMLContains(t, html, "<tbody>", "table body")
}

// Integration tests

func TestRenderComplexDocument(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	result := renderMust(t, NewGoldmarkRenderer(), `---
title: Test
tags:
  - one
  - two
---

## Section
`)

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
