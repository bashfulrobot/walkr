package site

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultTheme is the look every site had before themes existed.
const DefaultTheme = "field-guide"

// Theme is the page chrome a site is built with. A theme changes stylesheets,
// fonts and the logo only, never the rendered step content.
type Theme struct {
	Name string
	// CSS is the stylesheet loaded after style.css, relative to the asset base.
	// Empty for the default theme, which style.css already covers.
	CSS string
	// Logo is the rail logo, relative to the asset base. Empty means no logo.
	Logo string
}

var themes = map[string]Theme{
	DefaultTheme: {Name: DefaultTheme},
	"kong":       {Name: "kong", CSS: "assets/themes/kong.css", Logo: "vendor/kong/logomark.svg"},
}

// ThemeNames lists the valid theme names, sorted.
func ThemeNames() []string {
	names := make([]string, 0, len(themes))
	for name := range themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveTheme picks a theme. A non-empty flag wins over the manifest value,
// and an empty result for both gives DefaultTheme.
func ResolveTheme(flag, manifest string) (Theme, error) {
	name := flag
	if name == "" {
		name = manifest
	}
	if name == "" {
		name = DefaultTheme
	}
	t, ok := themes[name]
	if !ok {
		return Theme{}, fmt.Errorf("unknown theme %q, valid: %s", name, strings.Join(ThemeNames(), ", "))
	}
	return t, nil
}
