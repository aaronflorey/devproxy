---
status: partial
phase: 03-install-daemon-lifecycle-and-diagnostics
source: [03-VERIFICATION.md]
started: 2026-05-06T00:00:00Z
updated: 2026-05-06T00:00:00Z
---

## Current Test

[awaiting human testing]

## Tests

### 1. Install lifecycle (`devproxy install`)
expected: resolver/paths/services installed correctly; menubar installed only with `--with-menubar`
result: [pending]

### 2. Doctor runtime diagnostics (`devproxy doctor`)
expected: accurate health output for docker/launchd/admin socket/listeners/scutil/mkcert/CA/domain resolution
result: [pending]

### 3. Uninstall selective cleanup (`devproxy uninstall`)
expected: teardown first, then only selected cleanup categories removed
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
