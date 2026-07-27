---
name: walkr-author
description: Author a walkr walkthrough for a codebase — analyze a target repository and generate a `.walkr/` directory (walkthrough.yaml, glossary.yaml, steps/*.md) that the `walkr` CLI renders into an interactive teaching site. Use when the user says "author a walkr walkthrough", "generate a walkthrough for this repo", "/walkr-author", "build a .walkr directory", "onboard someone to this repo with walkr", "explain this codebase step by step for walkr", or points at a repo and asks for a guided tour / newcomer walkthrough / teaching site for it. This skill is portable — it is meant to be copied into a target repo's `.claude/skills/walkr-author/` and run there; it does not require the walkr binary itself to be present, only the target repo to analyze.
allowed-tools: ["Read", "Glob", "Grep", "Write", "Edit", "Bash"]
---

# walkr-author

You are authoring a `.walkr/` walkthrough for a target repository — a hand-written
markdown content set that the `walkr` CLI renders into a static, wizard-style teaching
site for a newcomer. **You do not generate HTML and you do not run the renderer.** Your only
job is to produce markdown + YAML that conforms *exactly* to the content-format contract,
because the renderer trusts authored content completely and does no content generation of
its own.

This skill is portable: it may be running inside the `walkr` repo itself, or it may
have been copied into some other repo's `.claude/skills/walkr-author/` to document
*that* repo. Either way, treat the repo you were asked to walk through as "the target repo" —
never assume it's walkr's own source.

## Step 0 — load the contract, every time

Before writing a single file, read **`references/content-format.md`**, bundled in this
skill's own directory (next to this file) — the same directory this SKILL.md lives in,
resolved as `<this-skill>/references/content-format.md`. That file is the complete,
locked contract for frontmatter keys, the three layouts, and the four directives. It is
authoritative over everything summarized below and over any prior memory of walkr's
format — if the two ever disagree, the bundled file wins.

Also read **`references/worked-example.md`** in this skill's directory — a full, correct,
line-by-line-annotated worked example (walkr's own dogfood walkthrough) showing what
compliant output actually looks like, including a demonstration of the `mark=`/footnote
line-counting rule. Pattern-match your output against it.

Do not invent a frontmatter key, directive, or layout that isn't in `content-format.md`.
If the target repo seems to need something the spec doesn't support (e.g. per-step
disclosure toggles, grouped/sectioned rail, fetched external docs) — it deliberately
doesn't exist in v1 (see that file's closing section, "What's deliberately not in v1").
Work within the three layouts instead of asking for a new one.

## Step 1 — analyze the target repository

Goal: build a mental model of the repo good enough to explain it to someone who has never
seen it, at three levels of resolution: **how it's wired together**, **the notable code
paths**, and **the config/ops surface**. Be language-agnostic — the checklist below applies
whether the target is Go, TypeScript, Python, Rust, a Helm chart repo, or anything else.

1. **Find the shape.** `Glob`/`Bash` (`ls`, `find`, `git ls-files`) the top-level layout.
   Identify: entry point(s) (`main.go`, `index.ts`, `__main__.py`, `cmd/`, a CLI framework's
   root command); the build/package manifest (`go.mod`, `package.json`, `Cargo.toml`,
   `pyproject.toml`, `pom.xml`, `Gemfile`, a `flake.nix`); top-level source directories and
   what each is responsible for (read directory names, README, doc comments — don't guess
   from convention alone, open representative files).
2. **Trace the wiring.** How do the pieces actually connect at runtime? Follow imports,
   dependency injection, routing tables, event buses, plugin registration, whatever the
   target's own idiom is. You're building the overview diagram's edges, not an exhaustive
   call graph — aim for the 4-8 node picture a newcomer needs to orient, matching the
   granularity of walkr's own overview diagram (skill → steps → render pipeline →
   HTML → wizard → diagrams).
3. **Find notable code paths.** 2-6 files/functions that are the conceptual core — the
   pipeline that does the repo's actual job, not every file. A "notable code path" is
   something a newcomer would ask "wait, how does *that* work?" about. Read the real
   source; don't summarize from a filename.
4. **Find config/ops surface.** k8s manifests, Dockerfiles, CI workflow files (GitHub
   Actions, GitLab CI, etc.), Terraform/Helm/Pulumi, `docker-compose.yml`, deploy scripts,
   release tooling. Treat these as first-class teaching material, not an afterthought —
   the spec gives config the exact same annotated-code treatment as source (see `layout:
   config` below), because misreading a manifest breaks production as easily as
   misreading code.
5. **Find the jargon.** Project-specific terms, acronyms, internal names, or domain
   vocabulary a newcomer wouldn't know cold (e.g. "render pipeline", "control plane",
   whatever the target repo's own vocabulary is). These become `glossary.yaml` entries.
   Definitions are **authored by you from what you actually read in the repo** — never
   fetch or paraphrase from external docs (pkg.go.dev, framework docs, etc.); that's an
   explicit non-goal of the format.

Use `Read`/`Grep`/`Glob` liberally during analysis; `Bash` only for read-only exploration
(`git log`, `git remote -v` to derive the `repo:` slug, `find`, `wc -l`) — this skill never
needs to modify the target repo outside of writing the new `.walkr/` directory.

## Step 2 — plan the sequence

One flat, ordered list of steps (there is no grouping key in this format — don't invent
one). Sequence:

1. **Big-picture overview(s)** — `layout: overview`, at least one, with the structure/
   wiring mermaid diagram. This is always step `order: 1`. A newcomer must see the whole
   shape before any code.
2. **Subsystem overview(s)** — more `layout: overview` steps if the repo has 2+ major
   subsystems worth their own "how this piece fits" diagram or explanation, before diving
   into their code.
3. **Notable code paths** — `layout: code-walk` steps, one per notable path identified in
   analysis. One concept per step — if a single file does two unrelated things, that's two
   steps, not one.
4. **Config/ops** — `layout: config` steps for manifests, CI, IaC.
5. **Closing recap** — exactly one final step, `layout: overview`, no mermaid block, no
   deep-dive. Plain prose recapping the throughline (structure → code → config, or whatever
   the target repo's actual arc was). This mirrors `layout: overview`'s optional diagram/
   deep-dive being just that — optional, not a separate layout.

Assign `order` sequentially (1, 2, 3, ...) across the *entire* list regardless of layout —
`order` is a single global sort key, not per-layout. Filenames are conventionally
`NN-slug.md` (`01-overview.md`, `02-render-pipeline.md`, ...) with `NN` matching `order`
for human readability, but the renderer sorts on frontmatter `order`, not filename, so
ties break on filename — keep them aligned anyway to avoid confusing future editors.

## Step 3 — write `walkthrough.yaml`

Optional but write it anyway — it's three fields and gives the rail a proper header:

```yaml
title: <repo's display name>
tagline: <short subtitle, 1-3 words>
repo: <org>/<repo>
```

Derive `repo:` from the target repo's git remote when you can (`git remote get-url origin`
via `Bash`, parsed to `org/repo`); fall back to the directory name if there's no remote.
Do not add a `groups[]` key — it doesn't exist in this format (see content-format.md,
"There is deliberately no `groups[]` key").

## Step 4 — write `glossary.yaml`

One top-level entry per jargon term you plan to reference via the glossary directive
(Step 5). Each entry:

```yaml
<def-id>:
  term: <canonical display term>
  definition: >
    One or two sentences, authored from what you read in the repo.
  learn_more: <optional URL>   # omit if there's nothing worth linking
```

`<def-id>` is the lookup key used as `def=<def-id>` in step bodies — it does not have to
match the bracketed visible text at the call site, and doesn't have to equal `term`
word-for-word either (see the worked example's `render-pipeline` entry: `def=render-pipeline`,
bracket text `[render pipeline]`, `term: render pipeline` — three things that happen to be
similar here but are three independent strings). Every `def=` id used anywhere in
`steps/*.md` must have a matching top-level key here, or the reader hits a broken popover.

## Step 5 — write each `steps/NN-slug.md`

### Frontmatter — required on every step, get every field right

```yaml
---
title: <headline, may contain exactly one *em* or _em_ span>
label: <short rail word/phrase, 1-2 words, deliberately NOT the same string as title>
kind: <rail subtitle AND the "Chapter 0N · {kind}" eyebrow — short, e.g. "Structure", "Code walk", "Config", "Summary">
order: <integer, global sort key across all steps>
layout: overview | code-walk | config
summary: <one plain sentence, no markdown, always-visible lede>
---
```

**Gotcha — quote any value containing a literal colon-space.** YAML frontmatter is plain
YAML: `title: "The render pipeline: *render.go*"` must be quoted because the unquoted
`: ` after "pipeline" reads as a nested mapping and fails to parse. `title: How this repo
is *wired together*` needs no quotes because it has no literal `: `. When in doubt, quote
it — quoting a string with no colon is harmless, an unquoted colon is a build break.

**Gotcha — `label` and `title` are two different strings, on purpose.** `title` is the
full headline (rendered big, may carry one `*em*` span). `label` is the short standalone
word the rail shows (`Overview`, `render.go`, `Deployment`, `Recap`) — never reuse the full
title as the label, and never leave label as a truncation of title; write it as its own
short noun phrase.

**Gotcha — one `*em*`/`_em_` span maximum in `title`.** The renderer emits it as a literal
`<em>` — it's a single inline emphasis word/phrase inside the title string, not a markdown
feature applied to the whole field.

### Body shape by layout — these are NOT interchangeable

| layout | body shape |
|---|---|
| `overview` | Plain GFM prose (paragraphs, glossary spans). Optionally **one** ` ```mermaid ` fenced block (give it `title="..."` for the pill caption), optionally **one** `:::deep{...}` block. Both optional parts, in either combination, including neither (closing recap: neither). |
| `code-walk` | **First** a GFM bullet list (the always-visible plain-language summary) — **then** one fenced code block with `path`/`mark` attributes, **immediately followed by** a matching GFM ordered list (the footnotes). Collapsed by default behind "Show annotated source." |
| `config` | **Only** the fenced code block (`path`/`mark`) + matching ordered list. **No bullet-list summary before it** — that's the one structural difference from `code-walk`. **Always fully expanded** — no toggle exists for `config`, ever. |

**Gotcha — don't give `config` a summary list.** It's tempting to copy the `code-walk`
shape and prepend bullets "for consistency." Don't — the spec defines `config` as landing
straight on the annotated block, unfiltered, because manifests should "read top-to-bottom
immediately" (content-format.md, `layout: config`). Config also never gets a
collapse/toggle — if you find yourself wanting to hide a manifest behind a "show more,"
that's a sign it should be trimmed to the annotated block, not deferred.

### Directives — the only four non-standard-GFM constructs the renderer understands

Everything else in a step body is plain CommonMark/GFM (paragraphs, lists, tables, links,
images, code spans, task lists) — use it freely. These four are special:

**1. Glossary term** — `[visible text]{def=some-id}` where `some-id` is a top-level key in
`glossary.yaml`. Renders a dotted-underline hover/click popover, content baked in at build
time (no client-side fetch).

**2. Deep-dive modal** (overview layout only, per the observed pattern) —
```markdown
:::deep{title="Why build the UI before the format?"}
Ordinary markdown paragraphs.
:::
```
Soft rule: **at most one per step.** The renderer won't stop you from stacking more, but
piling up optional depth on one screen defeats the "never a wall of text" goal — if you
have two things worth a deep-dive, that's a sign you need two steps.

**3. Mermaid diagram** —
````
```mermaid title="structure.mmd"
graph TB
  A --> B
```
````
`title` becomes the pill caption; falls back to `diagram.mmd` if omitted. Emit valid
Mermaid syntax (`graph TB`, `sequenceDiagram`, etc.) — the Go renderer never validates it,
so a syntax error only surfaces when a human opens the rendered page. Aim for 4-8 nodes at
the top-level overview; more than that stops being a "structure at a glance" diagram.

**4. Annotated code (two-level code-walk / config)** —
````
```go path="internal/render/render.go" mark=2,5,7
func RenderStep(s *Step) (string, error) {
    fm, body := splitFrontmatter(s.Raw)
    ...
}
```
1. First footnote — matches mark's first number.
2. Second footnote — matches mark's second number.
3. Third footnote — matches mark's third number.
````
- `path` is optional; when given it's split at the last `/` for dim-directory /
  bright-filename display. Omit it if there's no single canonical file (e.g. a manifest
  assembled inline rather than pulled from one path).
- `mark` is optional. **When present, its count must exactly equal the following ordered
  list's item count**, or the build fails naming the step and the mismatch. This is a hard
  contract check, not something to eyeball — see the algorithm below.
- If `mark` is absent, the block renders as plain highlighted code and a following
  ordered list (if any) renders as ordinary markdown, NOT as footnotes — don't accidentally
  drop `mark` and expect the list to still attach.
- Works on any fenced-code language; syntax colouring is hand-classified per language by
  the renderer, which is a renderer concern, not something this skill needs to handle.

**Algorithm for getting `mark=`/footnotes right, every time:**
1. Write the fence body first, exactly as it will appear.
2. Number *every* line inside the fence starting at 1, top to bottom — blank lines,
   closing braces, trailing comments, all of it counts as a line.
3. Decide which lines deserve a footnote and in what order you want to explain them
   (usually top-to-bottom, but the list order is whatever order you write the numbers in —
   they're positional, not sorted for you).
4. Set `mark=` to exactly those line numbers, in that order.
5. Write the ordered list immediately after the fence with **exactly** that many items, in
   the same order — item 1 explains `mark`'s first number, item 2 the second, and so on.
6. Count both sides before moving on. `references/worked-example.md` walks through this
   line-by-line for both a Go code-walk and a YAML config step — use it as the template.

## Teaching principles — check every step against these before moving on

- **One concept per step.** If you're tempted to use "and" in a step's `summary`, it's
  probably two steps.
- **Never a wall of text.** Prose paragraphs in `overview` steps stay short; heavy detail
  goes behind `code-walk`'s collapsed annotation or a `:::deep` modal, never inline.
- **Progressive disclosure, deliberately.** `code-walk` gives the plain-language bullet
  summary first, annotated source second, collapsed by default. `overview`'s optional
  depth is a modal, not more prose. Don't flatten this by writing everything as visible
  prose "to be safe."
- **Config is not a second-class citizen.** Manifests, CI, IaC get the identical
  `path`/`mark` + footnote treatment as source code — explain *why* a value is what it is
  (a replica count, a resource limit, a probe), not just *what* it is.
- **Jargon goes in the glossary, not a parenthetical.** The first time you'd write "(this
  means X)" inline, make it a glossary term instead.
- **Structure before code.** The very first step is always the overview diagram; nothing
  else comes before the reader has seen the whole shape.

## Step 6 — final self-check before declaring done

Walk every generated file against this list. Fix anything that fails — none of these are
matters of taste, they're the build-time and parse-time contract.

- [ ] Every frontmatter block has `title`, `label`, `kind`, `order`, `layout`, `summary`.
- [ ] Any frontmatter value containing a literal `: ` is quoted.
- [ ] `label` is never identical to `title` on any step.
- [ ] `order` values are unique integers across all steps, and match the intended
      big-picture → subsystems → code-walk → config → recap sequence.
- [ ] `layout` is exactly one of `overview` / `code-walk` / `config` — no other value.
- [ ] Every `overview` step has, at most, one mermaid block and one `:::deep` block, in
      any combination including neither.
- [ ] Every `code-walk` step has a bullet-list summary *before* its fenced code block.
- [ ] Every `config` step has **no** bullet-list summary before its fenced code block.
- [ ] Every fenced code block that has `mark=` is immediately followed by an ordered list
      whose item count equals the count of numbers in `mark=`, in matching order.
- [ ] Every fenced code block with no `mark=` has no ordered list glued to it (or if one
      follows for unrelated prose reasons, it's clearly not meant to be read as footnotes).
- [ ] Every `[text]{def=id}` in every step has a matching top-level `id:` key in
      `glossary.yaml`.
- [ ] `walkthrough.yaml` has only `title`/`tagline`/`repo` — no `groups[]`.
- [ ] The last step is `layout: overview`, has no mermaid block, and closes the loop
      (recaps what the walkthrough covered) rather than introducing new material.
- [ ] Nothing invents a frontmatter key or directive absent from
      `references/content-format.md`.

If any box fails, fix the specific file — don't rationalize the deviation, the renderer
enforces several of these at build time and will fail loudly rather than degrade quietly.

## Output location

Write to `.walkr/` at the root of the target repository (the same repo you analyzed
in Step 1), matching this layout exactly:

```
.walkr/
├─ walkthrough.yaml
├─ glossary.yaml
└─ steps/
   ├─ 01-overview.md
   ├─ 02-<slug>.md
   └─ ...
```

This skill does not invoke the `walkr` binary and does not need it installed to do
its job — it only needs to produce content that conforms to the spec. If the binary is
available in the environment (`walkr build .walkr -o /tmp/site` /
`walkr serve .walkr --open`), offer to run it so the user can preview the
result, but authoring the correct markdown/YAML is the actual deliverable.
