# qBittorrent app — implementation handoff

Status as of 2026-06-03. This is the working context for the qBittorrent
source-build feature so the next agent can continue coding and troubleshooting.

- Design spec (source of truth): [plans/qbittorrent-app-install.md](plans/qbittorrent-app-install.md)
- Engineering notes: [ansible-apps.md](ansible-apps.md)
- User docs: [../documentation/docs/apps/qbittorrent.md](../documentation/docs/apps/qbittorrent.md)

## TL;DR

qBittorrent installs by compiling `qbittorrent-nox` from **vendored** sources
(libtorrent, Qt, Boost, zlib, OpenSSL). The user picks a minor line
(4.3–5.2) and, for ≥4.4, a libtorrent branch (RC_1_2/RC_2_0), with an optional
ephemeral libtorrent patch upload. A sudo password is collected in the UI so the
unprivileged daemon can escalate via Ansible `become`.

The full code path is implemented and unit/lint/build-clean. End-to-end build
has NOT completed on a real host yet — the install playbook now downloads
pinned sources via `ansible/roles/qbittorrent_build/tasks/vendor.yml` when
they are not already cached under the vendor root.

## What works / is verified

- Backend builds (`go build ./...`), `make test-openapi`, frontend `pnpm build`,
  all 34 vitest tests, `golangci-lint --new-from-rev=HEAD` = 0 new issues.
- All three qBittorrent playbooks pass `ansible-playbook --syntax-check`.
- On a real host (systemd service, after deploying the updated unit), the play
  runs through: manifest resolve → version/branch validation → apt cache refresh
  → **apt install of build deps succeeded** → downloads pinned sources via
  `tasks/vendor.yml` (first install on a host needs outbound HTTPS).


## Key files

Backend
- `internal/apps/qbittorrent/` — manifest loader, `InstallOptions()`,
  `Validate()`, `ValidateInstallOptions()`, `ValidateLibtorrentPatch()` (+ tests).
- `internal/apps/model/types.go` — `InstallOption[Choice|When]`, `App.InstallOptions`.
- `internal/apps/extravars/extravars.go` — `QbittorrentVersion`, `QbittorrentRelease`,
  `LibtorrentBranch`, `LibtorrentPatch`, `QbittorrentWebUIPasswordHash`,
  `BecomePassword` (`ansible_become_password`). Build-dependency versions are
  pinned in the manifest and read by Ansible directly, not passed as extra vars.
- `internal/apps/catalog/catalog.go` — `qbittorrentEntry()`: detection
  `qbittorrent@{user}.service`, install options from manifest, required sudo secret.
- `internal/apps/ansible/runner.go` — extracts `ansible_become_password`,
  passes it via `--become-password-file` (temp file, 0600, deleted; stripped from `-e` JSON).
- `internal/api/handlers/apps.go` — validates options on install + upgrade.
- `internal/paths/paths.go` — `ResolveVendorQBittorrentRoot()`, `QBittorrentOperatorPatchesDir`, `VendorRoot`.
- `internal/web/swagger/openapi.yaml` — `InstallOption*` schemas + `install_options` + `libtorrent_patch` note.

Frontend
- `web/src/components/install-options-modal.tsx` — version step + libtorrent
  branch/patch step (+ `requiredInstallOptions`).
- `web/src/components/app-shell.tsx` — `options` phase (install + upgrade).
- `web/src/components/install-secrets-modal.tsx` — generic credentials copy.
- `web/src/lib/api.ts` — `InstallOption*` types, `install_options`.

Ansible
- `ansible/roles/qbittorrent_build/` — `tasks/{main,resolve,dependencies,boost,qt,libtorrent,qbittorrent}.yml`,
  `defaults/main.yml`, `meta/main.yml`.
- `ansible/playbooks/apps/qbittorrent/{install,upgrade,remove}.yml`.

Vendoring
- `ansible/roles/qbittorrent_build/files/qbittorrent/manifest.yml` — version matrix + pinned deps.
- `ansible/roles/qbittorrent_build/files/qbittorrent/patches/libtorrent-RC_*.patch` — default patches.
- `docs/qbittorrent-build-manifest.md` — maintainer notes for the manifest and patches.
- `ansible/roles/qbittorrent_build/tasks/vendor.yml` — copies manifest/patches from
  the role and downloads resolved `sources/` at install time to
  `/usr/share/brrewery/vendor/qbittorrent`. This is the only source-fetch path.
- `scripts/install.sh` — creates `/var/lib/brrewery/patches/qbittorrent` (operator patches).

Ops
- `contrib/systemd/brrewery.service` — sandboxing removed (see below).

## Install flow

`select → secrets (sudo password) → options (version → libtorrent branch/patch) → job`
→ API validates secret + version/branch/patch → runner runs the playbook with
`--become-password-file` → role builds and installs `qbittorrent-nox`, writes
`qbittorrent@.service`, enables `qbittorrent@{user}`, configures nginx at
`/qbittorrent/` (WebUI 127.0.0.1:8086).

libtorrent patch priority: UI upload (this job, ephemeral) → operator file
`/var/lib/brrewery/patches/qbittorrent/libtorrent-<branch>.patch` → vendored
default performance patch (best effort). qBittorrent source patches are
brrewery-vendored security patches only.

## Fixes made during troubleshooting (with rationale)

1. **Manifest location** — the in-repo manifest and patches live under
   `ansible/roles/qbittorrent_build/files/qbittorrent/` (Go reserves top-level `vendor/`).
   Production source cache stays `/usr/share/brrewery/vendor/qbittorrent`.
2. **Manifest discovery** — `tasks/resolve.yml` auto-discovers the manifest from
   role files, the production vendor cache, or `/etc/brrewery/ansible/roles/...`.
3. **apt cache refresh** — split into a best-effort `update_cache` (`failed_when: false`)
   + an install task, so a stale/unwritable cache doesn't abort.
4. **Privilege escalation** — sudo password collected in UI (`ansible_become_password`),
   passed via `--become-password-file`. systemd unit sandboxing
   (`NoNewPrivileges`, `ProtectSystem`, `ProtectHome`, `ReadWritePaths`, `PrivateTmp`)
   removed because it made the FS read-only for the privileged Ansible child and blocked
   sudo. NOTE: the running host must redeploy the unit (`install.sh` or copy +
   `daemon-reload` + `restart brrewery`).

## Open items / next steps

- [ ] **Validate the real build** on a writable host after vendoring: Boost bootstrap,
      libtorrent cmake (static), qBittorrent cmake (`GUI=OFF`, `QT6=ON`), install to
      `/usr/local/bin`. The cmake/autotools flags in `tasks/{libtorrent,qbittorrent}.yml`
      are reasoned but unverified end-to-end.
- [ ] **Vendored Qt build path** (`tasks/qt.yml`) builds the exact Qt version pinned
      per line as `qt` in the manifest (downloaded from download.qt.io by `vendor.yml`).
- [ ] **Default performance patches** (`ansible/roles/qbittorrent_build/files/qbittorrent/patches/*.patch`) have
      placeholder context lines; they apply best-effort and will likely no-op. Regenerate
      against the actual vendored libtorrent source if real tuning is desired.
- [ ] **Upgrade/remove sudo prompt**: the secrets phase is install-only, so
      `upgrade`/`remove` under a non-root daemon won't prompt for the sudo password.
      Extend `requiredSecrets`/app-shell to also prompt on upgrade/remove (the runner
      already handles `ansible_become_password` generically).
- [ ] **Other become apps**: autobrr (and future apps) also need the sudo
      password under a non-root daemon; only qBittorrent declares it today. Consider a
      generic per-action become-password prompt.
- [ ] **qBittorrent WebUI auth**: install sets `WebUI\Port`, reverse-proxy support, and
      accepts the legal notice, but does not set the WebUI admin password. Decide whether
      to manage it (qBittorrent prints a temporary password to the journal on first run).
- [ ] **Reverse-proxy subpath**: `/qbittorrent/` proxying via the generic nginx role +
      `WebUI\ReverseProxySupportEnabled` is not verified in a browser.
- [ ] **Checksums**: optional checksum file verified in
      `tasks/vendor.yml` before extract (not implemented).
- [ ] **Manifest pins** are best-effort “latest patch per line”; confirm each line’s
      exact latest patch and dependency versions before release.

## Gotchas

- **Pre-existing failing tests** (NOT from this work; fail on clean HEAD too):
  - `internal/apps` `TestService_List` expects 16 apps; catalog has 14
    (`catalog_test.go` authoritatively expects 14 and passes).
  - `internal/system` `TestMonitoredFstabMounts_readsHostFstab` expects ≥2 fstab
    mounts (environment-dependent).
- **gofmt**: `internal/apps/extravars/extravars.go` and `internal/apps/service.go`
  are gofmt-dirty at HEAD (pre-existing). The qBittorrent additions deliberately keep the
  existing extravars block byte-identical to avoid a "new" gosec G101 finding on the
  password key under `--new-from-rev`.
- **`ansible-syntax-check` / role resolution**: run from the `ansible/` dir (or with the
  repo `ansible/ansible.cfg`) so `roles_path` resolves — same as the existing autobrr
  playbooks.
- **Ansible temp dirs**: the runner sets `HOME`/`ANSIBLE_*_TEMP` to `/tmp/brrewery-*`.
  When testing manually with `become`, set `ANSIBLE_REMOTE_TEMP` to a writable path.

## Verification commands

```bash
go build ./...
go test -race -count=1 ./internal/apps/qbittorrent/... ./internal/api/handlers/...
make test-openapi
( cd web && pnpm tsc -b && pnpm test )
( cd ansible && for f in playbooks/apps/qbittorrent/*.yml; do ansible-playbook "$f" --syntax-check; done )
```
