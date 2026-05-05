# Architecture Patterns

**Domain:** macOS local vanity-domain proxy for Docker Compose
**Researched:** 2026-05-05

## Recommended Architecture

Use a **single daemon process with internal modules** (not microservices), plus thin clients (CLI + menu bar) over a local Unix socket API.

```text
Docker Engine events + periodic snapshot
  -> Discovery Pipeline
    -> Candidate Routes
      -> Route Resolution Engine (priority + deterministic tie-break)
        -> Immutable Active Route Registry (in-memory)
          -> Fan-out:
             1) DNS responder (*.suffix -> 127.0.0.1)
             2) HTTP/HTTPS proxy host router (Host header -> upstream)
             3) Admin API read-model (status/routes/conflicts/health)
                 -> CLI commands
                 -> Menu bar + local dashboard

Certificate Manager (mkcert-backed)
  <- receives hostname set deltas from Route Registry
  -> updates cert cache for HTTPS listener

Installer/Operator layer (install/doctor/status/uninstall)
  -> launchd, resolver files, prerequisites, diagnostics
```

This shape is the 2026 default for local-dev infra tools because:
- Local reliability is higher with one supervising daemon (fewer process boundaries to fail).
- Route decisions stay consistent when DNS/proxy/API all read the same in-memory registry.
- UX remains simple: one binary, one LaunchAgent, one socket contract.

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| **Discovery Adapter** | Subscribes to Docker events; performs startup/full refresh container listing and inspect | Docker Engine API, Route Resolution Engine |
| **Route Resolution Engine** | Applies eligibility rules, label/config precedence, port selection, conflict resolution, deterministic tie-break | Discovery Adapter, Route Registry, Event/Warning Store |
| **Route Registry (authoritative state)** | Holds current active routes + metadata (winner/losers, reasons, timestamps, cert hostnames) | DNS server, Proxy router, Admin API, Cert Manager |
| **DNS Server** | Resolves only managed suffix to `127.0.0.1`; no per-route coupling | Route Registry (suffix/pause state only) |
| **Proxy Router (HTTP+HTTPS)** | Host-header lookup, reverse proxying to selected localhost published port, friendly no-route/paused pages | Route Registry, Cert Manager, ReverseProxy transport |
| **Certificate Manager** | Manages local cert material (mkcert invocation, cache, rotation on hostname-set change) | Route Registry, HTTPS listener, filesystem |
| **Admin API (Unix socket)** | Read-only and operator actions (`refresh`, `pause`, `resume`), unified health/conflicts/errors | Route Registry, Event/Warning Store, Installer checks |
| **CLI Surface** | Install/uninstall/doctor/status/routes/logs; no business logic duplication | Admin API, Installer primitives |
| **Menu Bar Surface** | Health badge, route list, quick actions; never owns routing | Admin API |
| **Installer + launchd Integration** | `/etc/resolver/*`, LaunchAgent lifecycle, prerequisite checks (mkcert/ports/socket) | macOS system APIs/tools, daemon |

Boundary rule: **only Route Resolution Engine mutates the Route Registry**. Everyone else reads snapshots or submits explicit commands.

### Data Flow

1. **Boot / refresh**
   - Daemon starts, validates prerequisites, loads config.
   - Discovery Adapter performs full running-container snapshot first (avoid missing state before live stream catches up).
   - Snapshot enters Route Resolution Engine.

2. **Continuous updates**
   - Docker emits lifecycle events (`start`, `stop`, `die`, `destroy`, `rename`, `update`, etc.).
   - Discovery Adapter normalizes event -> container inspect data.
   - Route Resolution Engine recomputes affected project/service routes.

3. **Route decisioning**
   - Eligibility filter: running + published TCP port + not disabled.
   - Precedence application: disable hard-stop, then labels override config on same field.
   - Conflict resolution: highest priority wins; deterministic tie-break for equal priority.
   - Registry replaced atomically (copy-on-write snapshot), emitting change event.

4. **Serving path**
   - DNS answers managed suffix queries to `127.0.0.1`.
   - Client connects to local proxy on 80/443.
   - Proxy strips host port, route lookup by hostname in active registry, forwards to `127.0.0.1:{published_port}`.
   - If paused or missing route, return deterministic local explanatory response.

5. **TLS path**
   - Registry hostname delta triggers Certificate Manager reconciliation.
   - Cert Manager reuses existing certs when SAN set unchanged; regenerates only on hostname-set change.
   - HTTPS listener reads updated cert material without route-table ownership.

6. **Operator/UI path**
   - CLI and menu bar call Admin API over Unix socket.
   - API returns same route/conflict/health view used by dashboard and doctor.

Directionality principle: **Docker -> Resolution -> Registry -> (DNS/Proxy/API/UI)**. UI/tools never write route state directly.

## Patterns to Follow

### Pattern 1: Event-stream + periodic/full reconciliation
**What:** Use Docker events for low-latency updates, but always support full resync on startup and manual refresh.
**When:** Always (event streams can be interrupted or lose context).
**Example:**
```typescript
onDaemonStart() => fullSnapshotReconcile()
onDockerEvent(e) => reconcile(containerIDFrom(e))
onRefreshCommand() => fullSnapshotReconcile()
```

### Pattern 2: Copy-on-write registry snapshots
**What:** Compute new route table off-thread, then atomically swap active snapshot.
**When:** Every route recomputation.
**Example:**
```typescript
next = computeRoutes(current, change)
atomicSwap(activeRegistry, next)
publishRouteVersion(next.version)
```

### Pattern 3: Strict write ownership
**What:** Single writer (resolution engine), many readers (DNS/proxy/API).
**When:** Core state management.
**Example:**
```typescript
// Only resolver mutates:
resolver.apply(discoveryEvent)
// Consumers only read:
proxy.lookup(host)
dns.isManagedSuffix(qname)
```

### Pattern 4: Transport-adapter split in proxy
**What:** Separate route lookup/policy from HTTP transport behavior.
**When:** Proxy implementation.
**Example:**
```typescript
upstream = routePolicy.resolve(host)
reverseProxy.rewrite(setXForwarded, upstream)
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: DNS coupled to per-route existence
**What:** Returning NXDOMAIN when route absent.
**Why bad:** Breaks predictable local resolution and creates confusing browser/network behavior.
**Instead:** Resolve managed suffix consistently; let proxy return friendly no-route/paused response.

### Anti-Pattern 2: Route mutation from CLI/menu bar
**What:** Multiple writers adjusting route table directly.
**Why bad:** Race conditions and inconsistent state across surfaces.
**Instead:** CLI/UI send commands to daemon; daemon reconciles and publishes one authoritative snapshot.

### Anti-Pattern 3: One cert per request or lazy first-hit cert creation
**What:** Generate certs on demand during incoming traffic.
**Why bad:** User-facing TLS failures and latency spikes.
**Instead:** Reconcile certs during route updates before traffic arrives.

### Anti-Pattern 4: Treating Docker events as complete truth
**What:** Never re-snapshot after disconnects/restarts.
**Why bad:** Stale or orphaned routes.
**Instead:** startup full scan + explicit refresh + reconnect resync.

## Scalability Considerations

| Concern | At 100 users | At 10K users | At 1M users |
|---------|--------------|--------------|-------------|
| **Per-machine route count** | Usually tens of routes; in-memory maps trivial | Hundreds possible; still trivial for Go maps | N/A for local tool; scale is organizational distribution, not central QPS |
| **Event churn** | Container events modest | Higher churn in monorepo dev setups; debounce + targeted recompute | N/A centralized scale not applicable |
| **Cert operations** | Infrequent | Can spike on frequent compose up/down; batch hostname delta processing | N/A |
| **Operator supportability** | Manual debugging acceptable | Need consistent doctor/status telemetry format | Need release/upgrade and diagnostics discipline, not runtime sharding |

## Suggested Build Order (Roadmap Implications)

1. **Core state + discovery skeleton first**
   - Build Discovery Adapter, Route Resolution Engine, and in-memory Route Registry before any networking listeners.
   - Why: every other component depends on authoritative route state.

2. **Admin API + CLI visibility second**
   - Expose `status/routes/refresh/conflicts` early.
   - Why: makes DNS/proxy development debuggable and reduces blind troubleshooting.

3. **HTTP proxy before DNS**
   - Validate host-routing correctness via direct Host-header tests (or curl with `--resolve`) before system resolver changes.
   - Why: isolates application routing errors from OS DNS setup issues.

4. **DNS + resolver install next**
   - Add suffix resolution and friendly no-route behavior once proxy is proven.
   - Why: this is where end-to-end “vanity domain” experience becomes real.

5. **TLS/cert manager after HTTP path is stable**
   - Integrate mkcert reconciliation and HTTPS listener; keep HTTP fallback for diagnosis.
   - Why: TLS adds operational complexity and should sit on proven routing.

6. **launchd + install/doctor hardening**
   - Productionize lifecycle, restart behavior, permissions, socket ownership.
   - Why: reliability phase after functional correctness.

7. **Menu bar/dashboard last**
   - Build on stable Admin API contract.
   - Why: avoid UI churn caused by backend contract instability.

## Sources

- Docker events reference (real-time stream semantics and event types): https://docs.docker.com/reference/cli/docker/system/events/  
  **Confidence:** HIGH
- Docker Compose labels (canonical `com.docker.compose.project` and `com.docker.compose.service` labels): https://docs.docker.com/compose/compose-file/05-services/#labels  
  **Confidence:** HIGH
- Go `net/http/httputil.ReverseProxy` docs (`Rewrite`, `SetXForwarded`, security notes): https://pkg.go.dev/net/http/httputil#ReverseProxy  
  **Confidence:** HIGH
- Apple launchd daemon/agent guide (LaunchAgents/LaunchDaemons model and behavior constraints): https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html  
  **Confidence:** MEDIUM (archived doc, still directionally valid; confirm with current `launchd.plist` man pages during implementation)
- mkcert README (local CA + local-trust cert workflow and security caveats): https://github.com/FiloSottile/mkcert  
  **Confidence:** MEDIUM (official repo, but release cadence is slow; validate current compatibility at implementation time)
