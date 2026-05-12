---
phase: 01-discovery-domains-and-conflict-policy
plan: 06
subsystem: daemon
tags: [docker-events, watcher, reconcile]
requires:
  - phase: 05
    provides: startup refresh and shared snapshot projections
provides:
  - live Docker lifecycle event subscription
  - reconnect-driven full resync before watcher health recovery
affects: [status, doctor, dashboard, logs]
tech-stack:
  added: []
  patterns: [single-snapshot-update-path, reconnect-before-healthy]
key-files:
  created: [internal/daemon/events_test.go]
  modified: [internal/daemon/app.go, internal/daemon/events.go, internal/daemon/docker_runtime.go, internal/daemon/app_test.go, internal/daemon/reconciler_test.go, cmd/devproxy/daemon.go]
key-decisions:
  - "Live Docker events trigger the existing scan-and-reconcile path instead of mutating snapshot state directly."
requirements-completed: [DISC-02]
completed: 2026-05-10
---
# Phase 1 Plan 06: Docker Event Watcher Summary

**The daemon now subscribes to the live Docker container event stream, marks watcher health disconnected on stream failures, and performs a full resync before reporting healthy again after reconnect.**

## Verification
- `go test ./internal/daemon/... -run 'Test(App|Watcher|Reconciler)' -count=1`
- `go test ./...`

## Deviations from Plan
None.

## Self-Check: PASSED
