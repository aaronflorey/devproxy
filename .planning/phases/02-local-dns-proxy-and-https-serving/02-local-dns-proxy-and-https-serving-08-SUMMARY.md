---
phase: 02-local-dns-proxy-and-https-serving
plan: 08
subsystem: infra
tags: [https, http, listeners, runtime]
requires:
  - phase: 02-local-dns-proxy-and-https-serving
    provides: network runtime handler assembly and prepared TLS inventory
provides:
  - Concrete HTTP and HTTPS listener startup and shutdown lifecycle on the daemon runtime
  - Truthful bind health for both listeners using configured addresses
  - Managed HTTPS certificate selection that allows no-route and paused hosts to reach shared friendly responses
affects: [http-serving, https-serving, diagnostics]
tech-stack:
  added: []
  patterns: [listener lifecycle owned by runtime, TLS selection gated by managed suffix instead of active route]
key-files:
  created: []
  modified:
    - internal/daemon/network.go
    - internal/daemon/network_test.go
    - internal/proxy/https.go
    - internal/proxy/https_test.go
key-decisions:
  - "Runtime binds both listeners explicitly and tears down partial startup if one bind fails."
  - "HTTPS certificate selection now requires only a managed hostname; no-route and paused behavior remains in the shared HTTP handler after handshake."
patterns-established:
  - "Friendly managed-host fallback behavior is protocol-consistent across HTTP and HTTPS."
requirements-completed: [NET-05, NET-07, NET-08]
duration: 1 min
completed: 2026-05-05
---

# Phase 2 Plan 8: Listener Startup and HTTPS Fallback Summary

**The daemon runtime can now bind real HTTP/HTTPS listeners, and managed HTTPS misses no longer fail during TLS selection before the friendly fallback path runs.**

## Accomplishments
- Added runtime tests for listener startup on loopback high ports while preserving default `127.0.0.1:80` and `127.0.0.1:443` bind addresses.
- Implemented `Start()` and `Close()` on `NetworkRuntime` using `net.Listen` and `tls.Listen`.
- Removed the HTTPS active-route handshake gate so managed no-route hosts can complete certificate selection and receive the same 404/503 behavior as HTTP.

## Files Created/Modified
- `internal/daemon/network.go` - listener startup/shutdown lifecycle and bind health updates.
- `internal/daemon/network_test.go` - listener bind lifecycle regression tests.
- `internal/proxy/https.go` - managed-host certificate selection without active-route rejection.
- `internal/proxy/https_test.go` - managed no-route TLS selection regression coverage.

## Verification
- `go test ./internal/daemon ./internal/proxy -run 'Test(ReconcilerAppliesMergedOverrides|ReconcilerLabelOverridesConfigForPortAndScheme|NewNetworkRuntimePreparesCertificates|NetworkRuntimeCertificateReadyFromPreparedInventory|NetworkRuntimeStartBindsHTTPAndHTTPS|HTTPSListenerSelectsCertificateForManagedNoRouteHost)'`
- `go test ./internal/daemon ./internal/proxy ./internal/admin`

## Notes
- No git commit was created in this run.

## Self-Check: PASSED
