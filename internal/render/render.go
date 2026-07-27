// Package render turns one step's frontmatter+body (see
// internal/walkthrough) into the HTML fragment the site assembler embeds.
// See docs/ai/content-format.md for the directive syntax this package implements.
package render

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

// Result is one step's rendered output.
type Result struct {
	HTML      string
	DeepDives []DeepDive
}

// newStepMarkdown builds a fresh goldmark instance (GFM + the annotated-code
// node renderer) and returns the renderer alongside so its Path side channel
// can be read after Convert. One per RenderStep call — cheap, and keeps the
// Path field race-free without needing synchronization.
func newStepMarkdown() (goldmark.Markdown, *codeBlockRenderer) {
	cbr := &codeBlockRenderer{}
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // directive preprocessing injects trusted raw HTML
			renderer.WithNodeRenderers(util.Prioritized(cbr, 0)),
		),
	)
	return md, cbr
}

// inlineMarkdown renders a small markdown fragment (deep-dive bodies,
// footnote text, titles) without the annotated-code extension.
var inlineMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe()),
)

// RenderStep converts one step's frontmatter-stripped body into HTML.
// Directive order: deep-dive extraction, glossary expansion, diagram
// framing (all raw-text preprocessing), then goldmark (which applies the
// annotated-code directive via a custom node renderer). For layout
// code-walk/config, the content before/after the annotated code block is
// then wrapped per docs/ai/content-format.md.
func RenderStep(step walkthrough.Step, gl walkthrough.Glossary) (*Result, error) {
	md := step.Body
	md, deeps := extractDeepDives(md, step.ID)
	md = expandGlossaryTerms(md, gl)
	md = renderMermaidBlocks(md)

	stepMarkdown, cbr := newStepMarkdown()
	var buf bytes.Buffer
	if err := stepMarkdown.Convert([]byte(md), &buf); err != nil {
		return nil, fmt.Errorf("rendering step %s: %w", step.ID, err)
	}
	out := buf.String()

	switch step.Layout {
	case walkthrough.LayoutCodeWalk:
		out = wrapCodeWalk(out, cbr.Path, step.ID)
	case walkthrough.LayoutConfig:
		out = wrapConfig(out)
	}

	return &Result{HTML: out, DeepDives: deeps}, nil
}

// wrapCodeWalk splits the rendered HTML at codeBoundary — everything before
// is the always-visible summary; everything from the boundary on is the
// annotated source, collapsed behind a "Show annotated source" toggle.
// codeOpen is keyed per-step (stepID) in assets/app.js since a real
// walkthrough has more than one code-walk step, unlike the prototype.
func wrapCodeWalk(rendered, filePath, stepID string) string {
	before, after, found := strings.Cut(rendered, codeBoundary)
	if !found {
		// No fenced code block in this step's body — render as-is rather
		// than silently dropping content (an author error, not fatal).
		return rendered
	}
	dir, file := path.Split(filePath)
	dir = strings.TrimSuffix(dir, "/")
	var head string
	if filePath != "" {
		head = fmt.Sprintf(`<span class="codewalk__path"><span class="dim">%s/</span>%s</span>`, dir, file)
	} else {
		head = `<span class="codewalk__path"></span>`
	}
	key := fmt.Sprintf("codeOpen['%s']", stepID)
	return fmt.Sprintf(
		`<div class="codewalk">`+
			`<div class="codewalk__head">%s`+
			`<button class="codewalk__toggle" type="button" x-on:click="%s = !%s">`+
			`<span class="codewalk__toggle-chev" :class="{ 'is-open': %s }">&#9656;</span>`+
			`<span x-text="%s ? 'Hide annotated source' : 'Show annotated source'"></span>`+
			`</button></div>`+
			`<div class="codewalk__summary">%s</div>`+
			`<div class="codewalk__deep" x-show="%s" x-cloak x-transition>%s</div>`+
			`</div>`,
		head, key, key, key, key, before, key, after,
	)
}

// wrapConfig wraps the annotated block in the bordered .manifest container;
// unlike code-walk, config renders it always-expanded (no summary, no toggle).
func wrapConfig(rendered string) string {
	before, after, found := strings.Cut(rendered, codeBoundary)
	if !found {
		return rendered
	}
	return before + `<div class="manifest">` + after + `</div>`
}

// RenderTitle renders a step's title, which may contain one _em_ or *em*
// span, into <em>-wrapped inline HTML (no wrapping <p>).
func RenderTitle(title string) (string, error) {
	return renderInlineMarkdownBlock(title)
}

// renderInlineMarkdownBlock renders a markdown fragment and strips a single
// wrapping <p>...</p> if goldmark added one, so the result is safe to embed
// inline (titles, deep-dive bodies, footnote text).
func renderInlineMarkdownBlock(md string) (string, error) {
	var buf bytes.Buffer
	if err := inlineMarkdown.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	out := strings.TrimSpace(buf.String())
	if strings.HasPrefix(out, "<p>") && strings.HasSuffix(out, "</p>") && strings.Count(out, "<p>") == 1 {
		out = strings.TrimSuffix(strings.TrimPrefix(out, "<p>"), "</p>")
	}
	return out, nil
}
