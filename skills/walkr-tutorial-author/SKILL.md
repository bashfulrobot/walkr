---
name: walkr-tutorial-author
description: Author a walkr walkthrough for a topic from reference URLs, no code repository involved. Analyze one or more linked docs, specs, or help pages and generate a `.walkr/` directory (walkthrough.yaml, glossary.yaml, steps/*.md, optionally media/) that the `walkr` CLI renders into an interactive teaching site. Use when the user says "author a walkr tutorial about <topic>", "make a walkr walkthrough from these docs", "/walkr-tutorial-author", "teach <topic> as a walkr walkthrough", "build a walkr walkthrough from this spec/URL", "walk a new hire through <product/process>" with linked reference material, or wants a guided explainer of a concept, protocol, spec, or product UI sourced from docs rather than a git checkout. This skill is portable, copy it into any repo's `.claude/skills/walkr-tutorial-author/` and run it there; it does not require the `walkr` binary to be present. Distinct from `walkr-author`, which analyzes a code repository instead of external docs, do not use this skill when the ask is "explain this repo" or "onboard someone to this codebase."
allowed-tools: ["Read", "Glob", "Grep", "Write", "Edit", "Bash", "WebFetch", "WebSearch"]
---

# walkr-tutorial-author

You are authoring a `.walkr/` walkthrough for a **topic**, sourced from one or more
reference URLs the user gives you, a spec, official docs, a product's help pages,
with **no code repository involved**. The output is the same kind of hand-written
markdown + YAML content set that `walkr-author` produces for a repo; the only
difference is where the material comes from. **You do not generate HTML and you do
not run the renderer.** Your only job is to produce markdown + YAML that conforms
*exactly* to the content-format contract, because the renderer trusts authored
content completely and does no content generation of its own.

This skill is portable: it may be running inside the `walkr` repo itself, or it may
have been copied into some other repo's `.claude/skills/walkr-tutorial-author/` to
build an unrelated tutorial there. The target directory for the output `.walkr/` is
wherever the user wants the walkthrough to live, ask if it's not obvious (often the
current repo, sometimes a scratch directory for a throwaway explainer).

## Step 0: load the contract, every time

Before writing a single file, read **`references/content-format.md`**, bundled in
this skill's own directory (next to this file), the same directory this SKILL.md
lives in, resolved as `<this-skill>/references/content-format.md`. That file is the
complete, locked contract for frontmatter keys, the three layouts, the four
directives, the `media/` convention, and the source-attribution convention. It is
authoritative over everything summarized below and over any prior memory of walkr's
format, if the two ever disagree, the bundled file wins.

Do not invent a frontmatter key, directive, or layout that isn't in
`content-format.md`. The three layouts (`overview`, `code-walk`, `config`) and four
directives (glossary term, deep-dive modal, mermaid diagram, annotated code) are
exactly the same ones `walkr-author` uses for repos, a topic walkthrough is
authored with the identical toolkit, just pointed at docs instead of source files.

## Step 1: get the topic and sources

Confirm with the user, if not already given:
- The **topic** in one sentence (e.g. "OAuth 2.0 device-code flow", "how our
  Terraform Cloud workspace is set up").
- **One or more reference URLs** to source it from.
- Where the output `.walkr/` should be written (defaults to the current directory's
  `.walkr/` if this is being run inside a repo the walkthrough is *about*, e.g. an
  internal tool's docs; otherwise ask).

## Step 2: research the topic before writing anything

Same "no assumptions, cite sources" discipline as everywhere else: build an accurate
mental model from the actual linked material before drafting a single step.

1. `WebFetch` every reference URL. If a page links to further pages that are clearly
   part of the same subject (e.g. a spec's sub-sections, a docs site's sibling
   pages), fetch those too rather than guessing at their content. Use `WebSearch`
   only to fill a genuine gap the given URLs don't cover, never as a substitute for
   reading what the user actually linked.
2. Take notes as you go on: the big-picture shape of the topic (what are the major
   pieces and how do they relate, this becomes the overview diagram), the notable
   mechanisms/flows worth their own step (this becomes `code-walk` steps drawn from
   example requests/responses/config in the docs, not necessarily a real source
   file), any config/manifest/schema examples worth annotating (`config` steps), and
   jargon a newcomer wouldn't know cold (glossary candidates).
3. Track which URL each fact came from, you'll need this for the source-attribution
   line on each step (Step 6). Don't wait until the end to reconstruct this; note it
   per-fact as you research.
4. **Respect copyright.** Summarize and re-explain in your own words. Never bulk-copy
   paragraphs of prose from the source docs into a step body, short verbatim
   snippets (a code example, a config sample, a spec's exact field name) are fine
   and often necessary for a `code-walk`/`config` step; multi-sentence prose lifted
   wholesale is not.

## Step 3 (optional): capture screenshots of a live tool/UI

If the topic involves teaching a live tool or product UI (not just a spec/protocol),
you may illustrate steps with real screenshots instead of, or alongside, prose.

- Load the browser tools first if they're deferred: `ToolSearch` with query
  `"select:mcp__claude-in-chrome__tabs_context_mcp,mcp__claude-in-chrome__navigate,mcp__claude-in-chrome__computer,mcp__claude-in-chrome__read_page,mcp__claude-in-chrome__tabs_create_mcp"`.
- Call `tabs_context_mcp` first, then navigate/interact to reach the state worth
  screenshotting.
- Save captured images into `<walkthrough-dir>/media/` (create the directory if it
  doesn't exist) with descriptive filenames (`media/device-code-screen.png`, not
  `media/screenshot1.png`).
- Reference them from a step body with plain markdown, no directive needed:
  `![the device authorization screen](media/device-code-screen.png)`.
- Avoid triggering JS alerts/confirms/dialogs while navigating, they block the
  browser session. If a flow requires one, warn the user rather than clicking through it.
- This step is entirely optional. Most topic walkthroughs (specs, protocols,
  architecture explainers) need zero screenshots, don't force it.

## Step 4: decide diagrams vs. screenshots

- **Mermaid** (`\`\`\`mermaid` fenced blocks) for anything *generative*, an
  architecture, a sequence of calls, a state machine, a flow. These are drawn from
  your understanding of the topic, not a literal picture of something.
- **Screenshots** only for "this is literally what the button/screen looks like",
  a specific UI a reader needs to recognize on sight. Never approximate a UI with a
  hand-drawn mermaid box-and-arrow diagram, and never use a screenshot where a
  mermaid diagram would communicate the *structure* better.

## Step 5: plan the sequence

Same shape `walkr-author` uses for repos, adapted to a topic instead of a codebase.
One flat, ordered list (there is no grouping key in this format, don't invent one):

1. **Big-picture overview**, `layout: overview`, `order: 1`, with a mermaid diagram
   of the topic's overall shape (e.g. the actors and steps in an OAuth flow, the
   components of a Terraform Cloud workspace). A newcomer must see the whole shape
   before any specifics.
2. **Subsystem/phase overviews**, more `layout: overview` steps if the topic has
   2+ major phases or components worth their own diagram/explanation before diving
   into specifics (e.g. one step per phase of a multi-step protocol).
3. **Notable mechanisms**, `layout: code-walk` steps, one per mechanism worth
   walking through in detail (an example request/response pair, a code sample from
   the docs, a CLI invocation). One concept per step.
4. **Config/schema specifics**, `layout: config` steps for any manifest, schema, or
   settings example that should read top-to-bottom, annotated line-by-line.
5. **Closing recap**, exactly one final step, `layout: overview`, no mermaid block,
   no deep-dive. Plain prose recapping the throughline.

Assign `order` sequentially across the entire list regardless of layout. Filenames
are conventionally `NN-slug.md` matching `order`.

## Step 6: write `walkthrough.yaml`, `glossary.yaml`, and each `steps/NN-slug.md`

Follow `content-format.md` exactly, frontmatter keys (`title`, `label`, `kind`,
`order`, `layout`, `summary`), body shape per layout, and the four directives are
identical to what `walkr-author` produces for a repo. The only topic-specific
differences:

- `walkthrough.yaml`'s `repo:` field is a **generic subtitle**, not a repo slug,
  use the product/spec/topic name or source domain (e.g. `RFC 8628`,
  `Terraform Cloud`), per `content-format.md`'s "Media assets" section note on
  `repo` being renderer-agnostic display text.
- Glossary definitions are **authored by you from what you actually read**, in your
  own words, never paraphrased-then-passed-off verbatim from the source, and never
  fetched fresh at build time (the renderer has no network access and never will).
- **Every step ends with a source-attribution line** per `content-format.md`'s
  "Source attribution" section:
  ```markdown
  *Source: [Page title](https://the-actual-url)*
  ```
  List every URL a step actually drew from, comma-separated in the same line, if
  more than one. This is what lets a reader or future re-author verify or refresh
  the step later, don't skip it, and don't attribute a step to a URL you didn't
  actually use for that step's content.
- `code-walk`/`config` steps drawn from docs rather than a real repo file should
  usually omit `path=` (there's no canonical file) unless the docs themselves name
  a real file (e.g. a Terraform `.tf` filename shown in an example), see
  `content-format.md`'s note that `path` is optional.

Use the same gotchas `walkr-author` documents (quote any frontmatter value with a
literal `: `; `label` and `title` must differ; one `*em*`/`_em_` span max in
`title`; `mark=` count must exactly equal the following ordered list's item count),
they apply identically here since it's the same renderer contract.

**Run every step's prose through `/text-polish` before writing it to disk.** This
skill is drafting explanatory prose from source material, which is exactly the
failure mode `/text-polish` exists to catch, hedging, filler, AI-vocabulary tells,
rule-of-three padding. Polish the `summary` line and body paragraphs (not YAML
keys, code blocks, or the mermaid diagram source); markdown structure and
directives pass through untouched. Do this per step as you draft it, not as one
pass at the end, it's cheaper to polish a paragraph once than to rewrite a whole
walkthrough's voice after the fact.

## Teaching principles: check every step against these before moving on

- **One concept per step.** If a step's `summary` needs "and," it's probably two steps.
- **Never a wall of text.** Heavy detail goes behind `code-walk`'s collapsed
  annotation or a `:::deep` modal, never inline prose.
- **Progressive disclosure, deliberately.** Same collapsed-by-default /
  always-expanded split as `walkr-author` uses.
- **Jargon goes in the glossary, not a parenthetical.**
- **Structure before specifics.** The first step is always the overview diagram.
- **Attribute everything.** Every step traces back to a specific fetched URL, if
  you can't name the source for a claim, verify it before writing the step, don't
  guess.

## Step 7: final self-check before declaring done

Walk every generated file against this list. Fix anything that fails, none of
these are matters of taste, they're the build-time/parse-time contract plus this
skill's own sourcing discipline.

- [ ] Every frontmatter block has `title`, `label`, `kind`, `order`, `layout`, `summary`.
- [ ] Any frontmatter value containing a literal `: ` is quoted.
- [ ] `label` is never identical to `title` on any step.
- [ ] `order` values are unique integers matching the intended
      big-picture → subsystems/phases → mechanisms → config → recap sequence.
- [ ] `layout` is exactly one of `overview` / `code-walk` / `config`.
- [ ] Every `overview` step has, at most, one mermaid block and one `:::deep` block.
- [ ] Every `code-walk` step has a bullet-list summary *before* its fenced code block.
- [ ] Every `config` step has **no** bullet-list summary before its fenced code block.
- [ ] Every fenced code block with `mark=` is immediately followed by an ordered
      list whose item count equals the count of numbers in `mark=`, in matching order.
- [ ] Every `[text]{def=id}` has a matching top-level `id:` key in `glossary.yaml`.
- [ ] `walkthrough.yaml` has only `title`/`tagline`/`repo`, no `groups[]`.
- [ ] The last step is `layout: overview`, no mermaid block, closes the loop.
- [ ] Every step ends with a `*Source: [...](...)* ` line naming the URL(s) it
      actually drew from.
- [ ] No step's prose is a bulk copy of source-doc paragraphs, everything is
      re-explained in your own words; only short verbatim snippets (code, config,
      exact field names) are quoted directly.
- [ ] Every step's `summary` and body prose has been through `/text-polish`,
      no hedging, filler, or AI-vocabulary tells left in.
- [ ] Every screenshot referenced from a step lives under `media/` in the
      walkthrough directory and is referenced with a plain `![]()`, never a directive.
- [ ] Every diagram that's *generative* (architecture/flow/state machine) is a
      mermaid block, never a static image; screenshots are reserved for "this is
      literally what it looks like."
- [ ] Nothing invents a frontmatter key or directive absent from
      `references/content-format.md`.

If any box fails, fix the specific file, don't rationalize the deviation.

## Output location

Write to `.walkr/` at the directory the user specified in Step 1, matching this
layout exactly:

```
.walkr/
├─ walkthrough.yaml
├─ glossary.yaml
├─ media/              # only if you captured screenshots
└─ steps/
   ├─ 01-overview.md
   ├─ 02-<slug>.md
   └─ ...
```

This skill does not need the `walkr` binary installed to do its job, it only
needs to produce content that conforms to the spec. Authoring the correct
markdown/YAML is the actual deliverable.

## Step 8: build the site

Check whether `walkr` is on `PATH` (`Bash`: `which walkr` or `command -v walkr`).

- **Not found:** say so, and give the user the manual command
  (`walkr build <walkthrough-dir> -o ./site`) rather than trying to install it
  yourself.
- **Found:** ask the user where they want the built site written (a relative path
  is fine, e.g. `./site`; don't assume one). Then run
  `walkr build <walkthrough-dir> -o <their answer>` via `Bash`, using the same
  `<walkthrough-dir>` from Step 1, and report the resulting path. `walkr build`
  output is fully self-contained static HTML (Alpine.js, Mermaid.js, and fonts all
  embedded, plus `media/` if you captured screenshots), so mention it opens
  straight from `file://` or from any static host, no server needed. If the user
  instead wants to preview it live, `walkr serve <walkthrough-dir> --open` builds
  to a temp dir and serves it, and needs no output-path decision.
