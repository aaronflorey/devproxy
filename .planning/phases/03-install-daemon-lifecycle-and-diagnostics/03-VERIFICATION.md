---
phase: 03-install-daemon-lifecycle-and-diagnostics
verified: 2026-05-06T01:19:47Z
status: gaps_found
score: 3/5 must-haves verified
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 8/8
  gaps_closed: []
  gaps_remaining:
    - "Developer can run uninstall and choose to retain or remove config, state, logs, and certificates"
  regressions:
    - "Human UAT shows uninstall aborts on launchctl bootout exit status 5 before scoped cleanup executes"
gaps:
  - truth: "Developer can run uninstall and choose to retain or remove config, state, logs, and certificates."
    status: failed
    reason: "03-HUMAN-UAT test 3 fails with `stop daemon service: launchctl bootout ... exit status 5: Boot-out failed: 5: Input/output error`, so selected cleanup does not run."
    artifacts:
      - path: "internal/install/uninstall.go"
        issue: "Uninstall stops on StopDaemonService errors not classified as already-removed state."
      - path: "internal/install/launchd.go"
        issue: "Known-missing-state matcher omits Boot-out failed I/O patterns observed in UAT."
    missing:
      - "Treat launchd bootout 'already removed/invalid state' variants (including Boot-out failed: 5 where service is absent) as idempotent teardown or handle with explicit fallback check before aborting."
      - "Add regression test reproducing UAT bootout error path and asserting uninstall continues to resolver/scoped cleanup."
---

# Phase 3: Install, Daemon Lifecycle, and Diagnostics Verification Report

**Phase Goal:** Developers can install, run, inspect, troubleshoot, and uninstall devproxy reliably on macOS.
**Verified:** 2026-05-06T01:19:47Z
**Status:** gaps_found
**Re-verification:** Yes — based on completed `03-HUMAN-UAT.md` evidence

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | Developer can run `devproxy install` and have required paths/resolver/certs/services configured and started. | ? UNCERTAIN (WARNING) | UAT test 1 hit explicit prerequisite failures (`requires root privileges`, then `mkcert not found`). Failure messaging is correct, but successful privileged install path was not demonstrated in submitted UAT evidence. |
| 2 | Menu bar auto-start is installed only when `--with-menubar` is used. | ✓ VERIFIED | `internal/install/install.go` only installs/starts menubar inside `if opts.WithMenubar`; tests cover opt-in behavior. |
| 3 | `devproxy daemon` in foreground emits explicit startup failures for missing Docker/mkcert/listeners. | ✓ VERIFIED | `internal/daemon/app.go` fail-fast checks + targeted tests; UAT/doctor output also shows explicit dependency failures rather than silent fallback. |
| 4 | `status`, `routes`, `refresh`, `doctor`, and `logs` inspect live daemon state from the same local admin API source. | ✓ VERIFIED | CLI commands use `internal/adminapi/client.go`; admin server exposes `/status,/routes,/refresh,/logs,/doctor` on one UNIX socket. |
| 5 | Developer can run uninstall and selectively retain/remove config, state, logs, certificates. | ✗ FAILED (BLOCKER) | UAT test 3 shows uninstall aborts on `launchctl bootout ... Boot-out failed: 5: Input/output error` before scoped cleanup completes. This directly violates selective cleanup completion truth. |

**Score:** 3/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `internal/install/uninstall.go` | Teardown + scoped cleanup orchestration | ⚠️ HOLLOW — wired but behavior fails | Exists/substantive/wired, but real UAT shows teardown error path blocks selected cleanup. |
| `internal/install/launchd.go` | Idempotent launchd stop/uninstall helpers | ⚠️ PARTIAL | Missing-state matcher handles several strings but not observed `Boot-out failed: 5` path from UAT. |
| `internal/doctor/checks.go` | Runtime diagnostic checks | ✓ VERIFIED | Substantive and wired; reports explicit checks/failures. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `cmd/devproxy/uninstall.go` | `internal/install/uninstall.go` | prompted cleanup scope selection | ✓ WIRED | Prompt responses map into `CleanupScope` and call `Uninstall(...)`. |
| `internal/install/uninstall.go` | `internal/install/launchd.go` | stop/uninstall service lifecycle | ⚠️ PARTIAL | Wiring exists (`StopService`/`UninstallService`), but UAT shows stop path returns fatal error for a real bootout state, preventing full lifecycle completion. |
| `cmd/devproxy/doctor.go` | `internal/doctor/checks.go` | checker execution/rendering | ✓ WIRED | Command constructs checker and prints report. |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `cmd/devproxy/doctor.go` | `report.Checks` | `checker.Run(...)` probes host/runtime | Yes | ✓ FLOWING |
| `cmd/devproxy/uninstall.go` | `scope` | interactive prompts → `CleanupScope` | Yes, but flow is interrupted before cleanup in failing bootout path | ⚠️ FLOW BLOCKED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Targeted Phase 3 tests pass | `go test ./internal/install/... ./internal/doctor/... ./cmd/devproxy/...` | exit 0 | ✓ PASS |
| CLI surface includes lifecycle/diagnostic commands | `go run ./main.go --help` | shows `daemon install status routes refresh logs doctor uninstall` | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| OPS-01 | 03-03, 03-05 | Install baseline lifecycle | ? NEEDS HUMAN | Current UAT evidence shows prerequisite failures only; no successful install run captured. |
| OPS-02 | 03-03, 03-05 | Menubar opt-in only | ✓ SATISFIED | Flag-gated install path in `internal/install/install.go` + tests. |
| OPS-03 | 03-01 | Foreground daemon fail-fast errors | ✓ SATISFIED | Explicit dependency failure paths in daemon app and tests. |
| OPS-04 | 03-02 | Status command health visibility | ✓ SATISFIED | Thin admin client + command wiring verified. |
| OPS-05 | 03-02 | Routes + refresh operator flows | ✓ SATISFIED | Client-backed routes/refresh paths present and tested. |
| OPS-06 | 03-04, 03-06 | Doctor diagnostics | ✓ SATISFIED | Checks include required categories and emit explicit failures. |
| OPS-07 | 03-02 | Current-session logs | ✓ SATISFIED | Logs command + admin API event flow present. |
| OPS-08 | 03-04, 03-05 | Selective uninstall cleanup | ✗ BLOCKED | UAT test 3 aborts before scoped cleanup due launchctl bootout error. |
| OPS-09 | 03-01, 03-02 | Shared admin API socket source | ✓ SATISFIED | Commands consume one local admin API client/socket path. |

Orphaned requirements for Phase 3: **none**.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `cmd/devproxy/logs.go` | 19 | "Persisted history is not available in v1" | ℹ️ Info | Scope declaration, not a stub. |

### Gaps Summary

Phase 3 is **not achieved** yet. Human UAT falsified the prior “all verified” conclusion for uninstall lifecycle behavior. The uninstall path can fail on a real launchd bootout state (`Boot-out failed: 5`) before selected cleanup executes, which breaks OPS-08 and roadmap success criterion #5.

Install success-path evidence is still incomplete in UAT (mkcert missing on host), so OPS-01 remains a warning/uncertain item pending a prerequisite-satisfied rerun.

---

_Verified: 2026-05-06T01:19:47Z_  
_Verifier: the agent (gsd-verifier)_
