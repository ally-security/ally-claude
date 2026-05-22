# claude-3p-helper

A small Go binary that syncs MDM-style policy files into Claude-3p's
configLibrary — install connectors, plugins, and extensions across a
fleet of workstations from a single versioned YAML file in a repo.

```
claude-3p-helper sync <user>/<repo>/<path>   # apply a policy
claude-3p-helper models                      # list models in the active config
claude-3p-helper self-update                 # replace the binary with the latest release
claude-3p-helper version                     # print version + commit + Go version
```

## Why

Claude-3p has two policy surfaces: a managed-prefs path (macOS
`/Library/Managed Preferences/...`, Windows Group Policy registry keys)
that requires real MDM tooling, and a per-user `configLibrary/`
directory that's just JSON files on disk. This binary writes to the
per-user path, so a team can:

- Keep a YAML policy in git, with a doc-linked schema and per-vendor
  notes.
- Push it to every workstation via cron, a config-management runner,
  or whatever you already use.
- Verify the result with `models` and `--dry-run`.

It does **not** touch the Managed Preferences plist or the Windows
registry — that remains MDM territory.

## Install

Grab the binary for your platform from the latest
[release](https://github.com/ally-security/ally-claude/releases/latest):

```sh
# macOS arm64
curl -fsSL -o claude-3p-helper.tar.gz \
  https://github.com/ally-security/ally-claude/releases/latest/download/claude-3p-helper_<version>_darwin_arm64.tar.gz
tar xzf claude-3p-helper.tar.gz
mv claude-3p-helper /usr/local/bin/
```

Subsequent updates with `claude-3p-helper self-update`.

Or build from source:

```sh
go install github.com/anthropics/claude-3p-helper/cmd/claude-3p-helper@latest
```

## Usage

### `sync` — apply a policy

```sh
claude-3p-helper sync acme/claude-policy/examples/saas-bundle.yaml
```

The argument is `<user>/<repo>/<path>`. The resolver tries the local
filesystem first (relative or `~/`-expanded), then falls back to
`https://raw.githubusercontent.com/<user>/<repo>/<branch>/<path>`. So
during development you can point at a local file; in production it
fetches from your policy repo.

Flags:

| Flag | Default | Description |
|---|---|---|
| `--branch` | `main` | git branch when fetching from GitHub |
| `--dry-run` | off | print planned actions, don't write |
| `--no-activate` | off | don't mark the synced config as active |
| `--verbose` | off | debug-level logs to stderr |

By default the synced config becomes the active one (`_meta.json`
gets `activeConfigId` updated). Pass `--no-activate` to write the
config without making it active.

### `models` — local model inventory

```sh
claude-3p-helper models                       # active config
claude-3p-helper models --config saas-2026-05 # specific config
claude-3p-helper models --all                 # every config in the library
```

Lists `inferenceModels` from each config, formatting `labelOverride`
and a `[1M]` marker for entries with `supports1m: true`. Useful for
sanity-checking a freshly-synced Bedrock policy.

### `self-update`

```sh
claude-3p-helper self-update           # download + replace binary
claude-3p-helper self-update --check   # report whether an update exists
```

Queries the GitHub releases API, picks the archive matching the host
`GOOS`/`GOARCH`, verifies the downloaded archive against `checksums.txt`,
extracts the binary, and atomically swaps it over the running
executable. On Windows the previous binary is moved to `<path>.old`
first because Windows won't allow deleting a running executable.

## Policy schema

Two example policies live under [`examples/`](examples/):

- [`examples/saas-bundle.yaml`](examples/saas-bundle.yaml) — Anthropic
  inference + seven SaaS connectors (Notion, PostHog, Linear, HubSpot,
  Slack, Google Drive, Gmail).
- [`examples/bedrock.yaml`](examples/bedrock.yaml) — same connector
  bundle, but routed through Amazon Bedrock with a ten-entry model
  inventory using `labelOverride` for application-inference-profile
  and provisioned-throughput ARNs.

Both files have inline `Docs:` comments pointing at the relevant
section of the [3P configuration reference](https://claude.com/docs/cowork/3p/configuration)
and the [Bedrock-specific keys](https://claude.com/docs/cowork/3p/bedrock).

### Connector OAuth — DCR vs pre-registered

Verified against each vendor's public MCP docs (May 2026):

| Vendor | URL | OAuth model |
|---|---|---|
| Notion | `https://mcp.notion.com/mcp` | `oauth: true` (RFC 7591 DCR) |
| PostHog | `https://mcp.posthog.com/mcp` | `oauth: true` (DCR) |
| Linear | `https://mcp.linear.app/mcp` | `oauth: true` (DCR — explicit in their docs) |
| HubSpot | `https://mcp.hubspot.com/mcp` | pre-registered (HubSpot developer-portal app) |
| Slack | `https://mcp.slack.com/mcp` | pre-registered (Slack publishes a public clientId for the hosted MCP) |
| Google Drive | `https://drivemcp.googleapis.com/mcp/v1` | pre-registered (Cloud Console OAuth client) |
| Gmail | `https://gmailmcp.googleapis.com/mcp/v1` | pre-registered |

### Bedrock auth paths

Exactly one of:

- `inferenceBedrockBearerToken` — Bedrock console-issued API key
- `inferenceBedrockProfile` — named profile from `~/.aws/credentials`
- `inferenceBedrockSso{StartUrl,Region,AccountId,RoleName}` — interactive AWS SSO
- `inferenceCredentialHelper` — executable that prints a token to
  stdout at runtime (the right path for **STS-issued temporary
  credentials with rotating session tokens**)

## Where things land

The binary writes to a per-user directory. Paths are inferred from
the 3P doc plus reasonable defaults; verify against an actual Claude-3p
install before shipping widely.

| OS | Base |
|---|---|
| macOS | `~/Library/Application Support/Claude-3p/` |
| Windows | `%LOCALAPPDATA%\Claude-3p\` |
| Linux | `~/.config/Claude-3p/` |

Inside the base directory:

- `configLibrary/<id>.json` — the policy JSON
- `configLibrary/_meta.json` — `activeConfigId`
- `orgPlugins/<name>/` — unzipped plugin bundles
- `extensions/<name>.mcpb` — desktop extension files

`sync` is idempotent for plugins and extensions: if the on-disk
sha256 matches the manifest's `sha256:` field, the download is
skipped.

## Logging

`log/slog` text handler writing to stderr with RFC3339 timestamps and
key=value attributes. `--verbose` flips the level from INFO to DEBUG.

```
time=2026-05-20T19:46:56-06:00 level=INFO msg="sync starting" arg=examples/saas-bundle.yaml branch=main dryRun=false activate=true
time=2026-05-20T19:46:56-06:00 level=INFO msg="policy resolved" origin=local:examples/saas-bundle.yaml bytes=4717
time=2026-05-20T19:46:56-06:00 level=INFO msg="policy loaded" id=saas-baseline-2026-05 provider=anthropic connectors=7 plugins=0 extensions=0
time=2026-05-20T19:46:56-06:00 level=INFO msg="config written" id=saas-baseline-2026-05 path="…/configLibrary/saas-baseline-2026-05.json" bytes=2410 connectors=7
time=2026-05-20T19:46:56-06:00 level=INFO msg="config activated" id=saas-baseline-2026-05 metaPath="…/configLibrary/_meta.json"
```

Stdout stays clean — the policy-plan summary goes there so you can
pipe it to a reviewer.

## Development

```sh
go build ./cmd/claude-3p-helper
go test ./... -cover
gofmt -l .
go vet ./...
```

Releases are cut with GoReleaser via the `release` workflow — tag
`v*` and push to fire it. The CI workflow runs gofmt + vet + build +
test on PRs, plus a cross-compile matrix for darwin/linux/windows.

## Status

Pre-1.0; the schema closely mirrors the 3P configuration doc but the
`orgPlugins/` and `extensions/` directory names are inferred and
should be verified against a real Claude-3p install.
