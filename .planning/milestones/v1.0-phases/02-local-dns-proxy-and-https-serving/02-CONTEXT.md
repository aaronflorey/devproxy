# Phase 2: Local DNS, Proxy, and HTTPS Serving - Context

**Gathered:** 2026-05-05 (assumptions mode)
**Status:** Ready for planning

<domain>
## Phase Boundary

Make hostnames under the managed suffix resolve locally and serve active routes over HTTP and HTTPS through devproxy, including WebSocket traffic, trusted mkcert-backed TLS, clear no-route and paused responses, and the existing default that HTTP-to-HTTPS redirect stays off unless explicitly configured. Install lifecycle, resolver setup UX, daemon startup UX, and broader diagnostics remain Phase 3 scope.

</domain>

<decisions>
## Implementation Decisions

### Routing State
- **D-01:** DNS, HTTP, HTTPS, no-route, and paused request handling must all read from the reconciler's published `routing.Snapshot` rather than re-inspecting Docker or recomputing routes on the request path.

### Proxy Transport And Upstream Selection
- **D-02:** `routing.Route.Upstream` is the canonical proxy target and must be populated during reconciliation from the existing discovery and override metadata instead of the current hard-coded `127.0.0.1` plus `http` behavior.
- **D-03:** Existing override precedence remains in force for upstream selection: Docker labels win over config for overlapping route fields such as port and scheme.
- **D-04:** Use Go stdlib `net/http/httputil.ReverseProxy` as the Phase 2 proxy core for HTTP, HTTPS, and WebSocket upgrade traffic.
- **D-05:** Any listener or middleware wrapped around the reverse proxy must preserve `http.Hijacker` behavior so WebSocket upgrades continue to work.

### DNS Scope
- **D-06:** The in-process DNS responder should answer only for the configured managed suffix and return `127.0.0.1` for those matched hostnames.
- **D-07:** Explicit unmanaged domains may remain valid route declarations with warnings, but Phase 2 does not expand DNS management beyond the configured suffix.

### Certificate Strategy
- **D-08:** TLS certificate management should key off the active served hostname inventory derived from the winning routes in the snapshot.
- **D-09:** The preferred certificate unit is per project base hostname, covering `{project}.{suffix}` plus `*.{project}.{suffix}` when hostname depth stays one label below the project root.
- **D-10:** Reuse an existing project certificate across normal service churn and reissue only when a new project base hostname appears or the served hostname shape changes beyond current wildcard coverage.

### Friendly Responses And Pause State
- **D-11:** Managed hostnames that resolve locally but have no active winning route must receive a friendly no-route response from the local HTTP/HTTPS listeners.
- **D-12:** Paused routing must be represented as explicit daemon runtime state outside `routing.Snapshot`; pausing must not clear route data or disable DNS resolution.
- **D-13:** When routing is paused, managed hostnames should continue resolving normally and receive a friendly paused response from the local listeners.

### the agent's Discretion
- Exact internal package and file layout for DNS, proxy, and certificate components.
- Exact wording and payload format of the friendly no-route and paused responses.
- Certificate cache file naming and reload mechanics, as long as they preserve the per-project reuse rules above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Contract
- `.planning/ROADMAP.md` §Phase 2: Local DNS, Proxy, and HTTPS Serving — phase goal, dependency, and success criteria.
- `.planning/REQUIREMENTS.md` §DNS Proxy And HTTPS — NET-01 through NET-08.
- `.planning/PROJECT.md` §Context, §Constraints, §Key Decisions — macOS-only scope, managed suffix expectations, label precedence, redirect default, and certificate timing.

### Prior Architecture Signals
- `.planning/phases/01-discovery-domains-and-conflict-policy/01-discovery-domains-and-conflict-policy-05-SUMMARY.md` — locks the single-snapshot read-model and shared admin projection pattern that Phase 2 should extend.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/daemon/reconciler.go` — current place where route snapshots are built and where effective upstream metadata can be attached before publish.
- `internal/routing/types.go` — existing contracts for `Route`, `Upstream`, `RouteProvenance`, `Warning`, `Conflict`, and `Snapshot`.
- `internal/discovery/metadata.go` — already parses label-driven port and scheme overrides plus invalid-label warnings.
- `internal/routing/overrides.go` — already codifies label-over-config precedence for overlapping route fields.
- `internal/discovery/ports.go` — existing published-port selection logic for upstream target resolution.
- `internal/admin/status.go` and `internal/admin/routes.go` — current read models that should stay aligned with runtime routing behavior.

### Established Patterns
- Reconciliation produces an immutable snapshot that other surfaces consume rather than recalculating state ad hoc.
- Malformed metadata is handled with explicit warnings instead of silent fallback.
- Config defaults live in `internal/config/config.go` and are loaded centrally through Cobra/Viper in `cmd/devproxy/root.go`.

### Integration Points
- The reconcile path must evolve from discovery-only routing into the place that also supplies effective upstream, cert, and listener-facing routing data.
- Phase 2 listeners should consume `routing.Snapshot` plus a new paused-routing state instead of reaching into Docker or discovery directly.
- Certificate lifecycle should derive from the active winning hostnames already present in the snapshot.
- Future Phase 3 status and doctor flows should be able to report DNS/proxy/cert health from the same daemon-owned state surfaces introduced here.

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches.

</specifics>

<deferred>
## Deferred Ideas

None — analysis stayed within phase scope.

</deferred>
