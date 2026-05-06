---
phase: 04-menu-bar-and-dashboard-ux
plan: 04
subsystem: ui
tags: [menubar, dashboard, regression-tests, fallback]
requires:
  - phase: 04-menu-bar-and-dashboard-ux
    provides: dashboard + menubar action flows from plans 02-03
provides:
  - Regression coverage for daemon-offline, startup-toggle failure messaging, and fallback URL behavior
  - Dashboard route rendering that exposes HTTPS-unhealthy fallback reason text
affects: [dashboard, menubar, cli-lifecycle-tests]
tech-stack:
  added: []
  patterns: ["Degraded UI states must keep repair actions and explicit fallback copy visible"]
key-files:
  created: [.planning/phases/04-menu-bar-and-dashboard-ux/04-menu-bar-and-dashboard-ux-04-SUMMARY.md]
  modified:
    - internal/dashboard/server_test.go
    - internal/menubar/app_test.go
    - cmd/devproxy/lifecycle_test.go
    - internal/dashboard/templates/dashboard.html.tmpl
key-decisions:
  - "Expose RouteView.FallbackReason directly in the dashboard route list so HTTPS-readiness fallback is user-visible."
  - "Auto-approved human-verify checkpoint under workflow.auto_advance=true after automated verification passed."
patterns-established:
  - "UI degraded-path assertions live in unit/command tests before behavior polish changes."
requirements-completed: [UI-01, UI-02, UI-03, UI-04]
duration: 20 min
completed: 2026-05-06
---

# Phase 04 Plan 04: Menu-bar-and-dashboard-ux Summary

**Regression-hardened dashboard and menubar degraded flows with explicit HTTPS fallback messaging and fixed localhost launch contract coverage.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-05-06T03:00:00Z
- **Completed:** 2026-05-06T03:20:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added regression tests for daemon-offline dashboard rendering and preserved repair affordances.
- Added menubar and lifecycle coverage for startup failure status text and fixed dashboard launch wiring.
- Updated dashboard route UI to display HTTPS fallback reason when route open URL degrades to HTTP.

## Task Commits

Each task was committed atomically:

1. **Task 1: Add regression coverage for degraded UI flows and repair any failures** - `313a40e` (test), `c0028ee` (feat)
2. **Task 2: Verify native menu bar and browser dashboard behavior on macOS** - Auto-approved checkpoint (no code changes) because `workflow.auto_advance=true`.

## Files Created/Modified
- `internal/dashboard/server_test.go` - Adds offline-copy and HTTPS-fallback regression assertions.
- `internal/menubar/app_test.go` - Adds startup toggle failure message coverage in menu state summary.
- `cmd/devproxy/lifecycle_test.go` - Adds dashboard launch default URL contract regression coverage.
- `internal/dashboard/templates/dashboard.html.tmpl` - Renders `FallbackReason` alongside route upstream details.

## Decisions Made
- Rendered fallback reason in dashboard route rows instead of adding a separate section, minimizing UI surface changes while making degradation explicit.
- Kept launch URL contract checks at CLI test level to guard future drift from `127.0.0.1:45831` dashboard/log targets.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
Phase 04 plan execution is complete with degraded-path coverage and UI fallback visibility in place.

## Self-Check: PASSED
- Found file: `.planning/phases/04-menu-bar-and-dashboard-ux/04-menu-bar-and-dashboard-ux-04-SUMMARY.md`
- Found commit: `313a40e`
- Found commit: `c0028ee`

---
*Phase: 04-menu-bar-and-dashboard-ux*
*Completed: 2026-05-06*
