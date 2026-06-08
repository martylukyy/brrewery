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
Dependency versions are resolved by brrewery before Ansible runs, then fetched
and compiled by `ansible/roles/qbittorrent_build/tasks/vendor.yml`:

| Dependency | Resolution |
|------------|------------|
| qBittorrent patch | GitHub `release-{minor}.*` (`qbittorrent_release`) |
| Qt | Newest patch ≥ `qt.min` on download.qt.io (`qbittorrent_qt_version`) |
| zlib | Newest release on github.com/madler/zlib (`qbittorrent_zlib_version`) |
| Boost | Newest on archives.boost.io for `RC_2_0`; manifest `boost_rc_1_2` for `RC_1_2` (`qbittorrent_boost_version`) |
| OpenSSL | Newest 3.x on github.com/openssl/openssl (`qbittorrent_openssl_version`; 4.x excluded) |
| libtorrent | Tag from manifest for the selected branch |

The Go API reads the manifest for the install wizard and validation; Ansible
reads it to drive the build. Version resolution runs inside brrewery when a
package job starts.

## Maintainers

When upstream ships newer compatible releases, update `manifest.yml` (libtorrent
tags, `boost_rc_1_2` cap, Qt floors, etc.). Sources refresh automatically on the
next install.

## Patches

- The vendored `libtorrent-RC_*.patch` files are applied by default
  (best effort) when the user does not upload a custom libtorrent patch.
- A user-uploaded patch (Web UI) or an operator file under
  `/var/lib/brrewery/patches/qbittorrent/libtorrent-RC_*.patch` replaces the
  default for that build and **must** apply cleanly or the job fails.
- `qbittorrent-<version>-security.patch` files are brrewery-only and applied when
  the pinned release still requires them.
