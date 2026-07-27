---
title: "The render pipeline: *render.go*"
label: render.go
kind: Code walk
order: 2
layout: code-walk
summary: One file turns an authored step into HTML. Read the summary first — expand only if you want the annotated source.
---
- Reads a step's YAML frontmatter to pick a template (`overview`, `code-walk`, `config`).
- Parses the markdown body with goldmark, resolving custom directives (deep-dive, glossary terms, two-level code).
- Returns a rendered HTML fragment the site assembler embeds into `index.html`.

```go path="internal/render/render.go" mark=2,5,7
func RenderStep(s *Step) (string, error) {
    fm, body := splitFrontmatter(s.Raw)
    tmpl, err := templateFor(fm.Layout)
    if err != nil { return "", err }
    html := goldmark.Convert(body, WithDirectives())
    return tmpl.Exec(fm, html)
} // one concept per step — no walls of text
```
1. Frontmatter is parsed before the body — it decides the template and disclosure defaults before a single word of markdown is touched.
2. Custom directives (`:::deep`, glossary spans, two-level code fences) are goldmark extensions, not a second parser.
3. The whole function is deliberately small — it's the enforcement point for "never a wall of text."
