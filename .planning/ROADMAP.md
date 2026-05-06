# Roadmap: devproxy

## Overview

This roadmap delivers devproxy in four coarse phases that follow dependency order: deterministic discovery and route intent first, then local DNS/proxy/TLS data plane, then install and daemon operations, and finally menu bar/dashboard UX on top of the stable admin surface.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Discovery, Domains, and Conflict Policy** - Compute reliable route intent from running Compose containers and overrides.
- [x] **Phase 2: Local DNS, Proxy, and HTTPS Serving** - Make mapped domains resolve and serve HTTP/HTTPS traffic locally. (completed 2026-05-05)
- [ ] **Phase 3: Install, Daemon Lifecycle, and Diagnostics** - Make devproxy installable, operable, and debuggable on macOS.
- [ ] **Phase 4: Menu Bar and Dashboard UX** - Expose daemon health and route controls through macOS UI surfaces.

## Phase Details

### Phase 1: Discovery, Domains, and Conflict Policy
**Goal**: Developers can trust devproxy to discover eligible containers, compute deterministic domains, and resolve domain conflicts predictably without Compose edits.
**Depends on**: Nothing (first phase)
**Requirements**: DISC-01, DISC-02, DISC-03, DISC-04, DISC-05, DISC-06, DOMN-01, DOMN-02, DOMN-03, DOMN-04, DOMN-05, DOMN-06, DOMN-07
**Success Criteria** (what must be TRUE):
  1. Developer can start devproxy with running Compose projects and immediately see discovered routes generated from Compose metadata, with fallback name parsing only when labels are missing.
  2. Developer can start, stop, rename, or recreate containers and see route mappings update to match current eligible published-port containers.
  3. Developer can access services via default and root project domains, including Laravel Sail defaults like `laravel.test` and common companion subdomains.
  4. Developer can set route behavior through config and Docker labels, with label values taking precedence for overlapping fields and invalid label fields ignored with explicit warnings.
  5. Developer can observe deterministic winner/loser conflict outcomes and consistent conflict warnings across status, doctor, dashboard, and logs.
**Plans**: 5 plans

Plans:
**Wave 1**
- [ ] 01-01-PLAN.md — Bootstrap the Go module, CLI root, and shared config/routing contracts.

**Wave 2** *(blocked on Wave 1 completion)*
- [ ] 01-02-PLAN.md — Build discovery normalization for Docker metadata, eligibility, and port selection.
- [ ] 01-03-PLAN.md — Build domain generation and override precedence for default, root, Sail, and explicit domains.

**Wave 3** *(blocked on Wave 2 completion)*
- [ ] 01-04-PLAN.md — Resolve conflicts deterministically and publish immutable route snapshots.

**Wave 4** *(blocked on Wave 3 completion)*
- [ ] 01-05-PLAN.md — Wire reconciliation, Docker events, and shared status/routes/doctor/log read models.

### Phase 2: Local DNS, Proxy, and HTTPS Serving
**Goal**: Developers can resolve managed local domains and reliably reach active services over HTTP/HTTPS through devproxy.
**Depends on**: Phase 1
**Requirements**: NET-01, NET-02, NET-03, NET-04, NET-05, NET-06, NET-07, NET-08
**Success Criteria** (what must be TRUE):
  1. Developer can resolve hostnames under the managed suffix to `127.0.0.1` using the installed wildcard resolver.
  2. Developer can send HTTP or HTTPS requests to an active mapped hostname and receive upstream responses from the selected localhost published port, including WebSocket traffic.
  3. Developer can use trusted HTTPS certificates generated via mkcert, with certificates regenerated when served hostnames change and reused when unchanged.
  4. Developer can run with HTTP on port 80 and HTTPS on port 443, with redirect-to-HTTPS remaining off by default unless configured globally or per route.
  5. Developer receives clear friendly responses when no route exists for a managed hostname or when routing is paused.
**Plans**: 5 plans

Plans:
**Wave 1**
- [ ] 02-01-PLAN.md — Extend routing, reconciler, and config contracts for serving state, pause state, and redirect defaults.

**Wave 2** *(blocked on Wave 1 completion)*
- [ ] 02-02-PLAN.md — Build the managed-suffix DNS responder and hostname classification helpers.
- [ ] 02-03-PLAN.md — Build the HTTP proxy path with friendly no-route and paused responses plus WebSocket-safe proxying.

**Wave 3** *(blocked on Waves 1-2 completion as declared by plan dependencies)*
- [ ] 02-04-PLAN.md — Add certificate inventory reuse rules and mkcert-backed issuance.

**Wave 4** *(blocked on Waves 2-3 completion as declared by plan dependencies)*
- [ ] 02-05-PLAN.md — Wire HTTPS listeners and shared network runtime health projections.

### Phase 3: Install, Daemon Lifecycle, and Diagnostics
**Goal**: Developers can install, run, inspect, troubleshoot, and uninstall devproxy reliably on macOS.
**Depends on**: Phase 2
**Requirements**: OPS-01, OPS-02, OPS-03, OPS-04, OPS-05, OPS-06, OPS-07, OPS-08, OPS-09
**Success Criteria** (what must be TRUE):
  1. Developer can run `devproxy install` and have required config/state paths, resolver, certificates, daemon LaunchAgent, and required services configured and started.
  2. Developer can choose whether menu bar auto-start is installed, with menu bar LaunchAgent installed only when `--with-menubar` is used.
  3. Developer can run `devproxy daemon` in foreground and receive explicit startup failures when Docker, certificate prerequisites, or listener ports are unavailable.
  4. Developer can use `status`, `routes`, `refresh`, `doctor`, and `logs` to inspect live daemon health, route state, diagnostics, and current-session logs from the same local admin API source.
  5. Developer can run uninstall and choose to retain or remove config, state, logs, and certificates.
**Plans**: 7 plans

Plans:
**Wave 1**
- [x] 03-01-PLAN.md — Build the daemon-owned admin socket API and fail-fast foreground daemon startup path.

**Wave 2** *(blocked on Wave 1 completion)*
- [x] 03-02-PLAN.md — Add `status`, `routes`, `refresh`, and `logs` as thin clients of the daemon control plane.
- [x] 03-03-PLAN.md — Implement macOS install orchestration for paths, resolver wiring, launchd services, and optional menu bar auto-start.

**Wave 3** *(blocked on Waves 1-2 completion as declared by plan dependencies)*
- [x] 03-04-PLAN.md — Add doctor checks for live macOS/runtime health plus selective uninstall cleanup.

**Wave 4** *(gap closure after verification/UAT)*
- [x] 03-05-PLAN.md — Add explicit root-privilege lifecycle preflights and idempotent launchd teardown for install/uninstall.
- [x] 03-06-PLAN.md — Correct doctor managed-host probing and blocked-runtime diagnostics.

**Wave 5** *(gap closure after re-verification)*
- [x] 03-07-PLAN.md — Harden uninstall bootout missing-state handling so scoped cleanup survives the macOS `launchctl bootout ... exit status 5` path.

### Phase 4: Menu Bar and Dashboard UX
**Goal**: Developers can monitor devproxy and perform core control actions from the menu bar and local dashboard.
**Depends on**: Phase 3
**Requirements**: UI-01, UI-02, UI-03, UI-04
**Success Criteria** (what must be TRUE):
  1. Developer can view daemon health and active routes directly from the macOS menu bar app.
  2. Developer can trigger refresh, open dashboard/logs, run doctor, pause routing, and toggle start-at-login from menu bar controls.
  3. Developer can open a selected route from the menu bar over HTTPS when enabled for that route, otherwise HTTP.
  4. Developer can open a local dashboard that shows daemon health, active routes, recent conflicts, and recent daemon-session errors.
**Plans**: 4 plans
**UI hint**: yes

Plans:
**Wave 1**
- [ ] 04-01-PLAN.md -- Extend daemon/admin API contracts for menu controls, route-open metadata, and role-aware start-at-login state.

**Wave 2** *(blocked on Wave 1 completion)*
- [ ] 04-02-PLAN.md -- Add the localhost dashboard command and server-rendered UI backed only by admin API projections, including recent conflicts/errors and route links.

**Wave 3** *(blocked on Waves 1-2 completion as declared by plan dependencies)*
- [ ] 04-03-PLAN.md -- Add `devproxy menubar` runtime with systray actions for refresh, doctor, logs, pause/resume, route-open, dashboard launch, and role-aware start-at-login state.

**Wave 4** *(blocked on Waves 2-3 completion as declared by plan dependencies)*
- [ ] 04-04-PLAN.md -- Harden UI fallbacks, launch/open failure handling, and end-to-end dashboard/menubar integration coverage.

## Progress

**Execution Order:**
Phases execute in numeric order: 2 → 2.1 → 2.2 → 3 → 3.1 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Discovery, Domains, and Conflict Policy | 0/5 | Planned | - |
| 2. Local DNS, Proxy, and HTTPS Serving | 5/5 | Complete   | 2026-05-05 |
| 3. Install, Daemon Lifecycle, and Diagnostics | 0/4 | Not started | - |
| 4. Menu Bar and Dashboard UX | 3/4 | In Progress|  |
