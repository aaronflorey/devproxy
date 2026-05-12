# Phase 4: Menu Bar and Dashboard UX - Research

**Researched:** 2026-05-06
**Status:** Ready for planning

## Planning Question

What do we need to know to plan a macOS menubar runtime and local dashboard that stay thin over the existing daemon-owned UNIX-socket admin API?

## Recommendations

### 1. Keep both UI surfaces as thin admin API clients

- Reuse the Phase 3 control-plane pattern: UI code calls `internal/adminapi.Client`; it must not read reconciler, network, or launchd internals directly.
- Extend the admin API with explicit control endpoints instead of UI-local shell scripts, per D-01 and D-02.
- Add any new menu/dashboard data to the admin projections first, then consume it from the UI layers.

### 2. Use `github.com/getlantern/systray` for the menu bar runtime

- It matches the stack recommendation already recorded in `.planning/STACK.md`.
- Important constraint: it requires **CGO** and expects a macOS app bundle for best UX (`LSUIElement=1`, icon in bundle resources).
- The runtime model is `systray.Run(onReady, onExit)` with menu items updated from goroutines, which fits polling the admin API and reacting to click channels.
- Because this project is macOS-only in v1, the CGO/app-bundle tradeoff is acceptable.

### 3. Implement the dashboard as a local HTTP server in a dedicated `devproxy dashboard` command

- Serve dashboard HTML on localhost using Go stdlib (`net/http`, `html/template`, `embed`) rather than adding a frontend framework.
- The dashboard command should read admin API projections and render server-side HTML; this keeps the UX local, debuggable, and dependency-light.
- Give it a stable localhost address so the menu bar can open it deterministically.
- Add a dedicated logs/errors page or section so the menu bar “Open Logs” action has a concrete target even though v1 logs are session-scoped API data, not persisted files.

### 4. Expose explicit UI control contracts in the admin API

Recommended additions:

- `POST /routing/pause`
- `POST /routing/resume`
- `GET /startup`
- `POST /startup`
- Route metadata fields that tell the UI what URL to open and why a fallback was chosen

Suggested `POST /startup` request shape:

```json
{
  "role": "menubar",
  "enabled": true
}
```

Suggested `GET /startup` response shape:

```json
{
  "roles": [
    {
      "role": "daemon",
      "domain": "system",
      "label": "com.devproxy.daemon",
      "installed": true,
      "running": true,
      "toggleable": false,
      "status_message": "Managed by system launchd"
    },
    {
      "role": "menubar",
      "domain": "gui/<uid>",
      "label": "com.devproxy.menubar",
      "installed": true,
      "running": true,
      "toggleable": true,
      "status_message": "Starts at login"
    }
  ]
}
```

This preserves the split-role decision from Phase 3 and satisfies D-03.

### 5. Make route opening deterministic in the daemon projection

- UI code should not recalculate URL choice.
- Extend `admin.RouteView` (or equivalent API shape) with fields like:
  - `OpenURL`
  - `PreferredScheme`
  - `FallbackReason`
  - `HTTPSReady`
- The daemon/admin projection should choose `https://` only when the route is HTTPS-capable and runtime certificate/listener readiness is healthy; otherwise choose `http://` and provide the fallback reason, per D-04.

### 6. Add a bounded in-memory session issue buffer for dashboard error visibility

- Current `/logs` output is derived from route snapshot warnings/conflicts and active routes, not daemon-session failures.
- UI-04 requires recent current-session errors, so Phase 4 should add a small bounded buffer (for example, last 25 events) for:
  - refresh failures
  - start-at-login toggle failures
  - dashboard server startup failures
  - admin connectivity failures surfaced by UI actions
  - runtime/listener errors already known to the daemon
- Expose this through the admin API so both dashboard and menubar can render the same truth.

## Constraints and Pitfalls

### Do not hand-roll a second control plane

- Reuse `internal/adminapi/server.go` and `internal/adminapi/client.go`.
- Avoid direct `launchctl` calls from menu bar or dashboard code; control actions should be daemon/admin mediated.

### Be explicit about role ownership

- The system daemon remains the privileged routing process.
- The menu bar runtime is a user LaunchAgent.
- The UI must not imply that one toggle controls both roles simultaneously.

### Prefer clear failure states over silent fallback

- If dashboard bind/open fails, show a visible error state and add a session issue entry.
- If start-at-login toggle fails, return structured failure text to the caller and preserve current status.
- If HTTPS is unhealthy for a route, open HTTP intentionally and surface the reason.

### Keep the dashboard session-scoped

- No persisted history.
- No preference storage.
- No route mutation beyond already-approved control actions.

## Architectural Responsibility Map

| Layer | Owns | Must Not Own |
|-------|------|--------------|
| `internal/daemon` | session issues buffer, admin API state provider, mutation hooks | menu rendering, HTML templates |
| `internal/adminapi` | JSON contracts and HTTP handlers over unix socket | launchd business logic in the client |
| `internal/admin` | projection builders for status/routes/logs/startup views | socket transport or systray logic |
| `internal/dashboard` | localhost HTTP handlers, templates, CSS, browser-open helper | direct reconciler/network/launchd access |
| `internal/menubar` | systray menu composition, polling, action dispatch to admin API | route computation, direct launchd shelling |

## Recommended Build Order

1. Extend daemon/admin contracts for startup controls, route-open metadata, and session issues.
2. Build the localhost dashboard server and its logs/errors view.
3. Build the systray runtime against the stable contracts and dashboard URL.
4. Harden fallback/error states and add end-to-end UI integration coverage.

## Sources

- `.planning/STACK.md` — recommended Phase 4 stack and menubar notes.
- `.planning/research/ARCHITECTURE.md` — thin-client-over-admin-API architecture.
- `https://raw.githubusercontent.com/getlantern/systray/master/README.md` — systray API, CGO requirement, macOS app-bundle notes.
- `https://raw.githubusercontent.com/FiloSottile/mkcert/master/README.md` — confirms cert trust model that informs HTTPS-ready fallback messaging.
