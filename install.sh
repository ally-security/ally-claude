#!/bin/sh
# install.sh — install ally3p from GitHub Releases
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ally-security/ally-claude/main/install.sh | sudo bash
#
# Optional:
#   INSTALL_DIR=/custom/path sudo bash install.sh
#
# Downloads ally3p only. Helper binaries (google/slack/hubspot auth) are embedded
# and installed via: ally3p prereq --dir <bin-dir>

set -e

REPO="ally-security/ally-claude"
BIN_DIR="${INSTALL_DIR:-/usr/local/bin}"

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

echo "→ Fetching latest release..."
LATEST="$(fetch_latest_tag)"
echo "  Latest: $LATEST"

VERSION="${LATEST#v}"
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

ARCHIVE="ally3p_${VERSION}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"

echo "→ Downloading ally3p ($OS/$ARCH)..."
if ! curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE"; then
  echo "ERROR: could not download ${ARCHIVE}" >&2
  echo "       The latest release may predate ally3p — build from source:" >&2
  echo "         git clone https://github.com/${REPO}.git && cd ally-claude && make build" >&2
  echo "         sudo cp bin/ally3p ${BIN_DIR}/" >&2
  echo "       Or install a newer release when published: https://github.com/${REPO}/releases" >&2
  exit 1
fi

tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
EXTRACTED="$TMPDIR/ally3p"
if [ ! -f "$EXTRACTED" ]; then
  EXTRACTED="$(find "$TMPDIR" -type f -name ally3p | head -1)"
fi
if [ -z "$EXTRACTED" ] || [ ! -f "$EXTRACTED" ]; then
  echo "ERROR: could not find ally3p binary in $ARCHIVE" >&2
  exit 1
fi

if [ ! -w "$BIN_DIR" ]; then
  echo "ERROR: cannot write to $BIN_DIR" >&2
  echo "       run with sudo:" >&2
  echo "         curl -fsSL https://raw.githubusercontent.com/${REPO}/main/install.sh | sudo bash" >&2
  exit 1
fi

install -m 755 "$EXTRACTED" "$BIN_DIR/ally3p"
echo "  Installed $BIN_DIR/ally3p"

echo "→ Installing embedded helpers (google, slack, hubspot)..."
if ! "$BIN_DIR/ally3p" prereq --dir "$BIN_DIR"; then
  echo "WARN: prereq failed — run: sudo $BIN_DIR/ally3p prereq --dir $BIN_DIR" >&2
fi

echo ""
echo "✓ Installed to $BIN_DIR/ally3p"
echo ""
echo "Next steps:"
echo "  1. Sync your policy: ally3p claude sync my-policy.yaml"
echo "  2. ally3p claude login <policy.yaml>   # when ready"
echo "  3. Restart Claude Cowork 3P"
