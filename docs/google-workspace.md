# Google Workspace

| | |
|---|---|
| **URLs** | `https://gmailmcp.googleapis.com/mcp/v1`, drive, calendar, chat, people |
| **Auth** | Custom Google Cloud OAuth client + `headersHelper` |
| **Sign in** | `ally3p claude login google [policy.yaml]` |

One Google account covers all `google_service` entries. Tokens are stored as `google-user` in Keychain and auto-refresh.

## Policy

```yaml
servers:
  - google_service: gmail
    client_id: "YOUR_ID.apps.googleusercontent.com"
    client_secret: "GOCSPX-..."
    scope: "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.compose"

  - google_service: drive
    client_id: "YOUR_ID.apps.googleusercontent.com"
    client_secret: "GOCSPX-..."
    scope: "https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/drive.file"
```

`client_secret` is installed to Keychain on sync — never written to `configLibrary`.

## Google Cloud setup

1. Create OAuth client in [Google Cloud Console](https://console.cloud.google.com/)
2. Register redirect URI per service: `http://127.0.0.1:<port>/callback`

| Service | Port |
|---------|------|
| gmail | 53281 |
| drive | 53282 |
| calendar | 53283 |
| chat | 53284 |
| people | 53285 |

## Setup

```bash
ally3p claude sync ally.yaml
ally3p claude login google ally.yaml
# Restart Claude
```

## Verify

```bash
google-workspace-mcp-auth-gmail          # {"Authorization":"Bearer ..."}
google-workspace-mcp-auth-gmail verify   # MCP + redirect + headers
```

## Expected config

```json
{
  "name": "google-gmail",
  "url": "https://gmailmcp.googleapis.com/mcp/v1",
  "headersHelper": "/usr/local/bin/google-workspace-mcp-auth-gmail",
  "headersHelperTtlSec": 300
}
```

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `keychain read: exit status 44` | `ally3p claude login google ally.yaml` |
| `redirect_uri_mismatch` | Add redirect URI in Google Cloud for that service port |
| MCP permission denied | Enable Gmail MCP API + Developer Preview in GCP |
| Stale helpers | `sudo ally3p prereq` |

## IT: install creds without YAML

Once per Mac:

```bash
sudo google-workspace-mcp-auth install-credentials \
  --client-id '....apps.googleusercontent.com' \
  --client-secret 'GOCSPX-...'
```

## Keychain

| Account | Contents |
|---------|----------|
| `oauth-client-<service>` | OAuth client id + secret |
| `google-user` | Shared access/refresh token |
