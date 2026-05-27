# Figma

| Mode | URL | Auth |
|------|-----|------|
| Remote | `https://mcp.figma.com/mcp` | Dynamic OAuth → Claude Connect |
| Desktop | `http://127.0.0.1:3845/mcp` | Figma app running, no OAuth |

## Policy

```yaml
servers:
  - figma: remote          # shorthand (recommended)
  # - figma: desktop       # optional local MCP

  # Or explicit:
  # - name: figma
  #   url: https://mcp.figma.com/mcp
  #   oauth: true
```

Remote URLs get `oauth: true` automatically on sync if omitted.

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
  "oauth": true
}
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Connect fails | Ensure Claude Cowork 3P supports dynamic client registration |
| Desktop unreachable | Open Figma desktop app; enable MCP in settings |
