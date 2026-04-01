package content

import (
	"strings"
)

func isDraft(content []byte) bool {
	text := string(content)
	if !strings.HasPrefix(text, "---") {
		return false
	}

	end := strings.Index(text[3:], "---")
	if end == -1 {
		return false
	}

	frontmatter := text[3 : end+3]
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "draft: true" {
			return true
		}
	}

	return false
}
