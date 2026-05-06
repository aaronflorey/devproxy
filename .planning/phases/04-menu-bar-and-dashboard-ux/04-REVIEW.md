---
phase: 04-menu-bar-and-dashboard-ux
reviewed: 2026-05-06T00:00:00Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - cmd/devproxy/dashboard.go
  - cmd/devproxy/menubar.go
  - cmd/devproxy/lifecycle_test.go
  - internal/dashboard/server.go
  - internal/dashboard/templates.go
  - internal/dashboard/templates/dashboard.html.tmpl
  - internal/dashboard/templates/logs.html.tmpl
  - internal/dashboard/server_test.go
  - internal/menubar/app.go
  - internal/menubar/open.go
  - internal/menubar/runtime_darwin.go
  - internal/menubar/runtime_stub.go
  - internal/menubar/app_test.go
  - internal/adminapi/server.go
  - internal/adminapi/client.go
  - internal/daemon/app.go
  - internal/install/launchd.go
findings:
  critical: 2
  warning: 2
  info: 0
  total: 4
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-05-06T00:00:00Z  
**Depth:** standard  
**Files Reviewed:** 17  
**Status:** issues_found

## Summary

Reviewed Phase 4 menu bar + dashboard implementation and related tests with focus on correctness, security boundaries, and robustness. I found two shipping blockers and two warnings. The most serious issue is admin API client methods silently treating server-side failures as successful responses due to missing HTTP status handling.

## Critical Issues

### CR-01: Admin API client silently swallows non-2xx responses for most endpoints

**Classification:** BLOCKER  
**File:** `internal/adminapi/client.go:110-151`  
**Issue:** `fetchJSON` and `postJSON` decode response bodies without checking `resp.StatusCode`. If the server returns an error payload (`{"error":"..."}`) with 4xx/5xx, JSON decode into typed success structs can still succeed with zero values, causing false-success behavior (e.g., startup status appears empty instead of failing; pause/resume may look successful depending on caller logic).

**Fix:** Check HTTP status codes before decoding into success payloads, decode `ErrorResponse` for non-2xx, and return an explicit error.

```go
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
    var er ErrorResponse
    _ = json.NewDecoder(resp.Body).Decode(&er)
    if er.Error == "" {
        er.Error = fmt.Sprintf("unexpected status %d", resp.StatusCode)
    }
    return zero, fmt.Errorf("%s: %s", path, er.Error)
}
```

### CR-02: Dashboard shows “daemon unreachable” when daemon is reachable but routes are simply empty

**Classification:** BLOCKER  
**File:** `internal/dashboard/server.go:145-147`  
**Issue:** `data.DaemonError` is set to the daemon-offline message whenever `len(routes) == 0`, even after successful `Status()` and `Routes()` calls. This misreports system state and can trigger incorrect operator actions.

**Fix:** Only set `DaemonError` when daemon/API calls fail. Keep “No Active Routes” as a separate healthy-but-empty state.

```go
if routesErr == nil {
    data.NoActiveRoutes = len(routes) == 0
}
// remove: if data.NoActiveRoutes { data.DaemonError = errDaemonUnreachable }
```

## Warnings

### WR-01: Menubar runtime drops all action errors, making failures invisible

**Classification:** WARNING  
**File:** `internal/menubar/runtime_darwin.go:50-62`  
**Issue:** Click handlers call dispatcher methods with `_ = ...`, discarding all errors (refresh, pause/resume, open dashboard/logs, startup toggle, doctor). This creates silent failure paths and hides operational problems from users.

**Fix:** Capture errors and surface them (status line, notification, or internal issue log), e.g. a shared `handleActionErr(err)` path.

### WR-02: Lifecycle dashboard URL test contains tautological assertion

**Classification:** WARNING  
**File:** `cmd/devproxy/lifecycle_test.go:44-46`  
**Issue:** The test compares a constant string to itself (`got` and `want` are identical literals), so it cannot detect regressions. This weakens reliability of the Phase 4 URL contract test.

**Fix:** Assert against actual produced value (e.g., menubar/dashboard constant or computed command target), not two literals.

```go
if got := "http://" + listen + "/logs"; got != "http://127.0.0.1:45831/logs" {
    t.Fatalf("expected fixed logs URL, got %q", got)
}
```

---

_Reviewed: 2026-05-06T00:00:00Z_  
_Reviewer: the agent (gsd-code-reviewer)_  
_Depth: standard_
