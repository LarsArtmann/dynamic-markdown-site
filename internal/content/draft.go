package content

import (
	"strings"

	"gopkg.in/yaml.v3"
)

type frontmatter struct {
	Draft bool `yaml:"draft"`
}

func isDraft(content []byte) bool {
	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return false
	}

	end := strings.Index(text[3:], "---")
	if end == -1 {
		return false
	}

	frontmatterText := strings.TrimSpace(text[3 : end+3])
	if frontmatterText == "" {
		return false
	}

	var fm frontmatter
	err := yaml.Unmarshal([]byte(frontmatterText), &fm)
	if err != nil {
		return false
	}

	return fm.Draft
}
