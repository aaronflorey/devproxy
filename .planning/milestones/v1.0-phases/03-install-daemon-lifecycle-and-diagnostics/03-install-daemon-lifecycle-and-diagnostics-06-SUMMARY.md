---
phase: 03-install-daemon-lifecycle-and-diagnostics
plan: 06
subsystem: diagnostics
tags: [doctor, managed-host, tls, admin-socket]
requires:
  - phase: 03-04
    provides: doctor command and runtime status plumbing
provides:
  - managed-host-aware proxy probe behavior
  - blocked diagnostics when control-plane runtime status is unavailable
affects: [phase-03-operations, runtime-health]
tech-stack:
  added: []
  patterns: [control-plane-gated diagnostics, managed-host SNI probing]
key-files:
  created: []
  modified: [internal/doctor/checks.go, internal/doctor/checks_test.go]
key-decisions:
  - "Proxy reachability checks now depend on resolver-active + daemon runtime status gates."
  - "HTTP/HTTPS probes preserve managed hostnames while dialing loopback ports for realistic SNI/cert checks."
patterns-established:
  - "Doctor emits one shared blocking message when runtime status cannot be trusted."
requirements-completed: [OPS-06]
duration: 18min
completed: 2026-05-06
---

# Phase 3 Plan 6: Doctor Diagnostics Gap Closure Summary

**Doctor runtime checks now gate managed proxy probes on daemon status and use managed-host SNI instead of loopback IP probes.**

## Performance
- **Duration:** 18 min
- **Started:** 2026-05-06T00:06:00Z
- **Completed:** 2026-05-06T00:24:18Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Added regression tests for blocked runtime diagnostics and managed-host propagation.
- Implemented control-plane-aware gating so proxy checks do not run when daemon status is unavailable.
- Reworked HTTP/HTTPS probes to dial local ports while preserving hostname/SNI for realistic TLS outcomes.

## Task Commits
1. **Task 1: Lock managed-host doctor probe regressions in tests** - `135276b` (test)
2. **Task 2: Implement control-plane-aware managed HTTP and HTTPS doctor probes** - `082d090` (feat)

## Files Created/Modified
- `internal/doctor/checks_test.go` - Regression tests for runtime gating and managed-host probe args.
- `internal/doctor/checks.go` - Managed-host probe implementation and control-plane-aware blocking behavior.

## Decisions Made
- Runtime-health read failures now block proxy diagnostics with a shared, explicit message.
- Managed hostnames are used for HTTP/HTTPS probe URLs and TLS server name while still connecting to 127.0.0.1 listener ports.

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None.

## Next Phase Readiness
- Doctor output now aligns with real managed-host behavior and avoids misleading localhost-only TLS signals.

## Self-Check: PASSED
