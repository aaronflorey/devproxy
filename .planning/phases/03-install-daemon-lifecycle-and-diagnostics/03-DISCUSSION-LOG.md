# Phase 3: Install, Daemon Lifecycle, and Diagnostics - Discussion Log (Assumptions Mode)

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md.

**Date:** 2026-05-05
**Phase:** 03-install-daemon-lifecycle-and-diagnostics
**Mode:** assumptions
**Areas analyzed:** Control Plane Integration, Foreground Daemon Startup Gating, Install/Uninstall Scope and Lifecycle Shape, Launchd Role Separation

## Assumptions Presented

### Control Plane Integration
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| CLI operator commands should be thin clients over one daemon-owned admin API/read model surface. | Likely | `internal/admin/status.go`, `internal/admin/routes.go`, `internal/admin/doctor.go`, `internal/admin/logs.go`, `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md` |

### Foreground Daemon Startup Gating
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| `devproxy daemon` should fail fast and explicitly when startup prerequisites fail. | Confident | `internal/daemon/network.go`, `internal/certs/mkcert.go`, `internal/certs/mkcert_test.go`, `.planning/phases/02-local-dns-proxy-and-https-serving/02-CONTEXT.md` |

### Install/Uninstall Scope and Lifecycle Shape
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Install/uninstall should orchestrate macOS lifecycle integration and state layout while runtime routing internals remain separate. | Likely | `internal/daemon/reconciler.go`, `internal/daemon/network.go`, `internal/proxy/http.go`, `internal/proxy/https.go`, `internal/dns/server.go`, `.planning/REQUIREMENTS.md` |

### Launchd Role Separation
| Assumption | Confidence | Evidence |
|------------|-----------|----------|
| Daemon auto-start is baseline; menubar auto-start is opt-in with separate lifecycle handling. | Confident | `.planning/ROADMAP.md`, `.planning/REQUIREMENTS.md`, `cmd/devproxy/root.go` |

## Corrections Made

No corrections - all assumptions confirmed.

## External Research

- Launchd service strategy and privilege split: use system daemon domain for privileged core listeners and user agent domain for session UI concerns. (Sources: Apple daemon/agent docs, `launchd.plist(5)`)
- Admin UNIX socket guidance: explicit file ownership/mode controls, stale-socket cleanup, and launchd socket-management options. (Sources: `launchd.plist(5)`, `unix(4)`)
- Resolver validation guidance: verify through macOS system resolver pipeline and `/etc/resolver` semantics, not only `dig`/`nslookup`. (Sources: `resolver(5)`, operational resolver behavior references)
