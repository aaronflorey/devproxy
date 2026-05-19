# Troubleshooting

## Install or service commands ask for root

**Symptom**

`devproxy install`, `uninstall`, `start`, and `stop` fail with a root-privilege message.

**Cause**

These commands require access to `/usr/local/etc/devproxy`, `/var/lib/devproxy`, `/var/log/devproxy`, `/etc/resolver`, and `/Library/LaunchDaemons`.

**Fix**

Run the command with `sudo` or from an account that can elevate.

## `mkcert` is missing or trust-store setup fails

**Symptom**

`mkcert not found` or `mkcert trust store install failed`.

**Cause**

HTTPS setup runs `mkcert -install` during install.

**Fix**

Install `mkcert`, make sure it is on `PATH`, and rerun install.

## Launchd preflight errors

**Symptom**

Messages like:

- `launchd preflight failed: plist ... does not exist`
- `launchd preflight failed: program ... does not exist`
- `launchd preflight failed: program ... is not executable`
- `daemon plist ... must be owned by root:wheel`
- `permissions are ...; group/other write bits must be disabled`

**Cause**

The plist or staged binary is missing, has the wrong owner, or has unsafe permissions.

**Fix**

Reinstall the binary, then inspect the plist and executable:

```bash
plutil -lint /Library/LaunchDaemons/com.devproxy.daemon.plist
launchctl print system/com.devproxy.daemon
```

## Dashboard refuses to start

**Symptom**

`dashboard listen host must be localhost or 127.0.0.1`.

**Cause**

The dashboard is intentionally limited to loopback addresses.

**Fix**

Use the default `127.0.0.1:45831` or another loopback host.

## `status`, `routes`, or `doctor` cannot reach the daemon

**Symptom**

Errors such as `admin socket unreachable` or `connect admin socket ...`.

**Cause**

The daemon is not running, or the socket path does not match `/tmp/devproxy/admin.sock`.

**Fix**

Run:

```bash
devproxy status
devproxy doctor
launchctl print system/com.devproxy.daemon
```

## Managed hostname returns a friendly 404 or 503

**Symptom**

- `No route is active for ... in devproxy.`
- `Routing is paused for ... in devproxy.`

**Cause**

Either no route is active for that hostname, or routing is paused.

**Fix**

Check active mappings with `devproxy routes` and runtime state with `devproxy status`.

## Config loading fails

**Symptom**

`read config: ...` or `decode config: ...`.

**Cause**

The YAML path is wrong, unreadable, or invalid.

**Fix**

Re-run with a valid `--config` path and verify the YAML structure in [configuration](./configuration.md).

## Need more detail

Use these commands in order:

```bash
devproxy status
devproxy doctor
devproxy routes
devproxy logs
devproxy refresh
```

For install-time failures, also inspect:

- `/var/log/devproxy/daemon.stderr.log`
- `/var/log/devproxy/daemon.stdout.log`
- `launchctl print system/com.devproxy.daemon`
