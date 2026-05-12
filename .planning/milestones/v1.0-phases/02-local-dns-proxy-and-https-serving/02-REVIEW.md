---
phase: 02-local-dns-proxy-and-https-serving
reviewed: 2026-05-05T22:00:52Z
depth: deep
files_reviewed: 36
files_reviewed_list:
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-CONTEXT.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-01-PLAN.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-02-PLAN.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-03-PLAN.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-04-PLAN.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-05-PLAN.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-local-dns-proxy-and-https-serving-01-SUMMARY.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-local-dns-proxy-and-https-serving-02-SUMMARY.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-local-dns-proxy-and-https-serving-03-SUMMARY.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-local-dns-proxy-and-https-serving-04-SUMMARY.md
  - .planning/phases/02-local-dns-proxy-and-https-serving/02-local-dns-proxy-and-https-serving-05-SUMMARY.md
  - internal/routing/types.go
  - internal/daemon/reconciler.go
  - internal/daemon/reconciler_test.go
  - internal/config/config.go
  - internal/config/config_test.go
  - internal/dns/server.go
  - internal/dns/server_test.go
  - internal/proxy/http.go
  - internal/proxy/http_test.go
  - internal/admin/routes.go
  - internal/admin/logs.go
  - internal/certs/store.go
  - internal/certs/store_test.go
  - internal/certs/mkcert.go
  - internal/certs/mkcert_test.go
  - internal/proxy/https.go
  - internal/proxy/https_test.go
  - internal/daemon/network.go
  - internal/admin/status.go
  - internal/admin/status_test.go
  - internal/admin/doctor.go
  - internal/discovery/ports.go
  - internal/discovery/metadata.go
  - internal/routing/overrides.go
  - internal/routing/domains.go
findings:
  critical: 3
  warning: 1
  info: 0
  total: 4
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-05-05T22:00:52Z
**Depth:** deep
**Files Reviewed:** 36
**Status:** issues_found

## Summary

Reviewed the Phase 02 context plus the implementation for plans 02-01 through 02-05. The main problems are cross-plan integration gaps: the reconciler still publishes incomplete route metadata, HTTPS cannot actually reach the friendly no-route path once TLS is involved, and the daemon runtime is not wired to the certificate artifacts produced in plan 04.

## BLOCKER Issues

### CR-01: HTTPS no-route handling fails during TLS handshake

**Classification:** BLOCKER  
**File:** `internal/proxy/https.go:74-77`

**Issue:** `getCertificate` refuses every managed hostname that does not currently have an active route. That means a request like `https://missing.acme.test` never reaches `HandleHTTPS`; the handshake fails first with `no active managed route`, so the phase's promised friendly HTTPS no-route/paused behavior is impossible for managed hosts that fall under an existing certificate's coverage.

**Fix:** Select a certificate for managed hosts independently from route activity, then let the shared HTTP handler decide between proxy/no-route/paused.

```go
lookup := l.dnsLookup.LookupHostname(host)
if !lookup.Managed {
    return nil, fmt.Errorf("unmanaged host %s", host)
}

// choose any exact/wildcard cert that covers host
if cert := l.findCertificate(host); cert != nil {
    return cert, nil
}
return nil, fmt.Errorf("no certificate available for %s", host)
```

### CR-02: Reconciler still ignores override-driven route and upstream inputs

**Classification:** BLOCKER  
**File:** `internal/daemon/reconciler.go:47-63`

**Issue:** The reconciler only applies label fields, passes `0` as the config override port to `SelectPublishedPort`, and calls `GenerateDomains` with an empty `routing.RoutePreferences{}`. That drops config overrides and label-driven domain/root/domain-list preferences from the published snapshot, so DNS answers, proxy targets, and certificate inventory are computed from incomplete route metadata instead of the effective winning configuration required by the phase context.

**Fix:** Build merged route preferences before generating domains or selecting ports, and pass the effective override values through reconciliation.

```go
configPrefs := loadProjectServiceOverrides(...)
labelPrefs := routing.RoutePreferences{...
    Domain: labelDomain,
    Domains: labelDomains,
    Root: labelRoot,
    Port: labelPort,
    Scheme: labelScheme,
}
prefs, warnings := routing.MergeOverrides(configPrefs, labelPrefs)
selected, source, ok := discovery.SelectPublishedPort(c.Ports, routeOpts, valueOrZero(prefs.Port))
domains, domainWarnings := routing.GenerateDomains(candidate.Project, candidate.Service, prefs, ...)
```

### CR-03: Network runtime is not connected to stored mkcert outputs

**Classification:** BLOCKER  
**File:** `internal/daemon/network.go:27-31,43-60`

**Issue:** Plan 04 produces `StoredCertificate` path metadata, and `internal/proxy/https.go` can load it via `HTTPSListenerConfig.Stored`, but `NetworkRuntimeConfig` only accepts an in-memory `map[string]tls.Certificate` and `NewNetworkRuntime` never forwards stored certs. In the reviewed code there is no daemon-owned path from plan 04's prepared certificate state into the plan 05 runtime, so HTTPS cannot be assembled from the certificate artifacts this phase generated.

**Fix:** Extend `NetworkRuntimeConfig` to accept stored certificates and pass them through to `NewHTTPSListener`; set `CertificateReady` from either loaded in-memory certs or stored cert inventory.

```go
type NetworkRuntimeConfig struct {
    ManagedSuffix string
    Snapshot      func() routing.Snapshot
    RoutingPaused func() bool
    Certificates  map[string]tls.Certificate
    Stored        []certs.StoredCertificate
}

httpsHandler, err := proxy.NewHTTPSListener(proxy.HTTPSListenerConfig{
    ManagedSuffix: cfg.ManagedSuffix,
    Snapshot:      cfg.Snapshot,
    RoutingPaused: cfg.RoutingPaused,
    Certificates:  cfg.Certificates,
    Stored:        cfg.Stored,
})
```

## Warnings

### WR-01: Runtime health reports DNS as bound before any bind attempt

**Classification:** WARNING  
**File:** `internal/daemon/network.go:54-60`

**Issue:** `NewNetworkRuntime` initializes DNS health as `Enabled: true, Bound: true` even though no bind has happened yet. Until `SetDNSBindResult` runs, status/doctor surfaces will report a healthy DNS listener whether startup has succeeded, failed, or not been attempted.

**Fix:** Start all listener health in an unbound state and only flip `Bound` after a successful bind.

```go
runtime.health = NetworkRuntimeHealth{
    DNS:   ListenerHealth{Enabled: true, Bound: false, BindAddress: "127.0.0.1:53535"},
    HTTP:  ListenerHealth{Enabled: true, Bound: false, BindAddress: "127.0.0.1:80"},
    HTTPS: ListenerHealth{Enabled: true, Bound: false, BindAddress: "127.0.0.1:443"},
}
```

---

_Reviewed: 2026-05-05T22:00:52Z_  
_Reviewer: the agent (gsd-code-reviewer)_  
_Depth: deep_
