package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func themedTutorial(t *testing.T, root, rel, theme string) {
	t.Helper()
	tutorial(t, root, rel)
	write(t, filepath.Join(root, filepath.FromSlash(rel), ".walkr", "walkthrough.yaml"),
		"title: T "+rel+"\ntagline: tag\nrepo: r\ntheme: "+theme+"\n")
}

func readSite(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel), "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildAllHonorsPerSiteManifestTheme(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "plain")
	themedTutorial(t, root, "work", "kong")
	if _, err := BuildAll(root, root, Options{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(readSite(t, root, "plain"), "themes/kong.css") {
		t.Error("default site must not load the Kong stylesheet")
	}
	work := readSite(t, root, "work")
	if !strings.Contains(work, "../../_walkr/assets/themes/kong.css") {
		t.Error("kong site must load the shared Kong stylesheet")
	}
	if _, err := os.Stat(filepath.Join(root, "_walkr", "assets", "themes", "kong.css")); err != nil {
		t.Errorf("shared dir lacks the Kong stylesheet: %v", err)
	}
	res, err := Check(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK() {
		t.Errorf("mixed library should pass its own check: %+v", res)
	}
}

func TestThemeOptionOverridesEverySiteAndCheckFollowsIt(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "a")
	tutorial(t, root, "b")
	if _, err := BuildAll(root, root, Options{Theme: "kong"}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"a", "b"} {
		if !strings.Contains(readSite(t, root, rel), `data-theme="kong"`) {
			t.Errorf("%s: --theme kong must apply", rel)
		}
	}
	res, err := Check(root, Options{Theme: "kong"})
	if err != nil || !res.OK() {
		t.Errorf("check with the same theme must pass: %v %+v", err, res)
	}
	res, err = Check(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() {
		t.Error("check without the theme must report the kong build as stale")
	}
}

func TestBuildAllReportsUnknownManifestTheme(t *testing.T) {
	root := t.TempDir()
	themedTutorial(t, root, "bad", "Kong")
	rep, err := BuildAll(root, root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Failures) != 1 || !strings.Contains(rep.Failures[0].Err.Error(), "valid: field-guide, kong") {
		t.Errorf("expected one failure naming the valid themes, got %+v", rep.Failures)
	}
}

func TestPerSiteThemeCarriesItsOwnAssets(t *testing.T) {
	root := t.TempDir()
	themedTutorial(t, root, "work", "kong")
	if _, err := BuildAll(root, root, Options{PerSite: true}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"assets/themes/kong.css", "vendor/kong/logomark.svg", "vendor/kong/fonts/FunnelSans-Bold.ttf"} {
		if _, err := os.Stat(filepath.Join(root, "work", "site", filepath.FromSlash(rel))); err != nil {
			t.Errorf("per-site build lacks %s: %v", rel, err)
		}
	}
}
