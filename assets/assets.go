// Package assets embeds repo-walker's static assets — vendored Alpine.js,
// Mermaid.js, self-hosted fonts, and the wizard's own CSS/JS — into the Go
// binary, so `repo-walker build` needs no network access. See
// prototype/vendor and prototype/assets for where these came from.
package assets

import "embed"

//go:embed all:vendor
var Vendor embed.FS

//go:embed style.css
var StyleCSS []byte

//go:embed app.js
var AppJS []byte
