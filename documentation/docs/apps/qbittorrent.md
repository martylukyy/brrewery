---
id: qbittorrent
title: qBittorrent
---

# qBittorrent

brrewery installs qBittorrent by compiling `qbittorrent-nox` from source against
pinned copies of its build dependencies. On first install, the playbook downloads
those source archives from their upstream mirrors (cached under
`/usr/share/brrewery/vendor/qbittorrent` on the server). All build dependencies
including OpenSSL are compiled from source.

## System password

Building qBittorrent installs system packages and services, which requires root.
If brrewery runs as a non-root user, the install prompts for your **system (sudo)
password** so it can escalate with `sudo`. The password is used only for that run
(passed to Ansible via a temporary become-password file) and is never saved or
logged. If brrewery already runs as root, any value you enter is ignored.

## Choosing a version

When you install or upgrade qBittorrent, brrewery asks two questions:

1. **qBittorrent version.** One choice per supported release line — `4.3`, `4.4`,
   `4.5`, `4.6`, `5.0`, `5.1`, `5.2`. Each line builds the latest stable patch
   release for that line; older patch releases are not offered.
2. **libtorrent version.** For `4.4` and newer you can choose libtorrent `1.2`
   (the default) or `2.0`. The `4.3` line always uses libtorrent `1.2`, so this
   step is skipped.

You do not choose the Qt, Boost, zlib, or OpenSSL versions: brrewery resolves them before
Ansible runs — newest Qt patch compatible with your line (`qt.min`, from
download.qt.io), newest zlib (from github.com/madler/zlib), newest Boost
from archives.boost.io for libtorrent `2.0` (manifest-capped `1.86` for
libtorrent `1.2`), and newest OpenSSL 3.x (from github.com/openssl/openssl).
All are compiled locally.
Install and upgrade need outbound HTTPS to qt.io and archives.boost.io.

Builds compile from source, so installs and upgrades can take a while on slower
machines.

## Custom libtorrent patches

The libtorrent step lets you optionally upload a `.patch` file (a unified diff,
up to 512 KiB). This is useful for tuning libtorrent's `settings_pack` defaults
that qBittorrent does not expose in its WebUI.

- If you leave it empty, brrewery applies its **default performance patch** for
  the selected libtorrent branch.
- If you upload a patch, it is used **for that build only** and must apply
  cleanly, otherwise the install/upgrade fails. Uploaded patches are never saved
  to disk and never appear in the job log.
- Advanced operators can instead place a persistent patch on the server at
  `/var/lib/brrewery/patches/qbittorrent/libtorrent-RC_1_2.patch` or
  `…/libtorrent-RC_2_0.patch`; it is used when no patch is uploaded.

qBittorrent source patches are supplied by brrewery (security backports only)
and cannot be uploaded.

## After install

qBittorrent runs as a per-user systemd service (`qbittorrent@<user>.service`)
with the WebUI on `127.0.0.1:8086`, reverse-proxied at `/qbittorrent/`.
The unit sets `LD_LIBRARY_PATH` for vendored Qt under `/usr/local/qt6/lib`
and registers those paths in `/etc/ld.so.conf.d/brrewery-vendored.conf`.
The WebUI itself is served at `/` on that port; nginx strips the `/qbittorrent/`
prefix when proxying (unlike autobrr, qBittorrent has no subpath `baseUrl`).
