# Requirements: devproxy

**Defined:** 2026-05-05
**Core Value:** A developer can run `docker compose up` and immediately use predictable local domains for each routable service without editing Compose files, `/etc/hosts`, or local proxy configs.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Discovery And Routing

- [x] **DISC-01**: Developer can have devproxy discover running containers with published TCP ports on startup and refresh without changing Compose files
- [x] **DISC-02**: Developer can have devproxy keep route state current as containers start, stop, die, destroy, rename, and update
- [x] **DISC-03**: Developer can have devproxy infer Compose project and service names from Docker Compose labels, with container-name parsing only as a fallback when labels are unavailable
- [x] **DISC-04**: Developer can trust devproxy to route only eligible containers that are running, have a usable published TCP port, and are not disabled or ignored
- [x] **DISC-05**: Developer can rely on deterministic conflict resolution when multiple containers claim the same domain, using priority first and a stable tie-break when priorities match
- [x] **DISC-06**: Developer can see conflict warnings and losing candidates consistently in logs, `doctor`, `status`, and the dashboard

### Domains And Overrides

- [x] **DOMN-01**: Developer can access a discovered service at the default domain `{service}.{project}.{suffix}`
- [x] **DOMN-02**: Developer can access a configured root service at the project root domain `{project}.{suffix}`
- [x] **DOMN-03**: Laravel Sail users can have `laravel.test` map to the project root domain and common companion services receive their standard subdomains automatically
- [x] **DOMN-04**: Developer can override route behavior with Docker labels for enable/disable, domain, domains, root mapping, port, scheme, and priority
- [x] **DOMN-05**: Developer can define route overrides in config, while Docker labels take precedence for the same route fields in v1
- [x] **DOMN-06**: Developer can use explicit local-only custom domains, with public internet suffixes rejected and unmanaged suffixes allowed only with clear DNS warnings
- [x] **DOMN-07**: Developer can trust invalid label values to be ignored field-by-field with clear warnings instead of silent acceptance or daemon failure

### DNS Proxy And HTTPS

- [ ] **NET-01**: Developer can install a managed wildcard resolver so hostnames under the configured suffix resolve to `127.0.0.1`
- [x] **NET-02**: Developer can send HTTP requests for an active route and have devproxy proxy them to the selected localhost published port using the correct upstream scheme
- [ ] **NET-03**: Developer can use HTTPS for active routes with locally trusted certificates generated through `mkcert`
- [x] **NET-04**: Developer can have devproxy regenerate certificates when a project's served hostnames change and reuse valid certs when they do not
- [x] **NET-05**: Developer can use both HTTP on port `80` and HTTPS on port `443`, with HTTP-to-HTTPS redirect disabled by default and configurable globally or per route
- [ ] **NET-06**: Developer can use WebSocket traffic through devproxy for routed services
- [ ] **NET-07**: Developer can get a friendly no-route response when a hostname under a managed suffix has no active mapping
- [x] **NET-08**: Developer can pause routing and receive a friendly paused response for managed hostnames while DNS resolution continues normally

### Install Lifecycle And Diagnostics

- [ ] **OPS-01**: Developer can run `devproxy install` to create config and state directories, install the resolver, install certificates, install the daemon LaunchAgent, and start required services
- [ ] **OPS-02**: Developer can install the menu bar LaunchAgent only when `devproxy install --with-menubar` is used
- [ ] **OPS-03**: Developer can run `devproxy daemon` in the foreground and get a clear startup failure when Docker, certificate prerequisites, or listener ports are unavailable
- [ ] **OPS-04**: Developer can inspect daemon health, install state, and active route counts with `devproxy status`
- [ ] **OPS-05**: Developer can inspect active mappings with `devproxy routes` and trigger a full container rescan with `devproxy refresh`
- [ ] **OPS-06**: Developer can run `devproxy doctor` and get checks for Docker reachability, DNS, resolver configuration, port binding, mkcert, local CA, LaunchAgent state, proxy reachability, and example domain resolution
- [ ] **OPS-07**: Developer can stream current-session daemon logs with `devproxy logs`
- [ ] **OPS-08**: Developer can uninstall devproxy and be prompted whether to keep or remove config, state, logs, and certificates
- [ ] **OPS-09**: Developer can rely on a local admin API socket as the shared source for CLI, dashboard, and menu bar state

### Menu Bar And Dashboard

- [ ] **UI-01**: Developer can view daemon status and active routes from the macOS menu bar app
- [ ] **UI-02**: Developer can refresh routes, open the dashboard, open logs, run doctor, pause routing, and toggle start-at-login from the menu bar
- [ ] **UI-03**: Developer can open a route from the menu bar using HTTPS when HTTPS is enabled for that route, otherwise HTTP
- [ ] **UI-04**: Developer can open a local dashboard that shows daemon health, active routes, recent conflicts, and recent errors from the current daemon session

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Management

- **ADVN-01**: Developer can have devproxy automatically manage DNS for additional local-only suffixes beyond the primary configured suffix
- **ADVN-02**: Developer can inspect persisted daemon log history across restarts
- **ADVN-03**: Developer can manage richer dashboard preferences and route controls beyond the lightweight menu bar workflow

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Routing containers without published ports | Requires sidecar or network-namespace behavior that is outside the v1 safety and complexity budget |
| Mutating `docker-compose.yml` | Violates the zero-compose-change product promise |
| Replacing Traefik, Caddy, or production ingress stacks | The product is local-only developer infrastructure, not production routing |
| Linux or Windows support | v1 is intentionally macOS-only to keep resolver and service-management behavior reliable |
| Arbitrary TCP or UDP proxying | v1 is focused on HTTP and HTTPS vanity-domain routing |
| Public internet exposure or tunnels | Security and scope are intentionally limited to loopback-local development |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DISC-01 | Phase 1 | Complete |
| DISC-02 | Phase 1 | Complete |
| DISC-03 | Phase 1 | Complete |
| DISC-04 | Phase 1 | Complete |
| DISC-05 | Phase 1 | Complete |
| DISC-06 | Phase 1 | Complete |
| DOMN-01 | Phase 1 | Complete |
| DOMN-02 | Phase 1 | Complete |
| DOMN-03 | Phase 1 | Complete |
| DOMN-04 | Phase 1 | Complete |
| DOMN-05 | Phase 1 | Complete |
| DOMN-06 | Phase 1 | Complete |
| DOMN-07 | Phase 1 | Complete |
| NET-01 | Phase 2 | Pending |
| NET-02 | Phase 2 | Complete |
| NET-03 | Phase 2 | Pending |
| NET-04 | Phase 2 | Complete |
| NET-05 | Phase 2 | Complete |
| NET-06 | Phase 2 | Pending |
| NET-07 | Phase 2 | Pending |
| NET-08 | Phase 2 | Complete |
| OPS-01 | Phase 3 | Pending |
| OPS-02 | Phase 3 | Pending |
| OPS-03 | Phase 3 | Pending |
| OPS-04 | Phase 3 | Pending |
| OPS-05 | Phase 3 | Pending |
| OPS-06 | Phase 3 | Pending |
| OPS-07 | Phase 3 | Pending |
| OPS-08 | Phase 3 | Pending |
| OPS-09 | Phase 3 | Pending |
| UI-01 | Phase 4 | Pending |
| UI-02 | Phase 4 | Pending |
| UI-03 | Phase 4 | Pending |
| UI-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 34 total
- Mapped to phases: 34
- Unmapped: 0 ✅

---
*Requirements defined: 2026-05-05*
*Last updated: 2026-05-05 after roadmap mapping*
