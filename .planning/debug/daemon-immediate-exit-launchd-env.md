---
status: verifying
trigger: "Debug and fix likely daemon immediate-exit issue after install for macOS launchd service."
created: 2026-05-06T05:33:33+00:00
updated: 2026-05-06T05:42:00+00:00
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "launchd service immediately exits because required binaries are not found in launchd's restricted PATH, and doctor reports false-positive launchd health because it does not require running state"
  confirming_evidence:
    - "plistFor emits no EnvironmentVariables PATH block"
    - "checkLaunchd returns success on any successful launchctl print output without state parsing"
  falsification_test: "if PATH is added to plist and launchd check still passes for `state = exited` output, hypothesis is wrong"
  fix_rationale: "inject explicit PATH into launchd plist for daemon/menubar and require `state = running` in doctor launchd check so immediate exits are surfaced"
  blind_spots: "cannot run real launchd service lifecycle on Linux runner; validating via unit tests and simulated launchctl output"

hypothesis: launchd PATH omission plus launchd-state false positive causes install-success-but-daemon-dead behavior.
test: run targeted tests covering plist env emission and doctor launchd non-running detection.
expecting: install tests assert PATH in plist and doctor test fails on state=exited with output hints.
next_action: request human verification on macOS workflow after committing targeted fix.

## Symptoms

expected: daemon stays running after install, admin socket exists, listeners/proxy checks pass, managed domains resolve.
actual: install succeeds but no menubar icon, admin socket missing at /tmp/devproxy/admin.sock, listeners/proxy checks fail, managed domain resolution fails.
errors: doctor launchd ok but daemon runtime artifacts are missing.
reproduction: run install, then doctor shows launchd ok while socket/listener checks fail.
started: after recent install flow changes where install now succeeds.

## Eliminated

## Evidence

- timestamp: 2026-05-06T05:33:33+00:00
  checked: user symptom report
  found: launchd health appears false-positive and daemon runtime appears not alive.
  implication: service may be installed but exiting quickly; health check likely insufficient.

- timestamp: 2026-05-06T05:35:00+00:00
  checked: internal/install/launchd.go plistFor output
  found: launchd plist includes Label/ProgramArguments/RunAtLoad/KeepAlive only, with no EnvironmentVariables PATH.
  implication: launchd child process may miss Homebrew binary paths and fail dependency preflight at runtime.

- timestamp: 2026-05-06T05:35:30+00:00
  checked: internal/doctor/checks.go checkLaunchd
  found: checkLaunchd returns success whenever `launchctl print system/com.devproxy.daemon` exits 0, without checking `state` field.
  implication: exited/waiting services can be falsely reported as healthy launchd state.

- timestamp: 2026-05-06T05:41:00+00:00
  checked: targeted unit tests
  found: `go test ./internal/install ./internal/doctor ./cmd/devproxy` passed after adding PATH environment to plist and running-state validation in doctor launchd check.
  implication: code changes compile and targeted regressions are covered.

## Resolution

root_cause: ""
fix: "Added explicit launchd PATH EnvironmentVariables block to generated daemon/menubar plists and tightened doctor launchd check to require `state = running`, surfacing print output when not running."
verification: ""
verification: "Self-verified with targeted tests: go test ./internal/install ./internal/doctor ./cmd/devproxy"
files_changed: ["internal/install/launchd.go", "internal/install/launchd_test.go", "internal/doctor/checks.go", "internal/doctor/checks_test.go"]
