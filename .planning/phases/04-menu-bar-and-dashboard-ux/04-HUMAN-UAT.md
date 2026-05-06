---
status: human_needed
phase: 04-menu-bar-and-dashboard-ux
source: [04-VERIFICATION.md, 04-05-PLAN.md]
started: 2026-05-06T03:48:45Z
updated: 2026-05-06T03:48:45Z
checkpoint: checkpoint:human-verify
resume_signal: approved
---

## Current Test

Awaiting native macOS human verification for the two remaining UI checks.

## Tests

### 1. Native macOS menubar route-opening flow
expected: Active routes appear as selectable menu items and open daemon-provided OpenURL (https/http fallback preserved).
result: [pending]

```bash
# prereqs: macOS host, devproxy daemon running, at least one active managed route
devproxy menubar

# manually verify:
# - one item per active route above static actions
# - clicking HTTPS-ready route opens https://...
# - clicking degraded route opens http://... fallback
```

result

```text
Pending human execution.
```

### 2. Dashboard visual UX and degraded-state copy
expected: Health/routes/conflicts/current-session errors are legible; degraded copy appears only in true degraded/offline conditions.
result: [pending]

```bash
devproxy dashboard
# open http://127.0.0.1:45831/ in browser
# manually verify readability and degraded/offline copy behavior
```

result

```text
Pending human execution.
```

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Checkpoint

Type `approved` when both tests pass, or provide the concrete mismatch observed.
