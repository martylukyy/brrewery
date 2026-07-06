# Deluge build manifest and patches

Deluge is a Python application, so the only thing compiled from source is its
`libtorrent-rasterbar` (arvidn/libtorrent) dependency — built **with Python
bindings**, statically linked, and tuned with the same compiler flags as
qBittorrent. Deluge itself is installed from its GitHub source tag into a
self-contained virtualenv that also holds the built libtorrent module. Sources
are **not** committed in git; the Ansible role clones/downloads them during each
install or upgrade.

## In-repo layout

```text
ansible/roles/deluge_build/files/deluge/
  manifest.yml                      # one entry per Deluge version line
  patches/
    libtorrent-RC_1_1-Jamfile.patch # adds the install_module target to libtorrent 1.1 (1.3 line)
```

At install time the venv lives at `/opt/deluge`; the shared Boost source tree is
cached under `/opt`. The 1.3 line additionally builds a private CPython 2.7 and
OpenSSL 1.1 under `/opt/brrewery-python27` and `/opt/brrewery-openssl11`.

## manifest.yml

Each line builds the **latest patch** of a Deluge version series. Only the
Deluge release itself is resolved from upstream; every build **dependency** is
pinned per line in `manifest.yml` (lockfile-style, like a `package.json`) so
builds are reproducible instead of tracking upstream "latest". Each line is
self-contained — there is no shared `defaults` block:

| Selection | Source |
|-----------|--------|
| Deluge release | Resolved from the newest `deluge-{series}.*` tag on GitHub, passed as `deluge_release` (see `internal/apps/deluge`) |
| libtorrent branch | User choice (`libtorrent_branch`), else the line default |
| libtorrent version | Pinned per branch as `libtorrent.branches.<branch>.tag`, read from the manifest by Ansible and cloned by tag (RC_1_2 → `v1.2.20`, RC_2_0 → `v2.0.11`, RC_1_1 → `libtorrent-1_1_14`) |
| Boost | Pinned per libtorrent branch as `libtorrent.branches.<branch>.boost` (all currently `1_86_0`) |
| compiler flags | Pinned per line as `compiler_flags` |
| OpenSSL 1.1 / CPython 2.7 | Pinned as `openssl11_version` / `python27_version` on the legacy 1.3 line only |

| Deluge line | Python | libtorrent branches | Notes |
|-------------|--------|---------------------|-------|
| 2.2.x / 2.1.x / 2.0.x | system python3 | RC_1_2 (default), RC_2_0 | `pip` resolves the Deluge dependency stack |
| 1.3.x | vendored python2.7 | RC_1_1 | Python 2 only; CPython 2.7 + OpenSSL 1.1 built from source |

The Go API reads the manifest for the install wizard (version + libtorrent
branch pickers) and validation; Ansible reads it to drive the build.

## libtorrent build

The python bindings are built with Boost.Build (`b2 … install_module`), static
libtorrent + Boost, and `-O3 -mtune=native` (each line's `compiler_flags` —
**never** `-march=native`, matching the qBittorrent build). On Debian's system
OpenSSL 3 the static `libcrypto.a` references zstd/zlib symbols that b2 does not
link, so the runtime sonames are appended (`-l:libzstd.so.1 -l:libz.so.1`); the
legacy line instead links the clean vendored OpenSSL 1.1 directly.

### The 1.3 (Python 2) line

libtorrent RC_1_1 predates `install_module` and a modern toolchain, so three
fixes are applied at build time: the vendored Jamfile patch (adds the target),
`#include <map>` in two headers (GCC 14 transitive-include hygiene), and bringing
Boost's `_1.._N` placeholders into scope in the two binding sources that use them.

## Maintainers

The Deluge `.x` release still tracks the newest upstream patch automatically, so
no manifest edits are needed for new Deluge patch releases. Build **dependencies**
are pinned, so bump them explicitly when upstream ships a newer compatible
release: update a line's `libtorrent.branches.<branch>.{tag,boost}` (RC_1_2/RC_1_1
stay ≤ Boost 1.86 for `boost::asio::io_service`), its `compiler_flags`, or the
legacy line's `openssl11_version` / `python27_version`. Because each line is
self-contained, apply the change to every affected line. Also edit `manifest.yml`
when adding a new series or adjusting the per-line build profile.
