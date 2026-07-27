package walkthrough

import (
	"os"
	"path/filepath"
	"testing"
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

func TestLoad_OrdersStepsAndAppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "steps", "b.md"), "---\ntitle: Second\nlabel: Second\nkind: Config\norder: 2\nlayout: config\nsummary: s2\n---\nbody two\n")
	writeFile(t, filepath.Join(dir, "steps", "a.md"), "---\ntitle: First\nlabel: First\nkind: Structure\norder: 1\nlayout: overview\nsummary: s1\n---\nbody one\n")

	wt, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(wt.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(wt.Steps))
	}
	if wt.Steps[0].Title != "First" || wt.Steps[1].Title != "Second" {
		t.Fatalf("steps not ordered correctly: %+v", wt.Steps)
	}
	if wt.Manifest.Title != "Repo Walker" {
		t.Errorf("expected default manifest title, got %q", wt.Manifest.Title)
	}
}

func TestLoad_RejectsUnknownLayout(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "steps", "a.md"), "---\ntitle: X\nlabel: X\nlayout: bogus\norder: 1\n---\nbody\n")
	if _, err := Load(dir); err == nil {
		t.Fatal("expected an error for an unknown layout")
	}
}

func TestLoad_RejectsMissingLabel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "steps", "a.md"), "---\ntitle: X\nlayout: overview\norder: 1\n---\nbody\n")
	if _, err := Load(dir); err == nil {
		t.Fatal("expected an error for a missing label")
	}
}

func TestLoad_RejectsMissingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "steps", "a.md"), "no frontmatter here\n")
	if _, err := Load(dir); err == nil {
		t.Fatal("expected an error for missing frontmatter")
	}
}

func TestLoad_ManifestAndGlossary(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "walkthrough.yaml"), "title: My Repo\ntagline: Tour\nrepo: org/my-repo\n")
	writeFile(t, filepath.Join(dir, "glossary.yaml"), "widget:\n  term: widget\n  definition: A thing.\n")
	writeFile(t, filepath.Join(dir, "steps", "a.md"), "---\ntitle: X\nlabel: X\nlayout: overview\norder: 1\nkind: Overview\nsummary: s\n---\nbody\n")

	wt, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if wt.Manifest.Title != "My Repo" || wt.Manifest.Repo != "org/my-repo" {
		t.Errorf("manifest not loaded: %+v", wt.Manifest)
	}
	if wt.Glossary["widget"].Definition != "A thing." {
		t.Errorf("glossary not loaded: %+v", wt.Glossary)
	}
}
