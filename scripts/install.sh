#!/usr/bin/env bash
# tokless installer for macOS / Linux.
# Download this script from a trusted source, inspect it if needed, then run:
#   bash scripts/install.sh

set -euo pipefail

OWNER="wallentx"
REPO="tokless"
DEST="${HOME}/.local/bin"

ok()  { printf '\033[32m✔\033[0m %s\n' "$*"; }
err() { printf '\033[31m✖\033[0m %s\n' "$*" >&2; }

# OS + arch -> asset name.
case "$(uname -s)" in
  Linux*)  os="linux" ;;
  Darwin*) os="darwin" ;;
  *) err "Unsupported OS. Windows: use install.ps1 (irm … | iex)."; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  arch="x64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) err "Unsupported architecture: $(uname -m)."; exit 1 ;;
esac
asset="tokless-${os}-${arch}"
url="https://github.com/${OWNER}/${REPO}/releases/latest/download/${asset}"
sum_url="https://github.com/${OWNER}/${REPO}/releases/latest/download/SHA256SUMS"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  return 127
}

# Download + install.
mkdir -p "$DEST"
tmp="$(mktemp)"
sums="$(mktemp)"
trap 'rm -f "$tmp" "$sums"' EXIT
printf '\033[36m↓\033[0m Downloading %s…\n' "$asset"
if ! curl -fSL --progress-bar -o "$tmp" "$url" || [ ! -s "$tmp" ]; then
  err "Download failed ($asset). See https://github.com/${OWNER}/${REPO}/releases"
  exit 1
fi
printf '\033[36m↓\033[0m Downloading checksums…\n'
if ! curl -fSL --progress-bar -o "$sums" "$sum_url" || [ ! -s "$sums" ]; then
  err "Checksum download failed. Refusing to install unverified asset."
  exit 1
fi
expected="$(awk -v a="$asset" '$2 == a { print $1; found=1 } END { if (!found) exit 1 }' "$sums")" || {
  err "Checksum file does not list ${asset}. Refusing to install."
  exit 1
}
actual="$(sha256_file "$tmp")" || {
  err "sha256sum or shasum is required to verify ${asset}."
  exit 1
}
if [ "$actual" != "$expected" ]; then
  err "Checksum mismatch for ${asset}. Refusing to install."
  exit 1
fi
chmod +x "$tmp"
install -m 0755 "$tmp" "${DEST}/tokless"
ok "installed tokless $("${DEST}/tokless" --version 2>/dev/null) → ${DEST}/tokless"

# Ensure ~/.local/bin is on PATH for new shells.
case ":${PATH}:" in
  *":${DEST}:"*) : ;;
  *)
    case "$(basename "${SHELL:-bash}")" in
      zsh) rc="${ZDOTDIR:-$HOME}/.zshrc" ;;
      *)   rc="$HOME/.bashrc" ;;
    esac
    line="export PATH=\"${DEST}:\$PATH\""
    grep -qF "$DEST" "$rc" 2>/dev/null || printf '\n# tokless\n%s\n' "$line" >> "$rc"
    ok "Added ${DEST} to PATH in ${rc}."
    ;;
esac

# Run now, reconnecting the keyboard via /dev/tty so the picker works under a pipe.
if [ -r /dev/tty ]; then
  printf '\n'
  "${DEST}/tokless" </dev/tty || true
else
  ok "Run: tokless"
fi
