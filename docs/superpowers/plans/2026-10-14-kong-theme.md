# Kong theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. Steps use checkbox syntax.

**Goal:** Add a `kong` theme selectable by `--theme` or a `theme:` manifest key, with step content byte-identical across themes.

**Architecture:** `assets/style.css` keeps layout and gains tokens. `assets/themes/kong.css` overrides tokens and a few rules. `internal/site` resolves a `Theme` and the page template emits the extra link and logo only for non-default themes.

**Tech Stack:** Go 1.x, html/template, embed, Alpine.js, Mermaid.

**Spec:** `docs/superpowers/specs/2026-10-14-kong-theme-design.md`

## Global Constraints

- Default theme name `field-guide`; its `index.html` stays byte-identical to the pre-change output.
- Valid themes: `field-guide`, `kong`. Error text: `unknown theme "x", valid: field-guide, kong`.
- Precedence: flag, then manifest `theme:`, then `field-guide`.
- Kong colors: canvas `#000F06`, accent `#CCFF00` only on current step, progress fill, focus ring.
- Embedded directories must not be named `vendor`.

## Review Focus

- Manifest `theme: Kong` (wrong case) errors with the valid list.
- `build-all --theme kong` then `--theme kong --check` reports up to date.
- `build-all` without a flag on a library whose site pins `theme: kong` references the shared Kong CSS.
- Per-site build (`walkr build --theme kong`) carries the Kong CSS, logo and fonts beside `index.html`.
- Deep-dive modal bodies and glossary popups are identical across themes.

---

### Task 1: Tokenize style.css and Mermaid variables

**Files:** Modify `assets/style.css`, `assets/app.js`.

- [ ] Add to `:root` in style.css: `--code-str:#c9b78a; --code-num:#d68a5f; --veil:rgba(6,8,7,.72); --mark-wash:rgba(226,163,61,.08); --amber-glow:rgba(226,163,61,.16); --amber-line:rgba(226,163,61,.55); --teal-line:rgba(111,189,176,.6); --teal-wash:rgba(111,189,176,.05); --ghost:rgba(234,227,209,.025); --progress-fill:linear-gradient(90deg,var(--amber-deep),var(--amber-bright));` and `--mermaid-bg:#101512; --mermaid-primary:#171d19; --mermaid-text:#eae3d1; --mermaid-border:#e2a33d; --mermaid-line:#6fbdb0; --mermaid-secondary:#232b25; --mermaid-tertiary:#101512; --mermaid-font:'JetBrains Mono', monospace;`
- [ ] Replace each hardcoded literal with its token (`.st`, `.nm`, `.modal-veil`, `.is-marked`, active num shadow, link underline, `.term` underline, `.rail__repo` background, `.stage::before`, `.rail__progress-fill`).
- [ ] In app.js build `themeVariables` from `getComputedStyle(document.documentElement).getPropertyValue('--mermaid-*')` (trimmed), same keys as today.
- [ ] Run `go test ./...`; commit `refactor(assets): move hardcoded theme colors into tokens`.

### Task 2: Theme resolution in internal/site (TDD)

**Files:** Create `internal/site/theme.go`, `internal/site/theme_test.go`; modify `internal/site/site.go`, `internal/site/page.html.tmpl`, `internal/walkthrough/walkthrough.go`.

**Interfaces:** Produces `site.Theme{Name, CSS, Logo string}`, `site.DefaultTheme`, `site.ThemeNames() []string`, `site.ResolveTheme(flag, manifest string) (Theme, error)`, `site.Options.Theme string`, `walkthrough.Manifest.Theme string`.

- [ ] Write tests: flag beats manifest; manifest beats default; empty gives `field-guide` with empty `CSS` and `Logo`; `Kong` and `x` error with the valid list.
- [ ] Implement `theme.go` with a map of the two themes. `kong` has `CSS: "assets/themes/kong.css"`, `Logo: "vendor/kong/logomark.svg"`.
- [ ] `BuildWith` resolves the theme from `opts.Theme` and `wt.Manifest.Theme`, and puts `Theme` into `pageData`.
- [ ] Template: `<html lang="en"{{if .Theme.CSS}} data-theme="{{.Theme.Name}}"{{end}}>`; after the style link `{{if .Theme.CSS}}<link rel="stylesheet" href="{{.AssetBase}}{{.Theme.CSS}}" />{{end}}`; inside `.rail__brand` before the mark `{{if .Theme.Logo}}<img class="rail__logo" src="{{.AssetBase}}{{.Theme.Logo}}" alt="" width="44" />{{end}}`.
- [ ] Capture a golden default `index.html` before editing the template (build the dogfood walkthrough on `main`), assert equality in a test using a fixture directory under `internal/site/testdata`.
- [ ] Commit `feat(site): resolve a theme from flag or manifest`.

### Task 3: Kong assets and shared writer

**Files:** Create `assets/themes/kong.css`, `assets/thirdparty/kong/fonts.css`, `assets/thirdparty/kong/fonts/*.ttf`, `assets/thirdparty/kong/logomark.svg`, OFL files; modify `assets/assets.go`, `internal/site/site.go`.

- [ ] Copy from the kong-branding skill: Funnel Sans Regular, Italic, Bold, ExtraBold; Roboto Mono Regular, Italic, Bold; Space Grotesk Regular, Bold; the three OFL.txt files; `for-dark-backgrounds/svg/Kong-Logomark.svg` as `logomark.svg`.
- [ ] Write `fonts.css` with `@font-face` rules (relative `fonts/...`) and `kong.css` per the spec's Kong visual design. `kong.css` imports `../../vendor/kong/fonts.css`.
- [ ] `assets.go`: `//go:embed themes` as `Themes embed.FS`.
- [ ] `WriteSharedAssets` also copies `Themes` to `assets/themes`.
- [ ] Tests: shared output contains `assets/themes/kong.css`, `vendor/kong/logomark.svg`, every font; `assets_test.go` asserts the embed holds them.
- [ ] Commit `feat(assets): add the Kong theme stylesheet, fonts and logo`.

### Task 4: CLI flags and library plumbing

**Files:** Modify `main.go`, `buildall.go`, `internal/library/library.go`, tests.

- [ ] `--theme` on `build`, `build-all`, `serve`; validate early with `site.ResolveTheme(flag, "")` so a bad flag errors once.
- [ ] `library.Options.Theme`, passed to `site.Options.Theme` in `buildOne`; `Check` already reuses `BuildAll`.
- [ ] Tests: library with one default site and one `theme: kong` site; `Check` is OK after `BuildAll`; `--theme kong` build then check with the same option is OK, without it reports Stale.
- [ ] Commit `feat(cli): add --theme to build, build-all and serve`.

### Task 5: Content invariance test

**Files:** Create `internal/site/invariance_test.go`.

- [ ] Build `.walkr` fixture (use repo root `.walkr` plus a fixture with all three layouts and a deep-dive) in both themes. Extract every `<section id=... class="step ...">...</section>` and each deep-dive `modal__body` with a regexp. Assert equal strings, and assert the step count is above zero.
- [ ] Assert `render.RenderStorage` output (the Confluence path) has no theme parameter by building it once and comparing across a second call (documents the invariant).
- [ ] Commit `test(site): pin step content as theme independent`.

### Task 6: Docs, skills, brand audit, visual check

**Files:** Modify `docs/ai/content-format.md`, `skills/walkr-author/references/content-format.md`, `skills/walkr-tutorial-author/references/content-format.md`, `README.md`, `docs/user/README.md`.

- [ ] Document `theme:` (optional, `field-guide` or `kong`, flag overrides) identically in the three format files; usage in both READMEs.
- [ ] Run `python3 <kong-branding>/scripts/brand-audit.py assets/themes/kong.css` and fix findings.
- [ ] Build the dogfood site with each theme, screenshot with Playwright (overview, code-walk, config, diagram), fix visible defects.
- [ ] Final baseline: `go test ./... && go vet ./... && gofmt -l . && go run . build .walkr -o $CLAUDE_JOB_DIR/tmp/site`.
- [ ] Commit `docs: document the theme key and --theme flag`.
