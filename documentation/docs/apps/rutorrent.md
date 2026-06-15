---
id: rutorrent
title: ruTorrent
---

# ruTorrent

ruTorrent is the web front-end for [rTorrent](./rtorrent.md). brrewery installs
the latest stable [Novik/ruTorrent](https://github.com/Novik/ruTorrent) release
(a PHP application — no compilation) and wires it to rTorrent over the local SCGI
socket.

ruTorrent **depends on rTorrent**: install rTorrent first. The install verifies
rTorrent's SCGI socket is present before continuing.

## System password

Installing ruTorrent sets up system packages, a PHP-FPM pool and an nginx site,
which requires root, so the install prompts for your **system (sudo) password**.
That same password also becomes the ruTorrent login (see below). It is used only
for the run and is never saved or logged.

## Versions

There is no version choice: brrewery resolves the latest stable release from
GitHub (`/releases/latest`) and refreshes the source on upgrade. Install and
upgrade need outbound HTTPS to `api.github.com` and `github.com`.

## After install

ruTorrent is served at `/rutorrent/`. It runs under a dedicated PHP-FPM pool that
executes as your brrewery user so it can reach rTorrent's SCGI socket; nginx
serves the static assets and proxies PHP to that pool.

ruTorrent has no authentication of its own, so brrewery protects it with **HTTP
Basic auth**: log in with your brrewery admin username and your system password.

The PHP talks to rTorrent directly through its unix SCGI socket
(`$scgi_port = 0; $scgi_host = "unix://…/rpc.socket"`), so rTorrent must be
running for the UI to show torrents.

## Upgrade

Upgrade re-fetches the latest ruTorrent source. Your per-user UI settings under
`share/` are preserved, and the SCGI connection is reconfigured.

## Remove

Remove deletes the ruTorrent source (`/srv/rutorrent`), its PHP-FPM pool, the
nginx location and the basic-auth file, and reloads nginx and PHP-FPM. rTorrent
is left untouched.
