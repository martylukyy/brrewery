---
id: transmission
title: Transmission
---

# Transmission

brrewery installs Transmission by compiling `transmission-daemon` from the
upstream source tarball (`transmission-<version>.tar.xz`), always building the
latest stable release published on GitHub. The release tarball bundles all
third-party dependencies, so the build needs no git submodules — only a handful
of system `-dev` packages (curl, libevent, libsystemd, etc.) installed from apt.

## System password

Building Transmission installs system packages and services, which requires
root. If brrewery runs as a non-root user, the install prompts for your
**system (sudo) password** so it can escalate with `sudo`. That same password
also becomes your Transmission RPC password (see below). It is used only for the
run (passed to Ansible via a temporary become-password file) and is never saved
or logged. If brrewery already runs as root, any value you enter is ignored.

## Versions

There is no version choice: brrewery resolves the latest stable release from
GitHub (`/releases/latest`, which excludes betas) and rebuilds only when a newer
release than the one already installed is published. Install and upgrade need
outbound HTTPS to `api.github.com` and `github.com`.

Builds compile from source, so installs and upgrades can take a while on slower
machines.

## After install

Transmission runs as a per-user systemd service (`transmission@<user>.service`)
with the RPC/WebUI on `127.0.0.1:9091`, reverse-proxied at `/transmission/`. The
daemon already serves its WebUI under `/transmission/` (its `rpc-url` default),
so nginx does **not** strip the path prefix (unlike qBittorrent).

RPC authentication is enabled. Your username is your brrewery admin user and
your password is your system (sudo) password. Transmission stores the password
hashed — it hashes the plaintext on first start, after which the value in
`settings.json` begins with `{`. You can change it later in the WebUI under
**Edit Preferences → Remote**.

Downloads are saved to `~/Downloads/transmission` by default; change the
download location in the WebUI to point at your media/data path.

## Upgrades

Upgrading rebuilds Transmission only if a newer release exists, then restarts
the service. Your `settings.json` (including your password and download
location) is preserved across upgrades.
