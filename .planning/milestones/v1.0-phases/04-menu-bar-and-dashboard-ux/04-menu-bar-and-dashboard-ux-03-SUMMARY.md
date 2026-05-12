---
phase: 04-menu-bar-and-dashboard-ux
plan: 03
subsystem: ui
tags: [menubar, systray, admin-api, launchd, macos]
requires:
  - phase: 04-menu-bar-and-dashboard-ux
    provides: dashboard server and admin API route metadata from plans 01-02
provides:
  - macOS menubar command and runtime wired to daemon admin API actions
  - deterministic route/dashboard/log URL opening from daemon-projected data
  - role-aware startup toggle path constrained to menubar role
affects: [operator-ux, startup-controls, dashboard-launch]
tech-stack:
  added: [github.com/getlantern/systray]
  patterns: [thin-ui-over-adminapi, explicit-offline-copy, role-scoped-startup-toggle]
key-files:
  created: [cmd/devproxy/menubar.go, internal/menubar/app.go, internal/menubar/open.go, internal/menubar/icon.go, internal/menubar/runtime_darwin.go, internal/menubar/runtime_stub.go, internal/menubar/app_test.go]
  modified: [go.mod, go.sum]
key-decisions:
  - "Menubar action dispatch routes all mutations through admin API methods and never shells launchctl directly."
  - "Route open actions pass through daemon-provided OpenURL verbatim and opener validates only non-empty http(s) URLs before invoking open."
patterns-established:
  - "Menubar runtime remains thin: polling and click handlers call a pure dispatcher and shared menu state builder."
  - "Offline/degraded state uses approved explicit failure copy while keeping repair actions visible."
requirements-completed: [UI-01, UI-02, UI-03]
duration: 24 min
completed: 2026-05-06
---

# Phase 4 Plan 3: Menu Bar Runtime Summary

**Systray-based macOS menubar runtime now exposes daemon health, route-open entries, and admin-API-backed quick actions with deterministic dashboard/log URL launching.**

## Performance

- **Duration:** 24 min
- **Started:** 2026-05-06T03:10:00Z
- **Completed:** 2026-05-06T03:34:00Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Added RED tests that lock menu state rendering, dispatch behavior, startup role targeting, deterministic URL opening, and offline/degraded copy.
- Implemented `devproxy menubar` command and `internal/menubar` runtime/dispatcher/open helpers with explicit admin API action wiring.
- Added systray dependency and platform-specific runtime split (`darwin` implementation + non-darwin stub) to preserve cross-platform testability.

## Task Commits

1. **Task 1: Lock menu state, action dispatch, and fallback behavior in tests** - `375decb` (test)
2. **Task 2: Implement the systray runtime, command wiring, and route/dashboard launch actions** - `585c24e` (feat)

## Files Created/Modified
- `internal/menubar/app_test.go` - RED tests pinning required menu behavior and copy contracts.
- `internal/menubar/app.go` - Pure menu state builder and admin-backed dispatcher actions.
- `internal/menubar/open.go` - URL opener with http(s) validation and macOS `open` execution.
- `internal/menubar/runtime_darwin.go` - systray lifecycle, polling loop, and click handlers.
- `internal/menubar/runtime_stub.go` - non-macOS guard implementation.
- `internal/menubar/icon.go` - embedded icon bytes for systray.
- `cmd/devproxy/menubar.go` - Cobra subcommand registration and runtime bootstrap.
- `go.mod`, `go.sum` - added `github.com/getlantern/systray` dependency graph.

## Decisions Made
- Scoped startup toggles to `role=menubar` and retained daemon startup state as read-only status context.
- Preserved action-error visibility by returning exact admin API errors to callers instead of swallowing failures.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
Plan 04-04 can focus on packaging/polish and verification with menubar contracts now pinned and implemented.

## Self-Check: PASSED
