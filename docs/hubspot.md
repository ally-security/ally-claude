# HubSpot

| | |
|---|---|
| **URL** | `https://mcp.hubspot.com/anthropic` |
| **Auth** | [MCP Auth App](https://developers.hubspot.com/docs/apps/developer-platform/build-apps/integrate-with-the-remote-hubspot-mcp-server) + `headersHelper` |
| **Sign in** | `ally3p claude login hubspot [policy.yaml]` |

PKCE is required by HubSpot and handled automatically by `hubspot-mcp-auth` — nothing to configure in YAML.

## HubSpot app setup

1. Developer Portal → **Development → MCP Auth Apps** → **Create**
2. Redirect URLs (**both**):
   ```
   http://127.0.0.1:3119/callback
   http://localhost:3119/callback
   ```
3. Copy **Client ID** and **Client secret**

Scopes are determined at install time by HubSpot.

## Policy

```yaml
servers:
  - hubspot: true
    oauth:
      client_id: "YOUR_HUBSPOT_CLIENT_ID"
      client_secret: "YOUR_HUBSPOT_CLIENT_SECRET"
    callback_port: 3119

  # Or explicit:
  # - name: hubspot
  #   url: https://mcp.hubspot.com/anthropic
  #   oauth:
  #     client_id: "..."
  #     client_secret: "..."
  #   callback_port: 3119
```

## Setup

```bash
ally3p claude sync ally.yaml
ally3p claude login hubspot ally.yaml
# Restart Claude
```

## Verify

```bash
hubspot-mcp-auth
# → {"Authorization":"Bearer ..."}
```

Tokens auto-refresh via refresh token.

## Expected config

```json
{
  "name": "hubspot",
  "url": "https://mcp.hubspot.com/anthropic",
  "headersHelper": "/usr/local/bin/hubspot-mcp-auth",
  "headersHelperTtlSec": 300
}
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Redirect mismatch | Register both `127.0.0.1` and `localhost` callback URLs |
| `hubspot-mcp-auth not found` | `ally3p prereq` |
| Token expired | `ally3p claude login hubspot ally.yaml` |

## Keychain

| Account | Contents |
|---------|----------|
| `oauth-client` | HubSpot client id + secret |
| `user-token` | Access + refresh token (JSON) |
