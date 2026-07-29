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
