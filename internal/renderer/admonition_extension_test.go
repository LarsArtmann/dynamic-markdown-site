package renderer

import (
	"strings"
	"testing"
)

func TestAdmonitionExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		shouldContain  []string
		shouldNotMatch string
	}{
		{
			name: "critical alert",
			input: "> [!CRITICAL]\n> Never manually add users via Google Cloud Console.",
			shouldContain: []string{
				`class="admonition admonition-critical"`,
				`class="admonition-title"`,
				"Critical",
				"Never manually add users via Google Cloud Console.",
			},
		},
		{
			name: "note alert",
			input: "> [!NOTE]\n> This is a note.",
			shouldContain: []string{
				`class="admonition admonition-note"`,
				"Note",
				"This is a note.",
			},
		},
		{
			name: "warning alert",
			input: "> [!WARNING]\n> Be careful!",
			shouldContain: []string{
				`class="admonition admonition-warning"`,
				"Warning",
				"Be careful!",
			},
		},
		{
			name: "tip alert",
			input: "> [!TIP]\n> Use Terraform.",
			shouldContain: []string{
				`class="admonition admonition-tip"`,
				"Tip",
				"Use Terraform.",
			},
		},
		{
			name: "important alert",
			input: "> [!IMPORTANT]\n> Read this first.",
			shouldContain: []string{
				`class="admonition admonition-important"`,
				"Important",
				"Read this first.",
			},
		},
		{
			name: "caution alert",
			input: "> [!CAUTION]\n> Proceed with caution.",
			shouldContain: []string{
				`class="admonition admonition-caution"`,
				"Caution",
				"Proceed with caution.",
			},
		},
		{
			name: "multiline alert",
			input: "> [!WARNING]\n> First line.\n> Second line.",
			shouldContain: []string{
				`class="admonition admonition-warning"`,
				"First line.",
				"Second line.",
			},
		},
		{
			name: "regular blockquote unchanged",
			input: "> This is just a regular blockquote.",
			shouldNotMatch: `class="admonition`,
		},
		{
			name: "alert with inline code",
			input: "> [!NOTE]\n> Use `terraform apply` to deploy.",
			shouldContain: []string{
				`class="admonition admonition-note"`,
				"<code>",
			},
		},
	}

	r := NewGoldmarkRenderer()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := r.Render([]byte(tt.input))
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}

			html := string(result.HTML)

			for _, expected := range tt.shouldContain {
				if !strings.Contains(html, expected) {
					t.Errorf("expected HTML to contain %q\nGot: %s", expected, html)
				}
			}

			if tt.shouldNotMatch != "" && strings.Contains(html, tt.shouldNotMatch) {
				t.Errorf("expected HTML NOT to contain %q\nGot: %s", tt.shouldNotMatch, html)
			}
		})
	}
}
