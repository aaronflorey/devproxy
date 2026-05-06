---
phase: 02-local-dns-proxy-and-https-serving
verified: 2026-05-05T22:26:47Z
status: gaps_found
score: 1/5 must-haves verified
overrides_applied: 0
gaps:
  - truth: "Developer can send HTTP or HTTPS requests to an active mapped hostname and receive upstream responses from the selected localhost published port, including WebSocket traffic."
    status: failed
    reason: "Reconciler does not apply config/merged override preferences when generating domains and selecting upstream target port, so mapped hostnames and selected port/scheme are incomplete."
    artifacts:
      - path: "internal/daemon/reconciler.go"
        issue: "Uses empty routing.RoutePreferences{} and hardcoded config override port 0; ignores routing.MergeOverrides and config overrides."
    missing:
      - "Merge config + label route preferences before domain generation and port selection."
      - "Pass effective override port/scheme/domain/domains into snapshot route construction."
    requirements: [NET-02, NET-06]
  - truth: "Developer can use trusted HTTPS certificates generated via mkcert, with certificates regenerated when served hostnames change and reused when unchanged."
    status: failed
    reason: "Certificate planning/issuance code exists but is not wired into daemon network runtime creation path; runtime only accepts in-memory certificate map."
    artifacts:
      - path: "internal/daemon/network.go"
        issue: "NetworkRuntimeConfig lacks stored cert inventory input and NewNetworkRuntime does not pass StoredCertificate artifacts to HTTPS listener."
      - path: "internal/certs/store.go"
        issue: "Inventory builder is never consumed by runtime assembly."
      - path: "internal/certs/mkcert.go"
        issue: "mkcert issuance wrapper is never consumed by runtime assembly."
    missing:
      - "Wire certificate inventory/issuance outputs into NewNetworkRuntime and HTTPS listener assembly."
      - "Set CertificateReady from effective loaded cert sources, not only cfg.Certificates map length."
    requirements: [NET-03, NET-04]
  - truth: "Developer can run with HTTP on port 80 and HTTPS on port 443, with redirect-to-HTTPS remaining off by default unless configured globally or per route."
    status: failed
    reason: "No listener bind/start implementation exists in internal runtime path (only handler assembly and health structs), so actual port-80/443 serving is not implemented in this phase code."
    artifacts:
      - path: "internal/daemon/network.go"
        issue: "Contains no net.Listen/http.Server startup path for 80/443."
    missing:
      - "Implement runtime listener start/bind lifecycle for HTTP:80 and HTTPS:443 or wire existing listener launcher into this runtime."
    requirements: [NET-05]
  - truth: "Developer receives clear friendly responses when no route exists for a managed hostname or when routing is paused."
    status: failed
    reason: "HTTPS certificate selector blocks managed hosts without active routes during TLS handshake, preventing friendly HTTPS no-route/paused responses for those managed hosts."
    artifacts:
      - path: "internal/proxy/https.go"
        issue: "getCertificate requires lookup.ActiveRoute and returns error for managed hosts lacking active route."
    missing:
      - "Select cert for managed host independent of active-route check; delegate no-route/paused behavior to shared HTTP handler after handshake."
    requirements: [NET-07, NET-08]
---

# Phase 2: Local DNS, Proxy, and HTTPS Serving Verification Report

**Phase Goal:** Developers can resolve managed local domains and reliably reach active services over HTTP/HTTPS through devproxy.
**Verified:** 2026-05-05T22:26:47Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Developer can resolve hostnames under the managed suffix to `127.0.0.1` using the installed wildcard resolver. | ✓ VERIFIED | `internal/dns/server.go:41-50` returns authoritative A `127.0.0.1` for managed suffix hosts; tests in `internal/dns/server_test.go:10-43` pass. |
| 2 | Developer can send HTTP or HTTPS requests to an active mapped hostname and receive upstream responses from the selected localhost published port, including WebSocket traffic. | ✗ FAILED | Reconciler omits merged override inputs: `internal/daemon/reconciler.go:49` passes `0` override port and `:56` passes empty `RoutePreferences{}`. This breaks effective mapped-host/selected-port contract for override-driven routes. |
| 3 | Developer can use trusted HTTPS certificates generated via mkcert, with certificates regenerated when served hostnames change and reused when unchanged. | ✗ FAILED | Cert inventory/issuance exists (`internal/certs/store.go`, `internal/certs/mkcert.go`) but not wired into runtime: `internal/daemon/network.go:27-32` only accepts `map[string]tls.Certificate`; `NewNetworkRuntime` never consumes stored cert metadata. |
| 4 | Developer can run with HTTP on port 80 and HTTPS on port 443, with redirect-to-HTTPS remaining off by default unless configured globally or per route. | ✗ FAILED | Redirect default exists (`internal/config/config.go:39` false), but no bind/listener startup path found in `internal/` (no `net.Listen`/`ListenAndServe` usage), so actual port 80/443 serving is not implemented. |
| 5 | Developer receives clear friendly responses when no route exists for a managed hostname or when routing is paused. | ✗ FAILED | HTTP path is friendly (`internal/proxy/http.go:64-77,106-116`), but HTTPS handshake blocks managed/no-route hosts: `internal/proxy/https.go:74-77` requires active route before cert selection. |

**Score:** 1/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|---------|----------|--------|---------|
| `internal/dns/server.go` | managed-suffix resolver + managed lookup | ✓ VERIFIED | Exists, substantive, used by HTTP/HTTPS handlers. |
| `internal/proxy/http.go` | active-route proxy + friendly fallbacks | ✓ VERIFIED | Exists, substantive, wired through `NetworkRuntime` handlers. |
| `internal/proxy/https.go` | TLS listener over shared HTTP behavior | ⚠️ HOLLOW | Exists/wired, but TLS selector gate blocks friendly no-route/paused path for managed no-route hosts. |
| `internal/certs/store.go` | cert inventory/reuse logic | ⚠️ ORPHANED | Exists/substantive but not consumed by runtime assembly path. |
| `internal/certs/mkcert.go` | mkcert issuance wrapper | ⚠️ ORPHANED | Exists/substantive but not consumed by runtime assembly path. |
| `internal/daemon/reconciler.go` | effective route/upstream contract producer | ✗ FAILED | Publishes incomplete metadata due to ignored merged overrides/config override port. |
| `internal/daemon/network.go` | runnable network runtime assembly | ✗ FAILED | Handler assembly only; no listener bind/start and no stored-cert wiring. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/proxy/http.go` | `internal/dns/server.go` | managed hostname classification | ✓ WIRED | `dns.NewServer` + `LookupHostname` used in request path (`http.go:43,59`). |
| `internal/proxy/http.go` | `routing.Route.Upstream` | upstream target selection | ✓ WIRED | `lookup.Route.Upstream` -> `upstreamURL` -> reverse proxy (`http.go:74-83`). |
| `internal/proxy/https.go` | shared HTTP handler | no-route/paused behavior reuse | ✓ WIRED | `HandleHTTPS` delegates to `handler.HandleHTTP` (`https.go:57-59`). |
| `internal/proxy/https.go` | managed host cert selection | TLS handshake path | ✗ NOT_WIRED (intent) | `getCertificate` enforces `lookup.ActiveRoute` (`https.go:74-77`), preventing managed/no-route friendly flow. |
| `internal/daemon/reconciler.go` | `routing.MergeOverrides` + config overrides | effective route metadata | ✗ NOT_WIRED | No call to `routing.MergeOverrides`; no config override values in reconcile path. |
| `internal/daemon/network.go` | `internal/certs/store.go` / `internal/certs/mkcert.go` outputs | runtime cert readiness | ✗ NOT_WIRED | Runtime config does not ingest stored cert artifacts; plan-04 outputs disconnected from plan-05 runtime. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|---------|---------------|--------|--------------------|--------|
| `internal/proxy/http.go` | `lookup := dnsLookup.LookupHostname(host)` | `dns.Server` snapshot lookup | Yes | ✓ FLOWING |
| `internal/proxy/https.go` | `lookup := dnsLookup.LookupHostname(host)` in TLS selector | `dns.Server` snapshot lookup | Partially; blocked by active-route requirement | ⚠️ STATIC GATE |
| `internal/daemon/network.go` | `CertificateReady` | `len(cfg.Certificates) > 0` | No for stored cert path | ✗ DISCONNECTED |
| `internal/daemon/reconciler.go` | `domains`, `selected`, `scheme` | Labels only + empty prefs + override port 0 | No for merged override/config-driven mapping | ✗ DISCONNECTED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---------|---------|--------|--------|
| DNS/proxy/cert/admin phase tests execute | `go test ./internal/dns/... ./internal/proxy/... ./internal/certs/... ./internal/daemon/... ./internal/admin/...` | all packages `ok` | ✓ PASS |
| Runtime has concrete listener bind/start implementation | `grep` for `ListenAndServe|net.Listen|tls.Listen` in `internal/*.go` | no matches | ✗ FAIL |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|------------|-------------|-------------|--------|----------|
| NET-01 | 02-02 | Managed suffix resolves to 127.0.0.1 | ✓ SATISFIED | `internal/dns/server.go:41-50`, tests `internal/dns/server_test.go:10-43`. |
| NET-02 | 02-01, 02-03 | HTTP proxies active route to selected localhost port/scheme | ✗ BLOCKED | Reconciler does not merge override inputs (`internal/daemon/reconciler.go:49,56`) so selected mapping is incomplete. |
| NET-03 | 02-04, 02-05 | HTTPS with mkcert trusted certificates | ✗ BLOCKED | mkcert/store outputs are not wired into runtime (`internal/daemon/network.go:27-46`; no usage of `BuildCertificateInventory`/`Issue`). |
| NET-04 | 02-01, 02-04, 02-05 | Regenerate when hostname set changes; reuse when unchanged | ✗ BLOCKED | Inventory logic exists (`internal/certs/store.go`) but disconnected from runtime assembly path. |
| NET-05 | 02-01, 02-03, 02-05 | HTTP:80 + HTTPS:443, redirect off by default | ✗ BLOCKED | Redirect default present (`internal/config/config.go:39`), but no listener bind/start implementation found in `internal/`. |
| NET-06 | 02-03 | WebSocket traffic through devproxy | ✗ BLOCKED | HTTP proxy preserves upgrade headers (`internal/proxy/http_test.go:121-155`) but upstream route mapping source is incomplete due reconcile override gap. |
| NET-07 | 02-02, 02-03 | Friendly no-route response for managed hostnames | ✗ BLOCKED | HTTP friendly response works, HTTPS no-route blocked in TLS selector (`internal/proxy/https.go:74-77`). |
| NET-08 | 02-01, 02-02, 02-03 | Pause routing while DNS still resolves and friendly paused response | ✗ BLOCKED | DNS/pause logic exists, HTTP paused response exists; HTTPS paused/no-route path can be blocked by cert selector gate for non-active managed hosts. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/daemon/reconciler.go` | 56 | Empty `routing.RoutePreferences{}` passed into domain generation | 🛑 Blocker | Drops override-driven mapping data from published routes. |
| `internal/daemon/reconciler.go` | 49 | `SelectPublishedPort(..., 0)` hardcoded config override port | 🛑 Blocker | Ignores config-selected upstream port behavior. |
| `internal/proxy/https.go` | 75-77 | Active-route gate inside TLS cert selector | 🛑 Blocker | Prevents friendly HTTPS no-route behavior. |
| `internal/daemon/network.go` | 55 | DNS health initialized `Bound: true` before bind | ⚠️ Warning | Health/status can report false-positive DNS bound state. |

### Human Verification Required

None. Current blockers are code-level and directly observable.

### Gaps Summary

Phase 02 does not achieve the goal contract yet. DNS managed-suffix answering and HTTP fallback behavior are implemented, but three blocker integration gaps remain and align with the advisory review: (1) reconciler publishes incomplete effective routing metadata, (2) HTTPS no-route behavior is blocked at TLS handshake, and (3) certificate planning/issuance artifacts are disconnected from runtime assembly. Additionally, runtime listener binding for 80/443 is not implemented in internal runtime code. These block NET-02 through NET-08 goal-level completion.

---

_Verified: 2026-05-05T22:26:47Z_  
_Verifier: the agent (gsd-verifier)_
