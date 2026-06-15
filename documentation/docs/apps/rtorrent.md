---
id: rtorrent
title: rTorrent
---

# rTorrent

brrewery installs rTorrent by compiling it — together with its matching
[rakshasa libtorrent](https://github.com/rakshasa/libtorrent) — from source.
Unlike most apps, rTorrent offers a **version picker**: you choose which release
line to build at install time.

## System password

Building rTorrent installs system packages and a per-user service, which
requires root. If brrewery runs as a non-root user, the install prompts for your
**system (sudo) password** so it can escalate with `sudo`. It is used only for
the run (passed to Ansible via a temporary become-password file) and is never
saved or logged. rTorrent itself has no RPC password — its SCGI socket is local
and unauthenticated (see [ruTorrent](./rutorrent.md) for the web UI's auth).

## Versions

Pick one of these lines at install time:

| Line     | Builds                | libtorrent |
|----------|-----------------------|------------|
| `0.16.x` | newest `0.16` release | matching `0.16.x` |
| `0.15.x` | newest `0.15` release | matching `0.15.x` |
| `0.10.0` | `0.10.0`              | `0.14.0` |
| `0.9.8`  | `0.9.8`               | `0.13.8` |
| `0.9.6`  | `0.9.6`               | `0.13.6` |

For the `.x` lines brrewery resolves the newest patch release in the series from
GitHub at install time and pairs it with the libtorrent version upstream bundles
in that release. The pinned lines build an exact version. Install and upgrade
need outbound HTTPS to `api.github.com` and `github.com`.

The two legacy lines (`0.9.8`, `0.9.6`) are 2019/2015-era code; brrewery applies
the compatibility fixes they need to build with a current compiler and OpenSSL 3
(an OpenSSL-3 port for libtorrent 0.13.6, a `std::tr1`→C++11 port and a missing
`<locale>` include for rTorrent 0.9.6). Builds compile from source, so installs
and upgrades can take a while.

## After install

rTorrent runs as a per-user systemd service (`rtorrent@<user>.service`). Modern
lines (≥ 0.9.7) run headless in rTorrent's native daemon mode; 0.9.6, which has
no daemon mode, runs under `dtach` (a tiny PTY wrapper) instead. Either way the
dashboard's start/stop toggle controls the service.

It opens a local **SCGI socket** at `~/.local/share/rtorrent/rpc.socket` for a
web UI to drive it, stores torrents/session under `~/.local/share/rtorrent`, and
downloads to `~/Downloads/rtorrent`. rTorrent has no web UI of its own — install
[ruTorrent](./rutorrent.md) for one.

## Upgrade

Upgrade rebuilds the selected line (resolving a newer patch for the `.x` lines)
and restarts the service. Your `~/.rtorrent.rc` is left untouched.

## Remove

Remove stops and disables the service and deletes `~/.rtorrent.rc` and the
session directory. When no other rTorrent instance remains, the rtorrent binary,
the libtorrent libraries and the systemd unit are removed too. Your
`~/Downloads/rtorrent` data is kept.
