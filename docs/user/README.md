# walkr user manual

`walkr` turns a hand-authored markdown walkthrough into an interactive,
wizard-style static site that teaches a newcomer how something fits
together: a codebase, or a topic sourced from docs. Structure first, then
specifics, one concept per step. `walkr` never generates or analyzes content
itself; it only renders what you (or the `walkr-author`/`walkr-tutorial-author`
skills) write.

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

`./site/index.html` is now a fully self-contained page: no server, no
network calls (Alpine.js, Mermaid.js, and fonts are all embedded). Open it
directly from `file://` or drop `./site` on any static host.

## Authoring a walkthrough by hand

A walkthrough is a directory (`.walkr/` by default):

```
.walkr/
├─ walkthrough.yaml   # optional: title/tagline/repo shown in the rail header
├─ glossary.yaml       # hover/click term definitions
├─ media/               # optional: screenshots/diagrams referenced from steps
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
Your step body goes here: plain GFM, plus a few directives (see below).
```

- **`layout`** picks the template: `overview` (prose + optional diagram +
  optional deep-dive), `code-walk` (a summary list, then annotated source
  collapsed behind a toggle), or `config` (annotated source, always expanded,
  no toggle, for manifests/config you want visible immediately).
- **`title`** is the big heading (may contain one `*em*` word). **`label`**
  is the short word shown in the left-rail step list, deliberately a
  *different, shorter* string than `title`.

Three directives are available in the body:

- **Glossary term**: `` [term]{def=some-id} ``, looked up in `glossary.yaml`.
- **Deep-dive modal**: `` :::deep{title="..."} ... ::: `` for optional depth
  that would otherwise clutter the main flow.
- **Annotated code**: a fenced code block with `path="..."` and
  `mark=2,5,7` attributes, immediately followed by a matching ordered list.
  Each marked line gets a numbered badge, and list item *N* is that badge's
  footnote text.
- **Diagrams**: a plain ` ```mermaid title="name.mmd" ` fenced block.
- **Images**: a plain `![alt](media/whatever.png)`, no directive needed. Put the
  file under `.walkr/media/`; `walkr build` copies it byte-for-byte to
  `outDir/media/` alongside `index.html`.

**This is the quick tour, not the full contract.** For the exact, authoritative
frontmatter keys and directive syntax, including the gotchas (quoting a
`title` that contains a literal colon, how `mark=` line-counting works,
why `config` never gets a toggle), see
[`docs/ai/content-format.md`](../ai/content-format.md). That file is what the
renderer and the `walkr-author` authoring skill both implement against, so it
is always the source of truth if this manual and it ever disagree.

## Using an authoring skill instead

Rather than hand-writing frontmatter, point Claude Code at one of two portable
skills:

- **`walkr-author`** (`skills/walkr-author/`) analyzes a **code repository** and
  writes a conforming `.walkr/` directory for it. Copy it into a target repo's
  `.claude/skills/walkr-author/` and ask it to "author a walkr walkthrough for
  this repo".
- **`walkr-tutorial-author`** (`skills/walkr-tutorial-author/`) authors a
  walkthrough for a **topic plus one or more reference URLs** instead, no
  repository involved. It fetches and reads the linked docs, optionally
  captures screenshots of a live tool/UI into `media/`, and attributes every
  step to the source URL(s) it drew from. Ask it to "author a walkr tutorial
  about `<topic>`" with links to the docs/spec/help pages to source it from.

Both write to the same `.walkr/` layout and are governed by the same
`docs/ai/content-format.md` contract.

## Building a collection of tutorials

`walkr build-all [root]` builds every walkthrough under a directory tree. A tutorial is any `.walkr/` directory that contains `steps/`. Each site is written to a `site/` folder beside its `.walkr/`, and all the sites share one copy of the vendored libraries and stylesheet in `<root>/_walkr/`. That keeps a collection of 100 tutorials at about 8 MB instead of 350 MB. Builds are deterministic, so a rebuild only changes the tutorials you edited.

`site/` and `_walkr/` are generated, and every build regenerates them in full, so don't edit them by hand. One tutorial that fails to build is reported and the rest still build.

`walkr build-all --check` writes nothing. It rebuilds into a temporary directory, compares the result with what is on disk, and exits non-zero if any site or the shared assets are missing, stale, or extra. Use it in CI or a git hook to stop a stale site from being committed.

A shared site needs `_walkr/` to stay in the same place relative to it, so open sites from the collection, or copy the whole tree. For a fully standalone site, use `walkr build`, or pass `--per-site` to `build-all`.

## Publishing to Confluence

`walkr confluence publish` turns a walkthrough into a tree of Confluence pages: an optional section page, one tutorial page, and one child page per step. Each page keeps walkr's idea of a small amount of information at a time, with a previous and next pager, a Terms panel, and diagrams rendered to PNG. It uses native Confluence elements only, so the pages follow your site's theme and dark mode.

Publishing is one way. The walkthrough files are the source of truth, and a republish overwrites edits made in Confluence. Running it again updates the same pages in place, even after a step is renamed.

### Configure

One global file at `~/.config/walkr/config.yaml`:

```yaml
confluence:
  site: example.atlassian.net
  cloud_id: <site cloud id>
  email: you@example.com
  auth:
    token_ref: op://<vault>/<item>/<field>     # a pointer, never the token
  targets:
    my-walkthrough:
      dir: ~/path/to/.walkr
      space_key: <space key>
      parent_id: "<page id>"                   # where the section, or the tutorial page, goes
      section: Tutorials                       # optional
      diagrams: png                            # png, or source to skip rendering
```

The file never holds a secret. The parser rejects unknown keys, so a literal `token:` is an error. The Atlassian API token is resolved in memory through the 1Password SDK, using the service account token in `OP_SERVICE_ACCOUNT_TOKEN`. The API token must be a scoped token with read and write content, read space, read and write label, write attachment, and search scopes.

### Run

```
walkr confluence check --target my-walkthrough             # verify credentials, read-only
walkr confluence publish --target my-walkthrough --dry-run # show what would change
walkr confluence publish --target my-walkthrough           # publish, prints the start URL
```

Diagram rendering needs Chrome or Chromium on the machine that publishes. Without one, diagrams show as Mermaid source in a collapsed section and the command warns. Set `WALKR_CHROME` to choose the browser.

The `walkr-confluence-publish` skill wraps these steps for Claude Code. Design details are in `docs/ai/confluence-publish-design.md`.

## CLI reference

```
walkr build [dir] [-o ./site]   # default dir: .walkr
walkr build-all [root] [--check] [--per-site]   # build every walkthrough under root
walkr serve [dir] [--port N] [--open]
walkr init [dir]
walkr confluence check --target <name>      # verify Confluence credentials
walkr confluence publish --target <name> [--dry-run]
```
