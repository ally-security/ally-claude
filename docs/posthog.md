# PostHog

| | |
|---|---|
| **URL** | `https://mcp.posthog.com/mcp` |
| **Auth** | Dynamic OAuth (`oauth: true`) |
| **Sign in** | Claude **Connect** after sync |

PostHog routes to the correct US/EU region based on the account you sign in with.

## Policy

```yaml
servers:
  - posthog: true

  # Or explicit:
  # - name: posthog
  #   url: https://mcp.posthog.com/mcp
  #   oauth: true
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
  "oauth": true
}
```
