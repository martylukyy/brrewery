---
id: prowlarr
title: Prowlarr
---

# Prowlarr

brrewery installs [Prowlarr](https://wiki.servarr.com/prowlarr) from the official
self-contained .NET tarball published by the Servarr update service (not GitHub
releases). The linux/amd64 build is extracted to `/opt/Prowlarr` and owned by your
brrewery user.

## Runtime dependencies

Prowlarr's bundled .NET runtime needs ICU for globalization, so on Debian-based
hosts the install ensures `libicu-dev` (which pulls in the matching ICU runtime)
is present. SQLite is bundled with Prowlarr, so no system database package is
required.

## System password

Installing Prowlarr writes to `/opt`, installs a per-user systemd service, and
configures an nginx site — all of which require root. If brrewery runs as a
non-root user, the install prompts for your **account password** so it can
escalate with `sudo`. It is verified against your brrewery account before the
install runs, used only for that run, and never logged.

## After install

Prowlarr runs as a per-user systemd service (`prowlarr@<user>.service`) listening
on `127.0.0.1:9696`, reverse-proxied at `/prowlarr/`. Prowlarr's default URL base
is empty, so the install seeds `<UrlBase>prowlarr</UrlBase>` in
`~/.config/prowlarr/config.xml`; nginx then forwards the `/prowlarr/` subpath
unchanged (the same approach as autobrr, no prefix stripping).

Configuration and data live under `~/.config/prowlarr/` owned by your brrewery
user. brrewery seeds the bind address, port, URL base, a generated `ApiKey`, and
the authentication mode into `config.xml` with `force: false`, so an existing
config (and the login user Prowlarr stores in its database) is never overwritten
and re-running the install is safe.

Prowlarr's login form is enabled and **required for every request**
(`AuthenticationMethod=Forms`, `AuthenticationRequired=Enabled`). After Prowlarr
starts, the install calls Prowlarr's own API (`PUT /api/v1/config/host` — Prowlarr
uses the v1 API, unlike Sonarr/Radarr's v3) to create a login user whose
credentials match your brrewery account — the same username and the **account
password** you entered for the install. Prowlarr hashes and stores the password
itself, and the step is skipped on re-runs once the user already exists. Running
an upgrade re-downloads the latest tarball, replaces `/opt/Prowlarr`, and restarts
the service while leaving your configuration, data, and login untouched.

## First login

Open `/prowlarr/` and sign in with your brrewery account credentials (same
username and password). From there, add your indexers and connect Prowlarr to your
[Sonarr](sonarr.md) and [Radarr](radarr.md) apps so indexers sync automatically.
