# devproxy

## What This Is

DevProxy is a macOS-native developer tool for people running Docker Compose projects locally. It watches running containers, assigns local vanity domains like `acme-crm.test` and `api.acme-crm.test`, serves local DNS, and proxies HTTP and HTTPS traffic to the right published localhost port without requiring Compose file changes.

## Core Value

A developer can run `docker compose up` and immediately use predictable local domains for each routable service without editing Compose files, `/etc/hosts`, or local proxy configs.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Automatic route discovery from running Docker Compose containers with safe defaults and clear conflict handling
- [ ] Local DNS plus HTTP/HTTPS proxying so discovered services are reachable through development domains
- [ ] macOS install, daemon lifecycle, debugging, and status surfaces that make the system reliable to operate
- [ ] Laravel Sail-friendly behavior and optional Docker label/config overrides for non-default routing cases
- [ ] Menu bar visibility into active mappings and daemon health

### Out of Scope

- Proxying containers without published ports — v1 only routes services that already expose usable published TCP ports
- Mutating `docker-compose.yml` — the product must work without rewriting project config
- Replacing production ingress tools like Traefik or Caddy — this is local development infrastructure only
- Linux and Windows support — v1 is intentionally macOS-only
- Arbitrary TCP proxying — v1 only handles HTTP and HTTPS routing
- Public internet exposure — routes must remain local-development only

## Context

- The primary users are macOS developers using Docker Desktop, especially teams juggling multiple Compose projects locally.
- The product shape is a Go CLI plus a background daemon, with an optional macOS menu bar companion reading daemon state over a local socket.
- Route discovery relies first on Docker Compose labels, then falls back to container-name parsing when labels are unavailable.
- Routing only applies to running containers with published TCP ports, with deterministic conflict resolution and explicit warnings when multiple containers claim the same domain.
- Laravel Sail is a first-class target: `laravel.test` should map to the project root domain, and common companion services like Mailpit should get sensible subdomains automatically.
- Operational clarity matters as much as routing: warnings, conflicts, and health problems need to appear consistently in `doctor`, `status`, the dashboard, and logs.

## Constraints

- **Platform**: macOS only for v1 — installation, resolver configuration, LaunchAgents, and menu bar UX are all macOS-specific.
- **Runtime**: Use Docker Desktop and Docker Engine events — the daemon must react to container lifecycle changes and inspect running containers on startup.
- **Networking**: DNS resolves managed suffixes to `127.0.0.1`, and proxy listeners run locally on ports `80` and `443` — this keeps traffic local and predictable.
- **TLS**: `mkcert` is required for local certificate generation — install and startup should fail clearly when HTTPS prerequisites are missing.
- **Safety**: Prefer explicit failures and warnings over silent rewrites or guessed fallback behavior — debugging and trust are core product requirements.
- **Compatibility**: Do not require Compose-file edits or a sidecar proxy container for common cases — normal `docker compose up` must remain the default workflow.

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Product name and CLI command are `devproxy` | Keeps the tool discoverable and consistent across install, daemon, and operator flows | — Pending |
| Default development suffix is `.test` | `.test` is a standard local-development suffix and supports wildcard resolver behavior cleanly | — Pending |
| Docker labels override config for the same route fields | Container-local intent should win when both sources define routing behavior | — Pending |
| HTTP-to-HTTPS redirect is disabled by default | Avoid changing existing local workflows unless the user opts into redirect behavior | — Pending |
| Certificates are generated during route discovery | Routes should be ready before first request instead of failing lazily at open time | — Pending |
| Menu bar ships as a subcommand of the main binary | Keeps packaging and install flow simpler than maintaining a separate app bundle | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? -> Move to Out of Scope with reason
2. Requirements validated? -> Move to Validated with phase reference
3. New requirements emerged? -> Add to Active
4. Decisions to log? -> Add to Key Decisions
5. "What This Is" still accurate? -> Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check - still the right priority?
3. Audit Out of Scope - reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-05-05 after initialization*
