// Package assets embeds walkr's static assets — vendored Alpine.js,
// Mermaid.js, self-hosted fonts, and the wizard's own CSS/JS — into the Go
// binary, so `walkr build` needs no network access. See
// prototype/vendor and prototype/assets for where these came from.
package assets

import "embed"

// ThirdParty holds the vendored libraries and fonts. It is deliberately not
// named vendor: the Go module zip drops files nested two levels below any
// directory called vendor, so the fonts were silently missing from binaries
// built with go install. Built sites still write these files to a vendor/
// folder, so their URLs are unchanged.
//
//go:embed all:thirdparty
var ThirdParty embed.FS

//go:embed style.css
var StyleCSS []byte

//go:embed app.js
var AppJS []byte
