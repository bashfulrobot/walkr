---
name: walkr-confluence-publish
description: Publish an existing walkr walkthrough (a `.walkr/` directory) to Confluence as a tree of native pages, one tutorial page plus one child page per step, by running the `walkr confluence publish` command. Use when the user says "publish this walkthrough to Confluence", "/walkr-confluence-publish", "push the walkr tutorial to Confluence", "put this tutorial on Confluence", "republish the Confluence pages", or "update the Confluence version of this walkthrough". Requires the `walkr` binary and a configured Confluence target, unlike the authoring skills it is not portable. Do not use to author or edit walkthrough content (that is `walkr-author` or `walkr-tutorial-author`), and do not use to edit Confluence pages directly.
allowed-tools: ["Read", "Glob", "Grep", "Bash"]
---

# walkr-confluence-publish

You publish an existing `.walkr/` walkthrough to Confluence by running the `walkr` binary. All the logic lives in the binary (`walkr confluence check` and `walkr confluence publish`). Your job is to check the preconditions, show the plan, get a go-ahead, run it, and report what happened. You never write page content yourself and you never call the Atlassian MCP to publish.

## What it produces

A tree under the configured parent page, in the user's Confluence space:

- an optional section page (for example "Tutorials"), created once and never rewritten,
- one tutorial page per walkthrough, with the tagline, a start link, and the ordered chapter list,
- one child page per step, titled "<walkthrough title>: <step title>", with a lede panel, the content, a Terms panel, a previous and next pager, and Mermaid diagrams rendered to PNG and attached.

Full design and constraints are in `docs/ai/confluence-publish-design.md` in the walkr repo.

## Step 0: check the preconditions

Check, do not guess, and never print a secret.

1. `walkr` is on `PATH` (`command -v walkr`). If not, say so and point at `docs/user/README.md` for install options, do not try to install it.
2. The global config exists at `$XDG_CONFIG_HOME/walkr/config.yaml`, falling back to `~/.config/walkr/config.yaml`, and has a target for this walkthrough. If the file or target is missing, help the user write it from the example in the design doc. The config holds pointers only, an `op://` reference for the Atlassian API token, never the token itself.
3. The 1Password service account token is present in the environment. Check with `test -n "$OP_SERVICE_ACCOUNT_TOKEN"` (or the variable named by `auth.service_account_token_env`) and report only "set" or "not set".
4. Chrome or Chromium is optional. Without it, diagrams fall back to source in an expand. `WALKR_CHROME` sets the browser path, and `diagrams: source` in the target skips rendering on purpose.

## Step 1: verify access

Run `walkr confluence check --target <name>`. It resolves the token, lists spaces, and reads the target's parent page, and prints no secrets. Stop and explain if it fails, using the troubleshooting table below.

## Step 2: show the plan

Run `walkr confluence publish --target <name> --dry-run`. It looks pages up and writes nothing. Summarize it, how many pages would be created and how many updated. Before the first republish over pages that already exist, say plainly that publishing is one way and overwrites any edits made in Confluence.

## Step 3: publish

Only after the user says to go, run `walkr confluence publish --target <name>`. Report each created or updated page, every warning verbatim, and the "start here" URL. Warnings mean a link or term did not resolve, or diagrams fell back to source. Offer to fix the walkthrough content, not the Confluence pages.

## Rules

- Never print, echo, or log a secret. Never use `op read` or any command that prints a credential. The binary resolves the token itself.
- Publish only from the walkthrough files. Do not edit Confluence pages directly, since the next publish overwrites them.
- The token has no delete scope. If pages need removing, tell the user to delete them by hand in Confluence.
- Do not commit, push, or change the config unless asked.

## Troubleshooting

| Symptom | Meaning |
|---|---|
| `environment variable OP_SERVICE_ACCOUNT_TOKEN ... is not set` | The 1Password service account token is not in this shell's environment. |
| `config file ... not found` | No global config yet. Create it from the design doc example. |
| `401 ... scope does not match` | The Atlassian API token lacks a scope. Needed: read and write content, read space, read and write label, write attachment, search. |
| `401` on a read | Wrong email or token, or the token was revoked. |
| `409 ... Draft versioning is not supported` | Something tried to save a draft as a new version. Report it as a bug. |
| `no Chrome or Chromium found` | Install one, set `WALKR_CHROME`, or set `diagrams: source`. |
| A warning about a step link or term | The step names a chapter or glossary term that does not exist. Fix the markdown. |
