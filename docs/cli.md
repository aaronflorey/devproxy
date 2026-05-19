# CLI reference

## Global flags

- `--config <path>`: load YAML config from the given file.

## Commands

| Command | Purpose | Notable flags / output |
| --- | --- | --- |
| `print-config` | Print the effective config summary. | Prints `suffix=... root_services=...`. |
| `daemon` | Run the foreground daemon. | `--admin-socket`, `--http-address`, `--https-address`. |
| `install` | Install launchd integration on macOS. | `--with-menubar`; re-execs via `sudo` when needed. |
| `uninstall` | Remove launchd integration and local artifacts. | `--with-menubar`, `--yes`. |
| `start` | Start the installed daemon service. | Re-execs via `sudo` when needed. |
| `stop` | Stop the installed daemon service. | Re-execs via `sudo` when needed. |
| `status` | Show daemon health and route counts. | `--admin-socket`. |
| `doctor` | Run diagnostics. | `--admin-socket`, `--example-host`. |
| `routes` | List active route mappings. | `--admin-socket`. |
| `logs` | Print current daemon-session log events. | `--admin-socket`; no persisted history in v1. |
| `refresh` | Trigger a full container rescan. | `--admin-socket`. |
| `dashboard` | Run the local dashboard. | `--admin-socket`, `--listen`, `--open`. |
| `menubar` | Run the macOS menu bar runtime. | `--admin-socket`. |

## Command details

### `install`

```bash
sudo ./devproxy install --with-menubar
```

This writes the resolver file, bootstraps TLS prerequisites, stages the binary to `/usr/local/bin/devproxy`, and installs the system daemon. With `--with-menubar`, it also installs the per-user menu bar app and LaunchAgent.

### `daemon`

Defaults:

- admin socket: `/tmp/devproxy/admin.sock`
- HTTP: `127.0.0.1:80`
- HTTPS: `127.0.0.1:443`

### `dashboard`

Defaults to `127.0.0.1:45831`. The listen host must be `127.0.0.1` or `localhost`. If stdout is a TTY, the command opens the browser automatically on macOS unless `--open` is explicitly set.

### `doctor`

Checks include Docker, launchd, the admin socket, resolver state, HTTP/HTTPS listeners, `mkcert`, local CA state, and managed-domain resolution.

### `refresh`

Sends a POST to the local admin API and prints whether the daemon accepted and completed the rescan.
