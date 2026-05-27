# ally3p

Manage **Claude Cowork 3P** configuration from a single YAML file.  
Includes helper binaries for hosted MCP OAuth:

- `google-workspace-mcp-auth` — Google Workspace MCP (Gmail, Drive, Calendar, Chat, People)
- `slack-mcp-auth` — Slack hosted MCP (`https://mcp.slack.com/mcp`)
- `hubspot-mcp-auth` — HubSpot hosted MCP (`https://mcp.hubspot.com/anthropic`)

Both use Claude’s `headersHelper` pattern (bearer token on stdout), bypassing Claude’s built-in OAuth for providers where it is unreliable.

**Figma**, **PostHog**, and **Linear** use Claude’s built-in OAuth with dynamic client registration (`oauth: true`). **HubSpot** requires a custom MCP Auth App (client id + secret + PKCE), like Slack — not DCR.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/allysecurity/ally-claude/main/install.sh | bash
```

Installs `ally3p` to `/usr/local/bin`. Helper binaries are embedded and extracted with `ally3p prereq` (run automatically by `install.sh`).

## Quick start

```bash
# 1. Write your policy YAML (see example-policy.yaml)

# 2. Install prereqs (helper binaries)
ally3p prereq

# If you get a permissions error, run:
#   sudo ally3p prereq

# 3. Preview what will be written
ally3p claude sync my-policy.yaml --dry

# 4. Apply (opens Google + Slack sign-in if needed)
ally3p claude sync my-policy.yaml

# 5. Restart Claude Cowork 3P
```

One Google account = **one browser sign-in** for every `google_service` in your policy.  
Google tokens are stored as `google-user` in Keychain (shared across Gmail/Drive/Calendar).  
Slack user tokens (`xoxp-...`) are stored separately in Keychain (`slack-mcp-auth` / `user-token`).

## Run locally (dev)

```bash
# from repo root
make build

# install prereqs locally (no sudo)
./bin/ally3p prereq --dir ./bin

# preview generated Claude config (no writes)
./bin/ally3p claude sync example-policy.yaml --dry

# apply to your active Claude 3P configLibrary file
./bin/ally3p claude sync example-policy.yaml
```

Notes:
- `ally3p claude sync` auto-runs `ally3p prereq` on first non-dry run if Google wrappers are missing.
- Sync auto-detects helpers in `--helper-dir`, alongside the `ally3p` binary, `/usr/local/bin`, and `PATH`.
- `--dry` never writes files and never writes to Keychain.

## Policy YAML

```yaml
inference:
  provider: bedrock
  bedrock_region: us-east-1
  bedrock_token: "..."

banner:
  enabled: false

servers:
  - name: linear
    url: https://mcp.linear.app/mcp
    oauth: true

  # Figma remote — oauth: true auto-set on sync; auth via Claude Connect after restart
  - figma: remote
  # Or explicitly:
  # - name: figma
  #   url: https://mcp.figma.com/mcp
  #   oauth: true

  # PostHog — oauth: true auto-set on sync
  - posthog: true

  - name: slack
    url: https://mcp.slack.com/mcp
    oauth:
      client_id: "YOUR_SLACK_CLIENT_ID"
      client_secret: "YOUR_SLACK_CLIENT_SECRET"   # installed into Keychain on sync, not written to configLibrary
    callback_port: 3118

  - google_service: gmail
    client_id: "YOUR_ID.apps.googleusercontent.com"
    client_secret: "GOCSPX-..."   # installed into Keychain on sync, not written to configLibrary
    scope: "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.compose"

  - google_service: drive
    client_id: "YOUR_ID.apps.googleusercontent.com"
    client_secret: "GOCSPX-..."
    scope: "https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/drive.file"

  - google_service: calendar
    client_id: "YOUR_ID.apps.googleusercontent.com"
    client_secret: "GOCSPX-..."
    scope: "https://www.googleapis.com/auth/calendar.calendarlist.readonly https://www.googleapis.com/auth/calendar.events.freebusy https://www.googleapis.com/auth/calendar.events.readonly"
```

The sync command:
- Installs `client_secret` values from YAML into Keychain, then writes `configLibrary` **without secrets**
- Wires `headersHelper` for Google and Slack (not Claude’s `oauth: true` for those providers)
- Writes the active Claude `configLibrary` JSON in-place (preserves the config ID)

## Testing Google login

Test Google **before** relying on Claude. Each step should exit 0.

### 1. Sync policy (install client secrets + config)

```bash
make build
./bin/ally3p prereq --dir ./bin          # or: sudo ally3p prereq
./bin/ally3p claude sync ally.yaml --dry # preview
./bin/ally3p claude sync ally.yaml       # apply + browser sign-in if no token yet
```

Or sign in to a specific service:

```bash
./bin/ally3p claude login slack [ally.yaml]      # Slack only
./bin/ally3p claude login hubspot [ally.yaml]    # HubSpot only (PKCE)
./bin/ally3p claude login google [ally.yaml]     # Google (all scopes in policy)
./bin/ally3p claude login ally.yaml              # slack + hubspot + google in policy
./bin/ally3p claude login figma ally.yaml        # info: use Claude Connect after sync
./bin/ally3p claude login posthog ally.yaml      # info: use Claude Connect after sync
```

Services: `slack`, `hubspot`, `google`, `figma`, `posthog`, `all`.

### 2. Verify helper prints a Bearer token

Default command (what Claude calls as `headersHelper`):

```bash
google-workspace-mcp-auth-gmail
# or: /usr/local/bin/google-workspace-mcp-auth-gmail
```

Success looks like:

```json
{"Authorization":"Bearer ya29...."}
```

The helper **auto-refreshes** Google access tokens when they are within 60 seconds of expiry.

### 3. Run built-in verify (MCP + redirect URI + headers)

```bash
google-workspace-mcp-auth-gmail verify
```

Checks:
- Gmail MCP endpoint responds to `initialize`
- OAuth redirect URI is registered in Google Cloud (`http://127.0.0.1:53281/callback` for Gmail)
- Keychain has a user token and headers print successfully

Repeat for other services if needed:

```bash
google-workspace-mcp-auth-drive verify
google-workspace-mcp-auth-calendar verify
```

### 4. Call Gmail MCP with the token (end-to-end)

```bash
TOKEN=$(google-workspace-mcp-auth-gmail | python3 -c "import sys,json; print(json.load(sys.stdin)['Authorization'].split()[1])")

curl -sS -X POST "https://gmailmcp.googleapis.com/mcp/v1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":1,"params":{}}' | head -c 400
```

You should get a JSON-RPC result with Gmail tools (not `missing_token` or permission errors).

### 5. Confirm Claude config

```bash
jq '.managedMcpServers[] | select(.name | startswith("google"))' \
  ~/Library/Application\ Support/Claude-3p/configLibrary/*.json
```

Expect `headersHelper` pointing at `google-workspace-mcp-auth-<service>` and **no** `oauth` object on the entry.

### Google troubleshooting

| Symptom | Fix |
|---------|-----|
| `keychain read: exit status 44` | Run `./bin/ally3p claude login ally.yaml` |
| `redirect_uri_mismatch` in verify | Add `http://127.0.0.1:<port>/callback` in Google Cloud OAuth client |
| MCP `tools/call` permission denied | Enable Gmail MCP API + Developer Preview for your GCP project |
| Stale `/usr/local/bin` binary | `sudo ally3p prereq` to reinstall helpers |

Watch Claude logs:

```bash
tail -f ~/Library/Logs/Claude-3p/main.log | grep -i 'google-gmail\|custom3p-mcp-headers'
```

Success in Claude: `[custom3p-mcp] connected { name: 'google-gmail', auth: 'headers-helper' }`.

---

## Testing Slack login

Slack uses the same `headersHelper` pattern. **Do not use Claude’s Connect button for Slack** — Cowork’s built-in Slack OAuth token exchange is unreliable for custom apps.

### Slack app prerequisites (api.slack.com)

1. **Agents & AI Apps** → enable **Model Context Protocol**
2. **OAuth & Permissions** → redirect URLs (register **both**):
   ```
   http://127.0.0.1:3118/callback
   http://localhost:3118/callback
   ```
3. **User Token Scopes** (not bot): `chat:write`, `search:read.*`, channel history scopes, etc.

### 1. Sync policy (install client creds + config)

```bash
make build
./bin/ally3p prereq --dir ./bin
./bin/ally3p claude sync ally.yaml
```

Or sign in explicitly:

```bash
./bin/ally3p claude login slack ally.yaml
# or: slack-mcp-auth login
```

Browser opens → click **Allow** → token saved to Keychain (`slack-mcp-auth` / `user-token`).

### 2. Verify helper prints a Bearer token

```bash
slack-mcp-auth
```

Success:

```json
{"Authorization":"Bearer xoxp-..."}
```

Slack tokens **do not auto-refresh** today (unless your Slack app has token rotation enabled). Re-run login if the token is revoked or expires.

### 3. Call Slack MCP with the token (end-to-end)

```bash
TOKEN=$(slack-mcp-auth | python3 -c "import sys,json; print(json.load(sys.stdin)['Authorization'].split()[1])")

# initialize
curl -sS -X POST "https://mcp.slack.com/mcp" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -d '{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}'

# tools/list
curl -sS -X POST "https://mcp.slack.com/mcp" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"tools/list","id":2,"params":{}}' | head -c 500
```

Success: `serverInfo.name` = `Slack MCP` and ~19 tools listed.

### 4. MCP Inspector (optional visual test)

```bash
npx -y @modelcontextprotocol/inspector --cli https://mcp.slack.com/mcp \
  --transport http --method tools/list \
  --header "Authorization: Bearer $TOKEN"
```

Or open the UI:

```bash
npx -y @modelcontextprotocol/inspector
# Transport: Streamable HTTP → https://mcp.slack.com/mcp → paste bearer token
```

### 5. Full OAuth script (alternative)

```bash
export SLACK_CLIENT_ID="YOUR_CLIENT_ID"
export SLACK_CLIENT_SECRET="YOUR_CLIENT_SECRET"
export SLACK_REDIRECT_URI="http://localhost:3118/callback"
python3 scripts/test-slack-mcp-oauth.py
```

Runs browser OAuth, saves token to `~/.config/claude-3p/slack-mcp-token.json`, and tests `initialize` + `tools/list`.

### 6. Confirm Claude config

```bash
jq '.managedMcpServers[] | select(.name=="slack")' \
  ~/Library/Application\ Support/Claude-3p/configLibrary/*.json
```

Expect:

```json
{
  "name": "slack",
  "url": "https://mcp.slack.com/mcp",
  "headersHelper": "/usr/local/bin/slack-mcp-auth",
  "headersHelperTtlSec": 300
}
```

No `oauth` block — Slack connects automatically on Claude restart via `headersHelper`.

### Slack troubleshooting

| Symptom | Fix |
|---------|-----|
| `App is not enabled for Slack MCP server access` | Enable MCP at `https://api.slack.com/apps/<APP_ID>/app-assistant` |
| Claude Connect → `access_token` / `token_type` invalid | Expected — use `headersHelper` path, not Connect |
| `slack-mcp-auth not found` on sync | Run `ally3p prereq` |
| Token expired / revoked | `./bin/ally3p claude login ally.yaml` |

Watch Claude logs:

```bash
tail -f ~/Library/Logs/Claude-3p/main.log | grep -i slack
```

Success: `[custom3p-mcp] connected { name: 'slack', auth: 'headers-helper' }`.

---

## Figma MCP

Figma has two MCP endpoints. ally3p supports both via policy YAML.

| Mode | URL | Auth |
|------|-----|------|
| **Remote** (hosted) | `https://mcp.figma.com/mcp` | `oauth: true` — click **Connect** in Claude Cowork |
| **Desktop** (local) | `http://127.0.0.1:3845/mcp` | None — Figma desktop app must be running with MCP enabled |

### Policy YAML

```yaml
servers:
  # Shorthand (recommended)
  - figma: remote

  # Desktop (optional second entry)
  - figma: desktop

  # Explicit remote entry (same result as shorthand)
  - name: figma
    url: https://mcp.figma.com/mcp
    oauth: true
```

On sync, remote Figma URLs automatically get `oauth: true` if you omit it (dynamic client registration at Figma’s MCP OAuth endpoint).

### Setup steps (remote)

```bash
make build
./bin/ally3p claude sync ally.yaml --dry   # preview — expect oauth: true on figma
./bin/ally3p claude sync ally.yaml
# Restart Claude Cowork 3P
# Click Connect on the figma server in Claude
```

Or print setup instructions:

```bash
./bin/ally3p claude login figma ally.yaml
```

### Confirm Claude config

```bash
jq '.managedMcpServers[] | select(.name=="figma")' \
  ~/Library/Application\ Support/Claude-3p/configLibrary/*.json
```

Expect:

```json
{
  "name": "figma",
  "url": "https://mcp.figma.com/mcp",
  "transport": "http",
  "oauth": true
}
```

### Figma troubleshooting

| Symptom | Fix |
|---------|-----|
| Connect fails / no tools | Figma may restrict third-party OAuth for `mcp:connect`; ensure you are on Claude Cowork 3P with dynamic registration |
| Desktop MCP unreachable | Open Figma desktop app; enable MCP in Figma settings; confirm `curl http://127.0.0.1:3845/mcp` responds |
| Want both remote + desktop | Add two policy entries (`figma: remote` and `figma: desktop`) |

---

## PostHog MCP

PostHog’s hosted MCP (`https://mcp.posthog.com/mcp`) uses dynamic OAuth like Linear and Figma remote. PostHog routes you to the correct US/EU region based on the account you sign in with.

### Policy YAML

```yaml
servers:
  - posthog: true

  # Or explicitly:
  - name: posthog
    url: https://mcp.posthog.com/mcp
    oauth: true
```

On sync, PostHog URLs automatically get `oauth: true` if you omit it.

### Setup

```bash
./bin/ally3p claude sync ally.yaml --dry
./bin/ally3p claude sync ally.yaml
# Restart Claude Cowork 3P → click Connect on posthog
```

Or:

```bash
./bin/ally3p claude login posthog ally.yaml
```

### Confirm Claude config

```bash
jq '.managedMcpServers[] | select(.name=="posthog")' \
  ~/Library/Application\ Support/Claude-3p/configLibrary/*.json
```

Expect `oauth: true` and `url: https://mcp.posthog.com/mcp`.

---

## HubSpot MCP

HubSpot is **not** dynamic OAuth like Linear/Figma/PostHog. You must create an **MCP Auth App** in the [HubSpot Developer Portal](https://developers.hubspot.com/docs/apps/developer-platform/build-apps/integrate-with-the-remote-hubspot-mcp-server) and wire client credentials in your policy (same pattern as Slack).

| Item | Value |
|------|-------|
| MCP URL (Claude) | `https://mcp.hubspot.com/anthropic` |
| Auth | OAuth 2.0 + **PKCE** (required) |
| Redirect URIs | Register **both** `http://127.0.0.1:3119/callback` and `http://localhost:3119/callback` |

### 1. Create HubSpot MCP Auth App

1. HubSpot Developer Portal → **Development → MCP Auth Apps** → **Create**
2. Add redirect URLs (both):
   ```
   http://127.0.0.1:3119/callback
   http://localhost:3119/callback
   ```
3. Copy **Client ID** and **Client secret** from the app details page

Scopes are determined at install time by HubSpot (you do not pick them in the app UI).

### 2. Policy YAML

```yaml
servers:
  - hubspot: true
    oauth:
      client_id: "YOUR_HUBSPOT_CLIENT_ID"
      client_secret: "YOUR_HUBSPOT_CLIENT_SECRET"
    callback_port: 3119

  # Or explicit:
  - name: hubspot
    url: https://mcp.hubspot.com/anthropic
    oauth:
      client_id: "YOUR_HUBSPOT_CLIENT_ID"
      client_secret: "YOUR_HUBSPOT_CLIENT_SECRET"
    callback_port: 3119
```

On sync, `client_secret` goes to Keychain; configLibrary gets `headersHelper` (not `oauth`).

### 3. Sync + sign in

```bash
make build
./bin/ally3p prereq --dir ./bin
./bin/ally3p claude sync ally.yaml --dry
./bin/ally3p claude sync ally.yaml       # browser OAuth if no token yet
# Restart Claude Cowork 3P
```

Or sign in explicitly:

```bash
./bin/ally3p claude login ally.yaml      # all services in policy
# or: hubspot-mcp-auth login
```

### 4. Verify helper

```bash
hubspot-mcp-auth
```

Success:

```json
{"Authorization":"Bearer ..."}
```

Tokens auto-refresh via refresh token when near expiry.

### HubSpot troubleshooting

| Symptom | Fix |
|---------|-----|
| OAuth fails / redirect mismatch | Register both `127.0.0.1` and `localhost` callback URLs in MCP Auth App |
| `hubspot-mcp-auth not found` | Run `ally3p prereq` |
| Token expired | Re-run `./bin/ally3p claude sync ally.yaml` or `hubspot-mcp-auth login` |

---

## IT setup (Google — pick one)

### A. Keychain client secret (recommended)

**Once per Mac** (MDM script, onboarding, etc.):

```bash
sudo google-workspace-mcp-auth install-credentials \
  --client-id '....apps.googleusercontent.com' \
  --client-secret 'GOCSPX-...'
```

Stores OAuth client id + secret in macOS Keychain (`google-workspace-mcp-auth` / `oauth-client`).

**Claude policy** — secret stays out of `configLibrary`:

```json
{
  "name": "google-gmail",
  "transport": "http",
  "url": "https://gmailmcp.googleapis.com/mcp/v1",
  "headersHelper": "/usr/local/bin/google-workspace-mcp-auth-gmail",
  "headersHelperTtlSec": 300,
  "clientId": "YOUR_ID.apps.googleusercontent.com",
  "callbackPort": 53281,
  "scope": "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.compose"
}
```

The binary reads client secret from Keychain + `clientId`/`callbackPort`/`scope` from the config entry.  
Do **not** use Claude’s built-in `oauth: true` for Google hosted MCP.

Register redirect URI: `http://127.0.0.1:53281/callback` (per service port; see `list`).

### B. Credentials in Claude policy (legacy)

You can still put both `clientId` and `clientSecret` in `managedMcpServers.oauth` if you accept secrets on disk in the policy file.

### C. Enterprise credentials file

```bash
google-workspace-mcp-auth install-credentials --file \
  --client-id '....apps.googleusercontent.com' \
  --client-secret 'GOCSPX-...'
```

Writes `~/Library/Application Support/Claude-3p/google-workspace-oauth.json` (mode 600). Use on Linux or when Keychain is not available.

## Build

```bash
make build
```

Builds `bin/ally3p` and embeds google/slack/hubspot helpers for `ally3p prereq`. Releases ship **ally3p only**.

Install binaries + wrappers:

```bash
sudo ally3p prereq
# or: sudo make install
```

## Claude 3P config (per service)

Google:

```json
"headersHelper": "/usr/local/bin/google-workspace-mcp-auth-gmail"
```

Slack:

```json
"headersHelper": "/usr/local/bin/slack-mcp-auth"
```

Wrappers exist for Google because `headersHelper` cannot pass CLI arguments. Slack uses a single binary.

## Google services

```bash
google-workspace-mcp-auth list
```

| Service | Default port |
|---------|----------------|
| gmail | 53281 |
| drive | 53282 |
| calendar | 53283 |
| chat | 53284 |
| people | 53285 |

## Commands

### ally3p

| Command | Purpose |
|---------|---------|
| `prereq [--dir ./bin]` | Install `google-workspace-mcp-auth`, `slack-mcp-auth`, `hubspot-mcp-auth`, and Google wrappers |
| `claude sync <policy.yaml>` | Write configLibrary + Keychain creds + sign-in if needed |
| `claude sync <policy.yaml> --dry` | Preview JSON without writes |
| `claude login [policy.yaml]` | Sign in to all OAuth services in policy (slack, hubspot, google) |
| `claude login slack [policy.yaml]` | Sign in to Slack only |
| `claude login hubspot [policy.yaml]` | Sign in to HubSpot only (PKCE) |
| `claude login google [policy.yaml]` | Sign in to Google (all policy scopes) |
| `claude login figma [policy.yaml]` | Info: sync + Claude Connect (dynamic OAuth) |
| `claude login posthog [policy.yaml]` | Info: sync + Claude Connect (dynamic OAuth) |

Run `ally3p help` or `ally3p claude login` with no args for the full list.

### google-workspace-mcp-auth

| Command | Who | Purpose |
|---------|-----|---------|
| *(default)* / `headers` | Claude | Print Bearer header; auto sign-in + refresh if needed |
| `login` | User/IT | Force browser OAuth |
| `verify` | IT | Test MCP + redirect + headers |
| `logout` | User | Clear keychain user token |
| `install-credentials` | IT | Save OAuth client to Keychain (once per Mac) |
| `install-credentials --file` | IT | Write oauth JSON file instead |

### slack-mcp-auth

| Command | Who | Purpose |
|---------|-----|---------|
| *(default)* / `headers` | Claude | Print Bearer header from Keychain |
| `login` | User/IT | Browser OAuth → save `xoxp` token to Keychain |

### hubspot-mcp-auth

| Command | Who | Purpose |
|---------|-----|---------|
| *(default)* / `headers` | Claude | Print Bearer header from Keychain (auto-refresh) |
| `login` | User/IT | Browser OAuth + PKCE → save token to Keychain |

## Keychain

| Service | Account | Contents |
|---------|---------|----------|
| `google-workspace-mcp-auth` | `oauth-client-<service>` | OAuth client id + secret (from YAML sync) |
| `google-workspace-mcp-auth` | `google-user` | Shared Google user access/refresh token |
| `slack-mcp-auth` | `oauth-client` | Slack OAuth client id + secret (from YAML sync) |
| `slack-mcp-auth` | `user-token` | Slack user token (`xoxp-...`) |
| `hubspot-mcp-auth` | `oauth-client` | HubSpot MCP Auth App client id + secret |
| `hubspot-mcp-auth` | `user-token` | HubSpot access + refresh token (JSON) |
