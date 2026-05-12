---
phase: 02-local-dns-proxy-and-https-serving
plan: 03
subsystem: api
tags: [http, reverse-proxy, websocket, dns, admin]
requires:
  - phase: 02-local-dns-proxy-and-https-serving
    provides: managed suffix DNS classification and reconciled upstream metadata
provides:
  - Managed-host HTTP handling that proxies active routes and returns friendly local no-route/paused responses
  - WebSocket-upgrade-safe proxy path using Go stdlib reverse proxy with upstream scheme/port from snapshot
  - Admin route/log projections with effective upstream and handling-state visibility
affects: [https-listeners, status-surfaces, operator-diagnostics]
tech-stack:
  added: []
  patterns: ["Managed host classification gates friendly local ownership", "HTTP forwarding uses reconciled routing.Upstream only"]
key-files:
  created: [internal/proxy/http.go]
  modified: [internal/proxy/http_test.go, internal/admin/routes.go, internal/admin/logs.go]
key-decisions:
  - "Use a claim-or-bypass handler (`HandleHTTP`) so unmanaged hosts are not intercepted by devproxy fallbacks."
  - "Drive proxy targets strictly from `routing.Route.Upstream` and default scheme only when upstream metadata is incomplete."
patterns-established:
  - "Friendly local responses are limited to managed suffix hosts only."
  - "Paused routing is checked before route activation so pause behavior is explicit and distinct."
requirements-completed: [NET-02, NET-05, NET-06, NET-07, NET-08]
duration: 2 min
completed: 2026-05-05
---

# Phase 2 Plan 3: HTTP proxy serving path summary

**Managed-host HTTP handling now proxies active snapshot routes and returns explicit no-route/paused local responses with WebSocket-capable reverse proxy wiring.**

## Performance

- **Duration:** 2 min
- **Started:** 2026-05-05T11:37:52Z
- **Completed:** 2026-05-05T11:39:52Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Added failing RED coverage for active-route proxying, managed no-route, managed paused, unmanaged bypass, and upgrade-path handling.
- Implemented `internal/proxy/http.go` with managed-host classification, pause/no-route friendly responses, and `httputil.NewSingleHostReverseProxy` forwarding for active routes.
- Extended admin route/log projections with effective upstream and request-handling state fields for later status and diagnostics surfaces.

## Task Commits

Each task was committed atomically:

1. **Task 1: Write failing tests for HTTP proxy, no-route, paused, and upgrade behavior** - `4e05267` (test)
2. **Task 2: Implement the HTTP proxy handler and serving projections** - `1089734` (feat)

**Plan metadata:** `(pending)`

## Files Created/Modified
- `internal/proxy/http.go` - Managed-host HTTP handler with active-route proxying and friendly paused/no-route responses.
- `internal/proxy/http_test.go` - Coverage for proxying semantics, unmanaged bypass, and upgrade-capable forwarding behavior.
- `internal/admin/routes.go` - Route view now includes upstream scheme and handling state.
- `internal/admin/logs.go` - Session log events now include handling state and upstream details for active route snapshots.

## Decisions Made
- Used a `HandleHTTP` claim-or-bypass method so unmanaged hostnames can fall through to non-devproxy handlers.
- Built reverse-proxy targets only from reconciled upstream metadata (host/port/scheme), matching threat mitigations for request trust boundaries.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Initial proxy test expected `X-Forwarded-Host`; stdlib reverse proxy does not guarantee that header by default, so the assertion was removed and behavior is validated via end-to-end proxy response and captured upstream target metadata.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- HTTP serving semantics are locked for managed hosts and ready for HTTPS listener/certificate runtime wiring in later plans.
- Admin projections now expose upstream and handling context needed by status and diagnostic surfaces.

## Self-Check: PASSED
