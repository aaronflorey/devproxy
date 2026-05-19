# Development workflow

## Setup

```bash
mise install
```

The repository pins Go 1.26 in `mise.toml`.

## Common loop

```bash
mise run fmt
mise run test
mise run build
```

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd/devproxy/` | Cobra CLI commands. |
| `internal/config/` | Config schema and defaults. |
| `internal/install/` | macOS install/uninstall, resolver, and launchd helpers. |
| `internal/daemon/` | Runtime reconciliation, Docker watching, and network startup. |
| `internal/adminapi/` | Unix-socket admin API server and client. |
| `internal/dashboard/` | Local dashboard server and templates. |
| `internal/proxy/` | HTTP/HTTPS proxy handlers. |
| `internal/dns/` | Authoritative DNS server for managed suffixes. |
| `internal/doctor/` | Diagnostic checks used by `doctor`. |

## Conventions called out by the repo

- Keep changes focused.
- Preserve macOS-only assumptions unless the task explicitly expands platform support.
- Prefer explicit failures over fallback behavior.
- Conventional commit prefixes are expected for release automation (`feat:`, `fix:`, `docs:`, `chore:`).

## Manual smoke checks

Useful commands while iterating after install or against a running daemon:

```bash
go run . print-config
go run . status
go run . doctor
go run . routes
```
