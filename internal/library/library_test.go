package library

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const stepMD = "---\ntitle: Overview\nlabel: Overview\nkind: Structure\norder: 1\nlayout: overview\nsummary: s\n---\nbody\n"

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func tutorial(t *testing.T, root, rel string) {
	t.Helper()
	write(t, filepath.Join(root, filepath.FromSlash(rel), ".walkr", "steps", "01-overview.md"), stepMD)
	write(t, filepath.Join(root, filepath.FromSlash(rel), ".walkr", "walkthrough.yaml"), "title: T "+rel+"\ntagline: tag\nrepo: r\n")
}

func rels(ts []Tutorial) []string {
	var out []string
	for _, t := range ts {
		out = append(out, t.Rel)
	}
	return out
}

func TestDiscoverFindsTutorialsAndSkipsGeneratedAndHiddenDirs(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "a/two")
	tutorial(t, root, "a/one")
	tutorial(t, root, "deep/er/three")
	// none of these may be picked up
	write(t, filepath.Join(root, "a/one/site/decoy/.walkr/steps/01.md"), stepMD)
	write(t, filepath.Join(root, ".git/x/.walkr/steps/01.md"), stepMD)
	write(t, filepath.Join(root, "_walkr/.walkr/steps/01.md"), stepMD)
	write(t, filepath.Join(root, ".hidden/y/.walkr/steps/01.md"), stepMD)
	write(t, filepath.Join(root, "node_modules/z/.walkr/steps/01.md"), stepMD)
	if err := os.MkdirAll(filepath.Join(root, "notutorial/.walkr"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	want := "a/one,a/two,deep/er/three"
	if strings.Join(rels(got), ",") != want {
		t.Fatalf("Discover = %v, want %s", rels(got), want)
	}
}

func TestBuildAllSharesAssetsAndComputesRelativePaths(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "a/one")
	tutorial(t, root, "deep/er/three")

	rep, err := BuildAll(root, root, Options{})
	if err != nil || rep.Built != 2 || len(rep.Failures) != 0 {
		t.Fatalf("report = %+v, err %v", rep, err)
	}

	for rel, prefix := range map[string]string{
		"a/one":         "../../../_walkr/",
		"deep/er/three": "../../../../_walkr/",
	} {
		siteDir := filepath.Join(root, filepath.FromSlash(rel), "site")
		html, err := os.ReadFile(filepath.Join(siteDir, "index.html"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(html), `href="`+prefix+`assets/style.css"`) {
			t.Errorf("%s: wrong asset prefix, want %s", rel, prefix)
		}
		// the reference must actually resolve from where index.html lives
		if _, err := os.Stat(filepath.Join(siteDir, filepath.FromSlash(prefix), "assets", "style.css")); err != nil {
			t.Errorf("%s: prefix does not resolve: %v", rel, err)
		}
		for _, own := range []string{"vendor", "assets"} {
			if _, err := os.Stat(filepath.Join(siteDir, own)); err == nil {
				t.Errorf("%s: site carries its own %s/", rel, own)
			}
		}
	}
	if _, err := os.Stat(filepath.Join(root, "_walkr", "vendor", "mermaid.min.js")); err != nil {
		t.Errorf("shared assets not written: %v", err)
	}
}

func TestBuildAllRemovesLeftoversFromAnOlderLayout(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "a/one")
	write(t, filepath.Join(root, "a/one/site/vendor/old.txt"), "old")
	write(t, filepath.Join(root, "a/one/site/assets/old.css"), "old")

	if _, err := BuildAll(root, root, Options{}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"vendor/old.txt", "assets/old.css"} {
		if _, err := os.Stat(filepath.Join(root, "a/one/site", f)); err == nil {
			t.Errorf("leftover %s survived a rebuild", f)
		}
	}
}

func TestBuildAllPerSiteKeepsAssetsBesideEachSiteAndDropsShared(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "a/one")
	write(t, filepath.Join(root, "_walkr/stale.txt"), "x")

	if _, err := BuildAll(root, root, Options{PerSite: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "a/one/site/vendor/mermaid.min.js")); err != nil {
		t.Errorf("per-site build lacks its own vendor: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "_walkr")); err == nil {
		t.Error("per-site build left the shared directory behind")
	}
}

func TestBuildAllOneBrokenTutorialDoesNotStopTheRest(t *testing.T) {
	root := t.TempDir()
	tutorial(t, root, "good/one")
	write(t, filepath.Join(root, "bad/one/.walkr/steps/01.md"), "no frontmatter at all\n")

	rep, err := BuildAll(root, root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Built != 1 || len(rep.Failures) != 1 || rep.Failures[0].Rel != "bad/one" {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(root, "good/one/site/index.html")); err != nil {
		t.Errorf("good tutorial was not built: %v", err)
	}
}

func TestBuildAllErrorsWhenThereIsNothingToBuild(t *testing.T) {
	if _, err := BuildAll(t.TempDir(), t.TempDir(), Options{}); err == nil {
		t.Fatal("expected an error for an empty tree")
	}
}

func built(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	tutorial(t, root, "a/one")
	tutorial(t, root, "a/two")
	if _, err := BuildAll(root, root, Options{}); err != nil {
		t.Fatal(err)
	}
	return root
}

func kinds(r *CheckResult) map[string]DiffKind {
	m := map[string]DiffKind{}
	for _, d := range r.Diffs {
		m[d.Path] = d.Kind
	}
	return m
}

func TestCheckIsCleanRightAfterABuildAndNeverWrites(t *testing.T) {
	root := built(t)
	before, _ := os.ReadFile(filepath.Join(root, "a/one/site/index.html"))

	res, err := Check(root, Options{})
	if err != nil || !res.OK() {
		t.Fatalf("expected clean, got %+v, err %v", res, err)
	}
	after, _ := os.ReadFile(filepath.Join(root, "a/one/site/index.html"))
	if string(before) != string(after) {
		t.Error("Check modified the tree")
	}
}

func TestCheckReportsAnEditedSourceAsStale(t *testing.T) {
	root := built(t)
	write(t, filepath.Join(root, "a/one/.walkr/steps/01-overview.md"), strings.Replace(stepMD, "body", "changed body", 1))

	res, err := Check(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got := kinds(res)["a/one/site/index.html"]; got != Stale {
		t.Fatalf("diffs = %+v", res.Diffs)
	}
	if len(res.Diffs) != 1 {
		t.Errorf("only the edited tutorial should differ, got %+v", res.Diffs)
	}
}

func TestCheckReportsMissingAndExtraAndSharedDrift(t *testing.T) {
	root := built(t)
	if err := os.RemoveAll(filepath.Join(root, "a/two/site")); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "a/one/site/leftover.txt"), "x")
	write(t, filepath.Join(root, "_walkr/assets/style.css"), "/* tampered */")

	res, err := Check(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	k := kinds(res)
	if k["a/two/site"] != Missing {
		t.Errorf("missing site not reported: %+v", res.Diffs)
	}
	if k["a/one/site/leftover.txt"] != Extra {
		t.Errorf("extra file not reported: %+v", res.Diffs)
	}
	if k["_walkr/assets/style.css"] != Stale {
		t.Errorf("shared drift not reported: %+v", res.Diffs)
	}
}

func TestCheckReportsMissingSharedDirectory(t *testing.T) {
	root := built(t)
	if err := os.RemoveAll(filepath.Join(root, "_walkr")); err != nil {
		t.Fatal(err)
	}
	res, err := Check(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if kinds(res)["_walkr"] != Missing {
		t.Fatalf("diffs = %+v", res.Diffs)
	}
}

func TestCheckSurfacesBuildFailures(t *testing.T) {
	root := built(t)
	write(t, filepath.Join(root, "bad/one/.walkr/steps/01.md"), "no frontmatter\n")
	res, err := Check(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.OK() || len(res.Failures) != 1 {
		t.Fatalf("result = %+v", res)
	}
}
