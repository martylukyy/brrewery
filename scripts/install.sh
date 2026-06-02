#!/usr/bin/env bash
# brrewery host installer — idempotent bootstrap for production paths.
set -euo pipefail
export COREPACK_ENABLE_DOWNLOAD_PROMPT=0

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="$ROOT"
BINARY_DEST="/usr/local/bin/brrewery"
WEB_ROOT="/var/www/brrewery"
LIB_DIR="/var/lib/brrewery"
LOG_DIR="/var/log/brrewery"
ANSIBLE_DEST="/usr/share/brrewery/ansible"
SSL_DIR="/etc/ssl/brrewery"
NGINX_ETC="/etc/nginx"
REPO_URL="${BRREWERY_REPO_URL:-https://github.com/martylukyy/brrewery.git}"
REPO_REF="${BRREWERY_REPO_REF:-develop}"
CLONE_DIR="${BRREWERY_CLONE_DIR:-/tmp/brrewery-src}"
NODE_INSTALL_DIR="/usr/local/lib/nodejs"

if [[ "${EUID:-}" -ne 0 ]]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

bootstrap_source() {
  if [[ -f "$ROOT/Makefile" && -d "$ROOT/ansible" && -d "$ROOT/contrib" ]]; then
    SOURCE_DIR="$ROOT"
    return
  fi

  echo "==> Fetching brrewery source from GitHub"
  rm -rf "$CLONE_DIR"
  git clone --depth 1 --branch "$REPO_REF" "$REPO_URL" "$CLONE_DIR"
  SOURCE_DIR="$CLONE_DIR"
}

install_node_lts() {
  local arch
  case "$(uname -m)" in
  x86_64) arch="x64" ;;
  aarch64) arch="arm64" ;;
  *)
    echo "Unsupported architecture for Node.js LTS install: $(uname -m)" >&2
    exit 1
    ;;
  esac

  echo "==> Installing latest Node.js LTS from official mirror"
  local node_version
  node_version="$(
    curl -fsSL https://nodejs.org/dist/index.json | python3 -c '
import json, sys
for release in json.load(sys.stdin):
    if release.get("lts"):
        print(release["version"])
        break
'
  )"
  if [[ -z "$node_version" ]]; then
    echo "Failed to resolve latest Node.js LTS version" >&2
    exit 1
  fi

  local tarball="node-${node_version}-linux-${arch}.tar.xz"
  local url="https://nodejs.org/dist/${node_version}/${tarball}"
  local tmp_tar="/tmp/${tarball}"

  curl -fsSL "$url" -o "$tmp_tar"
  install -d -m 0755 "$NODE_INSTALL_DIR"
  rm -rf "${NODE_INSTALL_DIR:?}/node-${node_version}-linux-${arch}"
  tar -xJf "$tmp_tar" -C "$NODE_INSTALL_DIR"
  rm -f "$tmp_tar"

  ln -sf "${NODE_INSTALL_DIR}/node-${node_version}-linux-${arch}/bin/node" /usr/local/bin/node
  ln -sf "${NODE_INSTALL_DIR}/node-${node_version}-linux-${arch}/bin/npm" /usr/local/bin/npm
  ln -sf "${NODE_INSTALL_DIR}/node-${node_version}-linux-${arch}/bin/npx" /usr/local/bin/npx
  ln -sf "${NODE_INSTALL_DIR}/node-${node_version}-linux-${arch}/bin/corepack" /usr/local/bin/corepack
}

echo "==> Installing dependencies"
if command -v apt-get >/dev/null 2>&1; then
  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
    nginx git vnstat sudo ansible openssl make curl ca-certificates xz-utils python3 golang-go
else
  echo "Unsupported distro: apt-get is required." >&2
  exit 1
fi

install_node_lts

echo "==> Bootstrapping pnpm"
# Install exact project pnpm version globally to avoid interactive Corepack prompts
# when packageManager pinning is enforced during `pnpm install`/`pnpm build`.
npm install -g pnpm@latest

bootstrap_source

echo "==> Creating directories"
install -d -m 0750 "$LIB_DIR" "$LOG_DIR" "$WEB_ROOT" "$ANSIBLE_DEST" "$SSL_DIR"
install -d -m 0755 "$(dirname "$BINARY_DEST")"

echo "==> Building brrewery"
if [[ -f "$SOURCE_DIR/Makefile" ]]; then
  (cd "$SOURCE_DIR" && make build)
else
  echo "Missing Makefile in $SOURCE_DIR" >&2
  exit 1
fi

echo "==> Installing binary and ansible playbooks"
install -m 0755 "$SOURCE_DIR/brrewery" "$BINARY_DEST"
rm -rf "${ANSIBLE_DEST:?}"/*
cp -a "$SOURCE_DIR/ansible/." "$ANSIBLE_DEST/"

echo "==> Deploying web assets"
rm -rf "${WEB_ROOT:?}"/*
if [[ -d "$SOURCE_DIR/internal/web/dist" ]]; then
  cp -a "$SOURCE_DIR/internal/web/dist/." "$WEB_ROOT/"
elif [[ -d "$SOURCE_DIR/web/dist" ]]; then
  cp -a "$SOURCE_DIR/web/dist/." "$WEB_ROOT/"
fi

echo "==> TLS certificates"
if [[ ! -f "$SSL_DIR/fullchain.pem" ]]; then
  openssl req -x509 -nodes -days 3650 -newkey rsa:2048 \
    -keyout "$SSL_DIR/privkey.pem" \
    -out "$SSL_DIR/fullchain.pem" \
    -subj "/CN=brrewery.local"
  chmod 0640 "$SSL_DIR/privkey.pem"
  chmod 0644 "$SSL_DIR/fullchain.pem"
fi

echo "==> nginx configuration"
install -d -m 0755 "$NGINX_ETC/sites-available" "$NGINX_ETC/sites-enabled"
cp -a "$SOURCE_DIR/contrib/nginx/." "$NGINX_ETC/"
ln -sf ../sites-available/brrewery.conf "$NGINX_ETC/sites-enabled/brrewery.conf"
nginx -t
systemctl enable nginx
systemctl reload nginx || systemctl start nginx

echo "==> systemd unit"
install -m 0644 "$SOURCE_DIR/contrib/systemd/brrewery.service" /etc/systemd/system/brrewery.service
systemctl daemon-reload
systemctl enable brrewery
systemctl restart brrewery

if ! "$BINARY_DEST" create-admin 2>/dev/null; then
  echo "==> Create admin user (interactive)"
  "$BINARY_DEST" create-admin
fi

echo "==> brrewery installed"
echo "    Dashboard: https://127.0.0.1/ (self-signed TLS)"
echo "    Backend:   127.0.0.1:8080 (proxied at /api/)"
