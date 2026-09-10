#!/bin/sh
# One-line install:
#   curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
set -eu

REPO="${PLAINCORD_REPO:-Codezilla-jpg/plaincord}"
BIN_DIR="${PLAINCORD_BIN_DIR:-${PREFIX:-$HOME/.local/bin}}"
TMP="${TMPDIR:-/tmp}/plaincord-install-$$"
API="https://api.github.com/repos/${REPO}/releases/latest"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing: $1" >&2
    exit 1
  }
}

need curl
need uname
need mkdir
need python3

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *)
    echo "unsupported arch: $arch" >&2
    exit 1
    ;;
esac
case "$os" in
  linux|darwin) ;;
  *)
    echo "unsupported os: $os" >&2
    exit 1
    ;;
esac

asset="dis_${os}_${arch}"
mkdir -p "$TMP" "$BIN_DIR"
trap 'rm -rf "$TMP"' EXIT

echo "Fetching release metadata"
json=$(curl --proto '=https' --tlsv1.2 -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: plaincord-install" "$API")
meta=$(printf '%s' "$json" | python3 -c "
import json, sys
want = sys.argv[1]
rel = json.load(sys.stdin)
for a in rel.get('assets', []):
    if a.get('name') == want:
        digest = a.get('digest') or ''
        url = a.get('browser_download_url') or ''
        print(digest)
        print(url)
        sys.exit(0)
sys.exit(1)
" "$asset") || {
  echo "asset $asset not in latest release" >&2
  exit 1
}
digest=$(printf '%s\n' "$meta" | sed -n '1p')
url=$(printf '%s\n' "$meta" | sed -n '2p')
case "$digest" in
  sha256:*) ;;
  *)
    echo "release has no sha256 digest" >&2
    exit 1
    ;;
esac
case "$url" in
  https://github.com/*|https://objects.githubusercontent.com/*|https://release-assets.githubusercontent.com/*|https://*.githubusercontent.com/*)
    ;;
  *)
    echo "blocked download host: $url" >&2
    exit 1
    ;;
esac

echo "Downloading ${url}"
curl --proto '=https' --tlsv1.2 --proto-redir '=https' -fL --retry 3 --max-filesize 41943040 -o "$TMP/dis" "$url"

expect=${digest#sha256:}
if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "$TMP/dis" | awk '{print $1}')
else
  got=$(shasum -a 256 "$TMP/dis" | awk '{print $1}')
fi
if [ "$got" != "$expect" ]; then
  echo "sha256 mismatch" >&2
  exit 1
fi

chmod 755 "$TMP/dis"
mv "$TMP/dis" "$BIN_DIR/dis"
cp -f "$BIN_DIR/dis" "$BIN_DIR/DiscordCli"
chmod 755 "$BIN_DIR/DiscordCli"
rm -f "$BIN_DIR/dc"

echo "Installed:"
echo "  $BIN_DIR/dis"
echo "  $BIN_DIR/DiscordCli"
case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *)
    echo "Add to PATH: export PATH=\"$BIN_DIR:\$PATH\""
    ;;
esac
echo "Update later with: dis update"
