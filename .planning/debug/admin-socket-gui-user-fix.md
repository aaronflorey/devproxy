---
status: verifying
trigger: "Debug and fix the remaining admin socket permission issue on macOS."
created: 2026-05-06T00:00:00Z
updated: 2026-05-06T00:22:00Z
---

## Current Focus

hypothesis: On macOS `setAdminSocketAccess` only tries group `admin` best-effort chown (gid only), leaving owner root and relying on group membership, which fails for users not in `admin`.
test: Replace darwin path with explicit active GUI uid/gid resolution from `/dev/console` and enforce chown to that user; keep non-darwin unchanged.
expecting: Socket gets owned by active GUI user so non-root CLI can access even when not in `admin` group.
next_action: Commit minimal fix and report results.

reasoning_checkpoint:
  hypothesis: "setAdminSocketAccess uses admin-group fallback instead of active GUI user ownership, so non-admin desktop users cannot connect to root-owned socket"
  confirming_evidence:
    - "server.go currently chowns only gid to admin group and ignores failures"
    - "reported symptom is permission denied for non-root CLI even though socket exists"
  falsification_test: "If socket is already chowned to active GUI uid/gid and issue still reproduces, this hypothesis is wrong"
  fix_rationale: "Chowning darwin socket to active GUI uid/gid from /dev/console gives the intended desktop user direct rw ownership, removing fragile admin-group assumption"
  blind_spots: "Did not run on live macOS GUI session in this environment; relying on unit tests for behavior"

## Symptoms

expected: Non-root CLI should connect to daemon admin socket after reinstall.
actual: `/tmp/devproxy/admin.sock` exists but non-root CLI gets `permission denied`.
errors: permission denied on `/tmp/devproxy/admin.sock`.
reproduction: reinstall on macOS, launch daemon via launchd, run CLI as non-root user.
started: after reinstall; launchd itself reports OK.

## Eliminated

## Evidence

- timestamp: 2026-05-06T00:05:00Z
  checked: internal/adminapi/server.go:setAdminSocketAccess
  found: Function chmods 0660, then on darwin does lookupGroup("admin") and os.Chown(path, -1, gid) with all failures ignored.
  implication: Access policy depends on admin-group membership and keeps root owner, causing permission denied for non-admin GUI users.

- timestamp: 2026-05-06T00:07:00Z
  checked: internal/install/gui_user_darwin.go
  found: Installer resolves active GUI session via `/dev/console` stat and fails explicitly when uid <= 0.
  implication: Existing project pattern supports explicit GUI-user resolution and explicit failure semantics.

- timestamp: 2026-05-06T00:15:00Z
  checked: internal/adminapi/server.go and new gui_user_* files
  found: darwin path now resolves active GUI uid/gid from `/dev/console`, chmods 0600, chowns socket to uid/gid, and returns explicit errors on resolution/chown failure.
  implication: Socket access now targets the active desktop user directly instead of relying on admin group membership.

- timestamp: 2026-05-06T00:21:00Z
  checked: go test ./internal/adminapi ./internal/doctor ./cmd/devproxy
  found: All targeted tests passed after updating darwin/non-darwin socket ownership behavior and tests.
  implication: Change is compatible with admin API callers and related doctor/CLI paths.

## Resolution

root_cause: "On darwin, admin socket access relied on root:admin ownership (gid-only chown best-effort) instead of assigning ownership to the active GUI user; non-admin desktop users hit permission denied despite socket existing."
fix: "Changed darwin socket policy to resolve active GUI uid/gid from /dev/console and chown socket to that user, with restrictive 0600 mode and explicit error on GUI-user resolution/chown failure; kept non-darwin behavior unchanged (0660 with no darwin-specific ownership logic)."
verification: "Targeted tests passed: go test ./internal/adminapi ./internal/doctor ./cmd/devproxy"
files_changed:
  - internal/adminapi/server.go
  - internal/adminapi/gui_user_darwin.go
  - internal/adminapi/gui_user_other.go
  - internal/adminapi/server_test.go
