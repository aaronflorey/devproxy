# DevProxy documentation

DevProxy is a macOS-only Go CLI and daemon for Docker Compose projects. It watches containers, assigns vanity domains like `acme-crm.test`, serves local DNS, and proxies HTTP/HTTPS traffic to the right localhost port.

## What this docs set covers

- How to get the project running locally.
- How configuration is loaded and where defaults come from.
- The CLI surface, admin socket API, and dashboard.
- Development, testing, and troubleshooting workflows.

## Start here

1. [Getting started](./getting-started.md)
2. [CLI reference](./cli.md)
3. [Configuration](./configuration.md)
4. [Troubleshooting](./troubleshooting.md)

## For maintainers

- [Development workflow](./development.md)
- [Testing](./testing.md)
- [Architecture](./architecture.md)
- [Admin API](./api.md)

## Known limitations

- v1 is macOS only.
- Install, uninstall, start, and stop require root privileges and use `sudo` re-exec when needed.
- The dashboard only listens on `127.0.0.1` or `localhost`.
- `port_preference_order` and `serving.redirect_http_to_https` are present in config, but no runtime consumer was found in the repository.
