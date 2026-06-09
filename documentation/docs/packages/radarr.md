---
id: radarr
title: Radarr
---

# Radarr

brrewery installs [Radarr](https://radarr.video/) from the official
self-contained .NET tarball published by the Servarr update service (not GitHub
releases). The linux/amd64 build is extracted to `/opt/Radarr` and owned by your
brrewery user.

## Runtime dependencies

Radarr's bundled .NET runtime needs ICU for globalization, so on Debian-based
hosts the install ensures `libicu-dev` (which pulls in the matching ICU runtime)
is present. SQLite is bundled with Radarr, so no system database package is
required.

## System password

Installing Radarr writes to `/opt`, installs a per-user systemd service, and
configures an nginx site — all of which require root. If brrewery runs as a
non-root user, the install prompts for your **account password** so it can
escalate with `sudo`. It is verified against your brrewery account before the
install runs, used only for that run, and never logged.

## After install

Radarr runs as a per-user systemd service (`radarr@<user>.service`) listening on
`127.0.0.1:7878`, reverse-proxied at `/radarr/`. Radarr's default URL base is
empty, so the install seeds `<UrlBase>radarr</UrlBase>` in
`~/.config/radarr/config.xml`; nginx then forwards the `/radarr/` subpath
unchanged (the same approach as autobrr, no prefix stripping).

Configuration and data live under `~/.config/radarr/` owned by your brrewery
user. brrewery seeds the bind address, port, URL base, a generated `ApiKey`, and
the authentication mode into `config.xml` with `force: false`, so an existing
config (and the login user Radarr stores in its database) is never overwritten and
re-running the install is safe.

Radarr's login form is enabled and **required for every request**
(`AuthenticationMethod=Forms`, `AuthenticationRequired=Enabled`). After Radarr
starts, the install calls Radarr's own API (`PUT /api/v3/config/host`) to create a
login user whose credentials match your brrewery account — the same username and
the **account password** you entered for the install. Radarr hashes and stores the
password itself, and the step is skipped on re-runs once the user already exists.
Running an upgrade re-downloads the latest tarball, replaces `/opt/Radarr`, and
restarts the service while leaving your configuration, data, and login untouched.

## First login

Open `/radarr/` and sign in with your brrewery account credentials (same username
and password). From there, add your indexers (e.g. via
[Prowlarr](https://prowlarr.com/)), download clients, and movies.
