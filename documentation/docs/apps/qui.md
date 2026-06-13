---
id: qui
title: qui
---

# qui

[qui](https://github.com/autobrr/qui) is a fast, modern web interface for
qBittorrent that lets you manage one or more qBittorrent instances from a single
dashboard. brrewery installs it from the official autobrr release binary
(`linux_x86_64`) to `/usr/local/bin/qui`.

## System password

Installing qui creates a per-user systemd service and an nginx site, which
requires root. If brrewery runs as a non-root user, the install prompts for your
**account password** so it can escalate with `sudo`. The same password is used to
create the initial qui login account, so the credentials match your brrewery
admin user. It is verified against your brrewery account before the install runs,
used only for that run, and never saved or logged.

:::note
qui requires the account password to be at least 8 characters.
:::

## After install

qui runs as a per-user systemd service (`qui@<user>.service`) with the WebUI on
`127.0.0.1:7476`, reverse-proxied at `/qui/`. The service is configured with
`baseUrl = "/qui/"`, so nginx forwards the subpath unchanged (the same approach
as autobrr).

Configuration and the SQLite database live under `~/.config/qui/`
(`config.toml` and `qui.db`) owned by your brrewery user. The session secret is
generated at install time; changing it would invalidate stored qBittorrent
instance passwords, so it is left untouched on upgrade.

In-app update checks are disabled (`checkForUpdates = false`) because brrewery
manages upgrades — running an upgrade re-downloads the latest release binary and
restarts the service.

## First login

Open `/qui/`, sign in with your brrewery account credentials, then add your
qBittorrent instance(s) from the qui settings to start managing torrents.
