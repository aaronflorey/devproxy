---
phase: 02-local-dns-proxy-and-https-serving
plan: 07
subsystem: infra
tags: [https, tls, mkcert, daemon]
requires:
  - phase: 02-local-dns-proxy-and-https-serving
    provides: certificate inventory and mkcert issuance helpers
provides:
  - Runtime preparation of reusable or newly issued certificates from snapshot inventory
  - HTTPS listener assembly from stored certificate artifacts
  - Certificate readiness derived from effective prepared inventory, not inline TLS map length alone
affects: [https-listener, runtime-health]
tech-stack:
  added: []
  patterns: [snapshot-driven certificate preparation, fail-fast issuance during runtime construction]
key-files:
  created:
    - internal/daemon/network_test.go
  modified:
    - internal/daemon/network.go
key-decisions:
  - "Network runtime prepares certificates during construction using certificate inventory decisions from winning snapshot hostnames."
  - "Stored certificates are reused only when coverage still matches; otherwise runtime creation invokes the configured issuer and surfaces failures directly."
patterns-established:
  - "Runtime readiness reflects actual cert material prepared for HTTPS, whether reused from disk or newly issued."
requirements-completed: [NET-03, NET-04]
duration: 1 min
completed: 2026-05-05
---

# Phase 2 Plan 7: Runtime Certificate Preparation Summary

**`NewNetworkRuntime` now consumes certificate inventory decisions, reuses valid stored certs, and issues replacements when hostname coverage changes.**

## Accomplishments
- Added daemon runtime tests covering stored-cert reuse, issuance on coverage changes, and readiness derived from prepared inventory.
- Wired `certs.BuildCertificateInventory` and issuance callbacks into runtime construction.
- Passed prepared stored certificate metadata into the HTTPS listener so runtime assembly uses the existing mkcert/store primitives.

## Files Created/Modified
- `internal/daemon/network.go` - certificate inventory preparation, issuance wiring, and readiness updates.
- `internal/daemon/network_test.go` - regression coverage for reuse, issuance, and certificate readiness.

## Verification
- `go test ./internal/daemon ./internal/proxy -run 'Test(ReconcilerAppliesMergedOverrides|ReconcilerLabelOverridesConfigForPortAndScheme|NewNetworkRuntimePreparesCertificates|NetworkRuntimeCertificateReadyFromPreparedInventory|NetworkRuntimeStartBindsHTTPAndHTTPS|HTTPSListenerSelectsCertificateForManagedNoRouteHost)'`
- `go test ./internal/daemon ./internal/proxy ./internal/admin`

## Notes
- No git commit was created in this run.

## Self-Check: PASSED
