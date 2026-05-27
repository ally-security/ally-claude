#!/bin/sh
# install.sh — install ally3p from GitHub Releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/allysecurity/ally-claude/main/install.sh | bash
#
# Optional:
#   INSTALL_DIR=$HOME/.local/bin bash install.sh
#
# Downloads ally3p only. Helper binaries (google/slack/hubspot auth) are embedded
# and installed via: ally3p prereq --dir <bin-dir>

set -e

REPO="allysecurity/ally-claude"
BIN_DIR="/usr/local/bin"

if [ -n "$INSTALL_DIR" ]; then
  BIN_DIR="$INSTALL_DIR"
fi

OS="$(uname -s)"
case "$OS" in
  Darwin) OS="darwin" ;;
  *)
    echo "Unsupported OS: $OS (releases are macOS-only today)" >&2
    exit 1
    ;;
esac

ARCH="$(uname -m)"
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  arm64 | aarch64) ARCH="arm64" ;;
  *)
    echo "Unsupported arch: $ARCH" >&2
    exit 1
    ;;
esac

fetch_latest_tag() {
  json="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")"
  if command -v python3 >/dev/null 2>&1; then
    python3 - <<'PY' "$json"
import json, sys
print(json.loads(sys.argv[1])["tag_name"])
PY
    return
  fi
  echo "$json" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
}

echo "→ Fetching latest release..."
LATEST="$(fetch_latest_tag)"
if [ -z "$LATEST" ]; then
  echo "Failed to fetch latest release tag from GitHub." >&2
  exit 1
fi
echo "  Latest: $LATEST"

VERSION="${LATEST#v}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

ARCHIVE="ally3p_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"
echo "→ Downloading ally3p ($OS/$ARCH)..."
if ! curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE"; then
  echo "Failed to download $URL" >&2
  exit 1
fi
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
ALLY3P="$TMPDIR/ally3p"
if [ ! -f "$ALLY3P" ]; then
  ALLY3P="$(find "$TMPDIR" -name ally3p -type f | head -1)"
fi
if [ -z "$ALLY3P" ] || [ ! -f "$ALLY3P" ]; then
  echo "Could not find ally3p in archive $ARCHIVE" >&2
  exit 1
fi
install -m 755 "$ALLY3P" "$BIN_DIR/ally3p"
echo "  Installed $BIN_DIR/ally3p"

echo "→ Installing embedded helpers (google, slack, hubspot)..."
"$BIN_DIR/ally3p" prereq --dir "$BIN_DIR"

echo ""
echo "✓ ally3p + helpers installed to $BIN_DIR"
echo ""
echo "Next steps:"
echo "  1. Sync your policy: ally3p claude sync my-policy.yaml"
echo "  2. Restart Claude Cowork 3P"
