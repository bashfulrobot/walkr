package site

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSharedAssetsCarriesEveryTheme(t *testing.T) {
	dir := t.TempDir()
	if err := WriteSharedAssets(dir); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		"assets/style.css", "assets/themes/kong.css",
		"vendor/kong/logotype.svg", "vendor/kong/fonts.css", "vendor/kong/fonts/FunnelSans-Regular.ttf",
	} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("shared assets lack %s: %v", rel, err)
		}
	}
}
