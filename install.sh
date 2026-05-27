#!/bin/sh
# install.sh — install ally3p from GitHub Releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ally-security/ally-claude/main/install.sh | bash
#
# Optional:
#   INSTALL_DIR=$HOME/.local/bin bash install.sh
#
# Downloads ally3p only. Helper binaries (google/slack/hubspot auth) are embedded
# and installed via: ally3p prereq --dir <bin-dir>

set -e

REPO="ally-security/ally-claude"
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
  json="$(curl -fsSL \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${REPO}/releases/latest")" || {
    echo "ERROR: could not fetch releases for ${REPO}" >&2
    echo "       check https://github.com/${REPO}/releases" >&2
    exit 1
  }
  if [ -z "$json" ]; then
    echo "ERROR: empty response from GitHub releases API" >&2
    exit 1
  fi
  tag="$(printf '%s' "$json" | python3 -c 'import json,sys; print(json.load(sys.stdin)["tag_name"])')" || {
    echo "ERROR: could not parse release tag from GitHub API" >&2
    exit 1
  }
  printf '%s' "$tag"
}

download_archive() {
  name="$1"
  archive="${name}_${VERSION}_${OS}_${ARCH}.tar.gz"
  url="https://github.com/${REPO}/releases/download/${LATEST}/${archive}"
  echo "→ Downloading ${name} ($OS/$ARCH)..."
  if curl -fsSL "$url" -o "$TMPDIR/$archive"; then
    ARCHIVE="$archive"
    BINARY_NAME="$name"
    return 0
  fi
  return 1
}

echo "→ Fetching latest release..."
LATEST="$(fetch_latest_tag)"
echo "  Latest: $LATEST"

VERSION="${LATEST#v}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

ARCHIVE=""
BINARY_NAME=""
if ! download_archive "ally3p"; then
  if download_archive "claude-3p-helper"; then
    echo "  WARN: release uses legacy claude-3p-helper binary — install ally3p from a newer release when available" >&2
  else
    echo "ERROR: no release asset for ally3p_${VERSION}_${OS}_${ARCH}.tar.gz" >&2
    echo "       see https://github.com/${REPO}/releases/tag/${LATEST}" >&2
    exit 1
  fi
fi

tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
EXTRACTED="$TMPDIR/$BINARY_NAME"
if [ ! -f "$EXTRACTED" ]; then
  EXTRACTED="$(find "$TMPDIR" -type f \( -name ally3p -o -name claude-3p-helper \) | head -1)"
fi
if [ -z "$EXTRACTED" ] || [ ! -f "$EXTRACTED" ]; then
  echo "ERROR: could not find binary in $ARCHIVE" >&2
  exit 1
fi

install -m 755 "$EXTRACTED" "$BIN_DIR/ally3p"
echo "  Installed $BIN_DIR/ally3p"

if [ "$BINARY_NAME" = "ally3p" ]; then
  echo "→ Installing embedded helpers (google, slack, hubspot)..."
  "$BIN_DIR/ally3p" prereq --dir "$BIN_DIR"
else
  echo "  Skipping prereq (legacy release — build from source for full ally3p)" >&2
fi

echo ""
echo "✓ Installed to $BIN_DIR/ally3p"
echo ""
echo "Next steps:"
echo "  1. Sync your policy: ally3p claude sync my-policy.yaml"
echo "  2. Restart Claude Cowork 3P"
