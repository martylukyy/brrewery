# Handoff — libtorrent 2.1 (RC_2_1) option for Deluge

Status: **incomplete, do not ship as-is.** The option is wired end to end and the
build produces a working libtorrent 2.1 module, but selecting it yields a Deluge
that cannot start without an additional patch that is not in the role.

Date: 2026-08-04. Verified on this host: Debian 13, Deluge 2.2.0, Python 3.13.5.

---

## What works

libtorrent `v2.1.0` builds cleanly through the existing `deluge_build` path — no
build-system changes were needed:

- `bindings/python/Jamfile` at `v2.1.0` still defines the `install_module` target
  and the `python-install-path` feature that `libtorrent.yml` drives.
- `boost.yml` has no branch conditionals, so RC_2_1 reuses the staged Boost tree.
- Boost stays pinned at `1_86_0`: 2.1 needs >= 1.75 and does not use the
  `boost::asio::io_service` that caps RC_1_2/RC_1_1 at 1.86.
- The built module imports and reports `2.1.0.0`.

## What is broken

### 1. Deluge cannot construct its session (blocker)

`deluge/core/core.py:120` hardcodes a setting libtorrent 2.1 removed:

```python
settings_pack = {
    'peer_fingerprint': peer_id,
    'user_agent': user_agent,
    'ignore_resume_timestamps': True,   # removed in libtorrent 2.1
}
self.session = lt.session(settings_pack, flags=0)
```

Result: `deluged` exits 1 at startup with

```
Unable to start deluged: 'unknown name in settings_pack: ignore_resume_timestamps'
```

and `Restart=on-failure` crash-loops it. This is unpatched Deluge 2.2.0 against
libtorrent 2.1 — not a brrewery bug, and not fixed by choosing a different Deluge
line. Removing that single key is sufficient to get the daemon running; a session
then comes up healthy (DHT populated, RPC listening, WebUI 200).

If RC_2_1 is kept, this needs a role patch in `deluge_build/tasks/deluge.yml`
gated on `dz_libtorrent_branch == 'RC_2_1'`, following the existing base-path
patch pattern there (patch the installed venv copy, plus a grep-based build-time
guard so a silently-missed patch fails the build instead of the daemon).

### 2. ltConfig does not work on 2.1

```
Failed to start plugin: ltConfig
AttributeError: module 'libtorrent' has no attribute 'version_major'
```

`version_major` / `version_minor` are also gone in 2.1. Deluge disables the
plugin; it stays listed as available but never starts. Any plugin touching
removed libtorrent symbols has the same problem — patching Deluge core does not
help here.

### 3. The compatibility cut in the manifest is unverified

The manifest currently offers RC_2_1 on `2.2.x` and `2.1.x`, reasoned from
Deluge 2.1.0's "Remove libtorrent deprecated functions" changelog entry. That
entry covers deprecated *functions*, not removed `settings_pack` entries, so the
reasoning does not actually establish compatibility — issue 1 above disproves it.
Re-derive this cut (or drop the branch) before shipping. Only 2.2.x was built and
tested; 2.1.x was never exercised.

---

## Changes in the working tree

| File | Change |
|---|---|
| `ansible/roles/deluge_build/files/deluge/manifest.yml` | `RC_2_1` branch (`tag: v2.1.0`, `boost: 1_86_0`) on the 2.2.x and 2.1.x lines |
| `internal/apps/deluge/manifest.go` | `BranchRC21` const, `branchOrder` for stable picker ordering |
| `internal/apps/deluge/options.go` | Branch choices derived from the manifest instead of hardcoded; per-choice gating via `branchChoices` |
| `internal/apps/model/types.go` | `When` added to `InstallOptionChoice` |
| `web/src/lib/api.ts` | `when?` on the `InstallOptionChoice` TS type |
| `web/src/components/install-options-modal.tsx` | Filters branch choices by selected version; resets a selection the new version disallows |
| `internal/apps/deluge/manifest_test.go`, `versionresolve_test.go` | RC_2_1 manifest shape, per-choice `When`, `Validate` rejects `2.0.x + RC_2_1` |
| `web/src/components/install-options-modal.test.tsx` | Deluge fixture + 3 tests: gated choice offered / hidden / selection reset |

Tests pass (`go test ./internal/...`, 95 frontend tests). `make lint` was not run
— `golangci-lint` is not installed on this host.

The per-choice `When` mechanism is independently useful and worth keeping even if
RC_2_1 is dropped: without it, a branch added to only some lines is still offered
on all of them and only rejected server-side by `Validate`.

## Decision still open

1. **Drop RC_2_1** — only offer branches that work unpatched. Revert the manifest
   entry; optionally keep the per-choice `When` mechanism.
2. **Keep it + patch `core.py`** in the role, accepting that ltConfig and similar
   plugins stay broken.
3. **Keep it + patch + warn in the UI** — needs a description/warning field on
   `InstallOptionChoice`, which the model does not have today.

---

## State of this host

`/opt/deluge` currently runs Deluge 2.2.0 on libtorrent 2.1.0.0, **hand-patched**:
`ignore_resume_timestamps` was removed from
`/opt/deluge/lib/python3.13/site-packages/deluge/core/core.py` and its stale
bytecode dropped. This edit is not in the role — **the next rebuild reverts it and
the daemon returns to crash-looping.** Original at
`scratchpad/core.py.bak` (session-scoped, will not survive indefinitely; it is
just the stock file from the `deluge-2.2.0` tag).

ltConfig (`~/.config/deluge/plugins/ltConfig-2.1.1-py3.13.egg`) is installed and
available but disabled, per issue 2.

To get back to a supported state, reinstall on RC_1_2 or RC_2_0 from the UI.

## Fixed separately (not part of this handoff)

`ansible/playbooks/apps/deluge/install.yml` used `state: started` for `deluged`.
Since the playbook doubles as a re-install and the role wipes the venv underneath
a possibly-running daemon, `started` was a no-op that left deluged executing
deleted code — reporting the new version from disk while the process still held
the old `libtorrent.so`. It is now `state: restarted`. This is what made the 2.1
upgrade look successful when the daemon had never actually moved off 1.2.20.
