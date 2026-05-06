---
status: verifying
trigger: "Investigate and fix Phase install failure: sudo go run ./... install --with-menubar -> launchctl bootstrap system ... Input/output error"
created: 2026-05-06T00:00:00Z
updated: 2026-05-06T00:28:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "launchctl bootstrap fails because launchd plist Program is fixed to /usr/local/bin/devproxy, and install did not ensure that path exists with the currently running binary when invoked via go run"
  confirming_evidence:
    - "DaemonServiceConfig hardcodes Program to /usr/local/bin/devproxy"
    - "Installer flow had no pre-bootstrap step to stage/validate this binary path"
  falsification_test: "if installer stages executable to /usr/local/bin/devproxy before bootstrap and bootstrap failure persists with valid binary, hypothesis is wrong"
  fix_rationale: "staging current executable to the exact launchd Program path removes missing/stale binary mismatch and gives explicit install-time errors if staging cannot be done"
  blind_spots: "cannot run real macOS launchctl bootstrap in this Linux CI environment"

hypothesis: installer must stage current executable to /usr/local/bin/devproxy before launchd bootstrap.
test: run targeted install/internal and cmd/devproxy tests covering staging success and staging failure messaging.
expecting: tests pass and failure path now returns actionable staging error instead of opaque launchctl bootstrap I/O message.
next_action: commit minimal code/test changes for staging fix and report results.

## Symptoms

expected: install should succeed and bootstrap daemon service reliably.
actual: install fails at launchctl bootstrap with Input/output error.
errors: start daemon service: launchctl bootstrap system /Library/LaunchDaemons/com.devproxy.daemon.plist ... Input/output error
reproduction: run `sudo go run ./... install --with-menubar`
started: when running install via go run without guaranteed /usr/local/bin/devproxy binary.

## Eliminated

## Evidence

- timestamp: 2026-05-06T00:00:00Z
  checked: user symptom report
  found: bootstrap failure occurs after install invocation through go run
  implication: pre-bootstrap daemon executable path may be invalid or stale

- timestamp: 2026-05-06T00:08:00Z
  checked: internal/install/launchd.go DaemonServiceConfig
  found: launchd daemon Program is hardcoded to /usr/local/bin/devproxy
  implication: bootstrap depends on that absolute path existing and being executable

- timestamp: 2026-05-06T00:09:00Z
  checked: internal/install/install.go installer flow
  found: install performs paths/resolver/certs/service install/start without staging or validating /usr/local/bin/devproxy
  implication: go run execution can produce invalid Program target causing launchctl bootstrap failure

- timestamp: 2026-05-06T00:19:00Z
  checked: internal/install patch
  found: added PrepareDaemonBinary pre-bootstrap hook with default stageCurrentExecutable copy to /usr/local/bin/devproxy and explicit error wrapping
  implication: install now proactively satisfies launchd Program path or fails early with actionable context

- timestamp: 2026-05-06T00:27:00Z
  checked: go test ./internal/install ./cmd/devproxy
  found: targeted tests pass, including new staging failure-mode test
  implication: fix is validated for installer flow and command package behavior

## Resolution

root_cause: "Installer bootstrapped launchd services without ensuring hardcoded Program path (/usr/local/bin/devproxy) existed/current, so go run installs could hand launchd a missing/stale executable and fail with Input/output error."
fix: "Added pre-bootstrap daemon executable staging step that copies current executable to /usr/local/bin/devproxy and aborts with explicit staging error context if that step fails."
verification: "Targeted tests passed: go test ./internal/install ./cmd/devproxy"
files_changed: ["internal/install/install.go", "internal/install/launchd.go", "internal/install/install_test.go"]
