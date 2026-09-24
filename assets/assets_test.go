package assets

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// The Go module zip, which `go install` and the module proxy use, drops any
// file nested two levels or more below a directory named vendor. An embedded
// file lost that way is silently absent from installed binaries, which is how
// the fonts went missing. This mirrors that rule from golang.org/x/mod/zip.
func droppedByModuleZip(path string) bool {
	path = filepath.ToSlash(path)
	var i int
	switch {
	case strings.HasPrefix(path, "vendor/"):
		i = len("vendor/")
	case strings.Contains(path, "/vendor/"):
		i = strings.Index(path, "/vendor/") + len("/vendor/")
	default:
		return false
	}
	return strings.Contains(path[i:], "/")
}

func TestNoAssetIsDroppedByTheModuleZip(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if droppedByModuleZip(path) {
			t.Errorf("%s would be dropped from the module zip, so go install builds would lack it", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDroppedByModuleZipMatchesTheRealRule(t *testing.T) {
	for path, want := range map[string]bool{
		"vendor/alpine.min.js":         false,
		"vendor/fonts/a.woff2":         true,
		"assets/vendor/fonts/a.woff2":  true,
		"assets/vendor/mermaid.min.js": false,
		"thirdparty/fonts/a.woff2":     false,
		"style.css":                    false,
	} {
		if got := droppedByModuleZip(path); got != want {
			t.Errorf("droppedByModuleZip(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestEmbeddedFontsArePresent(t *testing.T) {
	for _, f := range []string{
		"thirdparty/fonts/fraunces-variable.woff2",
		"thirdparty/fonts/jetbrainsmono-variable.woff2",
		"thirdparty/fonts.css",
		"thirdparty/mermaid.min.js",
	} {
		if _, err := fs.Stat(ThirdParty, f); err != nil {
			t.Errorf("missing embedded %s: %v", f, err)
		}
	}
}
