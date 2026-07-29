package render

import (
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

func TestRenderStep_CodeWalk_AnnotatesMarkedLines(t *testing.T) {
	body := "- summary bullet one\n- summary bullet two\n\n" +
		"```go path=\"internal/render/render.go\" mark=1,2\n" +
		"func RenderStep(s *Step) (string, error) {\n" +
		"    return \"\", nil\n" +
		"}\n" +
		"```\n" +
		"1. first callout\n" +
		"2. second callout\n"

	step := walkthrough.Step{ID: "code-walk", Layout: walkthrough.LayoutCodeWalk, Body: body}
	res, err := RenderStep(step, walkthrough.Glossary{}, nil)
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}

	for _, want := range []string{
		`class="codewalk"`,
		`class="codewalk__summary"`,
		`class="codewalk__deep"`,
		`internal/render`,
		`render.go`,
		`class="mark">1<`,
		`class="mark">2<`,
		"first callout",
		"second callout",
		`class="kw">func<`,
	} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, res.HTML)
		}
	}
}

func TestRenderStep_Config_AlwaysExpandedNoToggle(t *testing.T) {
	body := "```yaml mark=1\n" +
		"replicas: 2\n" +
		"```\n" +
		"1. two replicas because\n"

	step := walkthrough.Step{ID: "config", Layout: walkthrough.LayoutConfig, Body: body}
	res, err := RenderStep(step, walkthrough.Glossary{}, nil)
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	if strings.Contains(res.HTML, "codewalk__toggle") {
		t.Errorf("config layout must not render a collapse toggle:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, `class="manifest"`) {
		t.Errorf("config layout must wrap in .manifest:\n%s", res.HTML)
	}
}

func TestRenderStep_MarkCountMismatchIsAnError(t *testing.T) {
	body := "```go mark=1,2\n" +
		"line one\n" +
		"```\n" +
		"1. only one callout\n"
	step := walkthrough.Step{ID: "bad", Layout: walkthrough.LayoutCodeWalk, Body: body}
	if _, err := RenderStep(step, walkthrough.Glossary{}, nil); err == nil {
		t.Fatal("expected an error when mark= count and footnote list length disagree")
	}
}

func TestRenderStep_GlossaryTermExpandsFromDefinition(t *testing.T) {
	gl := walkthrough.Glossary{
		"walkr": {Term: "walkr", Definition: "A CLI that renders authored markdown."},
	}
	step := walkthrough.Step{ID: "overview", Layout: walkthrough.LayoutOverview, Body: "See [walkr]{def=walkr} for details.\n"}
	res, err := RenderStep(step, gl, nil)
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	if !strings.Contains(res.HTML, `class="term"`) || !strings.Contains(res.HTML, "A CLI that renders authored markdown.") {
		t.Errorf("expected expanded glossary popover, got:\n%s", res.HTML)
	}
}

func TestRenderStep_DeepDiveExtractedAsModal(t *testing.T) {
	step := walkthrough.Step{
		ID:     "overview",
		Layout: walkthrough.LayoutOverview,
		Body:   "Intro text.\n\n:::deep{title=\"Why?\"}\nBecause reasons.\n:::\n",
	}
	res, err := RenderStep(step, walkthrough.Glossary{}, nil)
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	if len(res.DeepDives) != 1 {
		t.Fatalf("expected 1 deep dive, got %d", len(res.DeepDives))
	}
	dd := res.DeepDives[0]
	if dd.Title != "Why?" || !strings.Contains(string(dd.BodyHTML), "Because reasons.") {
		t.Errorf("unexpected deep dive: %+v", dd)
	}
	if !strings.Contains(res.HTML, "openModal('"+dd.ID+"')") {
		t.Errorf("step HTML missing trigger for %s:\n%s", dd.ID, res.HTML)
	}
	if strings.Contains(res.HTML, "Because reasons.") {
		t.Errorf("deep-dive body must be extracted out of the step HTML, not inline:\n%s", res.HTML)
	}
}

func TestRenderStep_FootnoteWithInlineElementStaysOneGridItem(t *testing.T) {
	// Regression: .footnotes li is a 2-column CSS grid (badge + text). A
	// footnote whose markdown contains an inline element (here, a code
	// span) followed by more text produces 3 sibling nodes after the badge
	// (an inline element plus a separate trailing text run) unless the
	// footnote body is wrapped in one element. Without the wrapper, the
	// third node overflows the 2-column grid template into the narrow
	// badge column, forcing every word after the code span onto its own
	// line. See codeblock.go's footnote-rendering comment for the full
	// mechanism.
	body := "```go mark=1\n" +
		"line one\n" +
		"```\n" +
		"1. `foo` bar baz qux\n"
	step := walkthrough.Step{ID: "footnote-inline", Layout: walkthrough.LayoutCodeWalk, Body: body}
	res, err := RenderStep(step, walkthrough.Glossary{}, nil)
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	want := `<li><span class="mark">1</span><span class="footnotes__text"><code>foo</code> bar baz qux</span></li>`
	if !strings.Contains(res.HTML, want) {
		t.Errorf("footnote text not wrapped in a single element:\nwant substring: %s\ngot:\n%s", want, res.HTML)
	}
}

func TestRenderStep_StepLinkExpandsToHashAnchorWhenIDKnown(t *testing.T) {
	step := walkthrough.Step{ID: "two-paths", Layout: walkthrough.LayoutOverview, Body: "See [Minimum Path]{step=minimum-path} first.\n"}
	res, err := RenderStep(step, walkthrough.Glossary{}, map[string]bool{"minimum-path": true})
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	if !strings.Contains(res.HTML, `<a href="#minimum-path">Minimum Path</a>`) {
		t.Errorf("expected hash anchor to known step, got:\n%s", res.HTML)
	}
}

func TestRenderStep_StepLinkFlagsUnknownID(t *testing.T) {
	step := walkthrough.Step{ID: "two-paths", Layout: walkthrough.LayoutOverview, Body: "See [Minimum Path]{step=no-such-step} first.\n"}
	res, err := RenderStep(step, walkthrough.Glossary{}, map[string]bool{"minimum-path": true})
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	if strings.Contains(res.HTML, `<a href="#no-such-step">`) {
		t.Errorf("unknown step id must not render as a working link:\n%s", res.HTML)
	}
	if !strings.Contains(res.HTML, "undefined step link: no-such-step") {
		t.Errorf("expected flagged span for unknown step id, got:\n%s", res.HTML)
	}
}

func TestRenderStep_MermaidBlockBecomesDiagramFrame(t *testing.T) {
	step := walkthrough.Step{
		ID:     "overview",
		Layout: walkthrough.LayoutOverview,
		Body:   "```mermaid title=\"structure.mmd\"\ngraph TB\n  A --> B\n```\n",
	}
	res, err := RenderStep(step, walkthrough.Glossary{}, nil)
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	for _, want := range []string{`class="diagram-frame"`, "structure.mmd", `class="mermaid"`, "graph TB"} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("missing %q in:\n%s", want, res.HTML)
		}
	}
}
