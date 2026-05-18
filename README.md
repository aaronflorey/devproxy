# DevProxy

[![CI](https://github.com/aaronflorey/devproxy/actions/workflows/ci.yaml/badge.svg)](https://github.com/aaronflorey/devproxy/actions/workflows/ci.yaml)
[![Release](https://img.shields.io/github/v/release/aaronflorey/devproxy?display_name=tag)](https://github.com/aaronflorey/devproxy/releases)
[![License](https://img.shields.io/github/license/aaronflorey/devproxy)](./LICENSE)

DevProxy is a macOS-native developer tool that watches Docker Compose projects, assigns local vanity domains like `acme-crm.test`, serves local DNS, and proxies HTTP and HTTPS traffic to the correct published localhost port.

## Status

- macOS only
- Built as a Go CLI with an optional menu bar runtime
- Designed for local Docker Desktop workflows

## Prerequisites

- macOS
- Docker Desktop or a compatible local Docker Engine socket
- `mkcert` installed and trusted for local HTTPS
- `sudo` access for install and uninstall flows

## Install

From source:

```bash
mise install
mise run build
sudo ./devproxy install --with-menubar
```

Homebrew release packaging is configured for `aaronflorey/homebrew-tap` and becomes usable after tagged releases are published.

## Development Setup

```bash
mise install
mise run test
```

Useful commands:

```bash
mise run fmt
mise run build
go run . status
go run . doctor
go run . routes
```

## Configuration

DevProxy reads configuration from environment variables and an optional YAML config file passed with `--config`.

Minimal example:

```yaml
domain_suffix: test
root_services:
  - app
  - web
serving:
  managed_suffix: test
  redirect_http_to_https: false
```

## Usage

Install the daemon and optional menu bar app:

```bash
sudo ./devproxy install --with-menubar
```

Inspect runtime health:

```bash
devproxy status
devproxy doctor
devproxy routes
```

Remove the installed services:

```bash
sudo devproxy uninstall --with-menubar --yes
```

## Releases

- Conventional commits drive release automation through `release-please`
- Tags are created as `vX.Y.Z`
- Tagged releases publish GitHub release artifacts with GoReleaser
- Homebrew formula updates are published to `aaronflorey/homebrew-tap`

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## Security

See [SECURITY.md](./SECURITY.md).
