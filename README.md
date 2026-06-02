# brrewery

Web interface for installing, managing, and monitoring packages on amd64 Linux hosts.

brrewery is a successor-style project to [swizzin](https://github.com/swizzin/swizzin), built with a Go backend and React frontend in the [autobrr](https://github.com/autobrr) project style.

## Quick start

```bash
# One-time host bootstrap (creates /var/lib/brrewery, nginx, systemd)
sudo ./scripts/install.sh

# Development (requires prod paths; see docs)
make dev
```

## Build

```bash
make build    # frontend + backend
make test
make test-openapi
```

## Documentation

- Internal engineering notes: [`docs/`](docs/)
- User-facing docs: Docusaurus project under `documentation/` (planned)

## License

GPL-2.0-or-later — see [LICENSE](LICENSE).
