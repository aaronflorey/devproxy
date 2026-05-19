# DevProxy Agent Notes

## Verification
- Use `mise` for local tooling. `mise.toml` pins Go `1.26`.
- CI parity is `mise run fmt && mise run test && mise run build`.
- `mise run fmt` is a check only: it fails when `gofmt -l .` prints files; it does not rewrite formatting.
- Trust `mise.toml` and `.github/workflows/ci.yaml` over `docs/testing.md`: that doc still mentions `test.sh` and `run.sh`, but those files do not exist.

## Runtime Shape
- This repo is intentionally macOS-only. Preserve launchd, `/etc/resolver`, menu bar, and local certificate assumptions unless the task is explicitly about platform expansion.
- Entry point is `main.go` -> `cmd/devproxy` Cobra commands.
- The real runtime split is:
  `internal/install` for resolver + launchd + staged binary install,
  `internal/daemon` for Docker watch/reconcile + DNS/HTTP/HTTPS listeners,
  `internal/adminapi` for the local Unix-socket API,
  `internal/dashboard` and `internal/menubar` as clients on top of that API.
- Default admin socket is `/tmp/devproxy/admin.sock`; the dashboard defaults to `127.0.0.1:45831`.

## High-Signal Paths
- Install paths come from `internal/install/paths.go`: `/usr/local/etc/devproxy`, `/var/lib/devproxy`, `/var/log/devproxy`, `/etc/resolver`, `/Library/LaunchDaemons`.
- Launchd labels are fixed in `internal/install/launchd.go`: `com.devproxy.daemon` and `com.devproxy.menubar`.
- `devproxy install` stages the CLI at `/usr/local/bin/devproxy`, writes `/etc/resolver/<suffix>`, runs `mkcert -install`, and installs the system daemon. `--with-menubar` also installs the per-user app/LaunchAgent.
- Several lifecycle commands re-exec with `sudo` when needed; prefer exercising the real CLI flows instead of manually editing launchd or resolver state.

## Config And Imports
- Config loading lives in `cmd/devproxy/root.go`: env vars use the `DEVPROXY_` prefix, and `--config` expects YAML.
- Keep the current module path/imports as `github.com/mochaka/devproxy` unless the task is explicitly a module rename. The repo directory/README use a different GitHub owner name.

## Release Automation
- Release automation expects conventional commit prefixes. Verified useful prefixes in repo docs/config: `feat:`, `fix:`, `docs:`, `chore:`.
- `release-please` drives tags from `release-please-config.json`; GoReleaser publishes darwin binaries only (`amd64`, `arm64`) and updates the Homebrew tap.
- `.goreleaser.yaml` runs `go test ./...` in `before.hooks`, and changelog entries exclude commits starting with `docs:` and `test:`.

## Useful Smoke Checks
- After install or when debugging a running daemon, the highest-signal commands are `go run . print-config`, `go run . status`, `go run . doctor`, `go run . routes`, and `go run . dashboard`.
