# Plex

Plex is installed, upgraded and removed from the official Plex download API
rather than the Plex apt repository, and is claimed through a web browser rather
than a terminal claim token.

Playbooks: `ansible/playbooks/apps/plex/{install,upgrade,remove}.yml`.
Manifest: `internal/apps/catalog/manifests/plex.yaml`.

## Install / upgrade (download API, not apt)

1. Query `https://plex.tv/api/downloads/5.json` and read `computer.Linux.version`
   plus the `computer.Linux.releases` list.
2. Map the host architecture to a Plex build — `x86_64` → `linux-x86_64`,
   `aarch64` → `linux-aarch64` — and select the release with `distro == debian`
   and that `build`. Other architectures fail with a clear message.
3. Download that `.deb` and install it with `dpkg --install` (never apt).
   Upgrade compares the installed package version to the API version and only
   re-installs when they differ.
4. The `.deb` postinst drops `/etc/apt/sources.list.d/plexmediaserver.list` so the
   OS package manager can upgrade Plex; brrewery owns upgrades, so that source is
   removed. The `plexmediaserver.service` system unit is enabled and started.

The manifest's `requires_account_password: true` exists only to collect the
operator's sudo (become) password for the `dpkg` step; it is verified against the
brrewery account before the playbook runs.

## Media access (plex user joins the brrewery group)

Plex keeps running as its own `plex` system user. The brrewery user's media
folders already carry read permission for that user's group, so install/upgrade:

- add the `plex` user to the brrewery user's primary group (`brrewery_group`)
  and restart `plexmediaserver.service` so the running server picks up the new
  supplementary group; and
- make a **single** permission change — the brrewery home directory itself from
  `0700` to `0750`, **non-recursive** — so the group can traverse into the media.
  This is the only chmod brrewery performs, and only when the home is exactly
  `0700` (any other mode is left as the operator set it). Nothing inside the home
  is touched; the media's existing group-read is relied upon.

`remove` reverts that home directory back to `0700`, but only if it is still the
`0750` brrewery set. Plex's own data (`/var/lib/plexmediaserver`) stays owned by
`plex`. If an older install switched Plex to run as the brrewery user via a
systemd override, the playbooks remove that override so the packaged `plex` user
applies again.

## Web access & claiming (served on its own port, like swizzin)

Plex's web client addresses the server by origin (`scheme://host`, **no path**)
and serves its API from the root, so it cannot be reverse-proxied under a subpath
like `/plex/`. Rather than carry a dedicated proxy vhost for it, brrewery follows
[swizzin](https://github.com/swizzin/swizzin), which does **not** reverse-proxy
Plex at all: it is reached directly on its own port at `:32400/web`.

So brrewery installs **no nginx config** for Plex. The manifest's `web_path` is
the port-relative form `:32400/web`; the dashboard's Plex link
(`appUrl` in `web/src/lib/app-link.ts`) opens the current host on that port —
`http://<your-brrewery-host>:32400/web`. install/upgrade actively delete any
proxy/vhost a previous brrewery version wrote.

To claim: from a device on the **same network** as the server, open
`http://<your-brrewery-host>:32400/web` (the dashboard's Plex link, or
`http://127.0.0.1:32400/web` on the server itself) and sign in to your Plex
account. Plex permits claiming from the local network, so no SSH tunnel or
`plex.tv/claim` token is needed. (Claiming from outside the LAN would require LAN
access or a tunnel, since brrewery deliberately uses no claim token.)

## Remove

Stops/disables the service, deletes any leftover Plex nginx config (reloading
nginx), `dpkg --purge`s the package (only when installed; a real purge failure
fails the job), removes the apt source, reverts the home directory mode, and
deletes the Plex library/metadata/cache at `/var/lib/plexmediaserver`.
