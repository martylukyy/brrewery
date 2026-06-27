# ruTorrent version manifest

ruTorrent is a PHP web UI with no compiled assets, so the only thing "built" is a
choice of which release tag to download. Unlike the qBittorrent/rtorrent build
manifests (which pin a toolchain), the ruTorrent manifest is a **compatibility
lockfile**: it pins, per rtorrent release series, the newest ruTorrent that still
works with that rtorrent. Sources are **not** committed in git; the Ansible role
downloads the chosen tag's source archive during each install or upgrade.

## Why it exists

ruTorrent's plugins drive rtorrent over XML-RPC/SCGI, so they are tied to the
rtorrent command surface. The `ratio` plugin is the canary: ruTorrent **≥ 5.3.2**
moved the ratio setters onto the rtorrent **0.16.x** `group.*` API and stopped
emitting a working call for the **0.15.x** `group2.rat_N.ratio.*.set("", value)`
surface. So an always-latest ruTorrent paired with an rtorrent ≤ 0.15.x build
leaves the ratio plugin dead out of the box ("Plugin failed to start"; the Ratio
Rules page throws `theWebUI.isCorrectRatio is not a function`). Upstream now
tracks rtorrent 0.16.x and won't keep supporting older rtorrent.
See [Novik/ruTorrent#3065](https://github.com/Novik/ruTorrent/issues/3065).

## In-repo layout

```text
ansible/roles/rutorrent/files/rutorrent/
  manifest.yml   # one entry per rtorrent series → compatible ruTorrent tag
```

## manifest.yml

`tasks/resolve.yml` reads the **installed** rtorrent version from its `-h` banner,
derives the series (`major.minor`), and selects the matching line. There is no
install-wizard picker — ruTorrent is pinned to whatever its rtorrent dependency
needs. Each line is one of:

| `resolve` | Resolution |
|-----------|------------|
| `latest`  | Newest upstream release tag from GitHub (the pre-fix behaviour) |
| `pinned`  | The manifest `tag` verbatim (lockfile-style) |

| rtorrent series | ruTorrent | Why |
|-----------------|-----------|-----|
| 0.16            | latest    | The `group.*` surface upstream now targets |
| 0.15            | `v5.3.1`  | Last release whose ratio plugin drives the 0.15.x setters (#3065) |
| 0.10, 0.9       | `v5.3.1`  | Predate the 0.16 `group.*` surface too |
| (anything else) | latest    | `default:` — newer/unknown rtorrent, or version unreadable |

The `default:` fallback means a detection miss never regresses a working 0.16+
pairing: it just behaves like the old always-latest install.

## Maintainers

The `0.16` and `default` lines track upstream automatically, so no edit is needed
for new ruTorrent patch releases there. Update `manifest.yml` only when:

- a newer ruTorrent is **verified** to work with a pinned rtorrent series (bump
  its `tag`), or
- brrewery adds a new rtorrent series that needs its own pin (add a line keyed by
  `rtorrent_series`).
