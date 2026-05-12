# Phase 3 Research: Install, Daemon Lifecycle, and Diagnostics

**Phase:** 3
**Researched:** 2026-05-05
**Status:** Ready for planning

## Standard Stack

- **Go 1.26.x** for the CLI, foreground daemon path, install helpers, and local admin transport.
- **`github.com/moby/moby/client`** for Docker ping/list/event reachability checks and daemon bootstrap wiring.
- **Go stdlib `net`, `net/http`, `os`, `os/exec`, and `encoding/json`** for the UNIX-socket admin API, launchctl process execution, filesystem setup, and JSON command output.
- **`github.com/spf13/cobra`** for `daemon`, `install`, `uninstall`, `status`, `routes`, `refresh`, `doctor`, and `logs` subcommands.
- **Existing `internal/admin/*`, `internal/daemon/*`, and `internal/certs/mkcert.go` packages** as the authoritative runtime, projection, and fail-loud certificate sources.

## Architecture Patterns

1. **One daemon-owned control plane**: per D-01 and D-09, the daemon should own a local UNIX-socket HTTP+JSON admin API, and every operator surface should consume it instead of recomputing route or health state in-process.
2. **Foreground daemon validates before serving**: per D-02 and D-03, `devproxy daemon` should check Docker reachability, mkcert prerequisites, socket lifecycle, and listener binds up front, then exit with explicit errors instead of half-starting.
3. **launchd role separation**: per D-06 through D-08, the networking daemon belongs in the system daemon domain, while any future menu bar process belongs in the user agent domain and is installed only behind `--with-menubar`.
4. **Installer owns macOS integration, not runtime semantics**: per D-04, install/uninstall should manage directories, resolver files, launchd plists, and cert bootstrap choices without re-implementing reconcile/proxy logic.
5. **Doctor verifies system state, not just config files**: per D-10, diagnostics should inspect resolver registration via `scutil --dns`-aligned state, launchctl service state, admin socket reachability, and example hostname resolution from the live daemon view.

## Phase-Specific Requirements To Preserve

- `devproxy install` must create config, state, and log paths; configure the managed-suffix resolver; bootstrap certificates; register the daemon service; and start required services.
- `devproxy install --with-menubar` must install a separate menu bar LaunchAgent, while default install must not.
- `status`, `routes`, `refresh`, `doctor`, and `logs` must all talk to the same local admin API socket.
- `devproxy logs` is current-session only; persisted multi-session history remains deferred.
- `uninstall` must prompt for selective retention/removal of config, state, logs, and certificates instead of deleting everything unconditionally.
- DNS diagnostics must validate the configured resolver is actually active in macOS resolver state, not merely present on disk.

## Do Not Hand-Roll

- Do not build separate state-recompute paths for CLI commands when `internal/admin/*` already defines the read models to publish.
- Do not use TCP localhost for the admin surface; keep it on a local UNIX socket with explicit file mode and stale-socket cleanup per D-09.
- Do not use deprecated `launchctl load` / `unload`; use `bootstrap`, `bootout`, `enable`, `disable`, `kickstart`, and `print` flows.
- Do not mutate `/etc/resolv.conf`; install a managed `/etc/resolver/<suffix>` file and validate system resolver activation separately.
- Do not silently continue when mkcert, Docker, or low-port binds are unavailable; fail with explicit startup/install diagnostics per D-02 and D-03.

## Common Pitfalls

1. **Split-brain operator truth**: if CLI commands rebuild snapshot or health locally, `status`, `doctor`, and `logs` will drift from the daemon.
2. **Stale admin socket files**: daemon restarts must remove dead socket files before binding, or the service can fail even though no process is listening.
3. **Wrong launchd domain**: installing the core daemon as a user agent breaks privileged port ownership; installing the menu bar as a system daemon breaks user-session UX.
4. **Resolver false positives**: a resolver file on disk is insufficient if macOS has not loaded or is not using it for the managed suffix.
5. **Over-destructive uninstall**: deleting config, state, logs, and certs without explicit user choice violates D-05 and makes recovery harder.

## Architectural Responsibility Map

| Tier | Owns | Does Not Own |
|------|------|--------------|
| CLI (`cmd/devproxy`) | command wiring, prompts, table/text rendering, admin API requests | daemon state recomputation, launchd internals |
| Admin API (`internal/adminapi`) | UNIX socket lifecycle, HTTP handlers, JSON contracts, stale-socket cleanup | route computation, Docker event watching |
| Daemon (`internal/daemon`) | runtime assembly, startup validation, snapshot/watcher/network health, refresh actions | launchd plist generation, uninstall prompts |
| Install (`internal/install`) | paths, resolver files, launchd plist materialization, bootstrap/bootout orchestration | route/read-model business logic |
| Diagnostics (`internal/doctor`) | system checks for Docker, resolver, launchd, listener reachability, example hostname probes | modifying route state, writing install artifacts |

## Plan Implications

- Start with a **daemon/control-plane plan** that creates the admin socket server, JSON contracts, and fail-fast foreground daemon entrypoint.
- Split **operator thin clients** from **macOS install orchestration** so command rendering and system integration can progress in parallel without sharing files.
- Finish with a **doctor/uninstall plan** that consumes the established control plane and install metadata to verify and clean up the full lifecycle safely.

---

*Derived from `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`, `.planning/PROJECT.md`, `.planning/phases/03-install-daemon-lifecycle-and-diagnostics/03-CONTEXT.md`, Apple `launchd` / `launchctl` / `resolver(5)` references, and mkcert usage docs.*
