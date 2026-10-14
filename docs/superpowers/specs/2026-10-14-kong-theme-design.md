# Kong theme for walkr

## Goal

Add a second theme, `kong`, for work and Kong projects. The current look becomes the
named theme `field-guide` and stays the default. Switching an existing site to `kong`
is a rebuild. No step, glossary, media, or manifest content changes.

## Hard constraint: content invariance

A theme may change only the page chrome (`page.html.tmpl`), CSS, fonts, and the logo.
It never changes rendered step HTML, deep-dive bodies, glossary output, or the
Confluence storage output.

- `render.RenderStep` and `RenderTitle` run without any theme input.
- The Confluence path (`internal/render/storage.go`, `internal/confluence`) takes no theme.
- A test enforces this (see Testing).

## Theme selection

- `--theme field-guide|kong` on `walkr build`, `walkr build-all` and `walkr serve`.
- Optional `theme:` key in `walkthrough.yaml` pins a single site.
- Precedence: flag, then manifest key, then `field-guide`.
- An unknown theme name is an error that lists the valid names.
- `build-all --theme kong --check` compares committed sites against a Kong build.
- The `theme:` key is consumed by the page template, which satisfies the rule in
  `walkthrough.Manifest` that every field is rendered by the UI.

## Architecture (token layer plus override sheet)

1. **Token refactor of `assets/style.css`.** Hardcoded colors move into custom
   properties in `:root`: the `.nm` code token (`#d68a5f`), the modal veil
   (`rgba(6, 8, 7, 0.72)`), the body backdrop gradients, and the rail repo tint.
   Visual output of `field-guide` does not change.
2. **`assets/themes/kong.css`.** Loads after `style.css`. Overrides tokens, font
   variables, and a few structural rules. The wizard grid and behavior are untouched.
3. **`internal/site`.** A `Theme` value (name, stylesheet path, logo path) is resolved
   once and added to `pageData`. The template emits a second `<link>` and a rail logo
   block only for themes that define them. For `field-guide` the generated
   `index.html` is byte-identical to today's output.
4. **Shared assets.** `WriteSharedAssets` always writes every theme's files:
   `assets/themes/kong.css`, `vendor/kong/logotype.svg`, and Kong fonts under
   `vendor/fonts/kong/`. One `_walkr/` directory serves a library that mixes themes.
5. **Mermaid.** `app.js` reads `--mermaid-*` custom properties through
   `getComputedStyle` to build `themeVariables`. `app.js` holds no theme names.
   `field-guide` defines the variables with today's values.
6. **Embedding.** Theme files live under `assets/` and are embedded. Directory
   naming avoids `vendor`, per the go-install fix in PR #13.

## Kong visual design

Source of truth: the `kong-cs:kong-branding` skill (2026 v1.1 tokens, fonts, logos).

- **Color.** Canvas `#000F06`. Text white `#FFFFFF`, secondary Bay `#B7BDB5`, muted
  `#858983`. Surfaces and borders from the neutral ramp (`#101110`, `#2D2E2C`,
  `#4A4D49`). Electric lime `#CCFF00` is the only accent.
- **Accent use.** Lime marks three things: the current step in the rail, the progress
  fill, and the keyboard focus ring. The logotype is lime by brand definition.
  No gradients, no second accent.
- **Type.** Funnel Sans for headings and body (ExtraBold titles, Regular body,
  Italic where the content uses emphasis). Roboto Mono for code. Space Grotesk for
  the nav buttons only. Sentence case, no tracked all-caps labels.
- **Layout.** Same shell and grid as `field-guide`. Flat dark canvas with no graph
  paper and no glows. Solid neutral hairlines replace dashed leader lines. The Kong
  logotype replaces the visible title at the top of the rail, so the rail says "Kong".
  The walkthrough title stays in the page for screen readers and the document title.
- **Code and diagrams.** Syntax token colors collapse to neutrals with at most one
  emphasized token color. Mermaid uses neutral fills with lime borders on primary nodes only.
- **Focus.** Visible lime focus ring on all interactive elements. Motion respects
  `prefers-reduced-motion`.
- **Assets.** Funnel Sans (Light omitted; Regular, Italic, Bold, ExtraBold), Roboto
  Mono (Regular, Bold), Space Grotesk (Regular, Bold) as bundled TTF, about 300 KB.
  Logotype SVG from `assets/logos/for-dark-backgrounds/svg/Kong-Logotype.svg`. Fonts
  carry their SIL OFL files. The logo is used unmodified. Internal Kong use needs no
  trademark disclaimer.

Design plan review: near-black with an acid-green accent is a common generated
default. It is kept here because the brand guide mandates it. Type, hairline rail, and
the single-accent rule come from the brief, not from a template.

## Components touched

| Path | Change |
|---|---|
| `assets/style.css` | colors to tokens, `--mermaid-*` vars |
| `assets/themes/kong.css` | new |
| `assets/kong/`, `assets/thirdparty/fonts/kong/` | logo, fonts, OFL files |
| `assets/assets.go` | embed theme CSS and logo |
| `assets/app.js` | Mermaid vars from CSS |
| `internal/site/` | `Theme`, `Options.Theme`, template block, shared asset writer |
| `internal/walkthrough/walkthrough.go` | `Manifest.Theme` |
| `internal/library/library.go` | pass theme through build and check |
| `main.go`, `buildall.go` | `--theme` flag, validation |
| `docs/ai/content-format.md` and both skill copies | document `theme:` |
| `README.md`, `docs/user/README.md` | usage |

## Error handling

- Unknown theme from flag or manifest: `unknown theme "x", valid: field-guide, kong`.
- Missing embedded theme file: build fails (guarded by the embed test).

## Testing

1. **Content invariance.** Build one fixture in both themes. Extract every step
   section and deep-dive body from each `index.html`. Assert byte equality. Assert the
   Confluence storage rendering is identical, since it takes no theme.
2. **Default stability.** `field-guide` `index.html` matches a golden file taken before the change.
3. **Resolution.** Flag beats manifest, manifest beats default, unknown name errors.
4. **Shared assets.** `WriteSharedAssets` output includes the Kong CSS, logo and fonts. A
   `build-all` library with one `kong` site and one default site passes `--check`.
5. **Embed.** Kong fonts and logo present in `assets.ThirdParty` and theme embed, in
   the same style as the PR #13 regression test.
6. **Brand audit.** `brand-audit.py assets/themes/kong.css` exits 0.
7. **Visual.** Playwright screenshots of the dogfood site in both themes, all three
   layouts (overview, code-walk, config) and a Mermaid diagram.
8. **Baseline.** `go test ./... && go vet ./... && gofmt -l .`

## Out of scope

Light Kong theme, per-step themes, Confluence theming, font subsetting, new layouts or
directives, any content-format change beyond the `theme:` manifest key.

## Rollout note

After the token refactor the shared `_walkr/assets/style.css` changes once. Existing
libraries show that single diff in `build-all --check` until rebuilt. Site `index.html`
files for the default theme do not change.
