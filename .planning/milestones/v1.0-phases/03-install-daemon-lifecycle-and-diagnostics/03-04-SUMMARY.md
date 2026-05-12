---
phase: 03-install-daemon-lifecycle-and-diagnostics
plan: 04
subsystem: infra
tags: [doctor, uninstall, launchd, resolver, diagnostics]
requires:
  - phase: 03-install-daemon-lifecycle-and-diagnostics
    provides: daemon admin socket, install lifecycle primitives
provides:
  - doctor diagnostics covering runtime and resolver evidence
  - selective uninstall orchestration with explicit cleanup scope
affects: [ops, cli, lifecycle]
tech-stack:
  added: []
  patterns: [dependency-injected checks, prompted cleanup scope]
key-files:
  created:
    - cmd/devproxy/doctor.go
    - cmd/devproxy/uninstall.go
    - internal/doctor/checks.go
    - internal/install/uninstall.go
  modified:
    - internal/adminapi/client.go
    - internal/install/launchd.go
    - internal/doctor/checks_test.go
    - internal/install/uninstall_test.go
key-decisions:
  - "Doctor validates resolver activation using scutil --dns evidence, not file-only heuristics."
  - "Uninstall always stops/unregisters services and removes resolver wiring before optional artifact cleanup."
patterns-established:
  - "Lifecycle teardown-first: stop/unregister services before destructive cleanup."
  - "Doctor reports named check results with explicit pass/fail and message text."
requirements-completed: [OPS-06, OPS-08]
duration: 3 min
completed: 2026-05-05
---

# Phase 03 Plan 04: Install Daemon Lifecycle And Diagnostics Summary

**Doctor diagnostics now verify Docker/launchd/admin/runtime/resolver state and uninstall now applies explicit keep/remove choices per artifact category.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-05-05T23:50:35Z
- **Completed:** 2026-05-05T23:53:52Z
- **Tasks:** 2
- **Files modified:** 8

## Accomplishments
- Added TDD coverage for doctor checks (resolver-state, launchd/admin/proxy reachability) and uninstall cleanup-scope behavior.
- Implemented `devproxy doctor` command wiring and reusable doctor check engine with scutil-aligned resolver validation.
- Implemented `devproxy uninstall` command flow and uninstaller orchestration with selective cleanup for config/state/logs/certificates.

## Task Commits

1. **Task 1: Define doctor and uninstall behaviors before implementation** - `e5b74b5` (test)
2. **Task 2: Implement doctor checks, uninstall orchestration, and CLI prompts** - `f4d1705` (feat)

## Files Created/Modified
- `internal/doctor/checks.go` - doctor check framework and default system checks.
- `cmd/devproxy/doctor.go` - CLI doctor command registration and report output.
- `internal/install/uninstall.go` - uninstall lifecycle orchestration and cleanup-scope application.
- `cmd/devproxy/uninstall.go` - interactive uninstall prompts and command wiring.
- `internal/install/launchd.go` - added stop/uninstall lifecycle helpers.
- `internal/adminapi/client.go` - added doctor API client method.

## Decisions Made
- Used named check results (`name/ok/message`) so doctor output remains explicit and script-friendly.
- Kept persisted multi-session log cleanup out of scope; uninstall cleanup only handles configured directories and cert artifacts.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 03 is complete and now includes full operator diagnostics and safe uninstall controls required by OPS-06 and OPS-08.

## Self-Check: PASSED
