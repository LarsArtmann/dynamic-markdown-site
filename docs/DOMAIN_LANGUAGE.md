# Domain Language

A **Unified Language** for Dynamic Markdown Site — shared across Developer, Reviewer, and AI.
Inspired by Domain-Driven Design (DDD) Ubiquitous Language.

## Glossary

| Term              | Definition                                                             | Context                          |
| ----------------- | ---------------------------------------------------------------------- | -------------------------------- |
| Content Node      | A node in the content tree — either a Directory or a File              | Core domain concept              |
| Directory Node    | A container of child content nodes, mapped from a filesystem directory | Navigation, tree structure       |
| File Node         | A markdown file with raw content, awaiting rendering                   | Content storage                  |
| Rendered File     | An immutable File Node combined with its rendered HTML, TOC, and metadata | Presentation layer           |
| URL Path          | A validated, traversal-safe URL path value object                      | HTTP routing, security           |
| Content Tree      | A hierarchical index of all content nodes, built from filesystem walk  | Repository layer                 |
| Repository        | The interface for content storage and retrieval (filesystem or blob)   | Storage abstraction              |
| Refresh           | Rebuilding the content tree from the source (filesystem or blob)      | Content lifecycle                |
| Frontmatter       | YAML metadata at the top of a markdown file (title, tags, draft, etc.) | File metadata                    |
| Draft             | A file marked `draft: true` in frontmatter, excluded from the site    | Content filtering                |
| TOC Item          | A table of contents entry with level, title, and anchor                | Navigation within a page         |
| Breadcrumb        | A navigation trail from root to the current page                       | Site navigation                  |
| Suggested Path    | A path similar to a 404 request, scored by Levenshtein distance        | Error recovery                   |
| Raw File          | A non-markdown file served directly (images, PDFs, fonts)              | Asset serving                    |
| Search Result     | A content node matched by query, with score and highlighted title      | Search feature                   |
| Live Reload       | SSE-based browser refresh triggered by filesystem changes in dev mode  | Developer experience             |
| HTML Cache        | An in-memory cache of rendered HTML content keyed by URL path          | Performance                      |

## Entities

| Term          | Definition                                           | Context                 |
| ------------- | ---------------------------------------------------- | ----------------------- |
| DirectoryNode | A directory with path, title, children, modified time | Tree structure, navigation |
| FileNode      | A markdown file with path, title, content, size      | Content storage         |
| RenderedFile  | A FileNode + rendered HTML + TOC + metadata          | Presentation            |

## Value Objects

| Term            | Definition                                    | Context                          |
| --------------- | --------------------------------------------- | -------------------------------- |
| URLPath         | A validated, safe URL path string             | Routing, security                |
| HTML            | A type marking pre-escaped HTML content       | Template rendering               |
| NodeKind        | An enum: Directory or File                    | Type discrimination              |
| Frontmatter     | Immutable metadata extracted from YAML header | File metadata                    |
| TOCItem         | A heading entry with level, title, anchor     | Navigation                       |
| Breadcrumb      | A path segment with title and active state    | Navigation                       |
| RefreshResult   | Statistics from a repository refresh operation| Operational                      |
| SuggestedPath   | A URLPath with similarity score               | 404 error recovery               |
| SearchResult    | A content match with score and snippet        | Search feature                   |
| RawFile         | File bytes with content type and modification time | Asset serving                |

## Events

| Term               | Definition                                        | Context                     |
| ------------------- | ------------------------------------------------- | --------------------------- |
| Content Refresh     | The content tree was rebuilt from source           | Repository lifecycle        |
| Cache Invalidation  | All cached rendered content was cleared            | Cache management            |
| Live Reload Notify  | A filesystem change was pushed to browsers via SSE | Dev mode                    |

## Commands

| Term              | Definition                                         | Context                  |
| ----------------- | -------------------------------------------------- | ------------------------ |
| Render            | Convert raw markdown bytes to HTML + metadata       | Rendering pipeline       |
| Refresh           | Rebuild content tree from filesystem or blob        | Repository management    |
| Search            | Find content nodes matching a text query            | Search feature           |
| Invalidate Cache  | Clear all cached rendered HTML                      | Cache management         |

## Bounded Contexts

| Context         | Description                                                    |
| --------------- | -------------------------------------------------------------- |
| Content Storage | Loading, indexing, and serving markdown and raw files          |
| Rendering       | Converting markdown to HTML with extensions (diagrams, alerts) |
| HTTP Serving    | Routing, middleware, response rendering                        |
| Search          | Full-text search over content titles and bodies                |
| Caching         | In-memory caching of rendered HTML                             |
| Dev Experience  | File watching, live reload, dev mode                           |

---

> **How to use this file:**
>
> - Keep terms concise — one clear sentence per definition
> - Update when new domain concepts emerge
> - Use these terms consistently in code, docs, and conversations
> - When in doubt about a word's meaning, check here first
