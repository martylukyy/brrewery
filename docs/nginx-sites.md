# nginx site layout

brrewery ships nginx configuration under `contrib/nginx/`, deployed to `/etc/nginx/` by `scripts/install.sh`.

## sites-available / sites-enabled

The dashboard vhost lives in:

- `sites-available/brrewery.conf`
- enabled via symlink: `sites-enabled/brrewery.conf` → `../sites-available/brrewery.conf`

The install script runs:

```bash
ln -sf ../sites-available/brrewery.conf /etc/nginx/sites-enabled/brrewery.conf
nginx -t && systemctl reload nginx
```

## Routing

| Path | Target |
|------|--------|
| `/`, `/login` | Static SPA shell (`index.html`) at `/var/www/brrewery`, `200` |
| `/assets/…`, other real files | Served directly from `/var/www/brrewery` |
| `/api/` | Go backend `127.0.0.1:8080` |
| `/health` | Go backend health endpoint |
| `/autobrr/` (and other installed apps) | Reverse-proxied via snippets in `/etc/nginx/brrewery/apps/` |
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

Installed apps add nginx location snippets under `/etc/nginx/brrewery/apps/` via the Ansible `brrewery_nginx_site` role. The dashboard vhost includes them before the SPA catch-all:

```nginx
include /etc/nginx/brrewery/apps/*.conf;
```

## nginxconfig.io snippets

Shared snippets are under `nginxconfig.io/` (`general.conf`, `security.conf`, `proxy.conf`, `ssl.conf`), following the [nginxconfig.io](https://github.com/digitalocean/nginxconfig.io) layout.

Per-app reverse-proxy snippets are installed by Ansible playbooks using the `brrewery_nginx_site` role.
