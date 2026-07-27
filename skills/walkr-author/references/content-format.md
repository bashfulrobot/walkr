# walkr content format

> Bundled copy: this file is a snapshot of `docs/ai/content-format.md` from
> the `walkr` repo, shipped inside this skill so it travels with it wherever
> the skill is copied. If you are working inside the `walkr` repo itself,
> `docs/ai/content-format.md` is the canonical original — re-sync this copy
> if that file changes. Everything below is otherwise verbatim.

This is the contract between the `walkr` renderer and anything that authors a
walkthrough (a human, or the `walkr-author` skill). Every key and directive below
was derived from the Phase 0 prototype (`prototype/`) — nothing here exists that the UI
doesn't render. If you're extending this format, the rule stays the same: build the UI
interaction first, then add the frontmatter key or directive that drives it.

## Layout

```
.walkr/
├─ walkthrough.yaml     # optional global manifest
├─ glossary.yaml         # hover/click term definitions
└─ steps/
   ├─ 01-overview.md
   ├─ 02-render-pipeline.md
   ├─ 03-deployment.md
   └─ 04-recap.md
```

## `walkthrough.yaml`

Optional. Three fields, all consumed by the rail header (`rail__mark`, `rail__tag`,
`rail__repo` in the prototype):

```yaml
title: Walkr          # rail__mark — big italic wordmark, top-left
tagline: Field Guide        # rail__tag — small caps line under the title
repo: walkr/walkr  # rail__repo — monospace pill, e.g. an org/repo slug
```

If absent, `walkr init` seeds sensible defaults and the CLI derives `repo` from
the target directory's git remote when it can.

There is deliberately no `groups[]` key. The prototype's rail is a flat, ordered list —
nothing in the UI groups steps into sections. Don't add that key until a UI needs it.

## Step frontmatter

Each file in `steps/` is one concept. Frontmatter:

| Key | Required | Consumed by |
|---|---|---|
| `title` | yes | `step__title`, the big heading (the `em`-wrapped word, if any, is written directly in the title string — see below) |
| `label` | yes | the short rail item text (`itinerary__title`) — one or two words, e.g. `Overview`, `render.go`. Deliberately separate from `title`: the rail needs a short standalone word, not the full headline. |
| `kind` | yes | rail item subtitle (`itinerary__kind`) *and* the eyebrow line (`Chapter 0N · {kind}`) |
| `order` | yes | sort key across `steps/*.md`; ties break on filename |
| `layout` | yes | picks the step template: `overview`, `code-walk`, or `config` (see below) |
| `summary` | yes | the always-visible lede sentence (`step__lede`), one sentence, no markdown |

`title` may contain a single `_em_` or `*em*` span — the renderer emits that inline
span as `<em>` inside `step__title`, matching the prototype's amber-italic word
(e.g. "How this repo is *wired together*").

Frontmatter is plain YAML, so if `title` (or any other value) contains a literal
`: `, quote the whole string (`title: "The render pipeline: *render.go*"`) — an
unquoted colon-space reads as a nested mapping and fails to parse.

### `layout: overview`

Body is plain GFM prose (paragraphs, glossary spans — see Directives). Optionally
followed by:

- one ` ```mermaid ` fenced block — rendered into the diagram frame. Give it a `title`
  attribute for the small pill caption (see Directives → diagrams).
- one `:::deep{...}` block — rendered as the "Go deeper" button + modal (see Directives).

Nothing else is layout-specific; `recap`-style closing steps (prose + glossary term,
no diagram) also use `layout: overview` — a diagram is optional, not a separate layout.

### `layout: code-walk`

Body is:

1. A GFM bullet list — the always-visible summary (`codewalk__summary`).
2. One fenced code block with `path` and (optionally) `mark` attributes, immediately
   followed by one GFM ordered list — the annotated source, collapsed by default behind
   "Show annotated source" (see Directives → annotated code).

### `layout: config`

Same annotated-code-block-plus-ordered-list pairing as `code-walk` step 2, but **always
expanded** — no summary list, no toggle. Use this for manifests/config that should read
top-to-bottom immediately.

## Directives

These are the only non-standard-GFM syntax the renderer understands. Everything else is
plain CommonMark/GFM (goldmark + the `github.com/yuin/goldmark/extension` GFM table).

### Glossary term

```markdown
[walkr]{def=walkr}
```

Renders the bracketed text as a dotted-underline `.term` span. `def` is looked up in
`glossary.yaml` at build time; the definition (and optional `learn_more` URL) is baked
directly into the page as the popover's content — there is no client-side glossary
fetch. Hover *or* click toggles the popover (see `prototype/assets/app.js`).

`glossary.yaml`:

```yaml
walkr:
  term: walkr
  definition: >
    A CLI that renders a hand-authored markdown walkthrough into a static,
    wizard-style teaching site. It never generates content itself.
  learn_more: https://github.com/bashfulrobot/walkr   # optional
```

### Deep-dive modal

```markdown
:::deep{title="Why build the UI before the format?"}
It's tempting to design frontmatter keys on paper first...

Building this prototype on hardcoded dummy data first means...
:::
```

Renders a `.deepen` button reading "✎ Go deeper: {title}" that opens a modal. The block
body is parsed as ordinary markdown (paragraphs only, in the prototype). One per step,
maximum — the prototype only ever shows one deep-dive per step; the renderer doesn't
enforce this, but the authoring skill should treat it as a soft rule to avoid stacking
too much optional depth on one screen.

### Diagram

```markdown
​```mermaid title="structure.mmd"
graph TB
  A --> B
​```
```

`title` becomes the small pill caption on the diagram frame (`diagram-frame__cap`). If
omitted, the renderer falls back to `diagram.mmd`. The fenced block's body is emitted
verbatim into a `<div class="mermaid">` — Mermaid.js (vendored, client-side) does the
rendering; the Go renderer never parses or validates diagram syntax.

### Annotated code (two-level code walk / config)

A fenced code block with two optional space-separated attributes on the info string,
after the language:

```markdown
​```go path="internal/render/render.go" mark=2,5,7
func RenderStep(s *Step) (string, error) {
    fm, body := splitFrontmatter(s.Raw)
    ...
}
​```
1. Frontmatter is parsed before the body — it decides the template and disclosure
   defaults before a single word of markdown is touched.
2. Custom directives are goldmark extensions, not a second parser.
3. The whole function is deliberately small.
```

- `path` (optional) — shown in the block header (`codewalk__path`); the part before the
  last `/` renders dim, the filename renders bright, matching the prototype's
  `internal/render` / `render.go` split.
- `mark=n1,n2,...` (optional) — 1-indexed source line numbers to badge. Each marked line
  gets the amber left-border treatment and a numbered badge, **in the order given**.
- The ordered list immediately following the fenced block supplies the footnote text,
  matched positionally to the `mark` list (first list item ↔ first `mark` number, etc.).
  List length must equal `mark` length or the build fails with a clear error naming the
  step and the mismatch — this is a build-time contract check, not a silent drop.
- If `mark` is absent, the code renders as a plain (non-annotated) highlighted block and
  no ordered list is consumed as footnotes — a following ordered list would render as
  ordinary markdown instead.

Language-agnostic: `path`/`mark` work on any fenced code language (Go in the code-walk
example, YAML in the config example). Syntax colouring is done by hand-classified spans
today (`.kw`/`.tp`/`.st`/`.nm`/`.pn`/`.cm` in `prototype/assets/style.css`) driven by a
small per-language classifier in the renderer — not a generic tokenizer, so new
languages need a classifier added in `internal/render` (documented there, not here).

## What's deliberately not in v1

- No `deep_default`/`dig_deeper`/`definitions` toggle keys — the prototype never varies
  disclosure behavior per step; `code-walk` is always collapsed-by-default, `config` is
  always expanded, `overview`'s deep-dive is always a modal. Add a key only when a UI
  needs a *per-step* variation, not before.
- No `group`/section headers in the rail.
- No fetching external docs (pkg.go.dev, etc.) for glossary content — authored only.
- No generic syntax-highlighting engine (chroma/Prism) — hand-classified spans per the
  small set of languages walkr itself needs to teach. Revisit if/when a real
  walkthrough needs a language the classifier doesn't cover.
