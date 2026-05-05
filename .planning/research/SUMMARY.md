# Project Research Summary

**Project:** devproxy  
**Domain:** macOS local vanity-domain proxy for Docker Compose  
**Researched:** 2026-05-05  
**Confidence:** HIGH

## Executive Summary

devproxy should be built as a macOS-first local infrastructure tool: one Go daemon that discovers Docker Compose services, computes deterministic host routes, and serves DNS + HTTP/HTTPS from a single authoritative route registry. The winning v1 shape is intentionally narrow: local-only routing on loopback, published ports only, no compose file mutation, and no sidecar proxy requirement. That focus is what creates the UX advantage (`docker compose up` and usable local domains with HTTPS).

The recommended implementation path is: establish reliable discovery and state reconciliation first, expose strong operator visibility early (`status`, `routes`, `doctor`), then layer HTTP routing, DNS integration, TLS, and lifecycle hardening. This sequencing reduces debugging ambiguity and avoids coupling networking complexity to immature core state logic.

The highest risks are operational, not algorithmic: macOS resolver behavior, privileged 80/443 binding, Docker event stream drift, and TLS trust-store mismatch. Mitigation is clear and should be designed in from day one: install-time gating, full-resync on stream reconnect, deterministic conflict policy, and health diagnostics that validate real system behavior (not just internal assumptions).

## Key Findings

### Recommended Stack

Research strongly supports a Go-first stack with stdlib networking primitives and minimal dependencies. The safest v1 approach is one static binary plus macOS-native integration points (launchd, `/etc/resolver`, mkcert), with optional UX layers (menubar) deferred until core reliability is proven.

**Core technologies:**
- **Go 1.26.x**: primary implementation language — best fit for long-running local daemons and strong networking stdlib.
- **`net/http` + `httputil.ReverseProxy`**: HTTP reverse proxy + admin API — modern `Rewrite` path avoids older `Director` pitfalls.
- **`moby/moby` Go client**: Docker discovery/events — official event stream + inspect/list support.
- **`miekg/dns`**: authoritative local DNS for managed suffix — proven and simple for in-process DNS handling.
- **Cobra + Viper**: CLI and config layering — practical command/config ergonomics for install/runtime ops.
- **mkcert CLI**: local CA and cert issuance — standard local HTTPS approach for dev workflows.
- **launchd + `/etc/resolver/<suffix>`**: macOS service and DNS integration — native and reliable for v1.

Critical version/strategy requirements:
- Use Go’s current `ReverseProxy` rewrite model (avoid legacy director assumptions).
- Treat privileged 80/443 binding as an install gate.
- Use startup snapshot + event stream reconciliation, not event stream alone.

### Expected Features

v1 success depends on table-stakes reliability over breadth. Users expect auto-discovery, deterministic naming, wildcard DNS, host-based proxying, trusted HTTPS, explicit conflict handling, and fast diagnostics. Differentiators matter, but only after core routing trust is established.

**Must have (table stakes):**
- Automatic Compose service discovery with deterministic domain generation.
- Wildcard local DNS for managed suffix + host-header HTTP routing.
- Trusted local HTTPS with mkcert lifecycle checks.
- Deterministic conflict resolution with surfaced winner/loser reasoning.
- Strong operational commands (`status`, `routes`, `doctor`, logs).
- macOS-native install/uninstall and local-only safety defaults.

**Should have (competitive):**
- Compose-change-free onboarding in common cases.
- Laravel Sail-first defaults for high-frequency workflow fit.
- Cross-surface conflict UX (CLI/dashboard/logs parity).
- Pause/resume routing mode with explicit paused responses.

**Defer (v2+):**
- Full menubar/dashboard polish as primary surface.
- Non-HTTP(S) protocols, path-routing DSL/middleware chains.
- Cross-platform support and public exposure/tunnel capabilities.

### Architecture Approach

Use a single daemon with strict write ownership: Discovery Adapter feeds a Route Resolution Engine, which alone mutates a copy-on-write in-memory Route Registry. DNS, proxy, admin API, CLI, and menubar are read-side consumers. Certificates are reconciled proactively from route hostname deltas. This keeps behavior deterministic across every user-facing surface.

**Major components:**
1. **Discovery + Resolution Core** — consumes Docker snapshot/events and computes eligible routes with deterministic conflict policy.
2. **Authoritative Route Registry** — atomic route snapshots read by DNS/proxy/API.
3. **DNS + Proxy Serving Plane** — wildcard suffix resolution and host-header forwarding to selected localhost published ports.
4. **Certificate Manager** — mkcert-backed cert lifecycle tied to hostname-set changes.
5. **Admin API + CLI/UX Surfaces** — unified diagnostics and operator actions over Unix socket.
6. **Installer/launchd Integration** — resolver setup, lifecycle management, and prerequisite gating.

### Critical Pitfalls

1. **Resolver appears configured but system traffic bypasses it** — validate via `scutil --dns` + system-resolver probes; include cache troubleshooting in `doctor`.
2. **Late discovery of 80/443 privilege or occupancy problems** — gate install on bind viability; report permission vs occupancy separately.
3. **Event-stream-only state leads to stale/ghost routes** — reconnect with backoff and force full reconcile on reconnect/start.
4. **TLS “enabled” without trust correctness** — split diagnostics into mkcert presence, CA install, cert material, and handshake validation.
5. **Opaque/non-deterministic domain conflicts** — enforce stable tie-break and expose loser reasons in all interfaces.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Discovery, Resolution, and Authoritative State
**Rationale:** Every downstream capability depends on correct route computation and deterministic conflict handling.  
**Delivers:** Docker snapshot/events ingestion, eligibility engine, deterministic naming, conflict policy, copy-on-write route registry.  
**Addresses:** Auto-discovery, deterministic naming, conflict handling, local-only defaults.  
**Avoids:** Event-stream drift and non-deterministic routing.

### Phase 2: Admin API + Operator Visibility
**Rationale:** Make internals observable before networking layers to cut debugging time.  
**Delivers:** Unix-socket admin API, `status/routes/refresh/doctor` baseline, health model, sync timestamps.  
**Uses:** Cobra/Viper, structured logging pattern, registry read models.  
**Implements:** Unified diagnostics plane consumed by all future UX surfaces.

### Phase 3: HTTP Proxy Serving Path
**Rationale:** Validate host-routing semantics before introducing OS DNS complexity.  
**Delivers:** Host-header lookup, reverse proxy forwarding, friendly no-route responses, forwarded-header normalization.  
**Addresses:** Core value of domain-to-service access.  
**Avoids:** Conflating DNS issues with routing bugs.

### Phase 4: DNS Integration + macOS Resolver Install
**Rationale:** DNS makes vanity-domain UX real once routing logic is stable.  
**Delivers:** Managed-suffix DNS server (`*.test`), `/etc/resolver` install flow, resolver-aware doctor checks.  
**Addresses:** Wildcard domain expectation and zero-config feel.  
**Avoids:** Resolver precedence/caching false negatives.

### Phase 5: HTTPS and Certificate Reconciliation
**Rationale:** HTTPS is table stakes but operationally fragile; add after HTTP/DNS confidence.  
**Delivers:** mkcert prerequisite checks, hostname-delta cert reconciliation, HTTPS listener, handshake validation in doctor.  
**Addresses:** Secure-context dev workflows and browser trust.  
**Avoids:** Browser warning churn and lazy cert-generation latency failures.

### Phase 6: launchd Hardening, Install/Uninstall Reliability, and Optional Menubar
**Rationale:** Productionize lifecycle and polish only after core path is dependable.  
**Delivers:** launchd robustness, privilege/occupancy gate hardening, uninstall cleanup flows, optional menubar atop stable API.  
**Addresses:** Always-on reliability and differentiator UX.  
**Avoids:** Split-brain observability and brittle post-reboot behavior.

### Phase Ordering Rationale

- Dependency-first sequencing: route state -> observability -> serving -> DNS -> TLS -> lifecycle polish.
- Architecture-aligned grouping keeps a single source of truth and minimizes cross-component race risk.
- Early diagnostics and deterministic conflict policy directly neutralize the highest-severity pitfalls.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 4 (DNS Integration):** macOS resolver precedence, cache behavior, and edge cases across network contexts need implementation-time validation.
- **Phase 5 (HTTPS):** browser trust-store differences (especially Firefox/NSS) and mkcert lifecycle edge cases need environment-specific testing guidance.
- **Phase 6 (launchd Hardening):** current launchd daemon/agent best-practice details should be validated against latest man pages and OS version behavior.

Phases with standard patterns (can likely skip extra research-phase):
- **Phase 1 (Discovery/Resolution):** well-established Docker event + reconcile patterns.
- **Phase 2 (Admin API/CLI visibility):** straightforward local daemon observability pattern.
- **Phase 3 (HTTP Proxy):** mature ReverseProxy host-routing patterns in Go stdlib.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Core recommendations are anchored in official Go/Docker/mkcert/macOS primitives; only optional dependencies are medium certainty. |
| Features | HIGH | Strong convergence across mature comparable tools (Traefik, OrbStack, Valet, DDEV, nginx-proxy). |
| Architecture | HIGH | Pattern fit is consistent with local infra tooling and dependency graph is coherent; minor uncertainty is launchd doc freshness. |
| Pitfalls | MEDIUM-HIGH | Critical risks are well-evidenced, but some macOS/Docker Desktop edge behaviors rely on issue-level evidence and require real-world validation. |

**Overall confidence:** HIGH

### Gaps to Address

- **launchd implementation details by macOS version:** validate with current `launchd.plist`/`launchctl` behavior during build.
- **Docker Desktop event-stream edge cases:** test disconnect/reconnect and Desktop restarts under load to harden resync guarantees.
- **Browser-specific trust behavior:** verify and document Firefox/NSS flows and enterprise-managed trust-store constraints.
- **Port-selection heuristics:** define explicit precedence and warning UX for ambiguous multi-port containers.
- **Unmanaged custom suffixes:** clarify operator UX for explicit domains outside managed resolver scope.

## Sources

### Primary (HIGH confidence)
- Docker official docs (`docker events`, Compose labels, port publishing) — discovery semantics, metadata model, exposure caveats.
- Go official docs (`net/http/httputil.ReverseProxy`) — safe rewrite/forwarding behaviors.
- mkcert official repository docs — local CA install and cert issuance model.
- Official docs for Traefik, OrbStack, Valet, DDEV, nginx-proxy — expected feature baseline and competitive patterns.

### Secondary (MEDIUM confidence)
- Apple launchd archived guide + resolver man pages — architecture direction is valid but needs current-OS verification.
- Viper/systray ecosystem docs — useful implementation options, not critical-path requirements.

### Tertiary (LOW confidence)
- Moby issue discussion on Docker Desktop event-stream discrepancies — directional warning, requires empirical validation in devproxy test matrix.

---
*Research completed: 2026-05-05*  
*Ready for roadmap: yes*
