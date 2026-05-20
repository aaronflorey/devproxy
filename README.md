# DevProxy

[![CI](https://github.com/aaronflorey/devproxy/actions/workflows/ci.yaml/badge.svg)](https://github.com/aaronflorey/devproxy/actions/workflows/ci.yaml)
[![License](https://img.shields.io/github/license/aaronflorey/devproxy)](./LICENSE)

DevProxy is a macOS-native developer tool that watches Docker Compose projects, assigns local vanity domains like `acme-crm.test`, serves local DNS, and proxies HTTP and HTTPS traffic to the correct published localhost port.

## What It Does

- macOS only
- Built as a Go CLI with an optional menu bar runtime
- Designed for local Docker Desktop workflows

## Documentation

- [Getting started](./docs/getting-started.md)
- [CLI reference](./docs/cli.md)
- [Configuration](./docs/configuration.md)
- [Troubleshooting](./docs/troubleshooting.md)
- [Architecture](./docs/architecture.md)

## Prerequisites

- macOS
- Docker Desktop or a compatible local Docker Engine socket
- `mkcert` installed and trusted for local HTTPS
- `sudo` access for install and uninstall flows

## Install

Install via Homebrew:

```bash
brew install aaronflorey/tap/devproxy
```

Or, if you use [`bin`](https://github.com/aaronflorey/bin):

```bash
bin install https://github.com/aaronflorey/devproxy
```

If you do not have `bin` installed, see [`aaronflorey/bin`](https://github.com/aaronflorey/bin).

Install the daemon and optional menu bar app:

```bash
sudo devproxy install --with-menubar
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
sudo devproxy install --with-menubar
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

## Development

Build from source:

```bash
mise install
mise run fmt
mise run test
mise run build
```

Useful commands:

```bash
go run . status
go run . doctor
go run . routes
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## Security

See [SECURITY.md](./SECURITY.md).
