# PostHog

| | |
|---|---|
| **URL** | `https://mcp.posthog.com/mcp` |
| **Auth** | `oauth: true` → Claude **Connect** after sync |

PostHog routes to the correct US/EU region based on the account you sign in with.

The Claude Code plugin may add headers such as `x-posthog-mcp-consumer: plugin`. Cowork 3P `managedMcpServers` does not allow `headers` and `oauth` on the same entry — use `oauth: true` here (same URL as the plugin).

## Policy

```yaml
servers:
  - name: posthog
    url: https://mcp.posthog.com/mcp
    transport: http
    oauth: true
```

## Setup

```bash
ally3p claude sync ally.yaml
# Restart Claude → click Connect on posthog
ally3p claude login posthog ally.yaml   # prints instructions
```

## Expected config

```json
{
  "name": "posthog",
  "url": "https://mcp.posthog.com/mcp",
  "transport": "http",
  "oauth": true
}
```
