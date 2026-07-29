package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func loadOneStepWalkthrough(t *testing.T, dir string) *walkthrough.Walkthrough {
	t.Helper()
	writeFile(t, filepath.Join(dir, "steps", "01-overview.md"),
		"---\ntitle: Overview\nlabel: Overview\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\nbody\n")
	wt, err := walkthrough.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return wt
}

func TestBuild_CopiesMediaDirWhenPresent(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "media", "shot.png"), "fake-png-bytes")
	writeFile(t, filepath.Join(src, "media", "nested", "diagram.svg"), "<svg/>")
	wt := loadOneStepWalkthrough(t, src)

	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "media", "shot.png"))
	if err != nil {
		t.Fatalf("expected media/shot.png in output: %v", err)
	}
	if string(got) != "fake-png-bytes" {
		t.Errorf("copied file content mismatch: got %q", got)
	}
	if _, err := os.ReadFile(filepath.Join(out, "media", "nested", "diagram.svg")); err != nil {
		t.Fatalf("expected nested media/nested/diagram.svg in output: %v", err)
	}
}

func TestBuild_StepSectionsCarryStepIndexForMermaidScoping(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "steps", "01-overview.md"),
		"---\ntitle: Overview\nlabel: Overview\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\nbody\n")
	writeFile(t, filepath.Join(src, "steps", "02-next.md"),
		"---\ntitle: Next\nlabel: Next\nkind: Structure\norder: 2\nlayout: overview\nsummary: s\n---\nbody\n")
	wt, err := walkthrough.Load(src)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("expected index.html in output: %v", err)
	}
	html := string(got)
	// app.js scopes mermaid rendering to `section[data-step-index="N"]` so
	// it can lazily (re)render a step's diagrams once that step is
	// actually shown, instead of rendering every .mermaid element up
	// front while most of them still sit under a display:none ancestor.
	// If this attribute goes missing, that lazy render silently no-ops
	// and diagrams past the first step render as tiny broken SVGs.
	for _, want := range []string{`data-step-index="0"`, `data-step-index="1"`} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %s in step section markup, got:\n%s", want, html)
		}
	}
}

func TestBuild_StepSectionsCarryIDForHashRouting(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "steps", "01-overview.md"),
		"---\ntitle: Overview\nlabel: Overview\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\nbody\n")
	writeFile(t, filepath.Join(src, "steps", "02-two-paths.md"),
		"---\ntitle: Two Paths\nlabel: Two Paths\nkind: Structure\norder: 2\nlayout: overview\nsummary: s\n---\nbody\n")
	wt, err := walkthrough.Load(src)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("expected index.html in output: %v", err)
	}
	html := string(got)
	// assets/app.js reads/writes location.hash against each step's `id`
	// (see s.id in the rail data and the section's static id attribute) so
	// deep-links and the browser Back/Forward buttons land on the right
	// chapter. If the section is missing its id, hash routing silently
	// stops working and every reload/back-navigation resets to chapter one.
	for _, want := range []string{`id="01-overview"`, `id="02-two-paths"`} {
		if !strings.Contains(html, want) {
			t.Errorf("expected %s on a step section, got:\n%s", want, html)
		}
	}
}

func TestBuild_StepLinkDirectiveBecomesHashAnchor(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "steps", "01-two-paths.md"),
		"---\ntitle: Two Paths\nlabel: Two Paths\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\n"+
			"Take the [Minimum Path]{step=02-minimum-path} for a quick look.\n")
	writeFile(t, filepath.Join(src, "steps", "02-minimum-path.md"),
		"---\ntitle: Minimum Path\nlabel: Minimum Path\nkind: Structure\norder: 2\nlayout: overview\nsummary: s\n---\nbody\n")
	wt, err := walkthrough.Load(src)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("expected index.html in output: %v", err)
	}
	html := string(got)
	if !strings.Contains(html, `<a href="#02-minimum-path">Minimum Path</a>`) {
		t.Errorf("expected cross-chapter step link to become a hash anchor, got:\n%s", html)
	}
}

func TestBuild_DeepDiveInlineHTMLNotDoubleEscaped(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "steps", "01-overview.md"),
		"---\ntitle: Overview\nlabel: Overview\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\n"+
			"body\n\n:::deep{title=\"Why?\"}\ncreate a <code>Gateway</code>.\n:::\n")
	wt, err := walkthrough.Load(src)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatalf("Build: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatalf("expected index.html in output: %v", err)
	}
	html := string(got)
	if strings.Contains(html, "&lt;code&gt;") {
		t.Errorf("deep dive body was double-escaped, raw <code> tags leaked into output as text:\n%s", html)
	}
	if !strings.Contains(html, "<code>Gateway</code>") {
		t.Errorf("expected literal <code>Gateway</code> element in output, got:\n%s", html)
	}
}

func TestBuild_NoMediaDirIsNotAnError(t *testing.T) {
	src := t.TempDir()
	wt := loadOneStepWalkthrough(t, src)

	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "media")); !os.IsNotExist(err) {
		t.Errorf("expected no media/ dir in output, got err=%v", err)
	}
}
