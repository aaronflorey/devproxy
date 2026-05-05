# Phase 2: Local DNS, Proxy, and HTTPS Serving - Discussion Log (Assumptions Mode)

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions captured in CONTEXT.md - this log preserves the analysis.

**Date:** 2026-05-05T11:02:51+00:00
**Phase:** 02-local-dns-proxy-and-https-serving
**Mode:** assumptions
**Areas analyzed:** Routing State, Proxy Transport And Upstream Selection, DNS Scope, Certificate Strategy, Friendly Responses And Pause State

## Assumptions Presented

### Routing State
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| DNS, HTTP, HTTPS, and paused/no-route responses should all read from the immutable `routing.Snapshot` published by the reconciler. | Confident | `internal/daemon/reconciler.go`, `internal/registry/snapshot.go`, `internal/admin/routes.go`, `internal/admin/status.go`, `.planning/phases/01-discovery-domains-and-conflict-policy/01-discovery-domains-and-conflict-policy-05-SUMMARY.md` |

### Proxy Transport And Upstream Selection
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Phase 2 should keep `routing.Route.Upstream` as the canonical proxy target and fill it from existing discovery and override metadata instead of the current hard-coded `127.0.0.1` plus `http` behavior. | Likely | `internal/routing/types.go`, `internal/discovery/metadata.go`, `internal/routing/overrides.go`, `internal/discovery/ports.go`, `internal/daemon/reconciler.go` |
| Go stdlib `net/http/httputil.ReverseProxy` is sufficient for HTTP, HTTPS, and WebSocket upgrade proxying in Phase 2 as long as listener wrappers preserve hijacking. | High after research | official Go 1.26 `httputil` docs and source, plus `internal/routing/types.go` and Phase 2 NET-06 requirements |

### DNS Scope
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| The in-process DNS responder should answer only for the configured managed suffix and return `127.0.0.1` there, while unmanaged explicit domains remain best-effort with warnings. | High after research | `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`, `internal/config/config.go`, `internal/routing/domains.go`, macOS resolver research |

### Certificate Strategy
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Certificate generation and reuse should key off the active served hostname set, with a likely per-project apex-plus-wildcard certificate strategy. | Medium-High after research | `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`, `internal/registry/snapshot.go`, `internal/routing/conflicts.go`, mkcert research |

### Friendly Responses And Pause State
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Friendly no-route and paused responses should come from the local listeners for managed hostnames, and paused state needs daemon state outside the current snapshot model. | Unclear | `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`, `internal/routing/types.go`, `internal/admin/status.go`, `internal/daemon/events.go` |

## Corrections Made

No corrections - all assumptions confirmed.

## External Research

- Go `httputil.ReverseProxy`: Go 1.26 already supports upgrade pass-through including WebSockets; preserve `http.Hijacker` support around the proxy. (Sources: `https://pkg.go.dev/net/http/httputil@go1.26.0`, `https://go.dev/src/net/http/httputil/reverseproxy.go`)
- mkcert strategy: a per-project certificate covering `{project}.{suffix}` and `*.{project}.{suffix}` is a better fit than a single global cert when service hostnames churn inside a project. (Sources: `https://github.com/FiloSottile/mkcert`, `https://github.com/FiloSottile/mkcert/issues/144`, `https://github.com/FiloSottile/mkcert/issues/60`, `https://github.com/FiloSottile/mkcert/issues/161`)
- macOS resolver behavior: `/etc/resolver/<suffix>` still supports suffix-specific routing to a local DNS listener; `scutil --dns` is the correct verification surface and cache behavior must be accounted for in diagnostics. (Sources: `https://man.freebsd.org/cgi/man.cgi?manpath=macOS+26.3&query=resolver&sektion=5`, `https://support.apple.com/en-us/101481`, `https://developer.apple.com/forums/thread/759115`)
