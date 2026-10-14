package assets

import (
	"io/fs"
	"testing"
)

func TestKongThemeFilesAreEmbedded(t *testing.T) {
	for _, path := range []string{
		"thirdparty/kong/logotype.svg",
		"thirdparty/kong/fonts.css",
		"thirdparty/kong/fonts/FunnelSans-ExtraBold.ttf",
		"thirdparty/kong/fonts/RobotoMono-Regular.ttf",
		"thirdparty/kong/fonts/SpaceGrotesk-Bold.ttf",
		"thirdparty/kong/fonts/OFL-FunnelSans.txt",
	} {
		if _, err := fs.Stat(ThirdParty, path); err != nil {
			t.Errorf("%s is not embedded: %v", path, err)
		}
	}
	if _, err := fs.Stat(Themes, "themes/kong.css"); err != nil {
		t.Errorf("themes/kong.css is not embedded: %v", err)
	}
}
