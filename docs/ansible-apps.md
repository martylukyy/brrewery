# Ansible app playbooks

App lifecycle is driven by Ansible playbooks under `ansible/playbooks/apps/<id>/`.

> Adding a whole app (catalog metadata + playbooks + icon) is documented in
> [adding-an-app.md](adding-an-app.md). The catalog itself is data-driven:
> apps are declared as embedded YAML manifests, not Go code.

## Layout

```text
ansible/
  ansible.cfg
  inventory/localhost.yml
  roles/common/
  playbooks/apps/<id>/
    install.yml
    upgrade.yml
    remove.yml
```

Installed on the host at `/usr/share/brrewery/ansible` by `scripts/install.sh`.

## MVP

Playbooks are **syntax-valid stubs** (local connection, placeholder `debug` task). CI runs:

```bash
make ansible-syntax-check
```

## M2 execution

The Go runner in `internal/apps/ansible` will invoke:

```bash
ansible-playbook --connection=local <playbook> -e @extra-vars.json
```

Extra-vars for app secrets are supplied from the API/UI per install and are **never** written to disk by brrewery.

Every install automatically receives `brrewery_user` (the logged-in brrewery admin OS username). Playbooks should include the `brrewery_user` role to resolve the matching primary group as `brrewery_group`. App data, config, and systemd user services must run as that user for future multi-user support.

### Privilege escalation

Playbooks that change system state run under play-level `become: true`. When brrewery runs unprivileged, the operator's sudo password is collected in the web UI (an install secret with key `ansible_become_password`) and handed to the runner. The runner writes it to a private temp file and invokes `ansible-playbook --become-password-file <file>` (deleted after the run); the password is never placed in the extra-vars JSON, on the process arguments, or in the job log. When brrewery already runs as root the supplied value is simply unused by sudo. The bundled systemd unit (`contrib/systemd/brrewery.service`) therefore must not set `NoNewPrivileges=true` or `ProtectSystem=strict`, which would block sudo and writes to `/usr`, `/etc`, and `/opt`.

After a successful run, install status is re-probed via filesystem detection only (no playbook marker files).

## qBittorrent (source build)

qBittorrent is compiled from vendored sources rather than installed from a release binary. See [plans/qbittorrent-app-install.md](plans/qbittorrent-app-install.md) for the full design and [qbittorrent-handoff.md](qbittorrent-handoff.md) for current status and open items.

- **Vendored build tree:** `ansible/roles/qbittorrent_build/files/qbittorrent/` holds the manifest and default patches in git; production caches downloaded sources under `/usr/share/brrewery/vendor/qbittorrent`. See [qbittorrent-build-manifest.md](qbittorrent-build-manifest.md). `tasks/vendor.yml` downloads and extracts build sources at install time.
- **Build role:** `ansible/roles/qbittorrent_build` resolves the manifest line for the requested minor and compiles the dependency versions pinned per line in the manifest — Boost (per libtorrent branch), Qt (`qt`), zlib (`zlib`), OpenSSL (`openssl`), libtorrent (`branches.<branch>.tag`) — and `qbittorrent-nox` from vendored sources. The role reads these pinned versions straight from the manifest (`resolve.yml`); only the qBittorrent patch release is resolved from GitHub at job start (passed as `qbittorrent_release`). Install/upgrade needs outbound HTTPS to GitHub, qt.io, and archives.boost.io to download the pinned sources.
- **Extra vars** passed by the install/upgrade API:
  - `qbittorrent_version` — major.minor line from the UI (e.g. `5.2`).
  - `qbittorrent_release` — resolved patch release (e.g. `5.2.1`) from GitHub `release-{minor}.*` tags; set by brrewery before Ansible runs.
  - `libtorrent_branch` — `RC_1_2` or `RC_2_0`; empty falls back to the line default (`RC_1_2`).
  - `libtorrent_patch` — optional base64 unified-diff applied to a single build. It is **ephemeral**: decoded to a `0600` temp file, applied, and deleted; never written under `/var/lib/brrewery` and never logged (`no_log`).
- **libtorrent patch priority** (one wins, before the libtorrent compile):
  1. Uploaded `libtorrent_patch` (this build only) — must apply cleanly or the job fails.
  2. Operator file `/var/lib/brrewery/patches/qbittorrent/libtorrent-<branch>.patch` — must apply cleanly.
  3. Vendored default `patches/libtorrent-<branch>.patch` — applied best effort.
- **qBittorrent source patches** are brrewery-only: vendored `patches/qbittorrent-<version>-security.patch` is applied when present. There is no Web UI or operator path for qBittorrent source patches.
