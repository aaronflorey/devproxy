# Phase 1 Research: Discovery, Domains, and Conflict Policy

**Phase:** 1
**Researched:** 2026-05-05
**Status:** Ready for planning

## Standard Stack

- **Go 1.26.x** for the main binary and daemon runtime.
- **`github.com/moby/moby/client`** for Docker list/inspect/events.
- **`github.com/spf13/cobra`** for CLI entrypoints.
- **`github.com/spf13/viper`** for config + env layering.
- **`log/slog`** for structured warnings and operator-visible diagnostics.

## Architecture Patterns

1. **Startup snapshot + live event stream**: full running-container inspection on boot, then incremental Docker event reconciliation.
2. **Single writer for route state**: only the route resolution engine mutates the authoritative registry snapshot.
3. **Copy-on-write registry updates**: compute the next route snapshot off to the side, then atomically swap it.
4. **Labels first, fallback parsing second**: trust canonical Compose labels when present; only parse container names when labels are absent.
5. **Field-level override precedence**: config provides defaults, Docker labels override matching fields, invalid label values are ignored with warnings instead of killing reconciliation.

## Phase-Specific Requirements To Preserve

- Discover only **running containers with published TCP ports**.
- Support **root-domain generation** only for configured root services or explicit root mapping.
- Preserve **Laravel Sail defaults**: `laravel.test` maps to `{project}.test`, and common companion services keep normal subdomains.
- Resolve domain conflicts by **priority first**, then **stable tie-break on container name**.
- Store conflict losers and warnings in a read model consumable by **status**, **doctor**, **logs**, and a future **dashboard**.
- Reject obviously public suffixes for explicit custom domains; allow unmanaged local suffixes only with explicit DNS warnings.

## Do Not Hand-Roll

- Do not parse raw Docker JSON with ad-hoc maps when typed inspect/list structures exist in the Moby client.
- Do not invent custom precedence rules outside the PRD/requirements order.
- Do not silently remap non-routable containers or invalid explicit domains.
- Do not couple later DNS/proxy serving logic into this phase; this phase should stop at authoritative route computation and visibility surfaces.

## Common Pitfalls

1. **Event-stream drift**: always re-sync from a fresh snapshot after disconnect/reconnect.
2. **Wrong port selection**: apply explicit precedence and warn when falling back to the first published TCP port.
3. **Conflict flapping**: tie-break must be deterministic and independent of Docker event ordering.
4. **Unmanaged explicit domains silently failing**: allow them, but record a warning explaining DNS is not managed for that suffix.
5. **Split-brain observability**: conflicts and warnings must come from one shared read model, not separate per-command recomputation.

## Architectural Responsibility Map

| Tier | Owns | Does Not Own |
|------|------|--------------|
| CLI (`cmd/devproxy`) | command wiring, config load, rendering route/health output | route decisions, Docker business logic |
| Discovery (`internal/discovery`) | Docker snapshot/events, Compose metadata extraction, candidate normalization | final conflict resolution across the full route set |
| Routing (`internal/routing`) | eligibility, port choice, domain generation, override merge, conflict policy | DNS serving, HTTP proxying, UI rendering |
| Registry (`internal/registry`) | immutable active snapshot + conflict read model | Docker I/O, proxy listeners |
| Daemon/Admin (`internal/daemon`, `internal/admin`) | refresh loop, watcher lifecycle, operator views over registry data | alternate routing rules or duplicate state |

## Plan Implications

- Split planning into: **contracts/foundation**, **discovery**, **domain/override logic**, **conflict + registry resolution**, and **watcher/visibility wiring**.
- Treat **port selection**, **domain generation**, and **conflict resolution** as test-first logic candidates.
- Ensure at least one plan writes the admin/read-model artifacts that future dashboard work can consume without redefining conflict semantics.

---

*Derived from `.planning/research/SUMMARY.md`, `ARCHITECTURE.md`, `FEATURES.md`, `PITFALLS.md`, and `.planning/PRD.md`.*
