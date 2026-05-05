# Feature Landscape

**Domain:** macOS local developer proxy for Docker Compose vanity domains
**Researched:** 2026-05-05

## Table Stakes

Features users expect. Missing = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Zero-config service discovery from running Compose containers | Competing workflows (Traefik Docker provider, OrbStack domains, nginx-proxy) all discover containers automatically from Docker metadata/events | Med | Must work without editing `docker-compose.yml` for common cases; parse Compose labels first, then safe fallback heuristics |
| Deterministic domain naming per project/service | Developers expect predictable hostnames they can memorize and share in docs (`service.project.tld`) | Med | Should always generate the same domain for same project/service and clearly show mappings |
| Local DNS for wildcard dev suffix | Tools like Valet/DDEV normalize wildcard local domains as baseline dev UX | Med | `*.test -> 127.0.0.1` (or configured suffix), resolver setup handled by installer |
| HTTP reverse proxy by Host header | Core value is “domain instead of localhost:port”; proxy routing is mandatory | Med | Route to published localhost TCP port only in v1; friendly no-route page when unmatched |
| HTTPS with trusted local certificates | Devs now expect local HTTPS (cookies, OAuth, secure-context APIs) to work without browser warnings | High | mkcert install + per-project cert issuance + renewal when hostnames change |
| Explicit conflict handling (same domain claimed by multiple containers) | Auto-discovery tools frequently collide in multi-project setups; silent overrides destroy trust | Med | Priority + deterministic tie-break + surfaced warning in CLI/status/logs |
| Simple override mechanism via labels | Advanced cases always need override knobs (domain, port, scheme, enable/disable) | Med | Keep label surface minimal and composable; invalid fields ignored with warnings, not crashes |
| Fast operational visibility (`status`, `routes`, `doctor`, logs) | Local infra tools fail often at install/network/cert layers; users need immediate diagnosis | Med | Health checks for Docker socket, DNS, ports, cert trust, launch state |
| macOS-native install/uninstall lifecycle | v1 is macOS-only, so users expect one command to install resolver/agent and one to cleanly remove | Med | LaunchAgent, resolver file, cert prerequisites, explicit uninstall prompts |
| Local-only safety defaults | Local tools are expected to avoid accidental internet exposure | Low | Bind to loopback by default; reject clearly public suffixes for explicit domains |

## Differentiators

Features that set product apart. Not expected, but valued.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Compose-change-free onboarding (no proxy sidecar, no label requirement for common case) | Strongest UX delta: `docker compose up` and domains just appear | High | This is the key product bet; competitors usually require labels, env vars, or dedicated proxy service |
| Laravel Sail first-class defaults (`laravel.test` -> `{project}.test`, Mailpit-style companion mapping) | Wins a concrete high-frequency workflow and reduces per-project glue config | Med | Keep framework-specific behavior narrow and explicit, not generic magic |
| Menu bar route/health control surface | Gives “always-on” visibility without terminal context switching | Med | Route list, quick-open URL, refresh, pause, doctor entrypoint |
| Pausable routing mode with explicit paused response | Safe temporary disable for debugging/network conflicts without tearing down DNS/install state | Low | Better than full stop/start churn during debugging |
| Domain conflict UX across all surfaces (CLI + dashboard + logs) | Most tools log conflicts but don’t make them obvious; strong trust/operability signal | Med | Include loser candidates and resolution reason |
| Support explicit custom local suffixes beyond managed DNS with warnings | Enables advanced setups without pretending DNS is managed automatically | Med | Allow route, but warn when suffix is unmanaged; still block obvious public internet domains |

## Anti-Features

Features to explicitly NOT build.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Compose file mutation/rewriting | Violates core product promise and creates scary config drift | Read Docker runtime metadata + optional labels/config overrides only |
| Proxying non-published container ports in v1 | Requires network namespace tricks/sidecars and adds major failure modes | Restrict v1 to published localhost ports; document clearly in doctor/status |
| Full ingress controller feature set (middlewares, auth chains, path routing DSL) | Becomes Traefik/Caddy clone and kills focus/time-to-value | Keep v1 host-based routing only; revisit after adoption evidence |
| Arbitrary TCP/UDP proxying | Expands blast radius and testing matrix far beyond vanity-domain use case | Keep protocol scope to HTTP/HTTPS |
| Public tunnel / internet exposure features | Security/compliance complexity is disproportionate for v1 local infra tool | Keep loopback-local by default; no remote exposure primitives |
| Cross-platform support (Linux/Windows) in v1 | OS-specific resolver/service-management multiplies implementation and support burden | Nail macOS reliability first; design abstractions for later ports |
| Heavy GUI app as primary interface | Slows iteration and duplicates CLI responsibilities | CLI/daemon first; menu bar as lightweight observer/controller |
| Auto-magic overrides of existing real/public domains | Creates dangerous DNS hijacking and confusing behavior | Reject known public suffixes; require explicit local-only domains |

## Feature Dependencies

```text
Docker event watcher + container inspection → Route eligibility engine
Route eligibility engine → Domain generation
Domain generation + port selection → Route registry
Route registry → DNS wildcard resolution
Route registry → HTTP reverse proxy
Route registry + hostname set → HTTPS cert generation
HTTPS cert generation → HTTPS listener
Daemon health model → status/routes/doctor/logs
Daemon local API socket → menu bar + dashboard
Conflict detection in registry → warnings in CLI/dashboard/logs
Install (resolver + LaunchAgent + mkcert) → reliable always-on operation
```

## MVP Recommendation

Prioritize:
1. Automatic Compose discovery + deterministic route generation (including conflict resolution)
2. Local DNS wildcard + HTTP routing for active routes
3. HTTPS via mkcert + operator visibility (`status`, `routes`, `doctor`)

Defer: Menu bar controls and pause-routing UX until core daemon reliability is proven. They are valuable differentiators, but weak core routing/DNS/cert reliability will negate their value.

## Sources

- Docker: `docker events` real-time lifecycle stream (official) — https://docs.docker.com/reference/cli/docker/system/events/ **(HIGH)**
- Docker Compose canonical labels (`com.docker.compose.project`, `com.docker.compose.service`) — https://docs.docker.com/reference/compose-file/services/ **(HIGH)**
- Traefik Docker provider (container discovery + label-driven routing) — https://doc.traefik.io/traefik/getting-started/docker/ and https://doc.traefik.io/traefik/reference/routing-configuration/other-providers/docker/ **(HIGH)**
- nginx-proxy auto reverse proxy via Docker socket + `VIRTUAL_HOST` env — https://github.com/nginx-proxy/nginx-proxy **(HIGH)**
- Caddy Docker Proxy label-based dynamic config + event-driven reload — https://github.com/lucaslorentz/caddy-docker-proxy **(HIGH)**
- OrbStack container/Compose domains + automatic HTTPS + custom domain labels — https://docs.orbstack.dev/docker/domains **(HIGH)**
- Laravel Valet `*.test` wildcard local-domain model + optional TLS secure command — https://laravel.com/docs/13.x/valet **(HIGH)**
- DDEV additional hostnames/wildcards, HTTPS via mkcert, conflict caveats — https://ddev.readthedocs.io/en/stable/users/extend/additional-hostnames and https://ddev.readthedocs.io/en/stable/users/install/configuring-browsers/ **(HIGH)**
