# Domain Pitfalls

**Domain:** macOS Docker Compose local vanity-domain proxy (DNS + HTTP/HTTPS)
**Researched:** 2026-05-05

## Critical Pitfalls

Mistakes that cause rewrites or major production-of-the-tool failures.

### Pitfall 1: "DNS looks configured" but traffic never reaches devproxy DNS
**What goes wrong:** `/etc/resolver/test` exists, but requests still do not resolve as expected (or intermittently resolve), so users blame proxy routing when the issue is actually resolver selection/caching.
**Why it happens:** macOS uses a multi-client “Super” resolver strategy; `/etc/resolver/*` is only one input, and app behavior differs (`ping`/browser via system resolver vs `dig`/`nslookup` behavior). DNS cache and resolver precedence can hide mistakes.
**Consequences:** High support burden, false bug reports, route trust collapse (“devproxy is flaky”).
**Prevention:**
- In `install` and `doctor`, verify with **system-resolver path** (not only raw DNS query):
  - `scutil --dns` contains supplemental resolver for suffix
  - probe lookup via system APIs (or `ping`-style check), not only `dig`
- Show explicit troubleshooting in `doctor`: cache flush command guidance and expected `scutil --dns` signatures.
- Keep resolver scope strict to configured suffix only (avoid global resolver side effects).
**Detection (warning signs):**
- `devproxy status` says healthy, but browser says `DNS_PROBE_FINISHED_NXDOMAIN`.
- `dig foo.test @127.0.0.1 -p 53535` works while browser/`ping` fails.
- Support reports “works after network toggle/reboot” (classic resolver state drift).

### Pitfall 2: Port 80/443 bind strategy ignored until late
**What goes wrong:** Daemon can’t bind privileged ports under normal user context; install appears successful, runtime fails after reboot or only in daemon mode.
**Why it happens:** On Darwin/BSD semantics, binding ports `<1024` requires elevated privileges/root-capable path; many projects only test foreground with sudo and miss launch/runtime reality.
**Consequences:** Non-functional core routing, brittle install/uninstall, user distrust.
**Prevention:**
- Treat low-port binding as a **first-class install gate**, not runtime surprise.
- Fail installation if 80/443 strategy is not viable, with actionable remediation.
- In `doctor`, separately report:
  - port occupancy conflict (another process already on 80/443)
  - permission model conflict (insufficient privileges)
- Keep fallback behavior explicit (no silent auto-port remap for v1).
**Detection (warning signs):**
- Works only when launched manually with sudo.
- LaunchAgent shows running, but no listeners on `127.0.0.1:80` / `:443`.
- Frequent “address already in use” or “permission denied” after install.

### Pitfall 3: Docker event stream treated as always-on and lossless
**What goes wrong:** Route table drifts from reality after Docker Desktop restart, socket hiccup, or stream EOF; stale routes keep serving wrong targets.
**Why it happens:** `docker events` is real-time stream, not guaranteed forever-healthy; only limited historical events are returned, and Desktop proxy behavior can differ from raw Engine behavior.
**Consequences:** Wrong routing, ghost routes, hard-to-reproduce bugs, restarts needed to recover.
**Prevention:**
- Implement reconnection loop with backoff + jitter.
- On reconnect, run full container re-inspection and reconcile route table atomically.
- Health should degrade when watcher disconnected beyond threshold.
- Keep `refresh` cheap and safe; expose “last successful Docker sync” timestamp.
**Detection (warning signs):**
- Logs show `unexpected EOF` / stream closed and no automatic recovery.
- `routes` includes containers no longer running.
- `status` still “healthy” while Docker actions fail.

### Pitfall 4: Trust model confusion for TLS (mkcert + local CA)
**What goes wrong:** HTTPS appears “enabled” but browsers still warn, Firefox differs from Safari/Chrome, or certs exist but chain isn’t trusted.
**Why it happens:** mkcert installs CA to trust stores with platform/tool-specific caveats; different apps may use different trust stores; teams confuse cert generation with trust installation.
**Consequences:** Users disable HTTPS, ignore warnings, or abandon tool due to perceived insecurity.
**Prevention:**
- Split TLS diagnostics into distinct checks:
  1) mkcert present
  2) CA installed in system trust store
  3) certificate files present for active hostnames
  4) handshake test against local proxy hostname
- Surface browser-specific notes (e.g., NSS/Firefox dependency) in `doctor` output.
- Never claim HTTPS healthy from file existence alone.
**Detection (warning signs):**
- “The local CA is not installed…” warnings.
- Works in one browser but not another.
- Cert files generated but browser shows untrusted issuer.

### Pitfall 5: Domain conflict policy is non-deterministic or opaque
**What goes wrong:** Multiple containers claim same domain and route “flaps” based on event order/startup timing.
**Why it happens:** Missing deterministic tie-break and poor conflict surfacing.
**Consequences:** Heisenbugs across teams/projects; impossible debugging.
**Prevention:**
- Enforce deterministic winner selection (priority, then stable tie-break).
- Emit conflict events to logs, `doctor`, dashboard, and `routes` state.
- Include losing candidates and why they lost.
**Detection (warning signs):**
- Route target changes across restarts with same container set.
- Different team members report different destination for same hostname.
- Conflict only visible in debug logs (not in user-facing surfaces).

## Moderate Pitfalls

### Pitfall 1: Route selection grabs the wrong published port
**What goes wrong:** Proxy forwards to DB/admin/secondary port instead of app HTTP port.
**Prevention:**
- Keep explicit port precedence policy and validate chosen port is actually routable HTTP(S)-ish unless explicitly overridden.
- Warn when fallback chooses “first published port”.
**Warning signs:**
- 502/connection reset for some services while container is healthy.
- Route points to known non-HTTP ports in `routes` output.

### Pitfall 2: Explicit custom domains outside managed suffix silently fail
**What goes wrong:** User sets `foo.internal` or other suffix; route exists but DNS never resolves locally.
**Prevention:**
- Allow route, but produce strong warning when suffix not managed by devproxy resolver.
- In `doctor`, list unmanaged explicit domains and required external DNS action.
**Warning signs:**
- Route table shows domain as active but all clients get NXDOMAIN.

### Pitfall 3: Security boundary drift from published-port assumptions
**What goes wrong:** Teams assume “local only” while Compose publishes to all interfaces (`0.0.0.0`) and services are reachable from LAN in some configurations.
**Prevention:**
- Warn when upstream chosen host binding is non-loopback.
- Document that devproxy doesn’t harden upstream container exposure; it routes what Docker publishes.
**Warning signs:**
- Service reachable from another machine on local network.

## Minor Pitfalls

### Pitfall 1: Debug surfaces disagree
**What goes wrong:** `status`, dashboard, and logs report different route/health state.
**Prevention:**
- Single internal health model; all UIs read from same source of truth.
**Warning signs:**
- “Healthy” status with visible runtime errors elsewhere.

### Pitfall 2: Uninstall leaves resolver/certs in surprising state
**What goes wrong:** Users remove tool but stale resolver/certs remain (or are removed unexpectedly).
**Prevention:**
- Interactive uninstall choices + explicit summary of what remains.
**Warning signs:**
- Post-uninstall DNS still affected; reinstall behaves unexpectedly.

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Milestone 1–2 (Discovery + Events) | Event-stream disconnect causes stale routes | Reconnect loop + full reconcile on reconnect + health degradation when unsynced |
| Milestone 3 (HTTP Proxy) | Wrong host/forwarded-header semantics reduce app compatibility | Normalize Host + X-Forwarded-* behavior, explicit tests for framework expectations |
| Milestone 4 (DNS) | Resolver misrouting/caching makes valid routes appear broken | `scutil --dns` checks, system-resolver probes, cache troubleshooting in doctor |
| Milestone 5 (HTTPS) | mkcert trust incomplete across environments/browsers | Multi-step TLS health checks + browser trust notes + fail-fast prerequisites |
| Milestone 6 (LaunchAgent + Ops) | Port privilege/occupancy discovered too late | Install-time bind validation + clear remediation + explicit failure modes |
| Milestone 7 (Menu bar UX) | Observability split-brain between UI and daemon | UI strictly consumes daemon state API + conflict/health parity checks |

## Sources

- Docker docs — Port publishing and mapping (default exposure, localhost binding caveats): https://docs.docker.com/engine/network/port-publishing/ (**HIGH**)
- Docker docs — `docker system events` semantics (stream behavior, event history limits): https://docs.docker.com/reference/cli/docker/system/events/ (**HIGH**)
- macOS resolver man page (multi-client DNS strategy, `/etc/resolver` semantics): https://www.manpagez.com/man/5/resolver/ (**MEDIUM**; mirror of Apple man content)
- mkcert official README (trust stores, CA install behavior, security warning for root CA key): https://github.com/FiloSottile/mkcert (**HIGH**)
- Go `httputil.ReverseProxy` source/docs notes (X-Forwarded spoofing and rewrite pitfalls): https://github.com/golang/go/blob/master/src/net/http/httputil/reverseproxy.go (**HIGH**)
- Docker Desktop/macOS-specific events-stream proxy discrepancy discussion (needs ongoing validation): https://github.com/moby/moby/issues/48536 (**LOW-MEDIUM**, issue-thread evidence)
