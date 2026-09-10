#!/bin/sh
# One-line install:
#   curl --proto '=https' --tlsv1.2 -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
set -eu

REPO="${PLAINCORD_REPO:-Codezilla-jpg/plaincord}"
BIN_DIR="${PLAINCORD_BIN_DIR:-${PREFIX:-$HOME/.local/bin}}"
TMP="${TMPDIR:-/tmp}/plaincord-install-$$"
BASE="https://github.com/${REPO}/releases/latest/download"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing: $1" >&2
    exit 1
  }
}

need curl
need uname
need mkdir

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
url="${BASE}/${asset}"
sums_url="${BASE}/SHA256SUMS"

mkdir -p "$TMP" "$BIN_DIR"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${url}"
curl --proto '=https' --tlsv1.2 --proto-redir '=https' -fL --retry 3 --max-filesize 41943040 -o "$TMP/dis" "$url"
curl --proto '=https' --tlsv1.2 --proto-redir '=https' -fL --retry 3 --max-filesize 1048576 -o "$TMP/SHA256SUMS" "$sums_url"

expect=$(awk -v name="$asset" '$NF==name {print $1; exit}' "$TMP/SHA256SUMS")
if [ -z "$expect" ]; then
  echo "no checksum for $asset" >&2
  exit 1
fi
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
