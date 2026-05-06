# Phase 4: Menu Bar and Dashboard UX - Context

**Gathered:** 2026-05-06 (assumptions mode)
**Status:** Ready for planning

<domain>
## Phase Boundary

Expose daemon health and route controls through macOS menu bar and local dashboard surfaces for UI-01, UI-02, UI-03, and UI-04; keep scope to UI/control-plane integration and avoid expanding into persisted history or broader preference management.
</domain>

<decisions>
## Implementation Decisions

### Control Plane Boundary
- **D-01:** Menu bar and dashboard remain thin clients over the daemon-owned UNIX-socket admin API; they must not read daemon internals directly or own routing logic.

### Menu Bar Controls
- **D-02:** Phase 4 must expose explicit control operations for routing pause/resume and start-at-login semantics rather than ad hoc shell-outs.
- **D-03:** Start-at-login behavior must respect split roles (system daemon vs user menu bar agent) and surface role-specific status/failure states clearly in UI actions.

### Route Opening Behavior
- **D-04:** Route opening protocol must derive from daemon-exposed route/runtime metadata: open `https://` when enabled for that route, otherwise `http://`; define deterministic fallback behavior when HTTPS listener/certificate readiness is unhealthy.

### Dashboard Scope
- **D-05:** Dashboard remains session-scoped in v1 and shows daemon health, active routes, recent conflicts, and recent current-session errors only; no persisted history/preferences in this phase.

### the agent's Discretion
- Keep action naming and endpoint shape aligned with existing admin API conventions and avoid introducing a second control surface.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

- .planning/ROADMAP.md
- .planning/REQUIREMENTS.md
- .planning/PROJECT.md
- .planning/STATE.md
- .planning/research/ARCHITECTURE.md
- internal/adminapi/server.go
- internal/adminapi/client.go
- internal/adminapi/types.go
- internal/daemon/app.go
- internal/daemon/reconciler.go
- internal/admin/status.go
- internal/admin/routes.go
- internal/routing/types.go
- internal/install/launchd.go
- internal/install/install.go
- cmd/devproxy/install.go
- cmd/devproxy/logs.go
- internal/admin/logs.go
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Existing admin control-plane transport and schema (`internal/adminapi/server.go`, `internal/adminapi/client.go`, `internal/adminapi/types.go`) already support menu bar/dashboard read projections and refresh action.
- Daemon snapshot projection path (`internal/daemon/app.go`) already unifies status/routes/doctor/logs from one source, which directly supports UI consistency.
- Existing launchd install primitives (`internal/install/launchd.go`, `internal/install/install.go`) already model daemon vs menubar roles and can be reused for start-at-login control semantics.

### Established Patterns
- Single-writer daemon state with thin clients over local API is established and should remain unchanged.
- Session-scoped logs and diagnostics are already the current user-facing model; dashboard should align to this.
- Explicit failure/reporting behavior is preferred over silent fallback, matching project constraints and prior phases.

### Integration Points
- Add/extend admin API mutator endpoints for UI control actions (pause/resume and start-at-login semantics).
- Extend route/status projection payloads if needed to support deterministic open-route protocol decisions.
- Menu bar command path (expected `devproxy menubar` launch role) should consume admin API only.
- Dashboard should read the same projections used by CLI (`status`, `routes`, `doctor`, `logs`) to satisfy UI-04 without duplicate logic.
</code_context>

<specifics>
## Specific Ideas

- Prefer role-explicit UX wording for startup controls so users can tell whether they are toggling the core daemon service, the menu bar helper, or both.
- Keep open-route action deterministic and explain fallback when HTTPS is unavailable.
</specifics>

<deferred>
## Deferred Ideas

- Persisted log history across daemon restarts remains out of scope for this phase (tracked in v2 requirements).
- Rich dashboard preferences and broader route controls remain out of scope for this phase (tracked in v2 requirements).

### Reviewed Todos (not folded)
None.
</deferred>
