# Phase 1: Discovery, Domains, and Conflict Policy - Context

**Gathered:** 2026-05-09T18:15:23Z
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers the authoritative route-intent layer for devproxy: discover eligible Docker Compose containers, compute deterministic hostnames and override behavior, resolve domain conflicts predictably, and publish one shared snapshot/read model for operators and later networking phases. It stops at route computation and visibility surfaces, not DNS or proxy serving.

</domain>

<decisions>
## Implementation Decisions

### Discovery And Eligibility
- Only running containers with published TCP ports are routable; UDP-only and exposed-only containers stay out of routing.
- Upstream port selection uses explicit override port first, then falls back to the first published TCP port with a warning when fallback is used.
- Normal ineligible containers are excluded quietly; warnings are reserved for malformed labels, rejected explicit domains, and ambiguous fallbacks.
- Route state updates should use incremental Docker lifecycle events, with a full resync after Docker disconnect/reconnect.

### Domains And Explicit Overrides
- The default generated hostname is `{service}.{project}.{suffix}`.
- The project root domain exists only for configured root services or explicit root-mapping labels; devproxy should not guess an implicit root winner.
- Laravel Sail gets first-class defaults: `laravel.test` maps to the project root and known companion services receive their conventional subdomains.
- Explicit custom domains reject public suffixes, allow unmanaged local suffixes with explicit warnings, and treat managed-suffix domains as first-class.

### Conflicts And Shared Read Models
- Domain conflicts resolve by highest priority first, then stable tie-break on container name when priorities match.
- The authoritative snapshot preserves winner, loser set, and conflict reason so all consumers read one shared conflict record.
- Warnings and conflicts are generated once during reconciliation and published through shared read models for `status`, `doctor`, `logs`, and the dashboard.
- Startup should build a full route snapshot first, then Docker event-driven updates should reuse the same reconciler path.

### OpenCode's Discretion
OpenCode has discretion over internal package boundaries and helper structure as long as the route snapshot stays authoritative, immutable for readers, and consistent across daemon and operator surfaces.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/discovery/metadata.go`, `eligibility.go`, and `ports.go` already centralize Compose metadata extraction, eligibility checks, label parsing, and published-port selection.
- `internal/routing/domains.go`, `overrides.go`, `conflicts.go`, and `types.go` provide pure routing functions and typed route/conflict/warning models.
- `internal/registry/snapshot.go` provides an immutable snapshot builder for active routes, conflicts, and warnings.
- `internal/daemon/reconciler.go` and `events.go` are the integration seam for snapshot rebuilds and Docker lifecycle updates.

### Established Patterns
- Discovery and routing logic are kept in small pure functions that return data plus warnings instead of mutating shared state directly.
- Reconciliation builds the next route set off to the side, then publishes one snapshot for all downstream readers.
- Operator-facing surfaces consume shared projections from daemon/admin packages instead of recomputing routing behavior in each command or UI.

### Integration Points
- `internal/daemon/reconciler.go` is the main join point between Docker container state, routing rules, and snapshot publication.
- `internal/registry/snapshot.go` feeds admin/status/doctor/log projections used by CLI and later UI surfaces.
- Docker lifecycle handling belongs on the watcher/reconciler path so startup refresh and live updates stay on one code path.

</code_context>

<specifics>
## Specific Ideas

No specific visual or interaction requirements beyond the roadmap and requirements set. Favor deterministic, operator-auditable behavior over clever fallback logic.

</specifics>

<deferred>
## Deferred Ideas

None - discussion stayed within phase scope.

</deferred>
