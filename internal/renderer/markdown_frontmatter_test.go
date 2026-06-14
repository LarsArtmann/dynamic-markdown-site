package renderer

import "testing"

func TestRenderWithFrontmatter(t *testing.T) {
	t.Parallel()

	renderer := NewGoldmarkRenderer()

	input := `---
title: Test Document
description: A test document
author: Test Author
tags:
  - tag1
  - tag2
draft: false
---

# Hello

Content.`

	result, err := renderer.Render([]byte(input))
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	if result.Metadata.Title != "Test Document" {
		t.Errorf("Metadata.Title = %q, want %q", result.Metadata.Title, "Test Document")
	}
	if result.Metadata.Description != "A test document" {
		t.Errorf("Metadata.Description = %q, want %q", result.Metadata.Description, "A test document")
	}
	if result.Metadata.Author != "Test Author" {
		t.Errorf("Metadata.Author = %q, want %q", result.Metadata.Author, "Test Author")
	}
	if len(result.Metadata.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Metadata.Tags))
	}
}

func TestRenderFrontmatterMissing(t *testing.T) {
	t.Parallel()

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
}
