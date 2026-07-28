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
(Alpine + Mermaid) handles navigation, modals, and diagrams, no server required once
it's built.

```mermaid title="structure.mmd"
graph TB
  F["walkr-author<br/>skill"] -->|writes| A[".walkr/<br/>steps"]
  A -->|parsed by| B["render pipeline<br/>(goldmark)"]
  B -->|emits| C["index.html<br/>+ fragments"]
  C -->|hydrated by| D["Alpine.js<br/>wizard"]
  D -->|renders diagrams via| E["Mermaid.js"]
```

:::deep{title="Why build the UI before the format?"}
It's tempting to design frontmatter keys and directive syntax on paper first. In
practice that produces fields nothing renders and directives the UI can't actually
express well.

Building the Phase 0 prototype on hardcoded dummy data first meant every eventual
frontmatter key and markdown directive was derived from something a real screen
needed: the overview diagram, the two-level code block, the annotated manifest,
the glossary popover, this very modal. Nothing speculative got added to the spec.
:::
