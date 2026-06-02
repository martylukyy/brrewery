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
INSTALL_LOG="/var/log/brrewery-install.log"
ANSIBLE_DEST="/usr/share/brrewery/ansible"
SSL_DIR="/etc/ssl/brrewery"
NGINX_ETC="/etc/nginx"
REPO_URL="${BRREWERY_REPO_URL:-https://github.com/martylukyy/brrewery.git}"
REPO_REF="${BRREWERY_REPO_REF:-develop}"
CLONE_DIR="${BRREWERY_CLONE_DIR:-/etc/brrewery}"
NODE_INSTALL_DIR="/usr/local/lib/nodejs"

if [[ "${EUID:-}" -ne 0 ]]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

: >"$INSTALL_LOG"
chmod 0640 "$INSTALL_LOG"

run_with_spinner() {
  local message="$1"
  shift

  local -a spinner=(⣷ ⣯ ⣟ ⡿ ⢿ ⣻ ⣽ ⣾)
  local i=0
  local pid
  local exit_code

  {
    printf '\n=== %s ===\n' "$message"
    "$@"
  } >>"$INSTALL_LOG" 2>&1 &
  pid=$!

  while kill -0 "$pid" 2>/dev/null; do
    printf "\r%s %s" "$message" "${spinner[i++ % ${#spinner[@]}]}"
    sleep 0.08
  done

  wait "$pid"
  exit_code=$?

  if [[ "$exit_code" -eq 0 ]]; then
    printf "\r%s ✓\n" "$message"
    return 0
  fi

  printf "\r%s ✗\n" "$message"
  echo "$message failed. Last 40 log lines ($INSTALL_LOG):" >&2
  tail -n 40 "$INSTALL_LOG" >&2 || true
  return "$exit_code"
}

bootstrap_source() {
  if [[ -f "$ROOT/Makefile" && -d "$ROOT/ansible" && -d "$ROOT/contrib" ]]; then
    SOURCE_DIR="$ROOT"
    return
  fi

  echo "==> Fetching brrewery source from GitHub"
  run_with_spinner "Fetching brrewery source" bash -c "
      rm -rf \"$CLONE_DIR\" &&
        git clone --depth 1 --branch \"$REPO_REF\" \"$REPO_URL\" \"$CLONE_DIR\"
    "
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

bootstrap_pnpm() {
  echo "==> Bootstrapping pnpm"
  # Install exact project pnpm version to match packageManager pin.
  npm install -g pnpm@11.5.0

  local npm_prefix
  npm_prefix="$(npm prefix -g)"
  local npm_bin_dir="${npm_prefix}/bin"
  export PATH="${npm_bin_dir}:/usr/local/bin:${PATH}"

  # Ensure pnpm is resolvable regardless of npm global prefix configuration.
  if [[ -x "${npm_bin_dir}/pnpm" ]]; then
    ln -sf "${npm_bin_dir}/pnpm" /usr/local/bin/pnpm
  fi
  if [[ -x "${npm_bin_dir}/pnpx" ]]; then
    ln -sf "${npm_bin_dir}/pnpx" /usr/local/bin/pnpx
  fi

  if ! command -v pnpm >/dev/null 2>&1; then
    echo "pnpm install succeeded but binary is not on PATH" >&2
    exit 1
  fi
  pnpm --version >/dev/null
}

echo "==> Installing dependencies"
if command -v apt >/dev/null 2>&1; then
  run_with_spinner "Installing dependencies" bash -c '
    apt update -qq &&
      DEBIAN_FRONTEND=noninteractive apt install -y -qq \
        nginx git vnstat sudo ansible openssl make curl ca-certificates xz-utils python3 golang-go
  '
else
  echo "Unsupported distro: apt is required." >&2
  exit 1
fi

install_node_lts
bootstrap_pnpm
bootstrap_source

echo "==> Creating directories"
install -d -m 0750 "$LIB_DIR" "$LOG_DIR" "$WEB_ROOT" "$ANSIBLE_DEST" "$SSL_DIR"
install -d -m 0755 "$(dirname "$BINARY_DEST")"

echo "==> Building brrewery"
if [[ -f "$SOURCE_DIR/Makefile" ]]; then
  run_with_spinner "Building frontend" bash -c "
      cd \"$SOURCE_DIR/web\" && pnpm install && pnpm build
    "
  rm -rf "$SOURCE_DIR/internal/web/dist"
  cp -r "$SOURCE_DIR/web/dist" "$SOURCE_DIR/internal/web/"
  run_with_spinner "Building backend" bash -c "cd \"$SOURCE_DIR\" && make backend"
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
install -d -m 0755 "$NGINX_ETC/nginxconfig.io"
install -m 0644 "$SOURCE_DIR/contrib/nginx/nginx.conf" "$NGINX_ETC/nginx.conf"
install -m 0644 "$SOURCE_DIR/contrib/nginx/nginxconfig.io/general.conf" "$NGINX_ETC/nginxconfig.io/general.conf"
install -m 0644 "$SOURCE_DIR/contrib/nginx/nginxconfig.io/security.conf" "$NGINX_ETC/nginxconfig.io/security.conf"
install -m 0644 "$SOURCE_DIR/contrib/nginx/nginxconfig.io/proxy.conf" "$NGINX_ETC/nginxconfig.io/proxy.conf"
install -m 0644 "$SOURCE_DIR/contrib/nginx/nginxconfig.io/ssl.conf" "$NGINX_ETC/nginxconfig.io/ssl.conf"
install -m 0644 "$SOURCE_DIR/contrib/nginx/sites-available/brrewery.conf" "$NGINX_ETC/sites-available/brrewery.conf"
rm -f "$NGINX_ETC/sites-enabled/default" "$NGINX_ETC/sites-available/default"
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
