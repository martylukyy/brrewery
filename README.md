<h1 align="center">
  <img alt="brrewery logo" src="web/public/logos/brrewery.webp" width="160px"/><br/>
  brrewery
</h1>

<p align="center">brrewery is a web interface for installing, managing, updating and removing apps on your server. Built on Go and React, driving battle-tested Ansible playbooks under the hood.</p>

<p align="center"><img alt="GitHub release (latest by date)" src="https://img.shields.io/github/v/release/autobrr/brrewery?style=for-the-badge">&nbsp;<img alt="GitHub all releases" src="https://img.shields.io/github/downloads/autobrr/brrewery/total?style=for-the-badge">&nbsp;<img alt="GitHub Workflow Status" src="https://img.shields.io/github/actions/workflow/status/autobrr/brrewery/release.yml?style=for-the-badge"></p>

## Table of Contents

1. [What Is brrewery?](#what-is-brrewery)
2. [Key Features](#key-features)
   - [Available Apps](#available-apps)
3. [Installation](#installation)
   - [Requirements](#requirements)
   - [Install Script](#install-script)
   - [Environment Variables](#environment-variables)
4. [Contributing](#contributing)
5. [Code of Conduct](#code-of-conduct)
6. [License](#license)

## What Is brrewery?

Setting up a seedbox or media server by hand means installing, configuring, and wiring together a dozen applications — download clients, the *arr suite, media servers, and the web server in front of them. brrewery takes care of all of that from the comfort of your browser.

brrewery runs as a small daemon on your server and serves a web dashboard through nginx. From the dashboard you can install, manage, update, and remove apps with a click — each operation is executed by an Ansible playbook, so installs are reproducible and idempotent. Instead of tracking state in files that can drift or be tampered with, brrewery detects what's installed by querying the filesystem for each app's executables and dependencies.

There is no database and no config file to maintain. The dashboard is always served by nginx at `/`, on port 80 for HTTP and port 443 for HTTPS, with automatic Let's Encrypt certificates when you provide a domain.

## Key Features

- Easy to use and mobile friendly web UI (with dark mode!) to manage everything
- One-command install script that sets up the daemon, nginx, and TLS
- App installs driven by Ansible playbooks — reproducible, idempotent, and easy to audit
- Installed apps detected from the filesystem, not from tamperable state files
- No database and no config file — nothing to migrate, nothing to corrupt
- Automatic HTTPS via Let's Encrypt (acme.sh), with a self-signed fallback
- Multi-user support with bcrypt-hashed credentials; app secrets (API keys, tokens) are never persisted — they are prompted for at install time only
- Built-in self-update and per-app updates
- System metrics on the dashboard, including network traffic via vnstat
- Built on Go and React, making brrewery lightweight with a single static binary

### Available Apps

- **Download clients:** qBittorrent, Deluge, Transmission, rTorrent, ruTorrent, SABnzbd
- **The \*arr suite:** Sonarr, Radarr, Lidarr, Prowlarr
- **Automation:** autobrr, qui
- **Media servers:** Jellyfin, Plex
- **Utilities:** Filebrowser

Missing an app? Playbooks live in [`ansible/playbooks/apps`](ansible/playbooks/apps) — adding a new one is the most common contribution. See [CONTRIBUTING.md](CONTRIBUTING.md) to get started.

## Installation

### Requirements

- A Debian 13 instance with amd64 architecture
- Root access
- Optionally: A domain that resolves to the host for automatic Let's Encrypt certificates

### Install Script

Download and run the installer as root:

```bash
git clone https://github.com/autobrr/brrewery.git
cd brrewery
sudo ./install.sh
```

The installer fetches the latest release, installs all dependencies (nginx, Ansible, vnstat, and friends), configures nginx and TLS, sets up the systemd service, and prompts you to create the initial admin user. When it finishes, open `https://your-server` and log in with the account you just created.

If you provide a domain during the install (or via `BRREWERY_DOMAIN`, see below), a Let's Encrypt certificate is issued and renewed automatically. Without a domain, the dashboard falls back to a self-signed certificate — you can re-run the installer at any time to add one later.

The installer is idempotent: re-running it upgrades brrewery to the latest release without touching your apps or users.

### Environment Variables

The install script accepts a few optional environment variables for non-interactive runs:

| Variable | Description |
| --- | --- |
| `BRREWERY_DOMAIN` | Domain to issue a Let's Encrypt certificate for. Must already resolve to the host. Empty skips issuance. |
| `BRREWERY_VERSION` | Release tag to install (e.g. `v1.2.0`). Empty resolves the newest published release, pre-releases included. |
| `BRREWERY_REPO_URL` | Repository to fetch releases from. Defaults to the official repo. |

Example:

```bash
sudo BRREWERY_DOMAIN=brrewery.example.com ./install.sh
```

The brrewery daemon itself needs no environment variables or CLI flags in production — everything is set up by the installer.

## Contributing

Whether you're fixing a bug, adding a feature, adding an app playbook, or improving documentation, your help is appreciated. Here's how you can contribute:

### Reporting Issues and Suggestions

- **Report Bugs:** Encountered a bug? Please open an issue with detailed steps to reproduce, expected behavior, and any relevant screenshots or logs.
- **Feature Requests:** Open an issue describing your idea and how it will improve `brrewery`.

### Code Contributions

Check out the full guide for contributing [here](CONTRIBUTING.md).

- **Fork and Clone:** Fork the `brrewery` repository and clone it to start working on your changes.
- **Branching:** Create a new branch for your changes. Use a descriptive name for easy understanding.
- **Coding:** Ensure your code is well-commented for clarity.
- **Commit Guidelines:** We appreciate the use of [Conventional Commit Guidelines](https://www.conventionalcommits.org/en/v1.0.0/#summary) when writing your commits.
  - There is no need for force pushing or rebasing. We squash commits on merge to keep the history clean and manageable.
- **Pull Requests:** Submit a pull request with a clear description of your changes. Reference any related issues.
- **Code Review:** Be open to feedback during the code review process.

## Code of Conduct

We follow a code of conduct that promotes respectful and harassment-free experiences. Please read [our Code of Conduct](CODE_OF_CONDUCT.md) before participating.

## License

brrewery is proudly open-source and is released under the [GNU Affero General Public License v3.0 (AGPLv3)](https://www.gnu.org/licenses/agpl-3.0.html). 