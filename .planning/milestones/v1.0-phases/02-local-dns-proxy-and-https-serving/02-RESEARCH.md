# Phase 2 Research: Local DNS, Proxy, and HTTPS Serving

**Phase:** 2
**Researched:** 2026-05-05
**Status:** Ready for planning

## Standard Stack

- **Go 1.26.x** for the daemon and listener runtime.
- **`github.com/miekg/dns`** for an in-process authoritative DNS server bound to the local resolver port.
- **Go stdlib `net/http`, `net/http/httputil`, and `crypto/tls`** for HTTP, HTTPS, WebSocket upgrades, and listener configuration.
- **`mkcert` CLI** as the certificate authority and leaf-certificate issuer for locally trusted HTTPS.
- **Existing Phase 1 snapshot/admin packages** as the authoritative routing and operator state source.

## Architecture Patterns

1. **Snapshot-fed serving**: DNS answers, HTTP proxying, HTTPS certificate selection, and friendly fallback responses should all read from the published `routing.Snapshot` plus explicit runtime pause state.
2. **Serve managed suffix only**: DNS answers must be limited to the configured managed suffix; unmanaged explicit domains remain route metadata only and should surface warnings instead of silently expanding DNS scope.
3. **Proxy transport follows resolved upstream**: reconciliation should compute the effective upstream host, port, and scheme once so request handlers do not re-run port or scheme selection.
4. **Per-project wildcard cert reuse**: derive served hostnames from winning routes, group them by project base hostname, and reuse certificates until coverage changes.
5. **Friendly local ownership on misses**: requests for managed hostnames with no active route or paused routing should terminate locally with explicit responses instead of generic connection failures.

## Phase-Specific Requirements To Preserve

- Bind DNS answers for managed hostnames to **`127.0.0.1`** only.
- Support **HTTP and HTTPS listeners simultaneously**, with redirect-to-HTTPS staying **off by default**.
- Preserve **Docker-label-over-config precedence** for effective upstream port and scheme selection.
- Support **WebSocket upgrade traffic** through the same proxy path as HTTP.
- Regenerate certificates only when the active hostname inventory changes beyond current wildcard coverage.
- Keep **DNS resolution active while routing is paused** so paused behavior is observable in the local listeners.

## Do Not Hand-Roll

- Do not build custom HTTP upgrade tunneling when `httputil.ReverseProxy` can preserve upgrade behavior.
- Do not generate one certificate per hostname when the hostname set can be covered by a project root plus wildcard pair.
- Do not inspect Docker or rebuild routes inside DNS or request handlers.
- Do not conflate missing route state with paused routing; they need different user-facing responses and operator diagnostics.
- Do not silently enable HTTPS redirect for existing routes; it must remain opt-in.

## Common Pitfalls

1. **Snapshot/request divergence**: if listeners read mutable partial state instead of the published snapshot, DNS and proxy answers can disagree.
2. **Wrong scheme forwarding**: upstream `https` routes need TLS transport configuration, not just `http://127.0.0.1:<port>` defaults.
3. **Broken WebSockets**: middleware that drops `http.Hijacker` or rewrites upgrade headers incorrectly will regress NET-06.
4. **Over-eager cert rotation**: treating any hostname churn as full reissue increases mkcert churn and slows reconcile loops.
5. **Bad managed-host detection**: no-route and paused responses should apply only to hostnames under the configured suffix, not to arbitrary Host headers.

## Architectural Responsibility Map

| Tier | Owns | Does Not Own |
|------|------|--------------|
| Reconciler (`internal/daemon`) | effective upstream metadata, served-host inventory, publishable runtime inputs | DNS packet parsing, HTTP handler details |
| DNS (`internal/dns`) | suffix match, A-record answers, managed-host detection helpers | route conflict resolution, proxy responses |
| Proxy (`internal/proxy`) | HTTP/HTTPS listeners, reverse proxy wiring, friendly no-route/paused responses | Docker inspection, route recomputation |
| TLS (`internal/certs`) | hostname inventory diffing, mkcert invocation, cert cache loading | redirect policy, request routing |
| Admin/runtime state | pause flag, listener/cert health summaries for later CLI/UI surfaces | direct network serving |

## Plan Implications

- Start with a **foundation plan** that extends snapshot and runtime contracts so every later networking component reads the same data.
- Keep **DNS serving**, **HTTP proxy behavior**, and **certificate lifecycle** in separate plans so tests stay focused and failures are easier to isolate.
- Add a dedicated **HTTPS/runtime wiring** plan after cert management so listener startup can consume prepared certificate state.
- End with an **admin/health integration** plan that exposes network runtime truth to future Phase 3 operator commands without redefining state.

---

*Derived from `.planning/PRD.md`, `.planning/REQUIREMENTS.md`, `.planning/PROJECT.md`, and `.planning/phases/02-local-dns-proxy-and-https-serving/02-CONTEXT.md`.*
