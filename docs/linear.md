# Linear

| | |
|---|---|
| **URL** | `https://mcp.linear.app/mcp` |
| **Auth** | Dynamic OAuth (`oauth: true`) |
| **Sign in** | Claude **Connect** after sync — no helper binary |

## Policy

```yaml
servers:
  - name: linear
    url: https://mcp.linear.app/mcp
    oauth: true
```

## Setup

```bash
ally3p claude sync ally.yaml
# Restart Claude Cowork 3P → click Connect on linear
```

## Expected config

```json
{
  "name": "linear",
  "url": "https://mcp.linear.app/mcp",
  "oauth": true
}
```
