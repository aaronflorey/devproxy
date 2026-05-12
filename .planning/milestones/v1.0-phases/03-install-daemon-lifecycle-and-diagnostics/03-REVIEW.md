---
phase: 03-install-daemon-lifecycle-and-diagnostics
reviewed: 2026-05-05T23:59:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - cmd/devproxy/commands.go
  - cmd/devproxy/daemon.go
  - cmd/devproxy/root.go
  - cmd/devproxy/status.go
  - cmd/devproxy/routes.go
  - cmd/devproxy/refresh.go
  - cmd/devproxy/logs.go
  - cmd/devproxy/install.go
  - cmd/devproxy/doctor.go
  - cmd/devproxy/uninstall.go
  - internal/adminapi/server.go
  - internal/adminapi/client.go
  - internal/daemon/app.go
  - internal/doctor/checks.go
  - internal/install/uninstall.go
findings:
  critical: 2
  warning: 4
  info: 0
  total: 6
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-05-05T23:59:00Z  
**Depth:** standard  
**Files Reviewed:** 15  
**Status:** issues_found

## Summary

Reviewed Phase 3 implementation changes spanning daemon lifecycle, admin API client/server, install/uninstall, and diagnostics commands. Multiple correctness defects were found in diagnostics and uninstall behavior, plus API error-handling gaps that can produce false-success outcomes.

## Critical Issues

### CR-01 (BLOCKER): Resolver diagnostics are hardcoded to `.test` and can falsely fail/pass

**File:** `internal/doctor/checks.go:221-230`

**Issue:** `readResolverState` hardcodes `ManagedSuffix: "test"` and only checks `scutil --dns` for `domain : test` / `.test`. For any configured suffix other than `test`, doctor output is incorrect (false failures or false passes), violating diagnostic correctness.

**Fix:** Thread configured suffix into resolver-state check and match against that suffix.

```go
func readResolverState(ctx context.Context, suffix string) (ResolverState, error) {
    cmd := exec.CommandContext(ctx, "scutil", "--dns")
    out, err := cmd.CombinedOutput()
    if err != nil { ... }

    normalized := strings.TrimPrefix(strings.TrimSpace(suffix), ".")
    needleA := "domain : " + normalized
    needleB := "domain : ." + normalized
    text := string(out)
    active := strings.Contains(text, needleA) || strings.Contains(text, needleB)

    return ResolverState{ManagedSuffix: normalized, ActiveResolver: active, ...}, nil
}
```

### CR-02 (BLOCKER): Uninstall can leave resolver file behind when suffix includes leading dot

**File:** `internal/install/uninstall.go:120-130`

**Issue:** `RemoveResolver` uses `cfg.Suffix` as-is, while install normalizes suffix by trimming leading dot (`WriteResolver`). If uninstall is called with `.test`, it removes `/etc/resolver/.test` instead of `/etc/resolver/test`, leaving active resolver config behind.

**Fix:** Normalize suffix during uninstall exactly the same way as install.

```go
suffix := strings.TrimSpace(strings.TrimPrefix(cfg.Suffix, "."))
if suffix == "" {
    suffix = "test"
}
path := filepath.Join(resolverDir, suffix)
```

## Warnings

### WR-01 (WARNING): Admin API client ignores HTTP status codes and may silently treat failures as success

**File:** `internal/adminapi/client.go:81-91`, `internal/adminapi/client.go:101-112`

**Issue:** `Refresh` and `fetchJSON` decode bodies without checking `resp.StatusCode`. Non-2xx responses (including structured error payloads) can be decoded into zero-value success-like structs without returning an error, producing misleading CLI output and broken control flow.

**Fix:** Validate status codes before decode; decode `ErrorResponse` on non-2xx and return explicit errors.

### WR-02 (WARNING): Refresh handler accepts malformed JSON payloads silently

**File:** `internal/adminapi/server.go:145-147`

**Issue:** Request decode error is ignored (`_ = json.NewDecoder(req.Body).Decode(&payload)`). Malformed JSON still triggers refresh with empty/default reason, masking bad client behavior.

**Fix:** Handle decode errors and return `400 Bad Request`.

```go
if err := json.NewDecoder(req.Body).Decode(&payload); err != nil && !errors.Is(err, io.EOF) {
    writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON payload"})
    return
}
```

### WR-03 (WARNING): HTTPS doctor check uses default TLS verification against `127.0.0.1`, causing likely false failures

**File:** `internal/doctor/checks.go:193-199`

**Issue:** `http.Get("https://127.0.0.1")` performs strict cert verification against IP SANs. Devproxy certs are expected for managed domains, not necessarily loopback IP SANs, so this can fail even when HTTPS proxy is healthy.

**Fix:** Check listener health from admin status as source of truth, or perform HTTPS probe with expected SNI host mapped to loopback and controlled TLS settings for diagnostics.

### WR-04 (WARNING): Uninstall command fails in non-interactive execution due to mandatory prompts

**File:** `cmd/devproxy/uninstall.go:44-53`

**Issue:** `promptCleanupScope` requires newline-terminated stdin answers. In CI/non-interactive shells, `ReadString('\n')` returns EOF and uninstall aborts before service teardown.

**Fix:** Add non-interactive flags (`--yes`, `--cleanup=...`) and default EOF handling to safe defaults instead of hard failure.

---

_Reviewed: 2026-05-05T23:59:00Z_  
_Reviewer: the agent (gsd-code-reviewer)_  
_Depth: standard_
