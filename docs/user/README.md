# walkr user manual

`walkr` turns a hand-authored markdown walkthrough into an interactive,
wizard-style static site that teaches a newcomer how a codebase fits
together — structure first, then code, then config, one concept per step.
`walkr` never generates or analyzes content itself; it only renders what you
(or the `walkr-author` skill) write.

## Install

**With Nix (recommended):**

```sh
nix run github:bashfulrobot/walkr -- --help
```

or add it to a flake/devShell via the exposed overlay:

```nix
inputs.walkr.url = "github:bashfulrobot/walkr";
# ...
overlays = [ inputs.walkr.overlays.default ];
# pkgs.walkr is now available
```

Build a local binary instead:

```sh
git clone https://github.com/bashfulrobot/walkr
cd walkr
nix build .#walkr   # ./result/bin/walkr
```

**With Go (no Nix):**

```sh
go install github.com/bashfulrobot/walkr@latest
```

## Quickstart

Scaffold a walkthrough, fill it in, and preview it:

```sh
walkr init                  # creates .walkr/ with a starter step
$EDITOR .walkr/steps/01-overview.md
walkr serve .walkr --open   # builds to a temp dir, serves it, opens your browser
```

When you're happy with it, build the self-contained static site:

```sh
walkr build .walkr -o ./site
```

`./site/index.html` is now a fully self-contained page — no server, no
network calls (Alpine.js, Mermaid.js, and fonts are all embedded). Open it
directly from `file://` or drop `./site` on any static host.

## Authoring a walkthrough by hand

A walkthrough is a directory (`.walkr/` by default):

```
.walkr/
├─ walkthrough.yaml   # optional: title/tagline/repo shown in the rail header
├─ glossary.yaml       # hover/click term definitions
└─ steps/
   ├─ 01-overview.md
   ├─ 02-something.md
   └─ ...
```

Each step is markdown with YAML frontmatter:

```markdown
---
title: How this repo is *wired together*
label: Overview
kind: Structure
order: 1
layout: overview
summary: One sentence, always visible, no markdown.
---
Your step body goes here — plain GFM, plus a few directives (see below).
```

- **`layout`** picks the template: `overview` (prose + optional diagram +
  optional deep-dive), `code-walk` (a summary list, then annotated source
  collapsed behind a toggle), or `config` (annotated source, always expanded,
  no toggle — for manifests/config you want visible immediately).
- **`title`** is the big heading (may contain one `*em*` word). **`label`**
  is the short word shown in the left-rail step list — deliberately a
  *different, shorter* string than `title`.

Three directives are available in the body:

- **Glossary term**: `` [term]{def=some-id} `` — looked up in `glossary.yaml`.
- **Deep-dive modal**: `` :::deep{title="..."} ... ::: `` for optional depth
  that would otherwise clutter the main flow.
- **Annotated code**: a fenced code block with `path="..."` and
  `mark=2,5,7` attributes, immediately followed by a matching ordered list —
  each marked line gets a numbered badge, and list item *N* is that badge's
  footnote text.
- **Diagrams**: a plain ` ```mermaid title="name.mmd" ` fenced block.

**This is the quick tour, not the full contract.** For the exact, authoritative
frontmatter keys and directive syntax — including the gotchas (quoting a
`title` that contains a literal colon, how `mark=` line-counting works,
why `config` never gets a toggle) — see
[`docs/ai/content-format.md`](../ai/content-format.md). That file is what the
renderer and the `walkr-author` authoring skill both implement against, so it
is always the source of truth if this manual and it ever disagree.

## Using the `walkr-author` skill instead

Rather than hand-writing frontmatter, point Claude Code at a repo with the
`walkr-author` skill (`skills/walkr-author/` in this repo; copy it into a
target repo's `.claude/skills/walkr-author/` to use it there) and ask it to
"author a walkr walkthrough for this repo". It analyzes the target
repository and writes a conforming `.walkr/` directory for you.

## CLI reference

```
walkr build [dir] [-o ./site]   # default dir: .walkr
walkr serve [dir] [--port N] [--open]
walkr init [dir]
```
