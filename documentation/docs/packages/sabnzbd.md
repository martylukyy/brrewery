---
id: sabnzbd
title: SABnzbd
---

# SABnzbd

brrewery installs [SABnzbd](https://sabnzbd.org/) from source, following the
upstream [install off modules](https://sabnzbd.org/wiki/installation/install-off-modules)
method. The `master` branch is cloned to `/opt/sabnzbd`, a Python virtualenv is
created at `/opt/sabnzbd/venv`, and the dependencies from `requirements.txt` are
installed into it. Translations are compiled with `tools/make_mo.py`.

## External utilities

The playbook also installs every Must-have and Optional helper SABnzbd looks for:

| Utility | Purpose | Source |
| --- | --- | --- |
| **par2** | repair/verify (Must-have) | apt (`par2`) |
| **unrar** | RAR extraction (Must-have) | compiled from the official RARLAB source — Debian main ships only the incompatible `unrar-free` |
| **unzip** | password-protected zip (`-P`) | apt (`unzip`) |
| **7zip** | provides `7z` / `7za` | apt (`p7zip-full`) |
| **nice** | external-tool CPU priority | apt (`coreutils`) |
| **ionice** | external-tool disk priority | apt (`util-linux`) |
| **notify-osd** | desktop notifications | apt (`notify-osd`) |

## System password

Installing SABnzbd creates a per-user systemd service and an nginx site, which
requires root. If brrewery runs as a non-root user, the install prompts for your
**account password** so it can escalate with `sudo`. The same password is set as
the SABnzbd WebUI password, so the credentials match your brrewery admin user. It
is verified against your brrewery account before the install runs, used only for
that run, and never logged.

## After install

SABnzbd runs as a per-user systemd service (`sabnzbd@<user>.service`) with the
WebUI on `127.0.0.1:8085`, reverse-proxied at `/sabnzbd/`. SABnzbd's default
`url_base` is empty, so the install sets `url_base = /sabnzbd` in
`~/.config/sabnzbd/sabnzbd.ini`; nginx then forwards the `/sabnzbd/` subpath
unchanged (the same approach as autobrr, no prefix stripping).

Configuration lives under `~/.config/sabnzbd/` owned by your brrewery user.
brrewery seeds `host`, `port`, `url_base`, and the WebUI login on first install;
SABnzbd manages the rest of the file (including the generated `api_key`) itself.
The WebUI password is written hex-encoded the same way SABnzbd's own
`encode_password` does, so passwords containing characters that are unsafe in
the ini format (`#`, spaces, commas, …) still authenticate correctly. Running an
upgrade pulls the latest `master`, refreshes the virtualenv, and restarts the
service.

## First login

Open `/sabnzbd/`, sign in with your brrewery account credentials, then walk
through SABnzbd's setup wizard to add your Usenet server(s). Because a WebUI
username and password are set, SABnzbd skips its DNS-rebinding hostname check, so
the login works whether you reach brrewery by IP or hostname.
