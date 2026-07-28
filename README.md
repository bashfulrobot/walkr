# walkr

`walkr` renders a hand-authored markdown walkthrough into an interactive,
wizard-style static site that teaches a newcomer how something fits
together — a codebase (via the `walkr-author` skill) or a topic sourced from
reference docs (via the `walkr-tutorial-author` skill). It never generates
or analyzes content itself — it only renders what a human or one of those
skills writes.

## Quickstart

```sh
nix run github:bashfulrobot/walkr -- init      # scaffold .walkr/
$EDITOR .walkr/steps/01-overview.md
nix run github:bashfulrobot/walkr -- serve .walkr --open
```

or install locally:

```sh
go install github.com/bashfulrobot/walkr@latest
walkr init
walkr build .walkr -o ./site   # self-contained static site, no server needed
```

Full install options (Nix flake, `go install`, building from source) and
authoring reference: **[docs/user](docs/user/README.md)**.

## Docs

- [User manual](docs/user/README.md) — install, quickstart, authoring a walkthrough, CLI reference.
- [Content format spec](docs/ai/content-format.md) — the authoritative frontmatter/directive contract the renderer and both authoring skills (`walkr-author`, `walkr-tutorial-author`) implement against.
