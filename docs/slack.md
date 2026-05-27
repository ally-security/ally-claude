# Slack

| | |
|---|---|
| **URL** | `https://mcp.slack.com/mcp` |
| **Auth** | Custom Slack app + `headersHelper` |
| **Sign in** | `ally3p claude login slack [policy.yaml]` |

**Do not use Claude Connect for Slack** — Cowork’s built-in Slack OAuth is unreliable for custom apps.

## Slack app setup

[api.slack.com](https://api.slack.com/apps):

1. **Agents & AI Apps** → enable **Model Context Protocol**
2. **OAuth & Permissions** → redirect URLs (**both**):
   ```
   http://127.0.0.1:3118/callback
   http://localhost:3118/callback
   ```
3. **User Token Scopes**: `chat:write`, `search:read.*`, channel history scopes, etc.
4. Enable MCP at `https://api.slack.com/apps/<APP_ID>/app-assistant`

## Policy

```yaml
servers:
  - name: slack
    url: https://mcp.slack.com/mcp
    oauth:
      client_id: "YOUR_SLACK_CLIENT_ID"
      client_secret: "YOUR_SLACK_CLIENT_SECRET"
    callback_port: 3118
```

## Setup

```bash
ally3p claude sync ally.yaml
ally3p claude login slack ally.yaml
# Restart Claude
```

## Verify

```bash
slack-mcp-auth
# → {"Authorization":"Bearer xoxp-..."}
```

Slack tokens do not auto-refresh unless your app has token rotation enabled.

## Expected config

```json
{
  "name": "slack",
  "url": "https://mcp.slack.com/mcp",
  "headersHelper": "/usr/local/bin/slack-mcp-auth",
  "headersHelperTtlSec": 300
}
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `App is not enabled for Slack MCP` | Enable MCP in app-assistant settings |
| Claude Connect token error | Expected — use `headersHelper`, not Connect |
| `slack-mcp-auth not found` | `ally3p prereq` |
| Token expired | `ally3p claude login slack ally.yaml` |

## Keychain

| Account | Contents |
|---------|----------|
| `oauth-client` | Slack client id + secret |
| `user-token` | Slack user token (`xoxp-...`) |
