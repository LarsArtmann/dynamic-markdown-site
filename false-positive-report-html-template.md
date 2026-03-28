# False Positive Report: html/template Detection

**Date:** 2026-03-27  
**Policy:** library-policy  
**Severity:** Moderate (low actual risk)

## Detection Summary

Policy flagged 4 files importing `html/template`:

| File                            | Line | Usage                                    |
| ------------------------------- | ---- | ---------------------------------------- |
| `internal/cache/html.go`        | 6    | Type declaration: `template.HTML`        |
| `internal/cache/html_test.go`   | 5    | Type cast: `template.HTML(html)`         |
| `internal/domain/file.go`       | 4    | Type declaration: `template.HTML`        |
| `internal/renderer/markdown.go` | 6    | Type cast: `template.HTML(buf.String())` |

## Why This Is a False Positive

### Actual Usage Pattern

The codebase uses `template.HTML` **only as a type marker** - it's `type HTML = string` (a string alias).

```go
// No template parsing
type RenderedContent struct {
    HTML template.HTML  // Just a string with semantic meaning
}

// No template parsing
return RenderResult{
    HTML: template.HTML(buf.String()),  // Simple string cast
}
```

### What Policy Assumes vs Reality

| Policy Assumes                              | Actual Behavior                           |
| ------------------------------------------- | ----------------------------------------- |
| Using `html/template` for HTML generation   | Using only the `HTML` type alias          |
| Template injection vulnerabilities possible | No template rendering - just type marking |
| Runtime HTML escaping needed                | No HTML generation from templates         |

### Security Impact

**Zero.** The code:

- Never calls `template.Parse()`, `template.Execute()`, or any rendering functions
- Never accepts user input into templates
- Only uses `template.HTML` to mark strings as pre-escaped HTML

## Root Cause

The policy detects `html/template` imports without analyzing whether template parsing/rendering actually occurs. The `HTML` type alias is exported from `html/template` but doesn't carry any of the package's security risks when used in isolation.

## Recommendation

1. **Update policy** to distinguish between:
   - `html/template` for template rendering (banned)
   - `html/template` for type aliases only (allowed)

2. **Alternative**: Define a project-local type to eliminate the dependency entirely:

```go
// internal/domain/types.go
package domain

// HTML represents pre-escaped HTML content.
type HTML string
```

3. **Risk level**: Low - consider downgrading `html/template` to "warn" rather than "ban" in future policy versions, since type-only usage is safe.

## Conclusion

This is a **false positive**. The policy correctly identifies the import but incorrectly assumes dangerous usage. The code poses no security risk and the `html/template` package is used only for its exported type alias.
