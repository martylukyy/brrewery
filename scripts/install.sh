#!/usr/bin/env bash
# brrewery host installer — idempotent bootstrap for production paths.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BINARY_DEST="/usr/local/bin/brrewery"
WEB_ROOT="/var/www/brrewery"
LIB_DIR="/var/lib/brrewery"
LOG_DIR="/var/log/brrewery"
ANSIBLE_DEST="/usr/share/brrewery/ansible"
SSL_DIR="/etc/ssl/brrewery"
NGINX_ETC="/etc/nginx"

if [[ "${EUID:-}" -ne 0 ]]; then
  echo "Run as root: sudo $0" >&2
  exit 1
fi

echo "==> Installing dependencies"
if command -v apt-get >/dev/null 2>&1; then
  apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq nginx ansible openssl
elif command -v pacman >/dev/null 2>&1; then
  pacman -Sy --noconfirm nginx ansible openssl
else
  echo "Unsupported distro: install nginx, ansible, and openssl manually." >&2
  exit 1
fi

echo "==> Creating directories"
install -d -m 0750 "$LIB_DIR" "$LOG_DIR" "$WEB_ROOT" "$ANSIBLE_DEST" "$SSL_DIR"
install -d -m 0755 "$(dirname "$BINARY_DEST")"

echo "==> Building brrewery"
if [[ -f "$ROOT/Makefile" ]]; then
  (cd "$ROOT" && make build)
else
  echo "Missing Makefile in $ROOT" >&2
  exit 1
fi

echo "==> Installing binary and ansible playbooks"
install -m 0755 "$ROOT/brrewery" "$BINARY_DEST"
rm -rf "${ANSIBLE_DEST:?}"/*
cp -a "$ROOT/ansible/." "$ANSIBLE_DEST/"

echo "==> Deploying web assets"
rm -rf "${WEB_ROOT:?}"/*
if [[ -d "$ROOT/internal/web/dist" ]]; then
  cp -a "$ROOT/internal/web/dist/." "$WEB_ROOT/"
elif [[ -d "$ROOT/web/dist" ]]; then
  cp -a "$ROOT/web/dist/." "$WEB_ROOT/"
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
cp -a "$ROOT/contrib/nginx/." "$NGINX_ETC/"
ln -sf ../sites-available/brrewery.conf "$NGINX_ETC/sites-enabled/brrewery.conf"
nginx -t
systemctl enable nginx
systemctl reload nginx || systemctl start nginx

echo "==> systemd unit"
install -m 0644 "$ROOT/contrib/systemd/brrewery.service" /etc/systemd/system/brrewery.service
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
