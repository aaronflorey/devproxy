---
phase: 01-discovery-domains-and-conflict-policy
plan: 08
subsystem: cli
tags: [status, doctor, admin-projection]
requires:
  - phase: 05
    provides: snapshot-backed admin read models
provides:
  - reusable status conflict and warning detail
  - doctor CLI consumption of shared /doctor projection
affects: [dashboard, logs, operator-diagnostics]
tech-stack:
  added: []
  patterns: [single-source-of-truth-read-model]
key-files:
  created: [cmd/devproxy/status_test.go, cmd/devproxy/doctor_test.go]
  modified: [cmd/devproxy/status.go, cmd/devproxy/doctor.go, internal/admin/status.go, internal/admin/status_test.go]
key-decisions:
  - "Status and doctor render snapshot-derived conflict and warning detail instead of recomputing policy in the CLI layer."
requirements-completed: [DISC-06]
completed: 2026-05-10
---
# Phase 1 Plan 08: Conflict Visibility Summary

**`status` and `doctor` now expose the same snapshot-backed warning and conflict detail already used by other operator surfaces, including loser container names and warning messages.**

## Verification
- `go test ./cmd/devproxy/... ./internal/admin/... -run 'Test(Status|Doctor)' -count=1`
- `go test ./...`

## Deviations from Plan
None.

## Self-Check: PASSED
