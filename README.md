# claude-3p-helper

Sync MDM-style policy files into Claude-3p's per-user `configLibrary/`
directory — install connectors, plugins, and extensions across a fleet
of workstations from a single versioned YAML in a repo.

## Why

Claude-3p reads policy from two places: a managed-prefs surface
(macOS `/Library/Managed Preferences/...`, Windows Group Policy
registry) that needs real MDM tooling, and a per-user `configLibrary/`
directory that's just JSON files on disk. This binary writes to the
per-user path so a team can keep policy in git and apply it on every
workstation with one command.

It does **not** touch Managed Preferences or the Windows registry —
that remains MDM territory.

## Install

```sh
export GITHUB_TOKEN=$(gh auth token)   # only while the repo is private
curl -fsSL https://raw.githubusercontent.com/ally-security/ally-claude/main/install.sh | sh
```

The script detects OS/arch, downloads the latest release, verifies
sha256, and installs to `/usr/local/bin` (or `~/.local/bin`).

Pin a version or change the destination:

```sh
curl -fsSL .../install.sh | sh -s -- --version v0.1.0 --dir ~/.local/bin
```

Subsequent updates with `claude-3p-helper self-update`.

## Use

```sh
claude-3p-helper sync user/repo/path/to/policy.yaml   # apply a policy
claude-3p-helper models                               # list models in the active config
claude-3p-helper self-update                          # replace the binary with the latest release
claude-3p-helper version
```

`sync` resolves the argument as a local file first, then falls back
to GitHub. For private policy repos, set `GITHUB_TOKEN` (or `GH_TOKEN`)
and the resolver fetches via the GitHub Contents API; otherwise it
uses `raw.githubusercontent.com`. By default the synced config becomes
active (`_meta.activeConfigId`); use `--no-activate` to opt out, or
`--dry-run` to preview without writing. `--verbose` enables
debug-level logs.

Two example policies live in [`examples/`](examples/):

- [`saas-bundle.yaml`](examples/saas-bundle.yaml) — Anthropic
  inference plus connectors for Notion, PostHog, Linear, HubSpot,
  Slack, Google Drive, and Gmail.
- [`bedrock.yaml`](examples/bedrock.yaml) — same bundle routed
  through Amazon Bedrock with a ten-entry model inventory.

Each file has inline `Docs:` comments pointing at the relevant 3P
config reference. After a sync, run `claude-3p-helper models` to see
exactly what landed.
