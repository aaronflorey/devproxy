# Phase 4: Menu Bar and Dashboard UX - Discussion Log (Assumptions Mode)

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions captured in CONTEXT.md — this log preserves the analysis.

**Date:** 2026-05-06
**Phase:** 04-Menu Bar and Dashboard UX
**Mode:** assumptions
**Areas analyzed:** Admin API boundary, Menu bar controls, Route opening behavior, Dashboard scope

## Assumptions Presented

### Admin API as sole UI boundary
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Keep menu bar + dashboard as thin clients over UNIX-socket admin API; avoid direct daemon/runtime coupling in UI code. | Confident | `.planning/phases/03-install-daemon-lifecycle-and-diagnostics/03-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `internal/adminapi/server.go`, `internal/adminapi/client.go`, `internal/adminapi/types.go`, `internal/daemon/app.go` |

### Control endpoints for UI actions
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Add explicit control operations for pause/resume and start-at-login semantics to satisfy UI-02. | Likely | `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `internal/adminapi/server.go`, `internal/adminapi/client.go`, `internal/daemon/reconciler.go`, `cmd/devproxy/install.go`, `internal/install/install.go`, `internal/install/launchd.go` |

### Route opening protocol policy
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Open selected route over HTTPS only when route is HTTPS-enabled; otherwise HTTP, based on daemon-exposed metadata with fallback on HTTPS-health failures. | Unclear | `.planning/REQUIREMENTS.md`, `internal/routing/types.go`, `internal/admin/routes.go`, `internal/admin/status.go` |

### Dashboard scope
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Keep dashboard session-scoped (health, routes, conflicts, recent current-session errors) with no persisted history/preferences in Phase 4. | Confident | `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/phases/03-install-daemon-lifecycle-and-diagnostics/03-CONTEXT.md`, `cmd/devproxy/logs.go`, `internal/admin/logs.go` |

## Corrections Made

No corrections — all assumptions confirmed.

## External Research

- Menu bar implementation and packaging: pure-Go systray still requires app-bundle/CGO/signing-notarization concerns; native macOS service-management model remains relevant for production UX.
  - Sources: Apple ServiceManagement docs, Apple notarization docs, getlantern/systray docs.
- Start-at-login UX for split daemon/agent roles: use role-explicit toggles, status-aware control flow, and clear user guidance for login items/background permissions.
  - Sources: Apple ServiceManagement docs, Apple launchd behavior docs.
