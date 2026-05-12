---
phase: 01-discovery-domains-and-conflict-policy
verified: 2026-05-10T01:53:00Z
status: passed
score: 7/7 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/7
  gaps_closed:
    - "Developer can start, stop, rename, or recreate containers and see route mappings update to match current eligible published-port containers."
    - "Developer can set route behavior through config and Docker labels, with label values taking precedence for overlapping fields and invalid label fields ignored with explicit warnings."
    - "Developer can observe deterministic winner/loser conflict outcomes and consistent conflict warnings across status, doctor, dashboard, and logs."
  gaps_remaining: []
  regressions: []
---

# Phase 1: Discovery, Domains, and Conflict Policy Verification Report

**Phase Goal:** Developers can trust devproxy to discover eligible containers, compute deterministic domains, and resolve domain conflicts predictably without Compose edits.
**Verified:** 2026-05-10T01:53:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | The codebase has one authoritative route model used by later discovery, conflict, and operator surfaces. | ✓ VERIFIED | `internal/routing/types.go:12-79` defines the shared candidate/route/conflict/snapshot contracts; `internal/daemon/reconciler.go:43-91` builds snapshots from that model; `internal/daemon/app.go:229-261` fans one snapshot into status, doctor, logs, and routes projections. |
| 2 | Developer can start devproxy with running Compose projects and immediately see discovered routes generated from Compose metadata, with fallback name parsing only when labels are missing. | ✓ VERIFIED | `internal/daemon/docker_runtime.go:26-57` scans running containers; `internal/discovery/metadata.go:29-47` prefers Compose labels and falls back to container-name parsing only when labels are absent; `internal/daemon/reconciler.go:47-80` turns those candidates into active routes. |
| 3 | Developer can start, stop, rename, or recreate containers and see route mappings update to match current eligible published-port containers. | ✓ VERIFIED | `cmd/devproxy/daemon.go:39-48` injects `daemon.DefaultDockerEvents`; `internal/daemon/app.go:144-146,279-355` starts a live watcher loop, reconnects on stream failure, rescans on supported events, and routes updates through `watcher.HandleEvent`; `internal/daemon/events.go:10-77` limits updates to supported lifecycle actions. `TestAppStartProcessesLiveDockerEvents` and `TestAppWatcherReconnectsWithFullResync` pass. |
| 4 | Developer can access services via default and root project domains, including Laravel Sail defaults like `laravel.test` and common companion subdomains. | ✓ VERIFIED | `internal/routing/domains.go:33-53` generates default and root hostnames; `internal/config/config.go:32-41` includes `laravel.test` in default root services; `internal/daemon/reconciler.go:68-80` publishes the generated hostnames into the active snapshot. |
| 5 | Developer can set route behavior through config and Docker labels, with label values taking precedence for overlapping fields and invalid label fields ignored with explicit warnings. | ✓ VERIFIED | `internal/discovery/metadata.go:49-106,129-130` parses label fields and emits per-field invalid-label warnings; `internal/routing/overrides.go:13-36` merges labels over config; `internal/routing/domains.go:34-37` now honors explicit `prefs.Root` before default root-service heuristics, including `false`. `TestDomainGeneration` and `TestReconcilerExplicitRootFalseSuppressesDefaultRootHostname` pass. |
| 6 | Developer can observe deterministic winner/loser conflict outcomes from one immutable snapshot. | ✓ VERIFIED | `internal/routing/conflicts.go:14-40` resolves conflicts by priority then stable container-name ordering; `internal/registry/snapshot.go:19-32` publishes immutable versioned snapshots; `internal/admin/routes.go:25-63` and `internal/admin/logs.go:28-40` consume snapshot conflicts/warnings without recomputing policy. |
| 7 | Developer can observe deterministic winner/loser conflict outcomes and consistent conflict warnings across status, doctor, dashboard, and logs. | ✓ VERIFIED | `internal/admin/status.go:8-77` exposes reusable conflict and warning detail; `cmd/devproxy/status.go:30-80` renders loser and warning detail from `client.Status()`; `cmd/devproxy/doctor.go:45-97` calls `client.Doctor()` and renders snapshot conflicts/warnings; `internal/dashboard/server.go:121-148,165-175` reads shared status/logs projections; `internal/admin/logs.go:31-40` emits warning/conflict events from the same snapshot. CLI and dashboard tests pass. |

**Score:** 7/7 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/daemon/app.go` | daemon startup wiring for refresh + live Docker watcher | ✓ VERIFIED | Substantive watcher loop exists (`watchDockerEvents`, `consumeDockerEvents`, `handleDockerEvent`, `resyncWatcher`) and is started from `Start()`. |
| `internal/daemon/events.go` | supported event filtering and watcher health transitions | ✓ VERIFIED | Defines supported Docker lifecycle actions, tracks disconnect/reconnect health, and delegates updates through the reconciler. |
| `internal/daemon/docker_runtime.go` | default Docker scan and event stream implementation | ✓ VERIFIED | Shells out to `docker ps`, `docker inspect`, and `docker events --format {{json .}}` with supported action filters. |
| `internal/routing/domains.go` | domain generation honoring explicit root preference | ✓ VERIFIED | Explicit `prefs.Root` overrides default root-service behavior before root hostname generation. |
| `internal/routing/overrides.go` | field-level config + label precedence merge | ✓ VERIFIED | Labels override config for all overlapping fields including `Root`. |
| `internal/admin/status.go` | reusable status projection with conflict/warning detail | ✓ VERIFIED | `StatusView` now carries `ConflictDetails` and `WarningDetails` copied from snapshot data. |
| `cmd/devproxy/status.go` | status CLI rendering shared conflict/warning detail | ✓ VERIFIED | Prints summary plus per-conflict loser names and per-warning messages from `StatusView`. |
| `cmd/devproxy/doctor.go` | doctor CLI rendering shared doctor projection | ✓ VERIFIED | Preserves runtime checks and additionally renders `/doctor` conflict/warning detail. |
| `internal/admin/logs.go` | log projection from shared snapshot | ✓ VERIFIED | Emits snapshot-derived warning/conflict log events used by logs/dashboard surfaces. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `cmd/devproxy/daemon.go` | `internal/daemon/app.go` | `AppConfig.DockerEvents = daemon.DefaultDockerEvents` | ✓ WIRED | Foreground daemon now injects the real Docker event source at startup. |
| `internal/daemon/app.go` | `internal/daemon/events.go` | `Start()` launches `watchDockerEvents()` | ✓ WIRED | `Start()` spins up the watcher after initial refresh when both `DockerEvents` and `DockerScan` are available. |
| `internal/daemon/events.go` | `internal/daemon/reconciler.go` | supported lifecycle events call `HandleEvent` / reconnect calls `RebuildSnapshot` | ✓ WIRED | Event handling delegates through `Watcher.HandleEvent()` and `Watcher.OnReconnect()` instead of mutating snapshots directly. |
| `internal/routing/overrides.go` | `internal/routing/domains.go` | merged `RoutePreferences.Root` drives root hostname decision | ✓ WIRED | `Reconciler` merges overrides, then passes `effectivePrefs` into `GenerateDomains()`, which now respects explicit `false`. |
| `internal/daemon/reconciler.go` | `internal/routing/domains.go` | `GenerateDomains()` on effective merged preferences | ✓ WIRED | Reconciler uses the single domain-generation path for config and label overrides. |
| `cmd/devproxy/status.go` | `internal/admin/status.go` | `client.Status()` → `ConflictDetails` / `WarningDetails` | ✓ WIRED | Status CLI renders the shared projection instead of counts-only output. |
| `cmd/devproxy/doctor.go` | `internal/admin/doctor.go` | `client.Doctor()` | ✓ WIRED | Doctor CLI now consumes `/doctor` and renders snapshot-backed detail. |
| `internal/dashboard/server.go` | shared admin projections | `client.Status()` + `client.Logs()` | ✓ WIRED | Dashboard consumes admin projections rather than recomputing conflict logic locally. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `internal/daemon/app.go` | `event`, `containers` | `DefaultDockerEvents()` stream + `DockerScan()` rescan | Yes | ✓ FLOWING |
| `internal/routing/domains.go` | `shouldRoot`, `domains` | merged `RoutePreferences` from config + labels via `Reconciler` | Yes | ✓ FLOWING |
| `internal/admin/status.go` | `ConflictDetails`, `WarningDetails` | `routing.Snapshot.Conflicts` and `.Warnings` | Yes | ✓ FLOWING |
| `cmd/devproxy/doctor.go` | `view.Conflicts`, `view.Warnings` | `/doctor` admin API response from `admin.BuildDoctor()` | Yes | ✓ FLOWING |
| `internal/admin/logs.go` | `result` | `routing.Snapshot.Warnings`, `.Conflicts`, `.Routes` | Yes | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Live Docker watcher updates snapshot and reconnect logic | `go test ./internal/daemon/... -run 'Test(App|Watcher|Reconciler)' -count=1` | `ok github.com/mochaka/devproxy/internal/daemon` | ✓ PASS |
| Explicit `root=false` precedence | `go test ./internal/routing/... -run 'TestDomainGeneration' -count=1` | `ok github.com/mochaka/devproxy/internal/routing` | ✓ PASS |
| Status and doctor render shared conflict/warning detail | `go test ./cmd/devproxy/... ./internal/admin/... -run 'Test(Status|Doctor)' -count=1` | `ok github.com/mochaka/devproxy/cmd/devproxy` / `ok github.com/mochaka/devproxy/internal/admin` | ✓ PASS |
| Phase implementation does not break repository tests | `go test ./...` | full suite passed | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| `DISC-01` | `01-02`, `01-05` | Discover running containers with published TCP ports on startup and refresh | ✓ SATISFIED | `internal/daemon/docker_runtime.go:26-57` plus `internal/daemon/reconciler.go:43-91`. |
| `DISC-02` | `01-05`, `01-06` | Keep route state current across lifecycle events | ✓ SATISFIED | `cmd/devproxy/daemon.go:39-48`; `internal/daemon/app.go:279-367`; daemon event tests pass. |
| `DISC-03` | `01-01`, `01-02` | Prefer Compose labels, fallback to name parsing only when labels unavailable | ✓ SATISFIED | `internal/discovery/metadata.go:29-47`. |
| `DISC-04` | `01-02` | Route only eligible running containers with usable published TCP ports and no disable/ignore block | ✓ SATISFIED | `internal/discovery/eligibility.go` and `internal/discovery/ports.go`, exercised through reconciler tests and full suite. |
| `DISC-05` | `01-04` | Deterministic conflict resolution by priority then stable tie-break | ✓ SATISFIED | `internal/routing/conflicts.go:14-40`. |
| `DISC-06` | `01-04`, `01-05`, `01-08` | Conflict warnings and losers visible consistently in logs, doctor, status, dashboard | ✓ SATISFIED | Shared snapshot projections feed `status`, `doctor`, dashboard, and logs. |
| `DOMN-01` | `01-03` | Default domain `{service}.{project}.{suffix}` | ✓ SATISFIED | `internal/routing/domains.go:33`. |
| `DOMN-02` | `01-03` | Configured root service gets `{project}.{suffix}` | ✓ SATISFIED | `internal/routing/domains.go:38-40`. |
| `DOMN-03` | `01-03` | Laravel Sail root + companion defaults | ✓ SATISFIED | `internal/config/config.go:32-41` plus standard default-domain generation path. |
| `DOMN-04` | `01-02`, `01-03`, `01-07` | Docker labels override route behavior fields including root mapping | ✓ SATISFIED | `internal/discovery/metadata.go:53-104`; `internal/routing/overrides.go:13-36`; `internal/routing/domains.go:34-37`. |
| `DOMN-05` | `01-01`, `01-03`, `01-07` | Config overrides exist and labels take precedence on overlap | ✓ SATISFIED | `internal/daemon/reconciler.go:51-69`; `TestReconcilerLabelOverridesConfigForPortAndScheme`; `TestReconcilerExplicitRootFalseSuppressesDefaultRootHostname`. |
| `DOMN-06` | `01-03` | Reject public suffixes; allow unmanaged local suffixes with warnings | ✓ SATISFIED | `internal/routing/domains.go:22-30,42-49`. |
| `DOMN-07` | `01-02`, `01-03` | Invalid label values ignored field-by-field with warnings | ✓ SATISFIED | `internal/discovery/metadata.go:53-104,129-130`. |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `internal/daemon/docker_runtime.go` | 85-128 | No parser-level unit test for malformed `docker events` JSON / abrupt CLI output | ⚠️ Warning | The stream reconnection path is tested with stub event streams, but CLI output parsing failures rely on indirect coverage only. |

### Human Verification Required

None.

### Gaps Summary

All three previously reported blocker gaps are closed in the current codebase:

1. **Live lifecycle updates are wired.** The daemon now subscribes to Docker events, rescans on supported lifecycle actions, and performs reconnect-driven full resyncs before reporting healthy.
2. **Override precedence is fixed.** Explicit `root=false` now suppresses default root-domain publication while preserving explicit `root=true` behavior.
3. **Conflict visibility is consistent across required surfaces.** `status`, `doctor`, dashboard, and logs all consume snapshot-backed projections for conflict and warning truth.

No remaining Phase 1 goal blockers were found in the current code.

---

_Verified: 2026-05-10T01:53:00Z_
_Verifier: the agent (gsd-verifier)_
