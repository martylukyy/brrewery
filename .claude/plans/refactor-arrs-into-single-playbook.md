# Refactor Servarr (*arr) playbooks into a single `servarr_app` role

## Context

`ansible/playbooks/packages/{sonarr,radarr,prowlarr}/{install,upgrade,remove}.yml`
are 9 playbooks that are ~95% identical (each install file is ~226 lines). Every
behavioral line is duplicated three times, so a fix or hardening change has to be
made in three places and is easy to get subtly wrong (the v1-vs-v3 API path is
exactly the kind of per-app detail that drifts). This change collapses the shared
logic into one parameterized role, `ansible/roles/servarr_app`, leaving each
`packages/<id>/{install,upgrade,remove}.yml` as a thin playbook that just sets the
per-app vars and includes the role. Two review-spotted hardening items are folded
in at the same time. Behavior must stay identical (the API-version split in
particular).

The only real per-app differences:

| app | id | display/binary | port | api ver | branch | download URL |
|-----|----|----|------|---------|--------|--------------|
| Sonarr | sonarr | Sonarr | 8989 | v3 | main | `https://services.sonarr.tv/v1/download/main/latest?version=4&os=linux&arch=x64` |
| Radarr | radarr | Radarr | 7878 | v3 | master | `https://radarr.servarr.com/v1/update/master/updatefile?os=linux&runtime=netcore&arch=x64` |
| Prowlarr | prowlarr | Prowlarr | 9696 | v1 | master | `https://prowlarr.servarr.com/v1/update/master/updatefile?os=linux&runtime=netcore&arch=x64` |

Everything else derives from these: install dir `/opt/<Display>`, binary
`/opt/<Display>/<Display>`, config dir `~/.config/<id>`, systemd unit
`<id>@<user>.service`, UrlBase `<id>`, nginx path `/<id>/`, API URL
`http://127.0.0.1:<port>/<id>/api/<ver>/config/host`.

## Hard constraints (verified)

- Thin playbook paths/names **must not change**: `internal/packages/catalog/catalog.go:98`
  derives `playbooks/packages/<id>/{install,upgrade,remove}.yml`, and
  `catalog_test.go:37-39` asserts those path strings contain the id. No Go test
  parses playbook *contents*.
- Role task files must live under `ansible/roles/` (not `playbooks/`): `make
  ansible-syntax-check` runs `ansible-playbook --syntax-check` on **every** `*.yml`
  under `playbooks/`, so a bare task-list file there would fail to parse as a play.
  `roles_path = roles` (ansible/ansible.cfg) resolves the new role automatically.

## New role: `ansible/roles/servarr_app/`

Mirrors `qbittorrent_build` conventions. Files:

### `meta/main.yml`
`galaxy_info` (role_name `servarr_app`, description), `dependencies: []`
(brrewery_user stays a play-level role, matching qbittorrent).

### `defaults/main.yml`
Centralises the derived conventions so thin playbooks only set the true per-app
inputs:
```yaml
servarr_install_dir: "/opt/{{ servarr_app_name }}"   # binary is <dir>/<name>
servarr_url_base: "{{ servarr_app_id }}"
servarr_apt_cache_valid_time: 3600                   # hardening item #2
```

### `handlers/main.yml`
`Reload systemd` (daemon_reload), notified by the unit install (install.yml) and
unit removal (remove.yml) — same handler qbittorrent_build defines. Role handlers
load automatically through `include_role`, so the thin playbooks need no `handlers:`
block.

### `templates/servarr@.service.j2`
Replaces the inline systemd unit (matches qbittorrent_build using a `.j2`):
```
[Unit]
Description={{ servarr_app_name }} service for %i
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=%i
ExecStart={{ servarr_install_dir }}/{{ servarr_app_name }} -nobrowser -data=/home/%i/.config/{{ servarr_app_id }}
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### `tasks/install.yml`
Exact port of the current install task list, parameterized. Order preserved:
1. `getent` passwd → `brrewery_passwd`; `set_fact brrewery_home`.
2. `stat` home → fail if missing.
3. `assert` `brrewery_user_password` defined & non-empty
   (`fail_msg: brrewery_user_password must be supplied for {{ servarr_app_id }} install`).
4. apt `libicu-dev` (when Debian) — keep the ICU comment; **add
   `cache_valid_time: "{{ servarr_apt_cache_valid_time }}"`** (hardening #2).
5. systemd stop `{{ servarr_app_id }}@{{ brrewery_user }}.service`, `failed_when: false`.
6. `get_url` `{{ servarr_download_url }}` → `/tmp/{{ servarr_app_id }}.tar.gz`.
7. remove install dir; ensure `/opt`; `unarchive` → `/opt`; chown recurse.
8. ensure `~/.config/{{ servarr_app_id }}` (0750).
9. `openssl rand -hex 16` → `servarr_api_key_gen` (`changed_when: false`).
10. write `config.xml` (inline `copy`, `force: false`, keep the
    "never clobber" comment) with `<Port>{{ servarr_port }}</Port>`,
    `<UrlBase>{{ servarr_url_base }}</UrlBase>`,
    `<ApiKey>{{ servarr_api_key_gen.stdout }}</ApiKey>`,
    `<Branch>{{ servarr_branch }}</Branch>`, Forms/Enabled as today.
11. install systemd unit via `template: src=servarr@.service.j2
    dest=/etc/systemd/system/{{ servarr_app_id }}@.service` → `notify: Reload systemd`.
12. systemd enable+start, `daemon_reload: true`.
13. `slurp` config → regex `<ApiKey>` → `set_fact servarr_api_key` (`no_log: true`).
14. **Wait-for-API GET** `…/{{ servarr_url_base }}/api/{{ servarr_api_version }}/config/host`
    → register `servarr_host_config`, `until status==200`, retries 30, delay 5.
    **Add `no_log: true`** (hardening #1 — response JSON contains the apiKey).
    `no_log` does not suppress `register`, so downstream `.json` use is unaffected.
15. `set_fact servarr_auth_overrides` (`no_log: true`).
16. PUT `…/config/host/{{ servarr_host_config.json.id }}` with
    `{{ servarr_host_config.json | combine(servarr_auth_overrides) }}`, register
    `servarr_auth_set`, `no_log: true`, same `when` guard (method != forms or
    username != brrewery_user).
17. restart + `wait_for` port, both `when: servarr_auth_set is changed`.
18. `include_role: brrewery_nginx_site` with `brrewery_package_id={{ servarr_app_id }}`,
    `brrewery_package_web_path=/{{ servarr_app_id }}/`,
    `brrewery_package_upstream=http://127.0.0.1:{{ servarr_port }}`.

### `tasks/upgrade.yml`
Port of current upgrade (lighter — no home-stat/fail, no password assert, no
handlers): getent + home fact → apt libicu-dev (Debian, **+ cache_valid_time**) →
get_url → stop → remove dir → unarchive → chown → enable+start (daemon_reload) →
nginx include_role.

### `tasks/remove.yml`
Port of current remove: getent + home fact → stop+disable (daemon_reload,
`failed_when: false`) → `include_role brrewery_nginx_site tasks_from: remove`
(`brrewery_package_id={{ servarr_app_id }}`) → remove `~/.config/<id>` → remove
install dir → list-unit-files shell (`changed_when: false`) → remove
`/etc/systemd/system/<id>@.service` when none remain (`notify: Reload systemd`).

### `tasks/main.yml`
Default entrypoint = install (`import_tasks: install.yml`) for parity with how
`qbittorrent_build`'s main.yml is the install flow; the thin playbooks select the
action explicitly via `tasks_from`, so this is just a sensible default.

## Thin playbooks (the 9 files, rewritten)

Each shrinks to vars + `roles: [brrewery_user]` + one `include_role`. Example
`packages/sonarr/install.yml`:
```yaml
---
- hosts: localhost
  connection: local
  become: true
  gather_facts: true
  vars:
    servarr_app_id: sonarr
    servarr_app_name: Sonarr
    servarr_port: 8989
    servarr_api_version: v3
    servarr_branch: main
    servarr_download_url: "https://services.sonarr.tv/v1/download/main/latest?version=4&os=linux&arch=x64"
  roles:
    - brrewery_user
  tasks:
    - name: Install Sonarr via servarr_app role
      ansible.builtin.include_role:
        name: servarr_app
        tasks_from: install
```
`upgrade.yml`/`remove.yml` are identical shells with `tasks_from: upgrade` /
`remove` (remove needs only `servarr_app_id`; upgrade needs id/name/port/url; the
download URL/branch/port are only consumed by the relevant action). For
simplicity and consistency each file sets the same per-app var block. radarr (7878,
v3, master) and prowlarr (9696, **v1**, master) follow the table above.

This pattern is proven: `packages/qbittorrent/install.yml` already does
`include_role: name=qbittorrent_build tasks_from=…` and passes syntax-check.

## Out of scope
- `lidarr` has a manifest + stub (debug-only) playbooks; the new role makes a real
  implementation trivial (v1, port 8686, branch master, lidarr.servarr.com) but
  that changes behavior and is outside this refactor. Flag as a follow-up.

## Verification
1. `make ansible-syntax-check` — parses all 9 thin playbooks + resolves the role.
2. `go test ./internal/packages/...` — catalog/path invariants stay green
   (paths unchanged).
3. Spot-diff intent: confirm prowlarr install/upgrade still hit `/api/v1/`, sonarr
   & radarr `/api/v3/`; config `<Branch>` = main/master/master; ports 8989/7878/9696.
4. Sanity: `grep -rn "api/v" ansible/playbooks/packages` should now find nothing
   (the version lives only in the role, driven by `servarr_api_version`).
