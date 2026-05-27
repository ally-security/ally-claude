#!/usr/bin/env python3
"""Manual Slack MCP OAuth test + MCP initialize/tools/list verification."""

from __future__ import annotations

import hashlib
import base64
import json
import os
import secrets
import sys
import urllib.parse
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread

CLIENT_ID = os.environ.get("SLACK_CLIENT_ID", "")
CLIENT_SECRET = os.environ.get("SLACK_CLIENT_SECRET", "")
REDIRECT_URI = os.environ.get("SLACK_REDIRECT_URI", "http://localhost:3118/callback")
PORT = int(os.environ.get("SLACK_CALLBACK_PORT", "3118"))
MCP_URL = "https://mcp.slack.com/mcp"

SCOPES = " ".join(
    [
        "chat:write",
        "search:read.public",
        "search:read.private",
        "search:read.im",
        "search:read.mpim",
        "search:read.files",
        "search:read.users",
        "channels:history",
        "groups:history",
        "im:history",
        "mpim:history",
        "users:read",
        "users:read.email",
    ]
)


def pkce_pair() -> tuple[str, str]:
    verifier = base64.urlsafe_b64encode(secrets.token_bytes(32)).rstrip(b"=").decode()
    challenge = base64.urlsafe_b64encode(
        hashlib.sha256(verifier.encode()).digest()
    ).rstrip(b"=").decode()
    return verifier, challenge


class CallbackHandler(BaseHTTPRequestHandler):
    code: str | None = None
    error: str | None = None

    def do_GET(self):  # noqa: N802
        parsed = urllib.parse.urlparse(self.path)
        if parsed.path != "/callback":
            self.send_response(404)
            self.end_headers()
            return
        qs = urllib.parse.parse_qs(parsed.query)
        if "error" in qs:
            CallbackHandler.error = qs["error"][0]
        if "code" in qs:
            CallbackHandler.code = qs["code"][0]
        self.send_response(200)
        self.send_header("Content-Type", "text/html")
        self.end_headers()
        self.wfile.write(b"<html><body><h1>Slack OAuth complete</h1><p>You can close this tab.</p></body></html>")

    def log_message(self, fmt, *args):  # noqa: A003
        return


def wait_for_callback(timeout: float = 300.0) -> str:
    server = HTTPServer(("127.0.0.1", PORT), CallbackHandler)
    thread = Thread(target=server.handle_request, daemon=True)
    thread.start()
    thread.join(timeout)
    server.server_close()
    if CallbackHandler.error:
        raise RuntimeError(f"Slack OAuth error: {CallbackHandler.error}")
    if not CallbackHandler.code:
        raise RuntimeError("Timed out waiting for OAuth callback")
    return CallbackHandler.code


def build_auth_url(code_challenge: str) -> str:
    params = {
        "client_id": CLIENT_ID,
        "redirect_uri": REDIRECT_URI,
        "response_type": "code",
        "scope": SCOPES,
        "code_challenge": code_challenge,
        "code_challenge_method": "S256",
    }
    return "https://slack.com/oauth/v2_user/authorize?" + urllib.parse.urlencode(params)


def exchange_code(code: str, verifier: str) -> dict:
    form = urllib.parse.urlencode(
        {
            "client_id": CLIENT_ID,
            "client_secret": CLIENT_SECRET,
            "code": code,
            "redirect_uri": REDIRECT_URI,
            "code_verifier": verifier,
        }
    ).encode()
    req = urllib.request.Request(
        "https://slack.com/api/oauth.v2.user.access",
        data=form,
        method="POST",
        headers={"Content-Type": "application/x-www-form-urlencoded"},
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode())


def mcp_request(token: str, method: str, params: dict | None = None, req_id: int = 1) -> dict:
    body = {"jsonrpc": "2.0", "method": method, "id": req_id}
    if params is not None:
        body["params"] = params
    data = json.dumps(body).encode()
    req = urllib.request.Request(
        MCP_URL,
        data=data,
        method="POST",
        headers={
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json",
            "Accept": "application/json, text/event-stream",
        },
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        raw = resp.read().decode()
    # streamable HTTP may return SSE; take last JSON line if needed
    if raw.startswith("event:"):
        for line in raw.splitlines():
            if line.startswith("data:"):
                raw = line[5:].strip()
    return json.loads(raw)


def main() -> int:
    if not CLIENT_ID or not CLIENT_SECRET:
        print("Set SLACK_CLIENT_ID and SLACK_CLIENT_SECRET", file=sys.stderr)
        return 2

    print(f"== Slack MCP OAuth test ==")
    print(f"redirect_uri: {REDIRECT_URI}")
    print(f"callback port: {PORT}")

    verifier, challenge = pkce_pair()
    auth_url = build_auth_url(challenge)
    print(f"\nOpen this URL in your browser and click Allow:\n{auth_url}\n")
    os.system(f'open "{auth_url}"')

    print("Waiting for callback...")
    code = wait_for_callback()
    print("Got authorization code, exchanging...")

    token_resp = exchange_code(code, verifier)
    print("Token response keys:", sorted(token_resp.keys()))
    if not token_resp.get("ok"):
        print(json.dumps(token_resp, indent=2))
        return 1

    token = token_resp.get("access_token")
    if not token and isinstance(token_resp.get("authed_user"), dict):
        token = token_resp["authed_user"].get("access_token")
    if not token:
        print("No access_token in response:", json.dumps(token_resp, indent=2))
        return 1

    print(f"Got user token prefix: {token[:12]}... token_type={token_resp.get('token_type')}")

    token_path = os.path.expanduser("~/.config/claude-3p/slack-mcp-token.json")
    os.makedirs(os.path.dirname(token_path), exist_ok=True)
    with open(token_path, "w") as f:
        json.dump({"access_token": token, "team": token_resp.get("team")}, f)
    os.chmod(token_path, 0o600)
    print(f"Saved token to {token_path}")

    for proto in ("2025-06-18", "2024-11-05"):
        try:
            init = mcp_request(
                token,
                "initialize",
                {
                    "protocolVersion": proto,
                    "capabilities": {},
                    "clientInfo": {"name": "slack-mcp-test", "version": "1.0"},
                },
            )
            print(f"\n== MCP initialize (protocol {proto}) ==")
            print(json.dumps(init, indent=2)[:1200])
            break
        except urllib.error.HTTPError as e:
            body = e.read().decode(errors="replace")
            print(f"\ninitialize failed HTTP {e.code} (protocol {proto}): {body[:500]}")
            init = None
    else:
        return 1

    tools = mcp_request(token, "tools/list", {}, req_id=2)
    print("\n== MCP tools/list ==")
    tool_names = [t.get("name") for t in tools.get("result", {}).get("tools", [])]
    print(f"tools ({len(tool_names)}):", ", ".join(tool_names[:15]))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
