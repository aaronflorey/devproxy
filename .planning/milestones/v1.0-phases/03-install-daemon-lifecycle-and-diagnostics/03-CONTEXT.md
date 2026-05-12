# Phase 3: Install, Daemon Lifecycle, and Diagnostics - Context

**Gathered:** 2026-05-05 (assumptions mode)
**Status:** Ready for planning

<domain>
## Phase Boundary

Make devproxy installable, operable, and debuggable on macOS by adding install/uninstall lifecycle flows, foreground daemon startup validation, and operator diagnostics/inspection commands that read from the shared daemon control plane. Keep discovery/routing and DNS/proxy/TLS data-plane behavior from Phases 1-2 intact.

</domain>

<decisions>
## Implementation Decisions

### Control Plane and Operator Surfaces
- **D-01:** `status`, `routes`, `refresh`, `doctor`, and `logs` must consume one daemon-owned admin API surface as thin clients, not separate state recomputation paths.

### Foreground Daemon Startup Behavior
- **D-02:** `devproxy daemon` must fail fast with explicit startup errors when Docker reachability, certificate prerequisites, or required listener binds are unavailable.
- **D-03:** Startup validation and runtime health reporting should extend existing daemon/network and cert failure signaling rather than introducing silent fallback or background-only retries.

### Install and Uninstall Lifecycle Shape
- **D-04:** `install` and `uninstall` scope is lifecycle orchestration (config/state/log path setup, resolver wiring, certificate bootstrap/removal choices, launchd registration and control), while routing/proxy runtime internals remain separate.
- **D-05:** Uninstall must prompt for retention/removal of config, state, logs, and certificates and apply only the user-selected cleanup scope.

### launchd and Session Roles
- **D-06:** Default install must set up daemon auto-start as required baseline behavior for Phase 3.
- **D-07:** Menu bar auto-start must be installed only when `devproxy install --with-menubar` is provided, via separate lifecycle handling from the daemon service.
- **D-08:** For privileged listener ownership and reliability, use launchd role separation: core networking daemon in system daemon domain, user-session UI behavior in agent domain.

### Admin Socket and Resolver Diagnostics
- **D-09:** Admin API must use a locally scoped UNIX socket with explicit ownership/mode controls and stale-socket cleanup on daemon restart.
- **D-10:** Doctor DNS validation must check macOS system resolver state (for example via `scutil --dns`-aligned behavior), not only raw DNS client tools.

### the agent's Discretion
- Exact command UX wording, output formatting, and table layouts for `status`, `routes`, `doctor`, and `logs`.
- Exact package boundaries and internal file layout for installer/service/socket helpers.
- Exact launchd plist key set and service labels, as long as they preserve the daemon-vs-menubar lifecycle constraints above.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase Contract
- `.planning/ROADMAP.md` (Phase 3 goal, dependency, requirements, success criteria)
- `.planning/REQUIREMENTS.md` (OPS-01 through OPS-09)
- `.planning/PROJECT.md` (constraints and key decisions: macOS-only, fail-loud behavior, no Compose mutation)

### Prior Locked Context
- `.planning/phases/02-local-dns-proxy-and-https-serving/02-CONTEXT.md`
- `.planning/phases/01-discovery-domains-and-conflict-policy/01-discovery-domains-and-conflict-policy-05-SUMMARY.md`
- `.planning/phases/02-local-dns-proxy-and-https-serving/02-local-dns-proxy-and-https-serving-05-SUMMARY.md`

### Runtime and Admin Anchors
- `cmd/devproxy/root.go`
- `internal/daemon/reconciler.go`
- `internal/daemon/events.go`
- `internal/daemon/network.go`
- `internal/admin/status.go`
- `internal/admin/routes.go`
- `internal/admin/doctor.go`
- `internal/admin/logs.go`
- `internal/proxy/http.go`
- `internal/proxy/https.go`
- `internal/dns/server.go`
- `internal/certs/mkcert.go`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/admin/*` already defines projection builders for status, routes, doctor, and logs, suitable for shared operator surfaces.
- `internal/daemon/network.go` already models listener bind success/failure and runtime health fields that can drive startup gating and diagnostics.
- `internal/certs/mkcert.go` already emits explicit prerequisite and issuance failures that Phase 3 can surface directly.
- `internal/daemon/reconciler.go` and `internal/registry/snapshot.go` already publish immutable routing snapshots that operator commands should read, not recompute.

### Established Patterns
- Fail-loud error handling is preferred over silent fallback for invalid runtime prerequisites.
- Shared read models are preferred over duplicated per-surface computation.
- Runtime and projection surfaces are layered by package, with CLI wiring at `cmd/devproxy` and domain logic in `internal/*`.

### Integration Points
- New install/uninstall and service-management commands should plug into CLI command wiring and call lifecycle helpers without mutating route reconciliation logic.
- Admin API transport (UNIX socket lifecycle, perms, stale cleanup) becomes the control-plane boundary consumed by CLI and later UI surfaces.
- Doctor/status/log commands should project daemon-owned health and route state, including resolver and listener checks aligned to macOS behavior.

</code_context>

<specifics>
## Specific Ideas

- Keep diagnostics opinionated and explicit: prioritize actionable failure messages over broad health summaries.
- Preserve current phase boundaries: do not expand into Phase 4 menu bar/dashboard UX beyond install-time opt-in lifecycle plumbing.

</specifics>

<deferred>
## Deferred Ideas

- Rich dashboard-level diagnostics visualization and interactive controls remain Phase 4 work.
- Persisted multi-session log history remains a v2 requirement and is not part of Phase 3 scope.

### Reviewed Todos (not folded)
None.

</deferred>
