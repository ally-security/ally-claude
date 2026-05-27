# ally3p

Manage **Claude Cowork 3P** from a YAML policy file. Sync writes Claude’s `configLibrary`, stores secrets in Keychain, and wires MCP servers.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/allysecurity/ally-claude/main/install.sh | bash
```

## Quick start

```bash
cp example-policy.yaml ally.yaml   # edit with your creds
ally3p claude sync ally.yaml --dry
ally3p claude sync ally.yaml
# Restart Claude Cowork 3P
```

See [`example-policy.yaml`](example-policy.yaml) for the full policy schema.

## Docs

| Service | Auth | Guide |
|---------|------|-------|
| Linear | Dynamic OAuth (Claude Connect) | [docs/linear.md](docs/linear.md) |
| Figma | Dynamic OAuth / desktop local | [docs/figma.md](docs/figma.md) |
| PostHog | Dynamic OAuth (Claude Connect) | [docs/posthog.md](docs/posthog.md) |
| Google Workspace | Custom OAuth + headersHelper | [docs/google-workspace.md](docs/google-workspace.md) |
| Slack | Custom OAuth + headersHelper | [docs/slack.md](docs/slack.md) |
| HubSpot | MCP Auth App + PKCE | [docs/hubspot.md](docs/hubspot.md) |

## Commands

```bash
ally3p prereq                          # extract embedded helpers
ally3p claude sync <policy.yaml>       # apply policy
ally3p claude sync <policy.yaml> --dry # preview
ally3p claude login slack [policy.yaml]
ally3p claude login hubspot [policy.yaml]
ally3p claude login google [policy.yaml]
ally3p claude login [policy.yaml]      # all OAuth services in policy
```

Run `ally3p help` for more.

## Dev

```bash
make build
./bin/ally3p prereq --dir ./bin
```

Releases ship **ally3p only**; helpers are embedded and extracted via `prereq`.
