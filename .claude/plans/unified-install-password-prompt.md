# Unified install password prompt

## Context

Installing autobrr and qBittorrent both ask the operator for the same secret —
their account password (the Linux user password, which equals the brrewery
dashboard password). Today that single concept is modelled two different ways:

| App | Secret key | Label | Verified? |
|-----|-----------|-------|-----------|
| autobrr | `brrewery_user_password` | "Brrewery password" | yes (`VerifyBrreweryPassword`) |
| qBittorrent | `ansible_become_password` | "Password" | **no** |

Two different keys, labels and behaviours produce two visibly different password
prompts and inconsistent validation (qBittorrent accepts any string). The keys
are actually the *same* credential: `extravars.ForInstall` already copies
`ansible_become_password` → `brrewery_user_password`, and the ansible runner
consumes `ansible_become_password` as the sudo (`become`) password — every real
playbook runs with `become: true`.

**Goal:** one canonical password prompt that all current and future apps reuse,
and always verify the entered value against the real account password — both
inline when the user submits the prompt *and* server-side when the install
starts (user chose "Both").

The React component ([install-secrets-modal.tsx](web/src/components/install-secrets-modal.tsx))
is already shared; the duplication lives in the backend catalog definitions and
the missing/uneven verification.

## Backend

### 1. Single canonical password secret — `internal/packages/catalog/catalog.go`
Add one shared definition and reuse it everywhere a password is needed:

```go
// passwordSecret is the single install-time password prompt shared by every
// package. It collects the operator's account password — the same value is the
// Linux user password, the sudo (become) password and the brrewery dashboard
// password — and is always verified against the brrewery account before install.
func passwordSecret() model.InstallSecret {
	return model.InstallSecret{
		Key:                    extravars.BecomePassword, // "ansible_become_password"
		Label:                  "Account password",
		Type:                   "password",
		VerifyBrreweryPassword: true,
	}
}
```

- `qbittorrentEntry()`: replace the inline `InstallSecret{Key: BecomePassword, …}`
  with `[]model.InstallSecret{passwordSecret()}` (adds the missing verification).
- autobrr entry: replace its `brrewery_user_password` secret with
  `passwordSecret()` (switches key to `ansible_become_password`).

**Why keying on `ansible_become_password` is correct:** it is the universal need
(sudo for any `become: true` playbook). `ForInstall` ([extravars.go:40](internal/packages/extravars/extravars.go))
already derives `brrewery_user_password` from it, so autobrr's `create-user`
([install.yml:105-124](ansible/playbooks/packages/autobrr/install.yml)) and
qBittorrent's WebUI hash ([qtresolve.go:171-181](internal/packages/qbittorrent/qtresolve.go))
keep working unchanged. Switching autobrr also *fixes a latent bug*: autobrr
previously never supplied a become password despite `become: true`.

Future apps opt in with `withInstallSecrets(entry(...), []model.InstallSecret{passwordSecret()})`.

### 2. Inline verify endpoint — `internal/api/handlers/auth.go` + `internal/api/server.go`
Add a side-effect-free endpoint that checks the *session user's* password:

- Handler `AuthHandler.VerifyPassword`: decode `{ "password": "..." }`; 400 if
  empty; resolve username via `h.auth.Username(r.Context())`; call
  `h.auth.VerifyPassword(username, password)` ([service.go:78](internal/auth/service.go));
  `401` ("Invalid credentials") on `auth.ErrInvalidPassword`, else `204 No Content`.
- Route: register `r.Post("/auth/verify-password", authHandler.VerifyPassword)`
  **inside the authenticated `r.Group`** ([server.go:68](internal/api/server.go)).
- `AuthHandler` needs no new deps (already holds `*auth.Service`, which exposes
  `Username`).

Server-side install validation already exists via `secrets.ValidateInstallSecrets`
([packages.go:81](internal/api/handlers/packages.go)) and now runs with
verification for both apps — that is the "at install start" half of "Both".

### 3. OpenAPI — `internal/web/swagger/openapi.yaml`
Add `/api/v1/auth/verify-password` (POST, `SessionAuth`, request body
`{password}`, responses `204` / `400` / `401`) next to `/auth/logout`
([openapi.yaml:402](internal/web/swagger/openapi.yaml)). Run `make test-openapi`.

## Frontend

Keep the single shared modal as THE password prompt; add inline verification.

### 4. API client — `web/src/lib/api.ts`
```ts
export async function verifyPassword(password: string): Promise<void> {
  await apiFetch<void>("/auth/verify-password", {
    method: "POST",
    body: JSON.stringify({ password }),
  });
}
```
(`apiFetch` already throws `ApiError` on non-2xx; a 204 body parses fine.)

### 5. `install-secrets-modal.tsx` — verify on submit
Make `handleSubmit` async:
1. Existing empty-field check.
2. For every secret with `verify_brrewery_password`, call `verifyPassword(value)`.
   On `ApiError` (401) set inline error "Incorrect password — enter your Linux /
   brrewery account password." and abort; do not call `onConfirm`.
3. On success call `onConfirm(values)` as today.

Add `verifying` state: disable the submit button and show "Verifying…" while the
request is in flight (guards against double-submit). Because secrets dedupe by
key ([app-shell.tsx:39](web/src/components/app-shell.tsx), `requiredSecrets`), a
multi-app install prompts and verifies once. No change needed to app-shell or the
phase flow.

## Tests

- **catalog_test.go** ([catalog_test.go:27-30](internal/packages/catalog/catalog_test.go)):
  autobrr's secret key is now `ansible_become_password`; assert key + label +
  `VerifyBrreweryPassword`. Add a parallel assertion for qBittorrent's secret now
  being verified.
- **install_test.go** ([internal/api/handlers/install_test.go](internal/api/handlers/install_test.go)):
  update any reference to the `brrewery_user_password` secret key for install
  payloads/expectations.
- **New** `auth` handler test: 204 on correct password, 401 on wrong, 400 on empty.
- **install-secrets-modal.test.tsx**: mock `verifyPassword`; assert it's called on
  submit, that a 401 shows the inline error and blocks `onConfirm`, and that
  success still calls `onConfirm`.
- `extravars`/`secrets`/qbittorrent enrich logic is unchanged, so their tests stay
  green; run them to confirm.

## Verification

1. `make lint` and `make test` (race) for touched Go packages:
   `./internal/packages/catalog/...`, `./internal/api/...`, `./internal/packages/secrets/...`.
2. `make test-openapi` (auth endpoint added).
3. Frontend: `pnpm --dir web test` for the modal + api specs.
4. `make build` to confirm the bundle + binary compile.
5. Manual smoke (optional, via `make dev`): select autobrr and qBittorrent
   together → one "Account password" prompt → wrong password shows inline error
   and blocks; correct password proceeds to qBittorrent options → install starts.
