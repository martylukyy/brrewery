# Plan: Add install / upgrade / remove support for Jellyfin

## Context

brrewery exposes apps as data-driven catalog entries (manifest + three Ansible
playbooks + icon). Jellyfin is already half-wired: its manifest
(`internal/apps/catalog/manifests/jellyfin.yaml`), its icon
(`web/public/apps/jellyfin.png`), and its catalog/detection plumbing exist, but
the three playbooks under `ansible/playbooks/apps/jellyfin/` are still the
generated `debug` **stubs**, so Install/Upgrade/Remove are no-ops in the UI.

This change replaces those stubs with real playbooks that install Jellyfin from
its **official APT repository** (`repo.jellyfin.org`), reverse-proxy it under
`/jellyfin/` (the path the manifest already declares), and cleanly remove it.
No Go changes are required — the catalog is data, and the manifest/icon/detection
already cover Jellyfin. The data-driven `catalog_test.go` invariants continue to
pass unchanged.

All external packaging facts below were verified against Jellyfin's
`install-debuntu.sh`, the live `repo.jellyfin.org` Packages indexes, the
`jellyfin-packaging` Debian `control`/`postinst`/`postrm`, and
`NetworkConfiguration.cs`.

### Design decisions (confirmed with the user)
- **Serve mode:** reverse proxy at `/jellyfin/` (consistent with Sonarr/Radarr),
  setting Jellyfin's `BaseUrl=/jellyfin`. (Not the Plex-style own-port approach.)
- **Live features:** add a backward-compatible, opt-in `brrewery_app_websocket`
  flag to the shared `brrewery_nginx_site` role and enable it only for Jellyfin,
  so dashboard / SyncPlay / remote-control WebSockets work through the proxy.
  Existing apps render identically (flag defaults off).

## Key facts (verified)

- **Repo/key:** download `https://repo.jellyfin.org/jellyfin_team.gpg.key`,
  `gpg --dearmor` → `/etc/apt/keyrings/jellyfin.gpg` (0644).
- **deb822 source** at `/etc/apt/sources.list.d/jellyfin.sources`:
  `Types: deb` / `URIs: https://repo.jellyfin.org/debian` (`/ubuntu` on Ubuntu) /
  `Suites: <codename>` (`ansible_distribution_release`) / `Components: main` /
  `Architectures: <dpkg --print-architecture>` / `Signed-By: /etc/apt/keyrings/jellyfin.gpg`.
  Also delete any legacy `/etc/apt/sources.list.d/jellyfin.list`.
- **Packages:** installing the `jellyfin` metapackage pulls `jellyfin-server`,
  `jellyfin-web`, `jellyfin-ffmpeg7` (current ffmpeg major = **7**).
- **Service:** `jellyfin.service` (system unit, `User=jellyfin`), HTTP on `:8096`;
  the `.deb` postinst auto-enables+starts it and creates the `jellyfin` system user.
- **Dirs:** config `/etc/jellyfin` (holds `network.xml`), data `/var/lib/jellyfin`,
  cache `/var/cache/jellyfin`, log `/var/log/jellyfin`, defaults `/etc/default/jellyfin`.
- **Subpath:** set `<BaseUrl>/jellyfin</BaseUrl>` in `/etc/jellyfin/network.xml`
  (default is self-closing `<BaseUrl />`); nginx **preserves** the `/jellyfin/`
  prefix (no strip). Edit the file with the service stopped, then start.
- **Purge:** `apt purge` of `jellyfin-server` rm's all four dirs and the
  `jellyfin` user, but does **not** remove the `.sources` file, the keyring, or
  `/etc/default/jellyfin` — those need manual cleanup.

## Files to change

### 1. `ansible/roles/brrewery_nginx_site` — opt-in WebSocket support
- `defaults/main.yml`: add `brrewery_app_websocket: false`.
- `templates/location.conf.j2`: inside the `location ^~ {{ web_path }}` block, add
  (gated on the new flag; `proxy.conf` already sets `proxy_http_version 1.1`):
  ```jinja
  {% if brrewery_app_websocket | bool %}
      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection $http_connection;
  {% endif %}
  ```
  `$http_connection` forwards the client's Connection header (map-free, keeps
  keepalive for non-WS requests). Default-off ⇒ zero change to existing apps.

### 2. `ansible/playbooks/apps/jellyfin/install.yml` (replace stub)
Pattern follows `plex/install.yml` (media-access group join) + `sonarr/install.yml`
(subpath proxy), using only `ansible.builtin` modules. `become: true`,
`gather_facts: true`, `roles: [brrewery_user]`.
1. Assert `ansible_os_family == "Debian"` and `dpkg --print-architecture` ∈
   {amd64, arm64, armhf}.
2. Ensure `/etc/apt/keyrings`; `get_url` the key to a temp file; `command:
   gpg --dearmor … -o /etc/apt/keyrings/jellyfin.gpg` guarded by
   `creates: /etc/apt/keyrings/jellyfin.gpg`; `chmod 0644`.
3. Remove legacy `jellyfin.list`; write `jellyfin.sources` (deb822, derived
   suite/arch/os as above).
4. `apt: name=jellyfin update_cache=true` (postinst starts the service).
5. `wait_for` `/etc/jellyfin/network.xml` to exist; `systemd: stop jellyfin`.
6. `replace` `<BaseUrl …/>`|`<BaseUrl>…</BaseUrl>` → `<BaseUrl>/jellyfin</BaseUrl>`.
7. Add `jellyfin` user to `brrewery_group`; open brrewery home `0700→0750`
   (non-recursive, only when exactly `0700`) — verbatim Plex media-access pattern.
8. `systemd: jellyfin enabled+started`; `wait_for` port `8096`.
9. `include_role: brrewery_nginx_site` with `app_id=jellyfin`,
   `web_path=/jellyfin/`, `upstream=http://127.0.0.1:8096`,
   `strip_path_prefix=false`, `websocket=true`.
10. `debug` "Jellyfin installed — finish setup at /jellyfin/".

### 3. `ansible/playbooks/apps/jellyfin/upgrade.yml` (replace stub)
Self-healing, parallel to install: assert Debian; re-assert key + `.sources`
(idempotent); `apt: name=[jellyfin, jellyfin-server, jellyfin-web,
jellyfin-ffmpeg7] state=latest update_cache=true`; stop → idempotent `BaseUrl`
`replace` → re-affirm group membership → start → `wait_for` 8096; re-assert the
nginx site (so an older install gains the WebSocket block).

### 4. `ansible/playbooks/apps/jellyfin/remove.yml` (replace stub)
Follows `plex/remove.yml`: `roles: [brrewery_user]`; stop+disable
`jellyfin.service` (`failed_when: false`); `apt: name=[jellyfin, jellyfin-server,
jellyfin-web, jellyfin-ffmpeg7] state=absent purge=true autoremove=true`; delete
`/etc/apt/sources.list.d/jellyfin.sources`, `…/jellyfin.list`,
`/etc/apt/keyrings/jellyfin.gpg`, `/etc/default/jellyfin`, and the four data dirs
(`/var/lib`, `/etc`, `/var/cache`, `/var/log` `/jellyfin`); revert brrewery home
`0750→0700` (only when still `0750`); remove the nginx site via
`brrewery_nginx_site` `remove.yml`.

### 5. `docs/jellyfin.md` (new) — engineering doc mirroring `docs/plex.md`
Covers: apt-repo install (key + deb822), the `jellyfin-ffmpeg7` dependency,
`BaseUrl=/jellyfin` + subpath proxy, the opt-in WebSocket flag, media-access
group join, and what purge does/doesn't clean (so remove deletes the repo/keyring/
defaults explicitly).

### No change needed
- `internal/apps/catalog/manifests/jellyfin.yaml` — `web_path: /jellyfin/` and
  detection (`jellyfin` binary at `/usr/bin/jellyfin` + `jellyfin.service`) are
  already correct; both artifacts exist after install.
- `web/public/apps/jellyfin.png` already present; catalog Go + tests unaffected.

## Verification
1. `cd ansible && find playbooks/apps/jellyfin -name '*.yml' -print0 | xargs -0 -n1
   ansible-playbook --syntax-check` (and the nginx role’s consumers) — or
   `make ansible-syntax-check` for the whole tree.
2. `go test -race -count=1 ./internal/apps/...` — confirms the data-driven catalog
   invariants (playbook paths, detection, icon) still hold for Jellyfin.
3. `make build` succeeds.
4. (Optional, on a Debian host) run the install playbook with
   `-e brrewery_user=<user>`, confirm `systemctl status jellyfin`, that
   `https://<host>/jellyfin/` loads through nginx with live/WebSocket features
   working, then run remove and confirm the service, packages, repo/keyring, and
   data dirs are gone and the home mode reverted.
5. `make precommit` (changed files) before committing.
