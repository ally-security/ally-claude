# Figma

| Mode | URL | Auth |
|------|-----|------|
| Remote | `https://mcp.figma.com/mcp` | `oauth: true` → Claude Connect |
| Desktop | `http://127.0.0.1:3845/mcp` | Figma app running, no OAuth |

## Policy

```yaml
servers:
  - name: figma
    url: https://mcp.figma.com/mcp
    transport: http
    oauth: true

  # Optional: local desktop MCP (no oauth)
  # - name: figma-desktop
  #   url: http://127.0.0.1:3845/mcp
  #   transport: http
```

## Setup (remote)

```bash
ally3p claude sync ally.yaml
# Restart Claude → click Connect on figma
ally3p claude login figma ally.yaml   # prints instructions
```

## Expected config (remote)

```json
{
  "name": "figma",
  "url": "https://mcp.figma.com/mcp",
  "transport": "http",
  "oauth": true
}
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Connect fails | Ensure Claude Cowork 3P supports dynamic client registration |
| Desktop unreachable | Open Figma desktop app; enable MCP in settings |
