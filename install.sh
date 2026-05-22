#!/bin/sh
# claude-3p-helper installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/ally-security/ally-claude/main/install.sh | sh
#   curl -fsSL https://raw.githubusercontent.com/ally-security/ally-claude/main/install.sh | sh -s -- --version v0.1.0
#   curl -fsSL https://raw.githubusercontent.com/ally-security/ally-claude/main/install.sh | sh -s -- --dir ~/.local/bin
#
# Flags:
#   --version <tag>   install a specific release (default: latest)
#   --dir <path>      destination directory (default: /usr/local/bin if writable,
#                     else $HOME/.local/bin)
#
# Env:
#   GITHUB_TOKEN / GH_TOKEN   personal access token, required for private repos.
#                             Easiest way to get one with the right scopes:
#                                 export GITHUB_TOKEN=$(gh auth token)

set -eu

REPO="ally-security/ally-claude"
BINARY="claude-3p-helper"
VERSION=""
DEST=""

while [ $# -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --dir)     DEST="$2"; shift 2 ;;
    -h|--help)
      sed -n '/^# Usage:/,/^# *$/p' "$0" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *)
      echo "unknown flag: $1" >&2
      exit 2
      ;;
  esac
done

# ── platform detection ────────────────────────────────────────────────
uname_s=$(uname -s)
uname_m=$(uname -m)

case "$uname_s" in
  Darwin) OS=darwin ;;
  Linux)  OS=linux ;;
  MINGW*|MSYS*|CYGWIN*)
    echo "Windows is not supported by this script. Download the .zip from:" >&2
    echo "  https://github.com/$REPO/releases/latest" >&2
    exit 1
    ;;
  *)
    echo "unsupported OS: $uname_s" >&2
    exit 1
    ;;
esac

case "$uname_m" in
  x86_64|amd64) ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *)
    echo "unsupported architecture: $uname_m" >&2
    exit 1
    ;;
esac

# ── prerequisites ─────────────────────────────────────────────────────
need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required tool: $1" >&2
    exit 1
  }
}
need curl
need tar

# sha256 verification is best-effort; pick whichever tool the host has.
SHA_CMD=""
if command -v sha256sum >/dev/null 2>&1; then
  SHA_CMD="sha256sum"
elif command -v shasum >/dev/null 2>&1; then
  SHA_CMD="shasum -a 256"
fi

# ── auth + fetch helpers ──────────────────────────────────────────────
TOKEN="${GITHUB_TOKEN:-${GH_TOKEN:-}}"

api_get() {
  # GET an api.github.com JSON endpoint and print the body.
  url="$1"
  if [ -n "$TOKEN" ]; then
    curl -fsSL -H "Authorization: token $TOKEN" -H "Accept: application/vnd.github+json" "$url"
  else
    curl -fsSL -H "Accept: application/vnd.github+json" "$url"
  fi
}

api_download() {
  # Download an asset by its api.github.com URL (works for private repos
  # when $TOKEN is set; the github.com/.../releases/download/... URLs
  # 404 anonymously on private repos).
  url="$1"; out="$2"
  if [ -n "$TOKEN" ]; then
    curl -fsSL -H "Authorization: token $TOKEN" -H "Accept: application/octet-stream" -o "$out" "$url"
  else
    curl -fsSL -H "Accept: application/octet-stream" -o "$out" "$url"
  fi
}

# Parse an asset's API url out of the release JSON by asset name.
# Relies on each asset object containing "url" before "name" (the
# default order in the GitHub releases API response).
asset_url() {
  name="$1"; json="$2"
  echo "$json" \
    | tr -d '\n' \
    | sed 's/{/\n{/g' \
    | grep "\"name\":[ ]*\"$name\"" \
    | sed -n 's/.*"url":[ ]*"\([^"]*\)".*/\1/p' \
    | head -n1
}

# ── resolve release ───────────────────────────────────────────────────
if [ -z "$VERSION" ]; then
  echo "→ resolving latest release for $REPO..."
  if ! release_json=$(api_get "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null); then
    echo "could not query releases (404 / 403). If $REPO is private, set" >&2
    echo "  export GITHUB_TOKEN=\$(gh auth token)" >&2
    echo "and re-run." >&2
    exit 1
  fi
else
  echo "→ fetching release metadata for $VERSION..."
  if ! release_json=$(api_get "https://api.github.com/repos/$REPO/releases/tags/$VERSION" 2>/dev/null); then
    echo "could not find release $VERSION" >&2
    exit 1
  fi
fi

VERSION=$(echo "$release_json" | sed -n 's/.*"tag_name":[ ]*"\([^"]*\)".*/\1/p' | head -n1)
if [ -z "$VERSION" ]; then
  echo "could not parse tag_name from release JSON" >&2
  exit 1
fi
VERSION_NUM=$(echo "$VERSION" | sed 's/^v//')
ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"

ARCHIVE_URL=$(asset_url "$ARCHIVE" "$release_json")
SUMS_URL=$(asset_url "checksums.txt" "$release_json")

if [ -z "$ARCHIVE_URL" ]; then
  echo "release $VERSION has no asset named $ARCHIVE" >&2
  exit 1
fi

# ── destination ───────────────────────────────────────────────────────
if [ -z "$DEST" ]; then
  if [ -w /usr/local/bin ] 2>/dev/null; then
    DEST=/usr/local/bin
  else
    DEST="$HOME/.local/bin"
  fi
fi
mkdir -p "$DEST"

# ── download + verify ─────────────────────────────────────────────────
TMP=$(mktemp -d 2>/dev/null || mktemp -d -t claude-3p-helper)
trap 'rm -rf "$TMP"' EXIT

echo "→ downloading $ARCHIVE"
api_download "$ARCHIVE_URL" "$TMP/$ARCHIVE"

if [ -n "$SHA_CMD" ] && [ -n "$SUMS_URL" ]; then
  echo "→ verifying sha256"
  api_download "$SUMS_URL" "$TMP/checksums.txt"
  expected=$(grep "  $ARCHIVE$" "$TMP/checksums.txt" | awk '{print $1}')
  if [ -z "$expected" ]; then
    echo "could not find $ARCHIVE in checksums.txt" >&2
    exit 1
  fi
  actual=$(cd "$TMP" && $SHA_CMD "$ARCHIVE" | awk '{print $1}')
  if [ "$expected" != "$actual" ]; then
    echo "sha256 mismatch:" >&2
    echo "  expected: $expected" >&2
    echo "  actual:   $actual" >&2
    exit 1
  fi
else
  echo "→ skipping checksum verification (no sha256sum/shasum or no checksums asset)"
fi

# ── extract + install ─────────────────────────────────────────────────
echo "→ extracting"
tar -xzf "$TMP/$ARCHIVE" -C "$TMP"
chmod +x "$TMP/$BINARY"

echo "→ installing to $DEST/$BINARY"
if mv "$TMP/$BINARY" "$DEST/$BINARY" 2>/dev/null; then
  :
else
  echo "  (sudo required to write to $DEST)"
  sudo mv "$TMP/$BINARY" "$DEST/$BINARY"
fi

# ── PATH check ────────────────────────────────────────────────────────
case ":$PATH:" in
  *":$DEST:"*) ;;
  *)
    echo
    echo "warning: $DEST is not on your \$PATH."
    echo "add this to your shell profile:"
    echo "  export PATH=\"$DEST:\$PATH\""
    ;;
esac

echo
echo "✓ installed $BINARY $VERSION → $DEST/$BINARY"
"$DEST/$BINARY" version 2>/dev/null || true
