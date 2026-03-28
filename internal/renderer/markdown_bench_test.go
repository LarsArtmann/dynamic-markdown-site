package renderer

import (
	"strings"
	"testing"
)

// createBenchmarkMarkdownSource creates markdown content of varying complexity.
func createBenchmarkMarkdownSource(complexity string) []byte {
	switch complexity {
	case "simple":
		return []byte(`# Simple Document

This is a simple paragraph with no special formatting.

## Section 1

Another paragraph here.
`)
	case "medium":
		return []byte(`---
title: Medium Complexity Document
description: A document with various markdown elements
author: Test Author
tags:
  - benchmark
  - testing
date: 2026-02-20
---

# Medium Document

## Introduction

This document has *italic*, **bold**, and ` + "`code`" + ` inline formatting.

### Lists

- Item 1
- Item 2
- Item 3

1. Numbered 1
2. Numbered 2
3. Numbered 3

### Code Block

` + "```go" + `
package main

func main() {
    println("Hello")
}
` + "```" + `

### Table

| A | B | C |
|---|---|---|
| 1 | 2 | 3 |
| 4 | 5 | 6 |

### Links

[Example](https://example.com)

### Blockquote

> This is a quote.
`)
	case "complex":
		return []byte(`---
title: Complex Document
description: A complex document with many elements for benchmarking
author: Benchmark Test
tags:
  - benchmark
  - testing
  - performance
  - markdown
date: 2026-02-20
draft: false
---

# Complex Document

## Introduction

This is a **complex** document with *many* ` + "`elements`" + ` for benchmarking the markdown renderer.

### Text Formatting

We have **bold**, *italic*, ` + "`inline code`" + `, ~~strikethrough~~, and [links](https://example.com).

### Lists

Unordered list:
- Item 1 with some text
- Item 2 with more text
- Item 3 with even more text
  - Nested item 1
  - Nested item 2
- Item 4

Ordered list:
1. First item
2. Second item
3. Third item
   1. Nested first
   2. Nested second
4. Fourth item

Task list:
- [x] Completed task
- [ ] Incomplete task
- [x] Another completed task

### Code Blocks

Go code:

` + "```go" + `
package main

import (
    "fmt"
    "net/http"
)

type Server struct {
    addr string
    mux  *http.ServeMux
}

func NewServer(addr string) *Server {
    s := &Server{
        addr: addr,
        mux:  http.NewServeMux(),
    }
    s.routes()
    return s
}

func (s *Server) routes() {
    s.mux.HandleFunc("/", s.handleIndex)
    s.mux.HandleFunc("/api", s.handleAPI)
}

func (s *Server) Run() error {
    return http.ListenAndServe(s.addr, s.mux)
}

func main() {
    s := NewServer(":8080")
    fmt.Println("Starting server on :8080")
    s.Run()
}
` + "```" + `

JavaScript code:

` + "```javascript" + `
const express = require('express');
const app = express();

app.get('/', (req, res) => {
    res.json({ message: 'Hello World' });
});

app.listen(3000, () => {
    console.log('Server running on port 3000');
});
` + "```" + `

Python code:

` + "```python" + `
from flask import Flask, jsonify

app = Flask(__name__)

@app.route('/')
def index():
    return jsonify({'message': 'Hello World'})

if __name__ == '__main__':
    app.run(port=5000)
` + "```" + `

### Tables

Simple table:

| Name  | Age | City     |
|-------|-----|----------|
| Alice | 30  | New York |
| Bob   | 25  | London   |
| Carol | 35  | Tokyo    |

Complex table:

| Feature        | Go | Python | JavaScript |
|----------------|-----|--------|------------|
| Static Typing  | Yes | No     | No         |
| Garbage Col.   | Yes | Yes    | Yes        |
| Concurrency    | Yes | Limited| Async      |
| Performance    | High| Medium | Medium     |

### Blockquotes

> This is a simple blockquote.

> This is a longer blockquote.
> It spans multiple lines.
>
> And has a second paragraph.
>
> — Attribution

### Links

[Basic link](https://example.com)

[Link with title](https://example.com "Example Site")

<https://example.com>

### Images

![Alt text](https://example.com/image.png)

![Image with title](https://example.com/logo.png "Logo")

### Definition Lists

Term 1
: Definition 1

Term 2
: Definition 2a
: Definition 2b

### Footnotes

This text has a footnote[^1].

[^1]: This is the footnote content.

### Horizontal Rule

---

## Conclusion

This document tests many markdown features for comprehensive benchmarking.
`)
	default:
		return []byte("# Empty\n\nNo content.")
	}
}

// BenchmarkRenderSimple benchmarks rendering simple markdown.
func BenchmarkRenderSimple(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	source := createBenchmarkMarkdownSource("simple")

	for b.Loop() {
		_, _ = renderer.Render(source)
	}
}

// BenchmarkRenderMedium benchmarks rendering medium complexity markdown.
func BenchmarkRenderMedium(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	source := createBenchmarkMarkdownSource("medium")

	for b.Loop() {
		_, _ = renderer.Render(source)
	}
}

// BenchmarkRenderComplex benchmarks rendering complex markdown.
func BenchmarkRenderComplex(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	source := createBenchmarkMarkdownSource("complex")

	for b.Loop() {
		_, _ = renderer.Render(source)
	}
}

// BenchmarkRenderWithCodeHighlighting specifically tests syntax highlighting performance.
func BenchmarkRenderWithCodeHighlighting(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	source := []byte(`---
title: Code Heavy Document
---

# Code Examples

` + "```go" + `
package main

import "fmt"

func main() {
    fmt.Println("Hello")
}
` + "```" + `

` + "```python" + `
def main():
    print("Hello")

if __name__ == "__main__":
    main()
` + "```" + `

` + "```javascript" + `
function main() {
    console.log("Hello");
}
main();
` + "```" + `
`)

	for b.Loop() {
		_, _ = renderer.Render(source)
	}
}

// buildTOCMarkdown generates markdown content with the specified number of headings.
func buildTOCMarkdown(numHeadings int) string {
	var sb strings.Builder
	for i := range numHeadings {
		depth := (i % 4) + 1
		for range depth {
			sb.WriteString("#")
		}
		sb.WriteString(" Heading " + string(rune('0'+i%10)) + "\n\nContent for section.\n\n")
	}
	return sb.String()
}

// BenchmarkRenderWithLargeTOC benchmarks documents with many headings.
func BenchmarkRenderWithLargeTOC(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	source := buildTOCMarkdown(100)

	for b.Loop() {
		_, _ = renderer.Render([]byte(source))
	}
}



// BenchmarkRenderConcurrent benchmarks concurrent rendering.
func BenchmarkRenderConcurrent(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	sources := [][]byte{
		createBenchmarkMarkdownSource("simple"),
		createBenchmarkMarkdownSource("medium"),
		createBenchmarkMarkdownSource("complex"),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = renderer.Render(sources[i%len(sources)])
			i++
		}
	})
}

// BenchmarkRenderWithFrontmatter benchmarks frontmatter parsing.
func BenchmarkRenderWithFrontmatter(b *testing.B) {
	renderer := NewGoldmarkRenderer()
	source := []byte(`---
title: Document with Frontmatter
description: A document with YAML frontmatter
author: Test Author
tags:
  - tag1
  - tag2
  - tag3
date: 2026-02-20
custom_field: custom_value
---

# Content

This is the body content.
`)

	for b.Loop() {
		result, _ := renderer.Render(source)
		_ = result.Metadata
	}
}
