---
phase: 03-install-daemon-lifecycle-and-diagnostics
plan: 07
subsystem: uninstall-lifecycle
tags: [launchd, uninstall, gap-closure, regression]
requires:
  - phase: 03-05
    provides: root-gated lifecycle uninstall scaffolding
provides:
  - bootout-exit-5 missing-state fallback with launchctl print verification
  - uninstall continuation for verified absent-service wrapper errors
affects: [phase-03-operations, uninstall-safety]
tech-stack:
  added: []
  patterns: [bootout-then-print verification, fail-loud launchd error handling]
key-files:
  created: [internal/install/launchd_test.go]
  modified: [internal/install/launchd.go, internal/install/uninstall.go, internal/install/uninstall_test.go]
key-decisions:
  - "Only suppress bootout exit-5 IO errors after launchctl print confirms service absence."
  - "Uninstall accepts the observed bootout wrapper as idempotent missing-state so resolver and scoped cleanup still run."
patterns-established:
  - "launchctl failures remain fatal unless a second signal proves idempotent missing state."
requirements-completed: [OPS-08]
duration: 19min
completed: 2026-05-06
---

# Phase 3 Plan 7: Uninstall Bootout Regression Gap Closure Summary

**Uninstall now survives the real macOS `bootout ... exit status 5` absent-service path by requiring a follow-up `launchctl print` missing-state confirmation before allowing cleanup to continue.**

## Performance
- **Duration:** 19 min
- **Started:** 2026-05-06T01:10:00Z
- **Completed:** 2026-05-06T01:29:19Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added regression tests for both the exact UAT wrapper string and StopService bootout/print decision flow.
- Implemented `StopService` fallback logic: bootout exit-5 IO errors now trigger `launchctl print {domain}/{label}` and are suppressed only for verified missing-service states.
- Updated uninstall missing-state detection to recognize the observed bootout wrapper so resolver removal and selected cleanup proceed.

## Task Commits
1. **Task 1: Lock the bootout exit-5 uninstall regression in tests** - `08769cf` (test)
2. **Task 2: Implement bootout fallback without masking real launchd failures** - `7d73576` (feat)

## Files Created/Modified
- `internal/install/launchd_test.go` - New bootout/print fallback regression coverage.
- `internal/install/uninstall_test.go` - Exact UAT wrapper regression proving cleanup continuation.
- `internal/install/launchd.go` - Bootout exit-5 fallback with print probe and strict fail-loud behavior.
- `internal/install/uninstall.go` - Missing-state classifier reused for observed wrapper handling.

## Decisions Made
- Keep non-missing bootout failures fatal even when the failure contains exit-5 IO text.
- Classify missing-state wrappers centrally through shared helper predicates.

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None.

## Next Phase Readiness
- Phase 3 uninstall diagnostics gap is closed and protected by deterministic regression tests.

## Self-Check: PASSED
