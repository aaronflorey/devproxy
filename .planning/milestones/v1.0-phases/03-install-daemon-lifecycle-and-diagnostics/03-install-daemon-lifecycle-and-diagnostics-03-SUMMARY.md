---
phase: 03-install-daemon-lifecycle-and-diagnostics
plan: 03
subsystem: infra
tags: [macos, install, launchd, resolver, mkcert, cobra]
requires:
  - phase: 03-01
    provides: daemon admin socket and foreground lifecycle baseline
provides:
  - macOS install orchestration for paths, resolver, cert prerequisite checks, and launchd services
  - `devproxy install` CLI with opt-in `--with-menubar` behavior
affects: [ops, install, daemon-lifecycle]
tech-stack:
  added: []
  patterns: [dependency-injected installer orchestration, split launchd daemon-vs-agent service configs]
key-files:
  created:
    - cmd/devproxy/install.go
    - internal/install/install.go
    - internal/install/paths.go
    - internal/install/resolver.go
    - internal/install/launchd.go
    - internal/install/install_test.go
  modified: []
key-decisions:
  - "Installer orchestration is implemented behind dependency-injected functions so filesystem, resolver, cert bootstrap, and launchd flows are test-pinned and observable."
  - "Daemon service is always installed/started in launchd system domain, while menu bar service is optional and targeted to launchd agent domain only when --with-menubar is set."
patterns-established:
  - "Install lifecycle in internal/install is orchestration-only and does not pull runtime reconciliation/proxy logic into CLI setup paths."
requirements-completed: [OPS-01, OPS-02]
duration: 14 min
completed: 2026-05-05
---

# Phase 03 Plan 03: Install orchestration Summary

**macOS install flow now provisions required devproxy directories, managed resolver, mkcert prerequisite check, and separated launchd daemon/menubar services via `devproxy install`.**

## Performance

- **Duration:** 14 min
- **Started:** 2026-05-05T23:50:00Z
- **Completed:** 2026-05-06T00:04:00Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Added RED tests that lock required install order, resolver handling, and launchd role split behavior.
- Implemented `internal/install` helpers for path creation, resolver materialization, launchd plist generation, and launchctl bootstrap.
- Added `devproxy install` command with default daemon install/start and opt-in menu bar install/start via `--with-menubar`.

## Task Commits

1. **Task 1: Lock install orchestration and launchd role behavior in tests** - `0c9b0ba` (test)
2. **Task 2: Implement install helpers and the install command** - `77389a5` (feat)

## Files Created/Modified
- `internal/install/install_test.go` - TDD tests pinning install flow expectations and launchd role separation.
- `internal/install/install.go` - Installer orchestration with dependency-injected filesystem/resolver/cert/launchd steps.
- `internal/install/paths.go` - Install path model and path creation helper.
- `internal/install/resolver.go` - Managed suffix resolver file writer using loopback DNS target and fixed port.
- `internal/install/launchd.go` - Daemon and menubar service config/plist generation plus launchctl bootstrap helper.
- `cmd/devproxy/install.go` - Cobra install command wiring with `--with-menubar` option.

## Decisions Made
- Used dependency injection in installer to make D-04/D-07/D-08 service lifecycle decisions directly testable.
- Kept launchd domain and label roles explicit (`com.devproxy.daemon` in system, `com.devproxy.menubar` in gui/<uid>) to preserve privileged daemon boundary.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Install lifecycle baseline for OPS-01/OPS-02 is in place and test-covered.
- Ready for 03-04 doctor/uninstall lifecycle completion plan.

## Self-Check: PASSED
