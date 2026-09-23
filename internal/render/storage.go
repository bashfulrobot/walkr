package render

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	ghtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

// This file renders a step into Confluence storage format (the XHTML plus
// ac: macros the REST API stores) instead of site HTML. It shares the
// directive parsing with RenderStep so both targets follow one spec, and it is
// pure: URLs and diagram image names arrive through StorageOptions, so it needs
// no network and its output can be compared to golden files.

// NavLink points at a sibling page.
type NavLink struct {
	Title string
	URL   string
}

// StorageOptions supplies what the renderer cannot know on its own.
type StorageOptions struct {
	Index int      // 1-based position of this step
	Total int      // number of steps
	Prev  *NavLink // nil on the first step
	Next  *NavLink // nil on the last step

	// StepURL resolves a {step=id} link to a page URL. When nil or when it
	// returns false the link degrades to plain text and a warning.
	StepURL func(id string) (string, bool)

	// Diagram maps Mermaid source to an attachment file name holding its
	// rendered PNG. When nil or false the source is shown in an expand.
	Diagram func(source string) (filename string, ok bool)
}

// StorageResult is one page body.
type StorageResult struct {
	Body     string
	Warnings []string
}

const (
	codeStart = "\x00WALKR-CODE-START\x00"
	codeEnd   = "\x00WALKR-CODE-END\x00"
)

// The whole Confluence output uses one soft accent and one neutral, both from
// Confluence's own background palette so they adapt to dark mode. Lozenges stay
// grey and panels carry no icons, to keep the pages quiet.
const (
	accentBG  = "#DEEBFF" // light blue: lede and start panels
	neutralBG = "#F4F5F7" // light gray: terms
)

var slotRe = regexp.MustCompile(`<p>@@WALKR:(\d+)@@</p>`)

type storageState struct {
	gl       walkthrough.Glossary
	opts     StorageOptions
	slots    []string
	terms    []string
	termSeen map[string]bool
	warnings []string
}

func (s *storageState) slot(xml string) string {
	s.slots = append(s.slots, xml)
	return fmt.Sprintf("\n\n@@WALKR:%d@@\n\n", len(s.slots)-1)
}

func (s *storageState) warnf(format string, args ...any) {
	s.warnings = append(s.warnings, fmt.Sprintf(format, args...))
}

// RenderStorage renders one step as a complete Confluence page body: eyebrow,
// lede, content, terms, sources, and the pager footer.
func RenderStorage(step walkthrough.Step, gl walkthrough.Glossary, opts StorageOptions) (*StorageResult, error) {
	st := &storageState{gl: gl, opts: opts, termSeen: map[string]bool{}}

	md := st.deepDives(step.Body)
	md = st.mermaid(md)
	md = st.terms_(md)
	md = st.stepLinks(md)

	body, err := st.convert(md, step.ID)
	if err != nil {
		return nil, fmt.Errorf("rendering step %s: %w", step.ID, err)
	}
	body = st.layoutWrap(body, step)
	body = strings.ReplaceAll(body, codeStart, "")
	body = strings.ReplaceAll(body, codeEnd, "")

	var content strings.Builder
	content.WriteString(eyebrow(step))
	content.WriteString(panel(accentBG, "<p><strong>"+esc(step.Summary)+"</strong></p>"))
	content.WriteString(body)
	content.WriteString(st.termsPanel())
	content.WriteString("<hr />")

	return &StorageResult{
		Body:     assemble(content.String(), pager(opts)),
		Warnings: st.warnings,
	}, nil
}

// RenderStorageIndex renders the walkthrough's parent page: the tagline, the
// summary of every chapter in order, and a link to start.
func RenderStorageIndex(m walkthrough.Manifest, steps []walkthrough.Step, stepURL func(id string) (string, bool)) (*StorageResult, error) {
	res := &StorageResult{}
	var c strings.Builder
	if m.Tagline != "" {
		c.WriteString("<p>" + statusMacro(m.Tagline, "") + "</p>")
	}
	if len(steps) > 0 {
		if u, ok := stepURL(steps[0].ID); ok {
			c.WriteString(panel(accentBG,
				`<p><strong>Start with chapter 1: </strong><a href="`+esc(u)+`">`+esc(PlainTitle(steps[0].Title))+`</a></p>`))
		}
	}
	c.WriteString("<h2>Chapters</h2><ol>")
	for _, s := range steps {
		u, ok := stepURL(s.ID)
		if !ok {
			res.Warnings = append(res.Warnings, "no URL for step "+s.ID)
			c.WriteString("<li><strong>" + esc(PlainTitle(s.Title)) + "</strong>: " + esc(s.Summary) + "</li>")
			continue
		}
		c.WriteString(`<li><a href="` + esc(u) + `">` + esc(PlainTitle(s.Title)) + `</a>: ` + esc(s.Summary) + `</li>`)
	}
	c.WriteString("</ol><hr />")
	res.Body = assemble(c.String(), "")
	return res, nil
}

var emphasisRe = regexp.MustCompile(`([_*])([^_*]+)([_*])`)

// PlainTitle strips the single _em_ or *em* span a step title may carry, since
// Confluence page titles are plain text.
func PlainTitle(title string) string {
	return strings.TrimSpace(emphasisRe.ReplaceAllString(title, "$2"))
}

func esc(s string) string { return html.EscapeString(s) }

func (s *storageState) deepDives(md string) string {
	return deepDiveRe.ReplaceAllStringFunc(md, func(m string) string {
		g := deepDiveRe.FindStringSubmatch(m)
		title, body := g[1], g[2]
		inner, err := s.convert(s.linkify(body), "deep")
		if err != nil {
			s.warnf("deep dive %q: %v", title, err)
			inner = "<p>" + esc(body) + "</p>"
		}
		return s.slot(expand("Go deeper: "+title, inner))
	})
}

func (s *storageState) mermaid(md string) string {
	return mermaidRe.ReplaceAllStringFunc(md, func(m string) string {
		g := mermaidRe.FindStringSubmatch(m)
		title, src := g[1], g[2]
		if title == "" {
			title = "diagram.mmd"
		}
		if s.opts.Diagram != nil {
			if name, ok := s.opts.Diagram(src); ok {
				return s.slot(`<p style="text-align: center;"><ac:image ac:alt="` + esc(title) + `"><ri:attachment ri:filename="` + esc(name) + `" /></ac:image></p>` +
					`<p style="text-align: center;"><em>` + esc(title) + `</em></p>`)
			}
		}
		return s.slot(expand("Diagram source ("+title+")", codeMacro("text", "", src)))
	})
}

// terms_ turns [text]{def=id} into bold text and records the term for the
// Terms panel.
func (s *storageState) terms_(md string) string {
	return glossaryRe.ReplaceAllStringFunc(md, func(m string) string {
		g := glossaryRe.FindStringSubmatch(m)
		text, id := g[1], g[2]
		if _, ok := s.gl[id]; !ok {
			s.warnf("undefined glossary term %q", id)
			return text
		}
		if !s.termSeen[id] {
			s.termSeen[id] = true
			s.terms = append(s.terms, id)
		}
		return "**" + text + "**"
	})
}

func (s *storageState) stepLinks(md string) string {
	return stepLinkRe.ReplaceAllStringFunc(md, func(m string) string {
		g := stepLinkRe.FindStringSubmatch(m)
		text, id := g[1], g[2]
		if s.opts.StepURL != nil {
			if u, ok := s.opts.StepURL(id); ok {
				return "[" + text + "](" + u + ")"
			}
		}
		s.warnf("no page URL for step link %q", id)
		return text
	})
}

// linkify applies term and step-link expansion to a fragment.
func (s *storageState) linkify(md string) string { return s.stepLinks(s.terms_(md)) }

func (s *storageState) convert(md, stepID string) (string, error) {
	cr := &storageCodeRenderer{}
	gm := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			ghtml.WithXHTML(),
			ghtml.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(cr, 0)),
		),
	)
	var buf bytes.Buffer
	if err := gm.Convert([]byte(md), &buf); err != nil {
		return "", err
	}
	out := slotRe.ReplaceAllStringFunc(buf.String(), func(m string) string {
		n, _ := strconv.Atoi(slotRe.FindStringSubmatch(m)[1])
		return s.slots[n]
	})
	return out, nil
}

// layoutWrap applies the step layout to the rendered body. In a code-walk the
// annotated source collapses into an expand, in a config it stays open.
func (s *storageState) layoutWrap(body string, step walkthrough.Step) string {
	if step.Layout != walkthrough.LayoutCodeWalk {
		return body
	}
	start := strings.Index(body, codeStart)
	end := strings.Index(body, codeEnd)
	if start < 0 || end < start {
		return body
	}
	inner := body[start+len(codeStart) : end]
	return body[:start] + expand("Show annotated source", inner) + body[end+len(codeEnd):]
}

func (s *storageState) termsPanel() string {
	if len(s.terms) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<p><strong>Terms</strong></p><ul>")
	for _, id := range s.terms {
		e := s.gl[id]
		b.WriteString("<li><strong>" + esc(e.Term) + "</strong>: " + esc(strings.Join(strings.Fields(e.Definition), " ")))
		if e.LearnMore != "" {
			b.WriteString(` <a href="` + esc(e.LearnMore) + `">Learn more</a>`)
		}
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	return panel(neutralBG, b.String())
}

func eyebrow(step walkthrough.Step) string {
	return "<p>" + statusMacro(fmt.Sprintf("Chapter %02d", step.Order), "") + " " + statusMacro(step.Kind, "") + "</p>"
}

func statusMacro(title, colour string) string {
	params := `<ac:parameter ac:name="title">` + esc(title) + `</ac:parameter>`
	if colour != "" {
		params += `<ac:parameter ac:name="colour">` + colour + `</ac:parameter>`
	}
	return `<ac:structured-macro ac:name="status" ac:schema-version="1">` + params + `</ac:structured-macro>`
}

func macro(name string, params [][2]string, richBody string) string {
	var b strings.Builder
	b.WriteString(`<ac:structured-macro ac:name="` + name + `" ac:schema-version="1">`)
	for _, p := range params {
		b.WriteString(`<ac:parameter ac:name="` + p[0] + `">` + esc(p[1]) + `</ac:parameter>`)
	}
	b.WriteString("<ac:rich-text-body>" + richBody + "</ac:rich-text-body></ac:structured-macro>")
	return b.String()
}

func panel(bg, richBody string) string {
	return macro("panel", [][2]string{{"bgColor", bg}}, richBody)
}

func expand(title, richBody string) string {
	return macro("expand", [][2]string{{"title", title}}, richBody)
}

// confluenceLang maps a fence language to a code macro language name. Names
// the macro does not know render as plain text, so an unknown one is safe.
var confluenceLang = map[string]string{
	"yaml": "yml", "yml": "yml", "javascript": "js", "js": "js", "python": "py", "py": "py",
	"shell": "bash", "sh": "bash", "bash": "bash", "html": "xml", "xml": "xml",
	"go": "go", "json": "json", "sql": "sql", "diff": "diff", "java": "java", "ruby": "ruby",
}

func codeMacro(lang, title, source string) string {
	if l, ok := confluenceLang[strings.ToLower(lang)]; ok {
		lang = l
	} else {
		lang = "text"
	}
	params := `<ac:parameter ac:name="language">` + lang + `</ac:parameter>`
	if title != "" {
		params += `<ac:parameter ac:name="title">` + esc(title) + `</ac:parameter>`
	}
	params += `<ac:parameter ac:name="linenumbers">true</ac:parameter>`
	// A CDATA section cannot contain "]]>", so split it across two sections.
	cdata := strings.ReplaceAll(source, "]]>", "]]]]><![CDATA[>")
	return `<ac:structured-macro ac:name="code" ac:schema-version="1">` + params +
		`<ac:plain-text-body><![CDATA[` + cdata + `]]></ac:plain-text-body></ac:structured-macro>`
}

func pager(o StorageOptions) string {
	var prev, next, mid string
	if o.Prev != nil {
		prev = `<p><a href="` + esc(o.Prev.URL) + `">← Previous: ` + esc(o.Prev.Title) + `</a></p>`
	}
	if o.Total > 0 {
		mid = `<p style="text-align: center;">` + statusMacro(fmt.Sprintf("%d of %d", o.Index, o.Total), "") + `</p>`
	}
	if o.Next != nil {
		next = `<p style="text-align: right;"><a href="` + esc(o.Next.URL) + `">Next: ` + esc(o.Next.Title) + ` →</a></p>`
	}
	return `<ac:layout-section ac:type="three_equal" ac:breakout-mode="default">` +
		cell(prev) + cell(mid) + cell(next) + `</ac:layout-section>`
}

func cell(inner string) string {
	if inner == "" {
		inner = "<p />"
	}
	return "<ac:layout-cell>" + inner + "</ac:layout-cell>"
}

func assemble(content, pagerSection string) string {
	return `<ac:layout><ac:layout-section ac:type="fixed-width" ac:breakout-mode="default"><ac:layout-cell>` +
		content + `</ac:layout-cell></ac:layout-section>` + pagerSection +
		`<ac:layout-section ac:type="fixed-width" ac:breakout-mode="default"><ac:layout-cell>` +
		`<p><small>Published from walkr. Edits made here are overwritten on the next publish.</small></p>` +
		`</ac:layout-cell></ac:layout-section></ac:layout>`
}

// storageCodeRenderer renders every fenced code block as a code macro. A block
// with mark= is followed by a numbered "Line N" list built from the footnotes.
type storageCodeRenderer struct{}

func (r *storageCodeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.render)
}

func (r *storageCodeRenderer) render(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	node := n.(*ast.FencedCodeBlock)
	path, marks, err := parseCodeAttrs(codeInfo(node, source))
	if err != nil {
		return ast.WalkStop, err
	}

	var src strings.Builder
	for i := 0; i < node.Lines().Len(); i++ {
		seg := node.Lines().At(i)
		src.WriteString(strings.TrimRight(string(seg.Value(source)), "\n"))
		if i < node.Lines().Len()-1 {
			src.WriteString("\n")
		}
	}

	io.WriteString(w, codeStart)
	io.WriteString(w, codeMacro(string(node.Language(source)), path, src.String()))
	if len(marks) > 0 {
		notes, err := takeFootnotes(node, source, marks)
		if err != nil {
			return ast.WalkStop, err
		}
		io.WriteString(w, "<ul>")
		for i, note := range notes {
			fmt.Fprintf(w, "<li><strong>Line %d</strong>: %s</li>", marks[i], note)
		}
		io.WriteString(w, "</ul>")
	}
	io.WriteString(w, codeEnd)
	return ast.WalkSkipChildren, nil
}
