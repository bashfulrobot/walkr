package render

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// codeBoundary is written immediately before a fenced code block's <pre>, so
// RenderStep can split "content before the code" from "the code itself" to
// apply the layout-specific wrapper (see docs/content-format.md, layouts
// code-walk and config). Not visible in output — split out before returning.
const codeBoundary = "\x00REPO-WALKER-CODE-BOUNDARY\x00"

var pathAttrRe = regexp.MustCompile(`path="([^"]*)"`)
var markAttrRe = regexp.MustCompile(`mark=([0-9,]+)`)

// codeBlockRenderer implements the annotated-code directive: a fenced code
// block with `path=`/`mark=` attributes, whose immediately-following ordered
// list supplies footnote text for the marked lines. It overrides goldmark's
// default FencedCodeBlock rendering entirely (registered at top priority),
// so it also has to render plain (non-annotated) fenced code blocks.
//
// Path is a side channel: the last-seen `path` attribute, read by RenderStep
// after Convert returns to build the codewalk__head bar. Safe because a
// fresh codeBlockRenderer (and goldmark.Markdown) is created per RenderStep
// call — see newStepMarkdown.
type codeBlockRenderer struct {
	Path string
}

func (r *codeBlockRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *codeBlockRenderer) render(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*ast.FencedCodeBlock)
	lang := string(node.Language(source))

	info := ""
	if node.Info != nil {
		info = string(node.Info.Segment.Value(source))
	}
	if m := pathAttrRe.FindStringSubmatch(info); m != nil {
		r.Path = m[1]
	}

	var markNums []int
	if m := markAttrRe.FindStringSubmatch(info); m != nil {
		for _, s := range strings.Split(m[1], ",") {
			num, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil {
				return ast.WalkStop, fmt.Errorf("invalid mark= attribute %q: %w", m[1], err)
			}
			markNums = append(markNums, num)
		}
	}

	badgeForLine := map[int]int{}
	for i, ln := range markNums {
		badgeForLine[ln] = i + 1
	}

	lineCount := node.Lines().Len()
	io.WriteString(w, codeBoundary)
	io.WriteString(w, `<pre class="code"><code>`)
	for i := 0; i < lineCount; i++ {
		seg := node.Lines().At(i)
		raw := strings.TrimRight(string(seg.Value(source)), "\n")
		classified := classifyLine(lang, raw)
		lineNo := i + 1
		if badge, marked := badgeForLine[lineNo]; marked {
			fmt.Fprintf(w, `<span class="ln is-marked">%s<span class="mark">%d</span></span>`+"\n", classified, badge)
		} else {
			fmt.Fprintf(w, `<span class="ln">%s</span>`+"\n", classified)
		}
	}
	io.WriteString(w, `</code></pre>`)

	if len(markNums) > 0 {
		list, ok := node.NextSibling().(*ast.List)
		if !ok {
			return ast.WalkStop, fmt.Errorf("fenced code block has mark=%v but is not followed by an ordered list of footnotes", markNums)
		}
		var footnotes []string
		for item := list.FirstChild(); item != nil; item = item.NextSibling() {
			text, err := extractListItemHTML(item, source)
			if err != nil {
				return ast.WalkStop, err
			}
			footnotes = append(footnotes, text)
		}
		if len(footnotes) != len(markNums) {
			return ast.WalkStop, fmt.Errorf("mark=%v names %d line(s) but the following list has %d item(s) — they must match 1:1", markNums, len(markNums), len(footnotes))
		}
		io.WriteString(w, `<ul class="footnotes">`)
		for i, f := range footnotes {
			fmt.Fprintf(w, `<li><span class="mark">%d</span> %s</li>`, i+1, f)
		}
		io.WriteString(w, `</ul>`)

		if parent := list.Parent(); parent != nil {
			parent.RemoveChild(parent, list)
		}
	}

	return ast.WalkSkipChildren, nil
}

// hasLines matches ast.Paragraph, ast.TextBlock, etc. — leaf block nodes
// that carry their raw source as a set of line segments.
type hasLines interface {
	Lines() *text.Segments
}

// extractListItemHTML renders one footnote list item's inline content as
// inline HTML, by walking down to its leaf line-bearing block(s) and
// re-rendering their raw markdown source.
func extractListItemHTML(item ast.Node, source []byte) (string, error) {
	var raw strings.Builder
	if err := appendNodeSource(item, source, &raw); err != nil {
		return "", err
	}
	if raw.Len() == 0 {
		return "", fmt.Errorf("empty footnote list item")
	}
	return renderInlineMarkdownBlock(raw.String())
}

func appendNodeSource(n ast.Node, source []byte, out *strings.Builder) error {
	// A node can structurally satisfy hasLines (it's a *ast.BaseBlock under
	// the hood, e.g. ast.ListItem) while still holding zero lines of its
	// own — the real text lives on its TextBlock/Paragraph child. Only
	// treat Lines() as authoritative when it's non-empty; otherwise recurse.
	if lb, ok := n.(hasLines); ok {
		lines := lb.Lines()
		if lines.Len() > 0 {
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				out.Write(seg.Value(source))
			}
			return nil
		}
	}
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if err := appendNodeSource(c, source, out); err != nil {
			return err
		}
	}
	return nil
}
