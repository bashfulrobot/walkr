package site

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

const invarianceSteps = "---\ntitle: Code\nlabel: Code\nkind: Code\norder: 2\nlayout: code-walk\nsummary: s\n---\n" +
	"- Walk **this** with a [thing]{def=thing} term.\n\n" +
	"```go path=main.go mark=1\nfunc main() {}\n```\n" +
	"1. The only line.\n\n" +
	":::deep{title=\"More\"}\nDeep body text.\n:::\n"

// pageContent returns every step section and modal body of a built page, the
// parts a theme must not change.
func pageContent(t *testing.T, indexHTML string) string {
	t.Helper()
	section := regexp.MustCompile(`(?s)<section id=.*?</section>`)
	modal := regexp.MustCompile(`(?s)<div class="modal__body">.*?</div>`)
	parts := append(section.FindAllString(indexHTML, -1), modal.FindAllString(indexHTML, -1)...)
	if len(parts) == 0 {
		t.Fatal("no step sections found in the built page")
	}
	return strings.Join(parts, "\n")
}

func TestStepContentIsIdenticalAcrossThemes(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "glossary.yaml"), "thing:\n  term: thing\n  definition: a thing\n")
	writeFile(t, filepath.Join(src, "steps", "01-overview.md"),
		"---\ntitle: Overview\nlabel: Overview\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\nbody with a [thing]{def=thing}\n")
	writeFile(t, filepath.Join(src, "steps", "02-code.md"), invarianceSteps)
	wt, err := walkthrough.Load(src)
	if err != nil {
		t.Fatal(err)
	}

	built := map[string]string{}
	for _, theme := range ThemeNames() {
		out := t.TempDir()
		if err := BuildWith(wt, out, Options{Theme: theme}); err != nil {
			t.Fatalf("%s: %v", theme, err)
		}
		b, _ := os.ReadFile(filepath.Join(out, "index.html"))
		built[theme] = pageContent(t, string(b))
	}
	if built["field-guide"] != built["kong"] {
		t.Error("step content differs between field-guide and kong")
	}
}
