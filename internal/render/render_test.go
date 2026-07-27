package render

import (
	"strings"
	"testing"

	"github.com/bashfulrobot/repo-walker/internal/walkthrough"
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
	res, err := RenderStep(step, walkthrough.Glossary{})
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
	res, err := RenderStep(step, walkthrough.Glossary{})
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
	if _, err := RenderStep(step, walkthrough.Glossary{}); err == nil {
		t.Fatal("expected an error when mark= count and footnote list length disagree")
	}
}

func TestRenderStep_GlossaryTermExpandsFromDefinition(t *testing.T) {
	gl := walkthrough.Glossary{
		"repo-walker": {Term: "repo-walker", Definition: "A CLI that renders authored markdown."},
	}
	step := walkthrough.Step{ID: "overview", Layout: walkthrough.LayoutOverview, Body: "See [repo-walker]{def=repo-walker} for details.\n"}
	res, err := RenderStep(step, gl)
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
	res, err := RenderStep(step, walkthrough.Glossary{})
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	if len(res.DeepDives) != 1 {
		t.Fatalf("expected 1 deep dive, got %d", len(res.DeepDives))
	}
	dd := res.DeepDives[0]
	if dd.Title != "Why?" || !strings.Contains(dd.BodyHTML, "Because reasons.") {
		t.Errorf("unexpected deep dive: %+v", dd)
	}
	if !strings.Contains(res.HTML, "openModal('"+dd.ID+"')") {
		t.Errorf("step HTML missing trigger for %s:\n%s", dd.ID, res.HTML)
	}
	if strings.Contains(res.HTML, "Because reasons.") {
		t.Errorf("deep-dive body must be extracted out of the step HTML, not inline:\n%s", res.HTML)
	}
}

func TestRenderStep_MermaidBlockBecomesDiagramFrame(t *testing.T) {
	step := walkthrough.Step{
		ID:     "overview",
		Layout: walkthrough.LayoutOverview,
		Body:   "```mermaid title=\"structure.mmd\"\ngraph TB\n  A --> B\n```\n",
	}
	res, err := RenderStep(step, walkthrough.Glossary{})
	if err != nil {
		t.Fatalf("RenderStep: %v", err)
	}
	for _, want := range []string{`class="diagram-frame"`, "structure.mmd", `class="mermaid"`, "graph TB"} {
		if !strings.Contains(res.HTML, want) {
			t.Errorf("missing %q in:\n%s", want, res.HTML)
		}
	}
}
