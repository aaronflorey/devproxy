---
status: completed
phase: 03-install-daemon-lifecycle-and-diagnostics
source: [03-VERIFICATION.md]
started: 2026-05-06T00:00:00Z
updated: 2026-05-12T00:00:00Z
---

## Current Test

Privileged macOS install success-path run for OPS-01 completed successfully.

## Tests

### 1. Install lifecycle (`devproxy install`)
expected: resolver, privileged paths, certificates, and daemon service install successfully on macOS with prerequisites satisfied
result: [passed]

```bash
# prereqs: macOS host, Docker Desktop running, mkcert installed, root privileges
sudo go run ./main.go install

# verify resolver + launchd artifacts
sudo launchctl print system/com.devproxy.daemon
sudo ls -la /etc/resolver/test
sudo ls -la /Library/LaunchDaemons/com.devproxy.daemon.plist
```

result

```text
Successful privileged macOS UAT captured. `launchctl print system/com.devproxy.daemon` showed the service running from `/Library/LaunchDaemons/com.devproxy.daemon.plist` with program `/usr/local/bin/devproxy`. `ls -la /etc/resolver/test` confirmed the resolver file exists. `ls -la /Library/LaunchDaemons/com.devproxy.daemon.plist` confirmed the daemon plist exists.
```

## Summary

total: 1
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 0

## Handoff

- Privileged macOS install UAT is complete for this phase.
- Evidence recorded: running launchd service, resolver file present, daemon plist present.
- No uninstall follow-up is currently required here; the prior `launchctl bootout ... exit status 5` issue is closed in code verification.
