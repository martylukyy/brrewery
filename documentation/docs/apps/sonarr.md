---
id: sonarr
title: Sonarr
---

# Sonarr

brrewery installs [Sonarr](https://sonarr.tv/) v4 from the official self-contained
.NET tarball published by Sonarr's own download service (not GitHub releases). The
linux/amd64 build is extracted to `/opt/Sonarr` and owned by your brrewery user.

## Runtime dependencies

Sonarr's bundled .NET runtime needs ICU for globalization, so on Debian-based
hosts the install ensures `libicu-dev` (which pulls in the matching ICU runtime)
is present. SQLite is bundled with Sonarr, so no system database package is
required.

## System password

Installing Sonarr writes to `/opt`, installs a per-user systemd service, and
configures an nginx site — all of which require root. If brrewery runs as a
non-root user, the install prompts for your **account password** so it can
escalate with `sudo`. It is verified against your brrewery account before the
install runs, used only for that run, and never logged.

## After install

Sonarr runs as a per-user systemd service (`sonarr@<user>.service`) listening on
`127.0.0.1:8989`, reverse-proxied at `/sonarr/`. Sonarr's default URL base is
empty, so the install seeds `<UrlBase>sonarr</UrlBase>` in
`~/.config/sonarr/config.xml`; nginx then forwards the `/sonarr/` subpath
unchanged (the same approach as autobrr, no prefix stripping).

Configuration and data live under `~/.config/sonarr/` owned by your brrewery
user. brrewery seeds the bind address, port, URL base, a generated `ApiKey`, and
the authentication mode into `config.xml` with `force: false`, so an existing
config (and the login user Sonarr stores in its database) is never overwritten and
re-running the install is safe.

Sonarr's login form is enabled and **required for every request**
(`AuthenticationMethod=Forms`, `AuthenticationRequired=Enabled`). After Sonarr
starts, the install calls Sonarr's own API (`PUT /api/v3/config/host`) to create a
login user whose credentials match your brrewery account — the same username and
the **account password** you entered for the install. Sonarr hashes and stores the
password itself, and the step is skipped on re-runs once the user already exists.
Running an upgrade re-downloads the latest v4 tarball, replaces `/opt/Sonarr`, and
restarts the service while leaving your configuration, data, and login untouched.

## First login

Open `/sonarr/` and sign in with your brrewery account credentials (same username
and password). From there, add your indexers (e.g. via
[Prowlarr](https://prowlarr.com/)), download clients, and series.
