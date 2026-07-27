package render

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/bashfulrobot/repo-walker/internal/walkthrough"
)

// DeepDive is one :::deep{...}::: block, rendered into the page's single
// shared modal container rather than inline in the step.
type DeepDive struct {
	ID       string
	Title    string
	BodyHTML string
}

var deepDiveRe = regexp.MustCompile(`(?m)^:::deep\{title="([^"]*)"\}\n(?s:(.*?))\n:::[ \t]*$`)

// extractDeepDives replaces each :::deep{title="..."}...::: block with the
// "Go deeper" trigger button, and returns the extracted blocks for the
// caller to render separately into the shared modal.
func extractDeepDives(md, stepID string) (string, []DeepDive) {
	var deeps []DeepDive
	n := 0
	out := deepDiveRe.ReplaceAllStringFunc(md, func(m string) string {
		groups := deepDiveRe.FindStringSubmatch(m)
		title, body := groups[1], groups[2]
		n++
		id := fmt.Sprintf("%s-deep-%d", stepID, n)
		bodyHTML, err := renderInlineMarkdownBlock(body)
		if err != nil {
			bodyHTML = html.EscapeString(body)
		}
		deeps = append(deeps, DeepDive{ID: id, Title: title, BodyHTML: bodyHTML})
		return fmt.Sprintf(
			`<button class="deepen" type="button" x-on:click="openModal('%s')"><span class="deepen__glyph">&#9998;</span> Go deeper: %s</button>`,
			id, html.EscapeString(title),
		)
	})
	return out, deeps
}

var glossaryRe = regexp.MustCompile(`\[([^\]]+)\]\{def=([\w-]+)\}`)

// expandGlossaryTerms replaces [text]{def=id} with the term span + baked-in
// popover markup, looked up from the glossary at build time.
func expandGlossaryTerms(md string, gl walkthrough.Glossary) string {
	return glossaryRe.ReplaceAllStringFunc(md, func(m string) string {
		groups := glossaryRe.FindStringSubmatch(m)
		text, id := groups[1], groups[2]
		entry, ok := gl[id]
		if !ok {
			return fmt.Sprintf(`<span class="term" title="undefined glossary term: %s">%s</span>`, html.EscapeString(id), html.EscapeString(text))
		}
		var more string
		if entry.LearnMore != "" {
			more = fmt.Sprintf(`<a class="term__pop-more" href="%s" target="_blank" rel="noopener">Learn more &rarr;</a>`, html.EscapeString(entry.LearnMore))
		}
		return fmt.Sprintf(
			`<span class="term" x-data="{ open: false }" x-on:mouseenter="open = true" x-on:mouseleave="open = false" x-on:click="open = !open">%s`+
				`<template x-if="open"><span class="term__pop" x-cloak><span class="term__pop-label">Definition</span>`+
				`<span class="term__pop-def">%s</span>%s</span></template></span>`,
			html.EscapeString(text), html.EscapeString(strings.TrimSpace(entry.Definition)), more,
		)
	})
}

var mermaidRe = regexp.MustCompile("(?m)^```mermaid(?:\\s+title=\"([^\"]*)\")?[ \\t]*\\n(?s:(.*?))\\n```[ \\t]*$")

// renderMermaidBlocks replaces ```mermaid fenced blocks with the diagram
// frame; Mermaid.js renders the raw diagram source client-side, so the Go
// renderer never parses diagram syntax, only lifts the optional title.
func renderMermaidBlocks(md string) string {
	return mermaidRe.ReplaceAllStringFunc(md, func(m string) string {
		groups := mermaidRe.FindStringSubmatch(m)
		title, body := groups[1], groups[2]
		if title == "" {
			title = "diagram.mmd"
		}
		return fmt.Sprintf(
			`<div class="diagram-frame"><span class="diagram-frame__cap">%s</span><div class="mermaid">%s</div></div>`,
			html.EscapeString(title), html.EscapeString(body),
		)
	})
}
