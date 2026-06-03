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
| `/` | Static SPA at `/var/www/brrewery` |
| `/api/` | Go backend `127.0.0.1:8080` |
| `/health` | Go backend health endpoint |
| `/autobrr/` (and other installed apps) | Reverse-proxied via snippets in `/etc/nginx/brrewery/packages/` |

HTTP (port 80) redirects to HTTPS (port 443). TLS material defaults to `/etc/ssl/brrewery/fullchain.pem` and `privkey.pem` (self-signed on first install).

## Package reverse proxies

Installed packages add nginx location snippets under `/etc/nginx/brrewery/packages/` via the Ansible `brrewery_nginx_site` role. The dashboard vhost includes them before the SPA catch-all:

```nginx
include /etc/nginx/brrewery/packages/*.conf;
```

## nginxconfig.io snippets

Shared snippets are under `nginxconfig.io/` (`general.conf`, `security.conf`, `proxy.conf`, `ssl.conf`), following the [nginxconfig.io](https://github.com/digitalocean/nginxconfig.io) layout.

Per-package reverse-proxy snippets are installed by Ansible playbooks using the `brrewery_nginx_site` role.
