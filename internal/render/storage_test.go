package render

import (
	"encoding/xml"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

var update = flag.Bool("update", false, "rewrite golden files")

var testGlossary = walkthrough.Glossary{
	"portfolio": {
		Term:       "portfolio",
		Definition: "An Asana view that groups projects.\nIt does not move them.",
		LearnMore:  "https://example.com/portfolio?language=en_US&x=1",
	},
}

func testStepURL(id string) (string, bool) {
	if id == "02-next" {
		return "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=222", true
	}
	return "", false
}

func testOpts() StorageOptions {
	return StorageOptions{
		Index:   2,
		Total:   4,
		Prev:    &NavLink{Title: "Overview", URL: "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=111"},
		Next:    &NavLink{Title: "Recap", URL: "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=333"},
		StepURL: testStepURL,
	}
}

const overviewBody = "You track many customers. A [portfolio]{def=portfolio} is optional, see [the next chapter]{step=02-next}.\n\n" +
	"```mermaid title=\"shape.mmd\"\ngraph TB\n  A --> B\n```\n\n" +
	":::deep{title=\"Why not more projects?\"}\nIt multiplies fast, see [portfolio]{def=portfolio}.\n\nSecond paragraph.\n:::\n\n" +
	"*Source: [Portfolios in Asana](https://help.asana.com/s/article/portfolio-management?language=en_US&x=1)*\n"

const codeWalkBody = "- The function is small.\n- It parses first.\n\n" +
	"```go path=\"internal/render/render.go\" mark=2,3\nfunc RenderStep() {\n    parse()\n    render()\n}\n```\n" +
	"1. Parse decides the layout.\n2. Render runs last, uses `goldmark`.\n\n" +
	"*Source: [Docs](https://example.com/docs)*\n"

const configBody = "```yaml path=\"walkthrough.yaml\" mark=1\ntitle: Walkr\ntagline: \"a]]>b\"\n```\n1. The wordmark.\n"

func step(id, layout, body string, order int) walkthrough.Step {
	return walkthrough.Step{
		ID: id, Title: "How it *fits together*", Label: "Overview", Kind: "Big picture",
		Order: order, Layout: walkthrough.Layout(layout), Summary: "One sentence lede.", Body: body,
	}
}

func TestRenderStorageGolden(t *testing.T) {
	cases := []struct {
		name string
		step walkthrough.Step
		opts func() StorageOptions
	}{
		{"overview-fallback", step("01-overview", "overview", overviewBody, 1), testOpts},
		{"overview-diagram-image", step("01-overview", "overview", overviewBody, 1), func() StorageOptions {
			o := testOpts()
			o.Diagram = func(string) (string, bool) { return "diagram-abc123.png", true }
			return o
		}},
		{"code-walk", step("03-code", "code-walk", codeWalkBody, 3), testOpts},
		{"config", step("04-config", "config", configBody, 4), func() StorageOptions {
			o := testOpts()
			o.Next = nil
			return o
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := RenderStorage(tc.step, testGlossary, tc.opts())
			if err != nil {
				t.Fatal(err)
			}
			assertWellFormed(t, res.Body)
			for _, leak := range []string{"@@WALKR", "\x00", "{def=", "{step="} {
				if strings.Contains(res.Body, leak) {
					t.Errorf("output leaks %q", leak)
				}
			}
			golden(t, tc.name, res.Body)
		})
	}
}

func TestRenderStorageWarnings(t *testing.T) {
	body := "See [gone]{step=99-missing} and [what]{def=nope}."
	res, err := RenderStorage(step("01-x", "overview", body, 1), testGlossary, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("warnings = %v", res.Warnings)
	}
	if strings.Contains(res.Body, "{step=") || strings.Contains(res.Body, "{def=") {
		t.Error("unresolved directive left in output")
	}
	assertWellFormed(t, res.Body)
}

func TestRenderStorageMarkMismatchIsAnError(t *testing.T) {
	body := "```go mark=1,2\nx\ny\n```\n1. only one\n"
	if _, err := RenderStorage(step("01-x", "config", body, 1), testGlossary, testOpts()); err == nil {
		t.Fatal("expected a mark/footnote mismatch error")
	}
}

func TestRenderStorageIndex(t *testing.T) {
	steps := []walkthrough.Step{step("01-a", "overview", "x", 1), step("02-next", "overview", "y", 2)}
	res, err := RenderStorageIndex(walkthrough.Manifest{Title: "Customer templates", Tagline: "Field Guide"}, steps, func(id string) (string, bool) {
		return "https://example.atlassian.net/wiki/pages/viewpage.action?pageId=" + id, true
	})
	if err != nil {
		t.Fatal(err)
	}
	assertWellFormed(t, res.Body)
	golden(t, "index", res.Body)
}

func TestPlainTitle(t *testing.T) {
	for in, want := range map[string]string{
		"How the setup *fits together*":    "How the setup fits together",
		"The render pipeline: _render.go_": "The render pipeline: render.go",
		"No emphasis here":                 "No emphasis here",
	} {
		if got := PlainTitle(in); got != want {
			t.Errorf("PlainTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

// assertWellFormed parses body as XML, since Confluence stores it as XML and
// rejects things HTML allows, such as an unclosed <br> or a named entity.
func assertWellFormed(t *testing.T, body string) {
	t.Helper()
	dec := xml.NewDecoder(strings.NewReader(`<root xmlns:ac="urn:ac" xmlns:ri="urn:ri">` + body + `</root>`))
	for {
		if _, err := dec.Token(); err != nil {
			if err.Error() == "EOF" {
				return
			}
			t.Fatalf("output is not well-formed XML: %v", err)
		}
	}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "storage", name+".xml")
	if *update {
		if err := os.WriteFile(path, []byte(got+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("missing golden %s (run with -update): %v", path, err)
	}
	if strings.TrimRight(string(want), "\n") != got {
		t.Errorf("output differs from %s (run with -update to accept)\n got: %s\nwant: %s", path, got, want)
	}
}
