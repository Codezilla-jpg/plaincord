#!/bin/sh
# One-line install:
#   curl -fsSL https://raw.githubusercontent.com/Codezilla-jpg/plaincord/main/install.sh | sh
set -eu

REPO="${PLAINCORD_REPO:-Codezilla-jpg/plaincord}"
BIN_DIR="${PLAINCORD_BIN_DIR:-${PREFIX:-$HOME/.local/bin}}"
TMP="${TMPDIR:-/tmp}/plaincord-install-$$"

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
url="https://github.com/${REPO}/releases/latest/download/${asset}"

mkdir -p "$TMP" "$BIN_DIR"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${url}"
if ! curl -fL --retry 3 -o "$TMP/dis" "$url"; then
  echo "Release binary not found. Build from source:" >&2
  echo "  git clone https://github.com/${REPO}.git && cd plaincord && go build -o dis ./cmd/dis" >&2
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
