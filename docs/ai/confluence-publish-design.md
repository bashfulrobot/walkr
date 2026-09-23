# Publishing walkr walkthroughs to Confluence

Status: config, secrets, Confluence client, storage-format renderer, and the publisher are built and tested, and a real walkthrough has been published. PNG diagram rendering is not built.

## Goal

Publish a walkr walkthrough to Confluence as native pages that keep walkr's idea: minimal information per page, walking a reader through a topic one concept at a time. Publishing is one command in the walkr binary, so it can run unattended.

## Decisions

| # | Decision | Choice |
|---|---|---|
| 1 | Who publishes | The Go binary, calling the Confluence Cloud REST v1 API directly through the api.atlassian.com gateway. The Atlassian Rovo MCP is not used for publishing, because it only exists inside a Claude session and has no attachment upload. |
| 2 | Page tree | A section page ("Tutorials", created once and never rewritten), one tutorial page per walkthrough under it (tagline, start link, ordered chapter list), and one child page per step under the tutorial page. The chapter list is explicit, since the children macro sorts alphabetically. |
| 3 | Diagrams | Render Mermaid to PNG at publish time with headless Chrome against the vendored mermaid.min.js, upload as an attachment, reference by filename. When Chrome is missing, or `diagrams: source` is set, the source goes in a code block inside an expand. The native panel diagram from the spike is dropped now that attachments work. |
| 4 | Glossary | A Terms panel at the bottom of each page, listing only the terms used on that page. |
| 5 | Sync | One way. The repo is the source of truth. Republish overwrites Confluence edits, and each page says so. |
| 6 | Style | Native Confluence elements only. Fonts and CSS cannot carry over. The palette is deliberately quiet: one soft blue for the lede and start panels, one light gray for Terms, grey lozenges, and no panel icons. Both colors come from Confluence's own background palette, so they follow dark mode. |
| 7 | Navigation | A pager row on each page (previous, "N of M" lozenge, next), plus the parent's chapter list. |
| 8 | Config | One global YAML file at ~/.config/walkr/config.yaml. It holds pointers to secrets, never values. |
| 9 | Secrets | The 1Password Go SDK with a service account, read from OP_SERVICE_ACCOUNT_TOKEN. The Atlassian API token is resolved from an op:// reference in memory and never printed. |
| 10 | Page identity | Labels of the form walkr-<target>-<step id>, found with CQL label search. |
| 11 | Page status | Published (current) pages. Drafts are not versioned, so a draft must be saved at its current version number, and label lookup of drafts is unverified. |

## Config file

```yaml
confluence:
  cloud_id: <site cloud id>
  email: <account email>
  auth:
    service_account_token_env: OP_SERVICE_ACCOUNT_TOKEN   # optional, this is the default
    token_ref: op://<vault>/<item>/<field>
  targets:
    my-walkthrough:
      dir: ~/path/to/.walkr
      space_key: <space key>
      parent_id: "<page id>"     # where the section, or without one the tutorial page, goes
      section: Tutorials         # optional page created under parent_id, tutorials go beneath it
      diagrams: png            # png or source
```

The parser rejects unknown keys, so a literal token: key is an error, and token_ref must start with op://. The Secret type prints [redacted] through every fmt, JSON, and slog path.

## API token scopes

The token is a scoped token, which only authenticates through the gateway URL https://api.atlassian.com/ex/confluence/<cloudId>. Scopes in use:

```
read:content:confluence
read:content-details:confluence
write:content:confluence
read:space:confluence
read:space-details:confluence
read:label:confluence
write:label:confluence
write:attachment:confluence
search:confluence
```

The v2 pages API is not available with these scopes, so the client uses v1 content endpoints. Atlassian may deprecate parts of v1. Space write and delete scopes and page delete are not needed, so test pages are deleted by hand.

## Verified against real Confluence

| Behavior | Result |
|---|---|
| Resolve the token through the service account, list spaces, read a page | Works |
| Create a draft page, read it back | Status draft, version 1 |
| Update a draft | Works only at the current version number, otherwise 409 |
| Update a published page | Version increments by one |
| Upload a PNG attachment, reference it in the body | Upload works, page stores an ac:image with ri:attachment, and the image renders inline |
| Label search on a published page | Finds it after about 16 seconds of indexing delay, immediately after later updates |
| Inline SVG through a data URI | Fails, stored but not displayed |
| Publish a nine chapter walkthrough, then publish again immediately | First run creates 11 pages (section, tutorial, 9 steps). The rerun updates all 11 and creates none, inside the label indexing delay |
| Label search on a draft | 0 results in an immediate query, inconclusive because of the indexing delay |

## Conventions

- Page title of a step is `<walkthrough title>: <step title with emphasis markers removed>`. Titles are unique per space, so the prefix avoids collisions.
- Labels are walkr-<target> on every page, walkr-<target>-index on the parent, and walkr-<target>-<step id> on each step. Labels are lowercase with no spaces.
- Body layout is eyebrow lozenges ("Chapter NN" and the step kind), a yellow lede panel from the step summary, the content, an optional Terms panel, a rule, the pager, and a small footer noting that edits are overwritten.

## Element mapping

| walkr | Confluence storage format |
|---|---|
| kind and order | Two status lozenges |
| summary | Yellow panel macro, bold text |
| Body prose, lists, tables | XHTML from goldmark |
| Mermaid block | Attachment image with a caption, or the source in an expand |
| :::deep | Expand macro titled "Go deeper: ..." |
| [term]{def=id} | Bold text, definition in the Terms panel |
| [text]{step=id} | Link to the child page URL |
| Annotated code (path, mark) | Code macro with line numbers, then a numbered "Line N" list. In a code-walk layout both sit in an expand |
| Source attribution line | Italic line with links, unchanged |

Unknown step links and undefined glossary terms render as plain text and produce warnings. A mark count that does not match the footnote list is a hard error, as in the site build.

## Publish flow

1. Load and validate the walkthrough, then the config and target.
2. Resolve the token through 1Password.
3. Find or create the section page by exact title. An existing section page is left untouched.
4. Find or create the tutorial page and each step page. A page is found by label first, then by exact title, and created only if neither exists. The title check matters because label search lags creation by about 16 seconds, so a fast rerun would otherwise duplicate pages. A renamed step is still found by its label and retitled.
5. Second pass, render every page with real links, using the IDs from step 4. Search is not used in this pass. Each page is saved as a minor edit so watchers are not notified.
6. If a diagram renderer is configured, render each Mermaid block to PNG and attach it, named by content hash. Any failure falls back to the source in an expand and adds a warning.

The command is `walkr confluence publish --target <name>`, with `--dry-run` to look pages up and report the plan without writing.

## Code layout

| Package | Role |
|---|---|
| internal/secrets | Secret type and the 1Password resolver |
| internal/config | The global config file |
| internal/confluence | REST client, check logic, and the publisher |
| internal/render | RenderStorage and RenderStorageIndex, next to the site renderer, sharing its directive parsing |
| main package | walkr confluence check and walkr confluence publish |

Tests use httptest and a fake resolver. RenderStorage has golden files plus a check that every output is well-formed XML. Integration tests run against real Confluence behind the integration build tag.

## Not built yet

- PNG diagram rendering with headless Chrome. Until then diagrams show as source in an expand.
- The nix vendorHash, stale since the 1Password SDK was added to go.mod.
- The walkr-confluence-publish skill wrapper, and pointers from walkr-author and walkr-tutorial-author.
- A section in docs/user/README.md.

## Risks and open items

- v1 endpoints may be deprecated. Moving to v2 needs granular page scopes, and attachment upload stays on v1.
- Code macro language names other than the mapped set fall back to plain text. Go is mapped but not confirmed to highlight.
- The full look of a real published walkthrough has not been reviewed by eye yet. The PNG attachment and pager were confirmed on test pages.
- Every republish saves a new minor version of every page, since Confluence normalizes storage and the body cannot be compared. A content hash label could skip unchanged pages.
- An optional link card to a hosted copy of the full site is not designed, because no hosting URL exists.
- Spike and test pages are left for manual deletion.
