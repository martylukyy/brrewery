# qBittorrent build manifest and patches

brrewery builds `qbittorrent-nox` from source against vendored copies of its
build dependencies. Source tarballs are **not** committed in git; the Ansible
role downloads and extracts them during each install or upgrade.

## In-repo layout

```text
ansible/roles/qbittorrent_build/files/qbittorrent/
  manifest.yml              # one entry per qBittorrent version line + pinned deps
  patches/
    libtorrent-RC_1_2.patch   # default settings_pack tuning (libtorrent 1.2)
    libtorrent-RC_2_0.patch   # default settings_pack tuning (libtorrent 2.0)
    qbittorrent-<version>-security.patch    # brrewery-supplied security backports (optional)
```

At install time, downloaded sources are cached under
`/usr/share/brrewery/vendor/qbittorrent/sources/` (not committed).

## manifest.yml

Each line builds the **latest stable patch** of a qBittorrent version series.
Only the qBittorrent patch itself is resolved from upstream; every build
**dependency** is pinned in `manifest.yml` (lockfile-style, like a `package.json`),
then fetched and compiled by `ansible/roles/qbittorrent_build/tasks/vendor.yml`:

Each line is self-contained: it pins all of its own dependency versions (there
is no shared `defaults` block).

| Dependency | Source |
|------------|--------|
| qBittorrent patch | Resolved from GitHub `release-{minor}.*`, passed as the `qbittorrent_release` extra var |
| Qt | Pinned per line as `qt`, read from the manifest by Ansible |
| zlib | Pinned per line as `zlib`, read from the manifest by Ansible |
| Boost | Pinned per line under the chosen libtorrent branch as `branches.<branch>.boost`, read from the manifest by Ansible |
| OpenSSL | Pinned per line as `openssl` (3.x), read from the manifest by Ansible |
| libtorrent | Pinned per line as `branches.<branch>.tag`, read from the manifest by Ansible |

The build role (`tasks/resolve.yml`) reads the pinned dependency versions
straight from the manifest line it loads — they are not passed as extra vars.
The Go API reads the same manifest for the install wizard and validation, and
before an app job starts it sets only the values Ansible cannot derive itself:
the resolved `qbittorrent_release` and the WebUI password hash.

## Maintainers

When upstream ships newer compatible releases, bump the pinned versions on the
affected line(s) in `manifest.yml`: `qt`, `zlib`, `openssl`, `compiler_flags`,
and each libtorrent branch's `tag` / `boost` (RC_1_2 stays capped at Boost
1.86). Sources refresh automatically on the next install.

## Patches

- The vendored `libtorrent-RC_*.patch` files are applied by default
  (best effort) when the user does not upload a custom libtorrent patch.
- A user-uploaded patch (Web UI) or an operator file under
  `/var/lib/brrewery/patches/qbittorrent/libtorrent-RC_*.patch` replaces the
  default for that build and **must** apply cleanly or the job fails.
- `qbittorrent-<version>-security.patch` files are brrewery-only and applied when
  the pinned release still requires them.
