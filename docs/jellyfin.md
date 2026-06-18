# Jellyfin

Jellyfin is installed, upgraded and removed from its **official APT repository**
(`repo.jellyfin.org`) and reverse-proxied under `/jellyfin/`, the same subpath
style as Sonarr/Radarr (and unlike Plex, which is reached on its own port).

Playbooks: `ansible/playbooks/apps/jellyfin/{install,upgrade,remove}.yml`.
Manifest: `internal/apps/catalog/manifests/jellyfin.yaml`.

## Install / upgrade (apt repository, not a one-off .deb)

1. Require a Debian-family host and an architecture Jellyfin publishes packages
   for — `dpkg --print-architecture` ∈ {`amd64`, `arm64`, `armhf`}.
2. Install the signing key: download `https://repo.jellyfin.org/jellyfin_team.gpg.key`
   and `gpg --dearmor` it to `/etc/apt/keyrings/jellyfin.gpg` (`0644`). The dearmor
   is guarded by `creates:` so re-runs don't churn the keyring.
3. Write a deb822 source at `/etc/apt/sources.list.d/jellyfin.sources` with
   `URIs: https://repo.jellyfin.org/<debian|ubuntu>` (from `ansible_distribution`),
   `Suites: <codename>` (`ansible_distribution_release`), `Components: main`,
   `Architectures: <dpkg arch>` and `Signed-By:` the keyring. Any legacy
   `/etc/apt/sources.list.d/jellyfin.list` is deleted first.
4. `apt install jellyfin` (with `update_cache`). The `jellyfin` metapackage pulls
   `jellyfin-server`, `jellyfin-web` and `jellyfin-ffmpeg7` (the current ffmpeg
   major is **7**). The `.deb` postinst creates the `jellyfin` system user and
   enables+starts `jellyfin.service` (HTTP on `:8096`).

Upgrade is the same shape but self-healing: it re-asserts the key and source
(so an install predating the deb822 source is repaired) and runs
`apt state=latest` over `jellyfin`, `jellyfin-server`, `jellyfin-web` and
`jellyfin-ffmpeg7` explicitly, since the metapackage version can lag its
components.

The manifest's `requires_account_password: true` collects the operator's account
password purely as the sudo (become) password for the apt/systemd steps; it is
verified against the brrewery account before the playbook runs. (This is why the
manifest carries the flag even though Jellyfin has no app-level login brrewery
provisions — same rationale as Plex.)

## Subpath proxy & live features (BaseUrl=/jellyfin)

nginx **preserves** the `/jellyfin/` prefix (`strip_path_prefix=false`), so
Jellyfin must serve under that base URL. Install stops the service, sets
`<BaseUrl>/jellyfin</BaseUrl>` in `/etc/jellyfin/network.xml` (the default is the
self-closing `<BaseUrl />`; the `replace` matches both that and an already-set
value, so it's idempotent), then starts the service. The file is generated on the
server's first start, so the playbook `wait_for`s it to exist before editing.

Jellyfin's dashboard, SyncPlay and remote control use WebSockets. The shared
`brrewery_nginx_site` role gained an opt-in `brrewery_app_websocket` flag
(default **off**, so every other app's vhost is byte-for-byte unchanged). When
enabled, the location adds:

```nginx
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection $http_connection;
```

`proxy.conf` already sets `proxy_http_version 1.1` (required for the upgrade).
`$http_connection` forwards the client's own `Connection` header, which avoids
needing an http-level `map` block (one can't live inside a `location` include) and
keeps keepalive intact for ordinary, non-WebSocket requests. Jellyfin enables the
flag; the upgrade playbook re-asserts the site so an older install gains the block.

## Media access (jellyfin user joins the brrewery group)

Jellyfin runs as its own `jellyfin` system user. Like Plex, install/upgrade:

- add the `jellyfin` user to the brrewery user's primary group
  (`brrewery_group`); the service is (re)started afterwards so it picks up the
  new supplementary group; and
- make a **single**, **non-recursive** permission change — the brrewery home
  directory from `0700` to `0750` — so the group can traverse into the media.
  Only done when the home is exactly `0700`; any other mode is left as the
  operator set it, and nothing inside the home is touched.

`remove` reverts that home directory back to `0700`, but only if it is still the
`0750` brrewery set.

## Remove

Stops/disables `jellyfin.service`, deletes the nginx location (reloading nginx),
then `apt purge … autoremove` over `jellyfin`, `jellyfin-server`, `jellyfin-web`
and `jellyfin-ffmpeg7`. `apt purge` of `jellyfin-server` removes the four data
dirs (`/var/lib`, `/etc`, `/var/cache`, `/var/log` `…/jellyfin`) and the
`jellyfin` user, but does **not** remove the deb822 source, the keyring or
`/etc/default/jellyfin` — so remove deletes those explicitly, and deletes the
four data dirs explicitly too (clean even if a purge is partial). Finally it
reverts the brrewery home mode.
