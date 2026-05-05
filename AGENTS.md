<!-- GSD:project-start source:PROJECT.md -->
## Project

**devproxy**

DevProxy is a macOS-native developer tool for people running Docker Compose projects locally. It watches running containers, assigns local vanity domains like `acme-crm.test` and `api.acme-crm.test`, serves local DNS, and proxies HTTP and HTTPS traffic to the right published localhost port without requiring Compose file changes.

**Core Value:** A developer can run `docker compose up` and immediately use predictable local domains for each routable service without editing Compose files, `/etc/hosts`, or local proxy configs.

### Constraints

- **Platform**: macOS only for v1 — installation, resolver configuration, LaunchAgents, and menu bar UX are all macOS-specific.
- **Runtime**: Use Docker Desktop and Docker Engine events — the daemon must react to container lifecycle changes and inspect running containers on startup.
- **Networking**: DNS resolves managed suffixes to `127.0.0.1`, and proxy listeners run locally on ports `80` and `443` — this keeps traffic local and predictable.
- **TLS**: `mkcert` is required for local certificate generation — install and startup should fail clearly when HTTPS prerequisites are missing.
- **Safety**: Prefer explicit failures and warnings over silent rewrites or guessed fallback behavior — debugging and trust are core product requirements.
- **Compatibility**: Do not require Compose-file edits or a sidecar proxy container for common cases — normal `docker compose up` must remain the default workflow.
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

## Recommended Stack
### Core Framework
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.26.x | Single static binary for CLI + daemon + optional menubar process | Best fit for long-running local networking daemons, low memory, first-class macOS support, and strong stdlib networking primitives. **Confidence: HIGH** |
| `net/http` + `net/http/httputil` (stdlib) | Go stdlib | HTTP reverse proxy and local admin API | In 2026 Go’s `ReverseProxy` supports the safer `Rewrite` path and warns against older `Director` behavior. Use stdlib instead of adding a proxy framework in v1. **Confidence: HIGH** |
| `github.com/spf13/cobra` | v1.9.x | CLI command surface (`install`, `daemon`, `status`, etc.) | Standard Go CLI stack; stable subcommand/flag model and ecosystem familiarity. **Confidence: HIGH** |
| `github.com/spf13/viper` | v1.20.x | Config file + env override loading | Mature config layering (file + env) that matches devproxy install/runtime needs. **Confidence: MEDIUM** |
### Database
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| No external DB in v1 (in-memory registry + YAML/JSON state files) | n/a | Route registry and small persisted state (cert metadata, install state, preferences) | This product is daemon-local infrastructure, not multi-user SaaS. SQLite/Postgres adds migration and corruption surface without user value in v1. **Confidence: MEDIUM** |
### Infrastructure
| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Docker Engine API via `github.com/moby/moby/client` | Current stable | Watch container lifecycle events + inspect running containers | Official Go client exposes event stream (`/events`) and container list/inspect APIs needed for discovery. **Confidence: HIGH** |
| `github.com/miekg/dns` | current stable | Local authoritative DNS server on `127.0.0.1:53535` for `*.test` | Battle-tested Go DNS server library; straightforward A/AAAA handling and muxing by zone. **Confidence: HIGH** |
| macOS `/etc/resolver/<suffix>` + resolver(5) model | macOS built-in | Route `*.test` DNS queries to local DNS listener | Native per-domain resolver routing on macOS; avoids brittle `/etc/hosts` mutation and supports wildcard domain behavior cleanly. **Confidence: HIGH** |
| `launchd` (`LaunchDaemon` for privileged bind, optional `LaunchAgent` for user UX) | macOS built-in | Service lifecycle, auto-restart, startup integration | Apple’s standard process manager. Critical point: privileged ports 80/443 require root-managed socket binding path; do not rely on user LaunchAgent for these listeners. **Confidence: HIGH** |
| `mkcert` CLI | v1.4.x line (current maintained release line) | Local CA bootstrap and per-domain cert issuance | De-facto local dev cert tool, simple UX (`mkcert -install`, SAN hostnames), trusted by browsers/system stores. **Confidence: HIGH** |
### Supporting Libraries
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/fsnotify/fsnotify` | v1.x | Watch config/state file changes for hot reload | Use only if you want immediate config reload without daemon restart; skip for first cut if `devproxy refresh` is enough. **Confidence: MEDIUM** |
| `go.uber.org/zap` | v1.x | Structured logging | Use when you need machine-parseable logs for `doctor`/`status`; otherwise stdlib `log/slog` is acceptable. **Confidence: MEDIUM** |
| `github.com/getlantern/systray` | current stable | Optional macOS menu bar UI from Go | Fastest path to optional menubar, but requires CGO and app bundle packaging. Use only behind `--with-menubar`. **Confidence: MEDIUM** |
## Alternatives Considered
| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| HTTP reverse proxy | Go stdlib `ReverseProxy` | Caddy embed / Traefik sidecar | Overkill for v1; brings extra config model and dependency surface when routing logic is already in your daemon. |
| DNS | `miekg/dns` in-process | dnsmasq/CoreDNS sidecar | Extra process management + install friction; in-process server is simpler and easier to debug for single-suffix local DNS. |
| Service manager | `launchd` plist install | Homebrew services as primary lifecycle | Homebrew services is operator tooling, not product-grade install contract; devproxy should own launchd setup directly. |
| Certs | mkcert CLI invocation | ACME/Let’s Encrypt flows | Internet CAs do not fit local `.test`/localhost workflows; adds pointless external dependency and failure modes. |
| Menubar | Optional `systray` companion | Electron/Tauri app for v1 | Much heavier runtime/packaging/notarization burden; slows v1 without core routing value. |
| Persistence | File-backed state | SQLite from day one | Premature complexity for local single-user tool; revisit only when query/history needs justify it. |
## Installation
# Core
# Optional
## v1 Prescriptive Build Notes (macOS integration)
## What NOT to use for v1
- **Do not require a sidecar proxy container** (breaks “just run compose up” goal).
- **Do not use `/etc/hosts` mutation as primary mechanism** (no wildcard support, conflict-prone).
- **Do not use user-only LaunchAgent for low ports** (port 80/443 privilege mismatch).
- **Do not adopt Kubernetes ingress patterns** (Traefik CRDs, Ingress resources) for a local Compose tool.
- **Do not build full GUI app first**; keep menubar optional and thin over local daemon API.
## Sources
- Docker events CLI/reference (real-time event stream + container event types): https://docs.docker.com/reference/cli/docker/system/events/  
- Docker Compose labels reference (canonical Compose labels incl. `com.docker.compose.project/service`): https://docs.docker.com/compose/compose-file/05-services/#labels  
- Moby Go client docs (`Events`, list/inspect APIs): https://pkg.go.dev/github.com/moby/moby/client  
- miekg/dns docs and server patterns: https://pkg.go.dev/github.com/miekg/dns and Context7 `/miekg/dns`  
- Go `httputil` docs (`ReverseProxy`, `Rewrite`, `Director` deprecation/security notes): https://pkg.go.dev/net/http/httputil  
- Cobra docs (command/flag structure): https://github.com/spf13/cobra  
- Viper docs (config file + env binding): https://github.com/spf13/viper  
- Apple launchd guidance (daemons/agents, launchd behaviors): https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html  
- macOS resolver file format (`/etc/resolver/*`, per-domain resolver behavior): https://manp.gs/mac/5/resolver  
- mkcert official README/usage: https://github.com/FiloSottile/mkcert  
- getlantern/systray README (CGO requirement, macOS bundle notes): https://github.com/getlantern/systray
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
