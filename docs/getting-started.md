# Getting started

## Prerequisites

- macOS.
- Docker Desktop or another local Docker Engine socket.
- `mkcert` installed and trusted locally.
- `sudo` access for install/uninstall/start/stop flows.
- Go 1.26 via `mise`.

## Bootstrap

```bash
mise install
mise run test
mise run build
```

`mise.toml` pins Go 1.26 and defines `fmt`, `test`, and `build` tasks.

## First successful run

Build the binary and install the daemon:

```bash
sudo ./devproxy install --with-menubar
```

Then verify the runtime:

```bash
devproxy status
devproxy doctor
devproxy routes
```

If you want the browser UI, run:

```bash
devproxy dashboard --open
```

## What to expect

- `status` prints daemon health, DNS, HTTP/HTTPS listener state, and route counts.
- `doctor` runs checks for Docker, launchd, the admin socket, resolver state, listeners, `mkcert`, and managed-domain resolution.
- `routes` lists active hostnames and upstreams.

## Next steps

- See [configuration](./configuration.md) for YAML and environment overrides.
- See [CLI reference](./cli.md) for command flags.
- See [troubleshooting](./troubleshooting.md) if install or startup fails.
