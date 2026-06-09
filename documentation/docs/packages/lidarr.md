---
id: lidarr
title: Lidarr
---

# Lidarr

brrewery installs [Lidarr](https://wiki.servarr.com/lidarr) from the official
self-contained .NET tarball published by the Servarr update service (not GitHub
releases). The linux/amd64 build is extracted to `/opt/Lidarr` and owned by your
brrewery user.

## Runtime dependencies

Lidarr's bundled .NET runtime needs ICU for globalization, so on Debian-based
hosts the install ensures `libicu-dev` (which pulls in the matching ICU runtime)
is present. SQLite is bundled with Lidarr, so no system database package is
required. (Audio fingerprinting via `fpcalc`/Chromaprint is an optional Lidarr
feature and is not installed here.)

## System password

Installing Lidarr writes to `/opt`, installs a per-user systemd service, and
configures an nginx site — all of which require root. If brrewery runs as a
non-root user, the install prompts for your **account password** so it can
escalate with `sudo`. It is verified against your brrewery account before the
install runs, used only for that run, and never logged.

## After install

Lidarr runs as a per-user systemd service (`lidarr@<user>.service`) listening on
`127.0.0.1:8686`, reverse-proxied at `/lidarr/`. Lidarr's default URL base is
empty, so the install seeds `<UrlBase>lidarr</UrlBase>` in
`~/.config/lidarr/config.xml`; nginx then forwards the `/lidarr/` subpath
unchanged (the same approach as autobrr, no prefix stripping).

Configuration and data live under `~/.config/lidarr/` owned by your brrewery
user. brrewery seeds the bind address, port, URL base, a generated `ApiKey`, and
the authentication mode into `config.xml` with `force: false`, so an existing
config (and the login user Lidarr stores in its database) is never overwritten and
re-running the install is safe.

Lidarr's login form is enabled and **required for every request**
(`AuthenticationMethod=Forms`, `AuthenticationRequired=Enabled`). After Lidarr
starts, the install calls Lidarr's own API (`PUT /api/v1/config/host` — Lidarr
uses the v1 API, unlike Sonarr/Radarr's v3) to create a login user whose
credentials match your brrewery account — the same username and the **account
password** you entered for the install. Lidarr hashes and stores the password
itself, and the step is skipped on re-runs once the user already exists. Running
an upgrade re-downloads the latest tarball, replaces `/opt/Lidarr`, and restarts
the service while leaving your configuration, data, and login untouched.

## First login

Open `/lidarr/` and sign in with your brrewery account credentials (same username
and password). From there, add your indexers (e.g. via
[Prowlarr](prowlarr.md)), download clients, and artists.
