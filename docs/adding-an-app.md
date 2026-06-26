# Adding an app

An app is defined by data, not code. Adding one to the catalog requires **no
changes to `internal/apps/catalog/catalog.go`** — you add a manifest, the
playbooks, and an icon.

## 1. Manifest

Create `internal/apps/catalog/manifests/<id>.yaml`. It is embedded into the
binary at build time and parsed once at startup.

```yaml
id: myapp                       # required, unique; drives all derived paths
name: My App                    # required, display name
description: What it does        # required
category: download              # download | arr | media | automation | tools
web_path: /myapp/               # optional; reverse-proxy subpath, omit if no web UI
# dependencies:                 # optional; other app ids that must be installed
#   - rtorrent
detection:                      # required; at least one check so install is detectable
  binaries:
    - myapp
  systemd_units:                # system units
    - myapp.service
  systemd_user_units:           # per-user template units; "{user}" is expanded
    - "myapp@{user}.service"
  paths:                        # files/dirs that must exist
    - /srv/myapp
# requires_account_password: true   # adds the shared, verified account-password prompt
# install_secrets:                  # extra install-time prompts (rare)
#   - key: my_secret
#     label: API key
#     type: password
# install_options:                  # static build/install choices (rare)
#   - key: my_option
#     label: Variant
#     type: select
#     choices:
#       - { value: a, label: A }
#       - { value: b, label: B }
```

Derived automatically from `id`:

- **Playbooks:** `ansible/playbooks/apps/<id>/{install,upgrade,remove}.yml`.
- **Icon:** resolved in the frontend by `id` (see step 3).

### The account password

Apps that provision a Linux service account (autobrr, qui, qBittorrent) set
`requires_account_password: true`. This adds the single shared password prompt —
the same value used as the Linux user password, the sudo (become) password, and
the dashboard password — which is always verified against the brrewery account
before install. Do not redeclare it under `install_secrets`.

## 2. Playbooks

Add `install.yml`, `upgrade.yml`, and `remove.yml` under
`ansible/playbooks/apps/<id>/`. See [ansible-apps.md](ansible-apps.md)
and an existing binary-release app (e.g. `autobrr` or `qui`) for the pattern:
resolve the brrewery user, install under `~/.config/<id>`, run a per-user systemd
unit, and wire up the nginx site via the `brrewery_nginx_site` role.

## 3. Icon

Icons live entirely in the frontend, keyed by app `id` — there is no served
`/apps/` asset folder. Drop the official logo as an SVG at
`web/src/assets/app-icons/<id>.svg`, then register it in
`web/src/lib/app-icons.ts` (import it with `?raw` and add an `APP_ICONS` entry).
It is inlined into the JS bundle as a data URI at build time. There is no text
or color fallback: an `id` with no registry entry renders nothing. To reuse
another app's logo (as `rtorrent` does with ruTorrent's), point the new `id` at
the same imported SVG.

## When you *do* need Go

Only for behavior that cannot be expressed as data:

- **Runtime-computed install options.** If an app's options depend on external
  state (qBittorrent derives its version list from the vendored build manifest),
  call `catalog.RegisterInstallOptions("<id>", provider)` from an `init()` in your
  app instead of declaring static `install_options`. The app must be
  imported somewhere in the binary (the app service and HTTP handlers already
  import the qbittorrent app) so its `init` runs.
- **Install-option validation / extra-var enrichment.** These are keyed by app
  id in `internal/api/handlers/apps.go` and `internal/apps/service.go`.

If you don't need any of the above, you never touch Go to add an app.
