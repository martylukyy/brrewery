# nginx site layout

brrewery ships nginx configuration under `contrib/nginx/`, deployed to `/etc/nginx/` by `scripts/install.sh`.

## sites-available / sites-enabled

The dashboard vhost **is** nginx's default site — it replaces the distro's
`default` vhost rather than living alongside it:

- `sites-available/default`
- enabled via symlink: `sites-enabled/default` → `../sites-available/default`

The default site name has no `.conf` extension, so the app-snippet glob inside the
vhost (`sites-enabled/*.conf`) never re-includes it; `nginx.conf` includes the vhost
explicitly by that name. The install script runs:

```bash
install -m 0644 contrib/nginx/sites-available/default /etc/nginx/sites-available/default
ln -sf ../sites-available/default /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx
```

## Routing

| Path | Target |
|------|--------|
| `/`, `/login` | Static SPA shell (`index.html`) at `/var/www/brrewery`, `200` |
| `/assets/…`, other real files | Served directly from `/var/www/brrewery` |
| `/api/` | Go backend `127.0.0.1:8080` |
| `/health` | Go backend health endpoint |
| `/autobrr/` (and other installed apps) | Reverse-proxied via `<id>.conf` location snippets enabled in `/etc/nginx/sites-enabled/` |
| anything else | `404` serving the SPA shell, so the in-app React 404 renders |

The dashboard is a TanStack Router SPA. Its **known routes** (`/`, `/login`) each
get an exact `location` that serves `index.html` with a `200`. Real static files
are served directly. Every other path hits `location /`'s `try_files … =404`,
which `error_page 404 @spa_notfound` routes to a named location that returns the
`index.html` shell **with the 404 status preserved** (no `=code`) plus
`Cache-Control: no-store` and the shared security headers, so the client router
boots and renders its `<NotFound/>` page against a true 404. Add a `location =
/<route>` line whenever a client route is added to `web/src/router.tsx`.

HTTP (port 80) redirects to HTTPS (port 443). TLS material defaults to `/etc/ssl/brrewery/fullchain.pem` and `privkey.pem` (self-signed on first install).

## App reverse proxies

Installed apps add nginx location snippets via the Ansible `brrewery_nginx_site` role. Each snippet is written to `sites-available/<id>.conf` and enabled by a symlink into `sites-enabled/`, mirroring the dashboard vhost's layout:

- `sites-available/radarr.conf`
- enabled via symlink: `sites-enabled/radarr.conf` → `../sites-available/radarr.conf`

These snippets are **location blocks, not server blocks** — the dashboard vhost includes the enabled ones inside its `server {}` before the SPA catch-all:

```nginx
include /etc/nginx/sites-enabled/*.conf;
```

Because that glob runs **inside** the `server {}` block, only `location`-level snippets may live in `sites-enabled/` as `.conf` files. The dashboard vhost itself is the lone server block, so it is enabled as the **default site** (`sites-enabled/default`, no `.conf` extension) and included explicitly from `nginx.conf` — that keeps the `*.conf` glob from re-including the vhost (which would nest a `server {}` inside a `server {}`) and keeps a `location` snippet from ever being loaded at the http context (which fails with *"location directive is not allowed here"*). Removing an app deletes both the symlink and the `sites-available` file and reloads nginx.

## Shared snippets

Shared snippets (`general.conf`, `security.conf`, `proxy.conf`, `ssl.conf`, adapted from [nginxconfig.io](https://github.com/digitalocean/nginxconfig.io)) live directly under `/etc/nginx/`. `nginx.conf` includes them explicitly — not via a glob, since `/etc/nginx/*.conf` would match `nginx.conf` itself and recurse.

Per-app reverse-proxy snippets are installed by Ansible playbooks using the `brrewery_nginx_site` role.
