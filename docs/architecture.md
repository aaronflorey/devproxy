# Architecture

## Runtime shape

DevProxy is organized around one macOS daemon plus local clients:

1. The CLI installs launchd jobs and configures the resolver.
2. The daemon watches Docker, builds a routing snapshot, and starts local DNS/HTTP/HTTPS listeners.
3. The admin API exposes the snapshot to `status`, `routes`, `logs`, `doctor`, `dashboard`, and `menubar`.

## Main pieces

| Package | Role |
| --- | --- |
| `internal/install` | Writes `/etc/resolver/<suffix>`, installs launchd plists, stages the binary, and handles cleanup. |
| `internal/daemon` | Reconciles Docker state into routes and starts the runtime services. |
| `internal/routing` | Generates hostnames, merges overrides, and resolves conflicts. |
| `internal/dns` | Answers A records with `127.0.0.1` for managed hosts. |
| `internal/proxy` | Reverse-proxies managed HTTP/HTTPS traffic and returns friendly 404/503 messages. |
| `internal/adminapi` | Local Unix-socket JSON API. |
| `internal/dashboard` | Loopback-only dashboard UI and JSON polling endpoints. |
| `internal/menubar` | macOS menu bar UI built on `systray`. |

## Request flow

```text
Docker events / scan
  -> daemon reconciler
  -> routing snapshot
  -> DNS + HTTP/HTTPS runtime
  -> admin API
  -> CLI, dashboard, menubar
```

## Install flow

`devproxy install`:

1. Creates install directories.
2. Writes `/etc/resolver/<suffix>` to point the managed suffix at `127.0.0.1:53535`.
3. Runs `mkcert -install`.
4. Stages the executable at `/usr/local/bin/devproxy`.
5. Installs and starts the system daemon.
6. Optionally installs the menubar app and LaunchAgent.

## Routing notes

- Root services map to `project.suffix`.
- Other services map to `service.project.suffix`.
- Explicit domains are accepted through overrides or labels.
- Public-looking domains are rejected by routing warnings.
