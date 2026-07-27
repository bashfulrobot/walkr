# Worked example: walkr's own dogfood walkthrough

This is `walkr`'s own `.walkr/` directory, reproduced here verbatim as a
known-good pattern to imitate. It is small on purpose (4 steps) — real walkthroughs will
have more steps per layer, but the shape (one flat `order` sequence, big-picture →
code-walk → config → recap) and the exact syntax of every directive are what to copy.

Read this alongside `references/content-format.md` (the rules) — this file is the
"what compliant output actually looks like" companion.

## `walkthrough.yaml`

```yaml
title: Walkr
tagline: Field Guide
repo: bashfulrobot/walkr
```

## `glossary.yaml`

```yaml
walkr:
  term: walkr
  definition: >
    A CLI that renders a hand-authored markdown walkthrough into a static,
    wizard-style teaching site. It never generates content itself — only
    renders it.

render-pipeline:
  term: render pipeline
  definition: >
    The goldmark-based Go code (internal/render) that parses a step's
    frontmatter and body, then emits the HTML fragment the wizard swaps in.

content-format-spec:
  term: content-format spec
  definition: >
    The single contract (docs/ai/content-format.md) both the Go renderer and the
    authoring skill obey — derived from what the Phase 0 prototype actually
    needed, nothing speculative.
  learn_more: https://github.com/bashfulrobot/walkr/blob/main/docs/ai/content-format.md
```

Note the top-level keys (`walkr`, `render-pipeline`, `content-format-spec`) are the
`def=` identifiers used in the body text — they don't have to match the visible bracketed
text or the `term:` field word-for-word (e.g. `def=render-pipeline` displays whatever text
is inside the `[...]` brackets at the call site, while `term: render pipeline` is what the
popover shows as the canonical name).

## `steps/01-overview.md` — big-picture, `layout: overview`, with diagram + one deep-dive

```markdown
---
title: How this repo is *wired together*
label: Overview
kind: Structure
order: 1
layout: overview
summary: Four moving pieces, one contract between them. Start here before touching any code.
---
[walkr]{def=walkr} reads a folder of authored steps and turns them into the
page you're looking at right now. The steps are plain markdown with frontmatter; a small
[render pipeline]{def=render-pipeline} turns that into HTML; the browser side
(Alpine + Mermaid) handles navigation, modals, and diagrams — no server required once
it's built.

​```mermaid title="structure.mmd"
graph TB
  F["walkr-author<br/>skill"] -->|writes| A[".walkr/<br/>steps"]
  A -->|parsed by| B["render pipeline<br/>(goldmark)"]
  B -->|emits| C["index.html<br/>+ fragments"]
  C -->|hydrated by| D["Alpine.js<br/>wizard"]
  D -->|renders diagrams via| E["Mermaid.js"]
​```

:::deep{title="Why build the UI before the format?"}
It's tempting to design frontmatter keys and directive syntax on paper first. In
practice that produces fields nothing renders and directives the UI can't actually
express well.

Building the Phase 0 prototype on hardcoded dummy data first meant every eventual
frontmatter key and markdown directive was derived from something a real screen
needed — the overview diagram, the two-level code block, the annotated manifest,
the glossary popover, this very modal. Nothing speculative got added to the spec.
:::
```

Notice: `title` has no literal `: ` in it, so it's unquoted. Plain prose paragraph, then
exactly one mermaid block, then exactly one deep-dive — that's the whole overview layout.

## `steps/02-render-pipeline.md` — a code path, `layout: code-walk`

```markdown
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

​```go path="internal/render/render.go" mark=2,5,7
func RenderStep(s *Step) (string, error) {
    fm, body := splitFrontmatter(s.Raw)
    tmpl, err := templateFor(fm.Layout)
    if err != nil { return "", err }
    html := goldmark.Convert(body, WithDirectives())
    return tmpl.Exec(fm, html)
} // one concept per step — no walls of text
​```
1. Frontmatter is parsed before the body — it decides the template and disclosure defaults before a single word of markdown is touched.
2. Custom directives (`:::deep`, glossary spans, two-level code fences) are goldmark extensions, not a second parser.
3. The whole function is deliberately small — it's the enforcement point for "never a wall of text."
```

Notice: `title` here *does* contain a literal `: ` (`"The render pipeline: *render.go*"`),
so the whole value is quoted — leaving it unquoted would parse as a nested YAML mapping
and fail the build. `label` (`render.go`) is a short standalone word for the rail, totally
different from `title`'s full headline — don't reuse one for the other.

Line-counting proof for `mark=2,5,7` (count every line inside the fence, 1-indexed,
starting immediately after the opening ` ```go ... `  line):

| fence line # | content | footnote |
|---|---|---|
| 1 | `func RenderStep(s *Step) (string, error) {` | (unmarked) |
| 2 | `    fm, body := splitFrontmatter(s.Raw)` | 1st list item |
| 3 | `    tmpl, err := templateFor(fm.Layout)` | (unmarked) |
| 4 | `    if err != nil { return "", err }` | (unmarked) |
| 5 | `    html := goldmark.Convert(body, WithDirectives())` | 2nd list item |
| 6 | `    return tmpl.Exec(fm, html)` | (unmarked) |
| 7 | `} // one concept per step — no walls of text` | 3rd list item |

`mark=2,5,7` has 3 numbers; the ordered list has exactly 3 items. That equality is a
hard build-time check — get it wrong and the build fails naming the step and the
mismatch.

## `steps/03-deployment.md` — ops/manifest, `layout: config`

```markdown
---
title: "Deploying it: the *k8s manifest*"
label: Deployment
kind: Config
order: 3
layout: config
summary: Config gets the same treatment as code — annotated inline, not left to speak for itself.
---
​```yaml mark=4,9,11
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 2
  template:
    spec:
      containers:
        - name: walkr
          readinessProbe:
            httpGet: { path: /healthz, port: 8080 }
          resources:
            limits: { cpu: "200m", memory: "128Mi" }
​```
1. Two replicas — this is a static-site server, so redundancy is cheap and mostly guards against node drain.
2. Without this, a slow-starting pod can receive traffic before the site is built and served.
3. Deliberately tight — the binary embeds all assets, so there's no separate asset-serving footprint to budget for.
```

Notice: **no bullet-list summary before the fence** — `config` skips straight to the
annotated block (that's the entire difference from `code-walk`'s body shape: code-walk
is summary-bullets-then-fence, config is fence-only). There is also no toggle to collapse
this — `config` steps render fully expanded, always. `path` is optional and omitted here
(the manifest has no single canonical file path in this repo); `kind: Config` reads as
the eyebrow ("Chapter 03 · Config") and the rail subtitle.

## `steps/04-recap.md` — closing recap, `layout: overview`, no diagram

```markdown
---
title: You've seen *the whole loop*
label: Recap
kind: Summary
order: 4
layout: overview
summary: Structure → code → config. The same four interactions repeat for every subsystem in a real walkthrough.
---
Every subsystem in a real walkthrough gets the same treatment: an overview diagram,
two-level code, annotated config, glossary terms, and deep-dives where the reasoning
needs more room than a paragraph.

Everything in this walkthrough is now generated from plain markdown — every frontmatter
key and directive here is written down as the [content-format spec]{def=content-format-spec},
and this page itself is proof the renderer reproduces the Phase 0 prototype from authored
content, not hardcoded HTML.
```

Notice: the recap is `layout: overview` with **no** mermaid block and **no** deep-dive —
both are optional parts of the overview layout, not separate layouts. A recap is just an
overview step that happens to skip the optional parts.
