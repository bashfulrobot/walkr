# walkr

Pure-Go CLI (`github.com/bashfulrobot/walkr`) that renders a hand-authored `.walkr/` markdown walkthrough into a static teaching site.

## Rules

- walkr renders content. It never generates or analyzes content.
- `docs/ai/content-format.md` is the content contract. The renderer (`internal/render`) implements it. The skills `walkr-author` and `walkr-tutorial-author` each bundle a copy at `skills/<name>/references/content-format.md`. A format change updates all three in one commit.
- No new layouts, directives, frontmatter keys or UI features unless asked.
- Every behavior change gets a test. Behavior that is correct by design gets a regression test that pins it.
- Keep the existing error-wrapping style, names and `internal/` package layout.
- `.walkr/` is a walkthrough about walkr itself, so a successful `just check` includes a render of it.

## Before finishing

Run `just check`. It runs `go vet`, `gofmt -l` (fails on any listed file), `go test ./...` and a dogfood build of `.walkr`.

## Release and install

There are no tags or releases. Machines build from `main`:

- Mac (donkeykong): `./go-tools.sh --upgrade`.
- NixOS (nixerator): `nix flake update walkr`, then `just rebuild`. A change to `go.mod` or `go.sum` needs a fresh `vendorHash` in `nix/walkr.nix` first.
- The bundled skills reach machines through a version bump of the `dk` plugin in claude-skills.
