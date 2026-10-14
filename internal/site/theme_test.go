package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bashfulrobot/walkr/internal/walkthrough"
)

func TestResolveTheme(t *testing.T) {
	cases := []struct {
		name, flag, manifest, want string
	}{
		{"default", "", "", "field-guide"},
		{"manifest", "", "kong", "kong"},
		{"flag beats manifest", "field-guide", "kong", "field-guide"},
		{"flag only", "kong", "", "kong"},
	}
	for _, c := range cases {
		got, err := ResolveTheme(c.flag, c.manifest)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.Name != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got.Name, c.want)
		}
	}
}

func TestResolveThemeDefaultHasNoExtraChrome(t *testing.T) {
	got, _ := ResolveTheme("", "")
	if got.CSS != "" || got.Logo != "" {
		t.Errorf("default theme must add no stylesheet or logo, got %+v", got)
	}
}

func TestResolveThemeRejectsUnknownNames(t *testing.T) {
	for _, name := range []string{"Kong", "x"} {
		_, err := ResolveTheme(name, "")
		if err == nil {
			t.Fatalf("%q: expected an error", name)
		}
		want := `unknown theme "` + name + `", valid: field-guide, kong`
		if err.Error() != want {
			t.Errorf("got %q, want %q", err.Error(), want)
		}
	}
	if _, err := ResolveTheme("", "Kong"); err == nil {
		t.Error("manifest value Kong must error")
	}
}

func TestBuildDefaultThemeMatchesGolden(t *testing.T) {
	wt, err := walkthrough.Load(filepath.Join("testdata", "default", "walkr"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := Build(wt, out); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(out, "index.html"))
	want, _ := os.ReadFile(filepath.Join("testdata", "default", "index.golden.html"))
	if string(got) != string(want) {
		t.Error("field-guide index.html differs from the golden file")
	}
}

func TestBuildKongThemeAddsChromeOnly(t *testing.T) {
	wt, err := walkthrough.Load(filepath.Join("testdata", "default", "walkr"))
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := BuildWith(wt, out, Options{Theme: "kong"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(out, "index.html"))
	html := string(b)
	for _, want := range []string{`data-theme="kong"`, `assets/themes/kong.css`, `vendor/kong/logomark.svg`} {
		if !strings.Contains(html, want) {
			t.Errorf("kong index.html lacks %q", want)
		}
	}
	for _, rel := range []string{"assets/themes/kong.css", "vendor/kong/logomark.svg", "vendor/kong/fonts.css"} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("per-site build lacks %s: %v", rel, err)
		}
	}
}
