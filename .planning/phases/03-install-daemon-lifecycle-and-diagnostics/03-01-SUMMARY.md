---
phase: 03-install-daemon-lifecycle-and-diagnostics
plan: 01
subsystem: infra
tags: [daemon, admin-api, unix-socket, cobra, diagnostics]
requires:
  - phase: 02-local-dns-proxy-and-https-serving
    provides: Runtime health and listener assembly for DNS/HTTP/HTTPS serving
provides:
  - Foreground daemon bootstrap with explicit prerequisite failure paths
  - Local UNIX-socket HTTP+JSON admin control plane for status/routes/doctor/logs/refresh
  - Extensible CLI command registration path for daemon and future operator commands
affects: [phase-03-plan-02, phase-03-plan-04, phase-04-dashboard]
tech-stack:
  added: []
  patterns: [single daemon-owned control-plane state provider, fail-fast startup validation before serving, command registration helper in cmd layer]
key-files:
  created:
    - cmd/devproxy/commands.go
    - cmd/devproxy/daemon.go
    - internal/adminapi/server.go
    - internal/daemon/app.go
  modified:
    - cmd/devproxy/root.go
    - internal/admin/status.go
    - internal/adminapi/server_test.go
    - internal/daemon/app_test.go
key-decisions:
  - "Decoupled internal/admin projection builders from daemon package types via neutral admin DTOs to remove import cycles while preserving output shape."
  - "Kept admin API on local UNIX socket with stale-socket cleanup and mode 0600 to enforce local-only control plane access."
  - "Validated daemon prerequisites (Docker, mkcert, listener binds) before serving any admin endpoint to preserve fail-fast startup semantics."
patterns-established:
  - "admin package accepts transport-agnostic health/read-model inputs"
  - "daemon owns state and publishes one projection source through adminapi handlers"
requirements-completed: [OPS-03, OPS-09]
duration: 16 min
completed: 2026-05-05
---

# Phase 3 Plan 1: Daemon Control Plane and Fail-Fast Bootstrap Summary

**Foreground daemon startup with explicit prerequisite validation and a single UNIX-socket admin API publishing shared status/routes/doctor/logs read models.**

## Performance

- **Duration:** 16 min
- **Started:** 2026-05-05T23:20:00Z
- **Completed:** 2026-05-05T23:36:08Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Added RED contracts/tests for stale socket cleanup, JSON API payloads, command registration scaffold, and explicit bootstrap failure semantics.
- Implemented daemon app startup orchestration, including Docker/mkcert/bind prechecks, runtime startup, and admin socket lifecycle.
- Wired root command registration through a helper and added `devproxy daemon` as a foreground command path for Phase 3 operator flows.

## Task Commits

Each task was committed atomically:

1. **Task 1: Lock the admin socket and daemon bootstrap contracts** - `1ea84df` (test)
2. **Task 2: Implement the daemon app, socket server, and foreground command** - `82dbf8f` (feat)

_Note: Task 1 is TDD RED; Task 2 provides the GREEN implementation for the plan behavior._

## Files Created/Modified
- `internal/adminapi/types.go` - Admin JSON response/request contracts and command-registry contract primitives.
- `internal/adminapi/server.go` - UNIX-socket admin server, stale socket removal, mode 0600 enforcement, and status/routes/doctor/logs/refresh handlers.
- `internal/daemon/app.go` - Fail-fast daemon bootstrap, runtime startup/closure, and authoritative state projection provider.
- `cmd/devproxy/daemon.go` - Foreground daemon Cobra command with signal-aware run loop.
- `cmd/devproxy/commands.go` and `cmd/devproxy/root.go` - Centralized subcommand registration path.
- `internal/admin/status.go` - Neutral watcher/runtime input DTOs to decouple from `internal/daemon` package types.

## Decisions Made
- Chose neutral DTO inputs in `internal/admin` instead of daemon-typed inputs to eliminate `daemon -> admin -> daemon` import cycles while retaining projection behavior.
- Kept route projection generation in adminapi via `admin.RoutesFromSnapshot` so all operator clients consume one daemon-provided snapshot source.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed daemon/admin import cycle with neutral admin DTOs**
- **Found during:** Task 2 (daemon app implementation)
- **Issue:** `internal/daemon/app.go` needed admin projection builders, but `internal/admin/status.go` imported daemon types, creating a package cycle.
- **Fix:** Replaced daemon-specific status inputs in `internal/admin/status.go` with package-local DTOs and updated daemon mapping logic.
- **Files modified:** `internal/admin/status.go`, `internal/admin/status_test.go`, `internal/daemon/app.go`, `internal/adminapi/server_test.go`
- **Verification:** `go test ./internal/adminapi/... ./internal/daemon/... ./cmd/devproxy/...` and `go test ./...`
- **Committed in:** `82dbf8f`

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Required for correctness and completion. No scope creep beyond planned daemon/admin surface.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Daemon control plane now exists for `status`, `routes`, `refresh`, `doctor`, and `logs` thin clients in Plan 03-02.
- No blockers identified for remaining Phase 3 plans.

## Self-Check: PASSED
