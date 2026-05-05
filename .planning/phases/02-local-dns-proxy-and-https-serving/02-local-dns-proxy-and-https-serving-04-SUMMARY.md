---
phase: 02-local-dns-proxy-and-https-serving
plan: 04
subsystem: infra
tags: [tls, mkcert, certificates, routing]
requires:
  - phase: 02-local-dns-proxy-and-https-serving
    provides: routing snapshot served hostnames and upstream metadata
provides:
  - Deterministic certificate inventory grouped by project root from winning served hostnames
  - Reuse versus reissue decisions based on wildcard coverage and hostname shape changes
  - mkcert wrapper with explicit issuance and missing-binary error reporting
affects: [https-listener, runtime-health, diagnostics]
tech-stack:
  added: []
  patterns: [snapshot-derived SAN planning, explicit external command error propagation]
key-files:
  created:
    - internal/certs/store.go
    - internal/certs/mkcert.go
    - internal/certs/store_test.go
    - internal/certs/mkcert_test.go
  modified: []
key-decisions:
  - "Certificate inventory derives hostnames only from winning route ServedHostnames (fallback Hostname) under the managed suffix."
  - "Wildcard reuse is allowed only for one-label descendants of the project root; deeper hostnames force reissue planning."
  - "mkcert invocation failures are always returned as explicit errors, including missing binary guidance."
patterns-established:
  - "Per-project certificate planning keyed by {project}.{suffix}"
  - "Coverage checks are deterministic via exact-SAN plus single-label wildcard matching"
requirements-completed: [NET-03, NET-04]
duration: 1 min
completed: 2026-05-05
---

# Phase 2 Plan 4: Certificate Inventory and mkcert Issuance Summary

**Project-root certificate inventory with wildcard reuse and explicit mkcert failure surfacing for HTTPS hostname coverage.**

## Performance

- **Duration:** 1 min
- **Started:** 2026-05-05T11:44:43Z
- **Completed:** 2026-05-05T11:46:16Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added RED tests for certificate lifecycle behavior: reuse, new-project issuance, deeper-hostname reissue, and mkcert error surfacing.
- Implemented deterministic certificate inventory derivation from active winning-route hostnames grouped by project root.
- Implemented mkcert issuance wrapper that returns explicit errors when mkcert is missing or command execution fails.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing tests for hostname grouping and certificate reuse** - `f852feb` (test)
2. **Task 2: Implement certificate inventory and mkcert issuance helpers** - `567daf0` (feat)

## Files Created/Modified
- `internal/certs/store_test.go` - RED tests for project wildcard reuse, new root issuance, and deeper-hostname reissue.
- `internal/certs/mkcert_test.go` - RED test for explicit mkcert command failure handling.
- `internal/certs/store.go` - Hostname inventory grouping, wildcard coverage checks, and reuse/reissue decisions.
- `internal/certs/mkcert.go` - mkcert command wrapper returning cache-friendly certificate path metadata and explicit errors.

## Decisions Made
- Derived certificate inventory strictly from reconciled route snapshot hostnames to avoid stale/losing route leakage.
- Treated wildcard SAN coverage as valid only for single-label subdomains, matching x509 wildcard semantics and phase rules.
- Wrapped mkcert execution with explicit error messages to fail fast for HTTPS prerequisites.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Certificate planning and issuance primitives are in place for HTTPS listener wiring in Plan 02-05.
- No blockers identified.

## Self-Check: PASSED
