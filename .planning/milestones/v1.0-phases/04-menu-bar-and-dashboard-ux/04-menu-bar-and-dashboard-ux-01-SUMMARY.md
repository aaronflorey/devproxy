---
phase: 04-menu-bar-and-dashboard-ux
plan: 01
subsystem: ui
tags: [admin-api, launchd, routing, menubar, dashboard]
requires:
  - phase: 03-install-daemon-lifecycle-and-diagnostics
    provides: daemon-owned unix-socket admin API and launchd service split
provides:
  - daemon-mediated pause/resume and startup-role control endpoints
  - deterministic route open metadata (scheme/url/fallback)
  - bounded current-session issue projection for UI clients
affects: [menu-bar-runtime, dashboard-server, ui-control-plane]
tech-stack:
  added: []
  patterns: [thin clients over admin API, fixed role enum validation, bounded in-memory issue buffer]
key-files:
  created: []
  modified:
    - internal/adminapi/types.go
    - internal/adminapi/server.go
    - internal/adminapi/client.go
    - internal/admin/status.go
    - internal/admin/routes.go
    - internal/admin/logs.go
    - internal/daemon/app.go
    - internal/install/launchd.go
    - internal/adminapi/server_test.go
    - internal/adminapi/client_test.go
    - internal/admin/status_test.go
key-decisions:
  - "Expose startup controls through role-validated /startup API instead of direct UI launchctl access."
  - "Derive route OpenURL from daemon snapshot/runtime readiness and attach explicit fallback reasons."
patterns-established:
  - "Control actions are daemon-mediated and role-scoped (daemon|menubar)."
  - "Session issues are bounded to 25 newest-first entries for UI payload safety."
requirements-completed: [UI-01, UI-02, UI-03, UI-04]
duration: 5 min
completed: 2026-05-06
---

# Phase 4 Plan 01: Control-plane API contracts for menu/dashboard Summary

**Daemon admin API now provides role-aware startup controls, pause/resume actions, deterministic route-open metadata, and bounded session issue payloads for both UI surfaces.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-05-06T02:14:57Z
- **Completed:** 2026-05-06T02:19:39Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Added RED tests that lock `/routing/pause`, `/routing/resume`, `/startup`, route-open metadata, and bounded session-issue behavior.
- Implemented new admin API server/client contracts for pause/resume and startup status/toggle with strict role validation (`daemon`, `menubar`).
- Extended daemon/admin projections to emit deterministic `OpenURL` + `FallbackReason` and expose bounded newest-first current-session issues.

## Task Commits

1. **Task 1: Pin the new UI API contracts and failure semantics in tests** - `f9cb823` (test)
2. **Task 2: Implement daemon-backed UI controls, route-open metadata, and session issue projections** - `825c56d` (feat)

## Files Created/Modified
- `internal/adminapi/server.go` - Added `/routing/pause`, `/routing/resume`, `/startup`, `/issues` handlers and role validation.
- `internal/adminapi/client.go` - Added Pause/Resume/Startup client methods.
- `internal/admin/routes.go` - Added `OpenURL`, `PreferredScheme`, `FallbackReason`, `HTTPSReady` projection fields.
- `internal/admin/logs.go` - Added bounded `SessionIssue` model and projection helper.
- `internal/daemon/app.go` - Added authoritative in-memory issue buffer and startup/pause control wiring.
- `internal/install/launchd.go` - Added startup role status inspection and menubar startup toggle helper.

## Decisions Made
- Used a fixed startup role enum in API request validation to mitigate label/path tampering risk at the UI→daemon boundary.
- Kept daemon role non-toggleable via UI and returned explicit failure messaging for attempted daemon toggles.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected daemon-toggle error construction during verification**
- **Found during:** Task 2
- **Issue:** `fmt.Errorf(msg)` triggered `go test ./...` static check failure (`non-constant format string`).
- **Fix:** Replaced with `errors.New(msg)` for constant-safe error construction.
- **Files modified:** `internal/daemon/app.go`
- **Verification:** `go test ./internal/adminapi/... ./internal/admin/... ./internal/daemon/... ./internal/install/... && go test ./...`
- **Committed in:** `825c56d`

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** No scope creep; fix was required to satisfy full test/build verification.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
Phase 4 UI clients can now consume stable daemon-owned control/read contracts for dashboard and menubar implementation in Plan 04-02.

## Known Stubs
None.

## Self-Check: PASSED
- FOUND: `.planning/phases/04-menu-bar-and-dashboard-ux/04-menu-bar-and-dashboard-ux-01-SUMMARY.md`
- FOUND: commit `f9cb823`
- FOUND: commit `825c56d`

---
*Phase: 04-menu-bar-and-dashboard-ux*
*Completed: 2026-05-06*
