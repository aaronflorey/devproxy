---
status: investigating
trigger: "Dashboard says it can't connect to the daemon; start/stop works; there are no logs; daemon does not pick up Docker containers."
created: 2026-05-07T12:00:00Z
updated: 2026-05-07T12:00:00Z
---

## Current Focus

hypothesis: Two bugs combine to cause all symptoms:
1. `listRunningContainerIDs` uses `cmd.CombinedOutput()` which captures stderr. If Docker outputs any warning/deprecation to stderr, it gets parsed as a container ID, causing `docker inspect` to fail on the invalid ID. This makes the Docker scan fail and the daemon crash-loop via launchd KeepAlive.
2. The dashboard handler shows "can't connect to daemon" when there are 0 routes even when Status() succeeded, misleading the user into thinking the daemon is unreachable.

test: Run targeted tests on docker_runtime, daemon, and dashboard packages; verify `cmd.Output()` fix doesn't break container ID parsing; verify dashboard no longer shows misleading error for 0-route case.

expecting: All targeted tests pass; Docker scan correctly handles containers even if Docker outputs to stderr; dashboard distinguishes between "daemon unreachable" and "no routes."

next_action: Apply fixes to internal/daemon/docker_runtime.go and internal/dashboard/server.go

reasoning_checkpoint:
  hypothesis: "stderr contamination from docker ps combined output causes scan failure; misleading dashboard error for 0-route case"
  confirming_evidence:
    - "listRunningContainerIDs at docker_runtime.go:112 uses CombinedOutput which mixes stderr into container ID list"
    - "dashboard/server.go:145 sets DaemonError on NoActiveRoutes even when Status() succeeded"
  falsification_test: "If docker ps never outputs to stderr on the target macOS environment AND the user has containers with published ports, this hypothesis is wrong"
  fix_rationale: "Use Output() instead of CombinedOutput() to isolate stdout (container IDs) from stderr (warnings); remove misleading DaemonError assignment when Status succeeds"
  blind_spots: "Cannot reproduce macOS-specific Docker stderr behavior in this Linux test environment"

## Symptoms

expected: Daemon detects running Docker containers with published ports, dashboard shows routes and logs.
actual: Dashboard shows "DevProxy can't reach the daemon right now", no routes, no logs.
errors: Dashboard displays errDaemonUnreachable message.
reproduction: Start daemon via launchd on macOS with running Docker containers; open dashboard.
started: after daemon start

## Eliminated

## Evidence

- timestamp: 2026-05-07T12:00:00Z
  checked: internal/daemon/docker_runtime.go:listRunningContainerIDs line 112
  found: Uses cmd.CombinedOutput() which captures both stdout and stderr from docker ps
  implication: If Docker outputs any stderr (warnings, deprecation notices), it contaminates container ID list and causes docker inspect to fail

- timestamp: 2026-05-07T12:05:00Z
  checked: internal/daemon/app.go:refreshFromDocker
  found: When DockerScan returns error, watcher.OnDisconnect() is called and error propagates to Start(), causing daemon to exit
  implication: With launchd KeepAlive=true, daemon crash-restarts endlessly while Docker outputs stderr warnings

- timestamp: 2026-05-07T12:10:00Z
  checked: internal/dashboard/server.go:handleDashboard line 145
  found: DaemonError is set to errDaemonUnreachable when data.NoActiveRoutes is true, even though the Status() call succeeded
  implication: Dashboard misleadingly reports "can't connect to daemon" when daemon IS reachable but has 0 routes

- timestamp: 2026-05-07T12:15:00Z
  checked: internal/daemon/docker_runtime.go:DefaultDockerScan and reconciler with real 28-containers test
  found: Docker scan and reconciler work correctly in clean environment (no stderr from docker ps)
  implication: Core route generation logic is correct; issue is specifically stderr contamination in container ID parsing

- timestamp: 2026-05-07T12:20:00Z
  checked: git log for recent changes
  found: Commit c0d4b1c added DockerScan to daemon command and refreshFromDocker to Start(); previously DockerScan was nil and Refresh() was a no-op clearing routes
  implication: Fix was recently applied; prior builds would have had broken Docker scanning entirely

## Resolution

root_cause: "Six bugs found across three investigation phases. Phase 1 (Docker scan): listRunningContainerIDs used CombinedOutput() capturing stderr, contaminating container ID list when Docker outputs warnings; dashboard showed misleading 'daemon unreachable' for 0-route case. Phase 2 (menubar/dashboard wiring): offlineMenuState dead code always used generic message ignoring actual errors; dashboard handler swallowed actual connection errors; menubar periodic refresh used _ = stateInfo preventing recovery from offline. Phase 3 (uninstall/cert crash): prepareStoredCertificates made mkcert issuance FATAL, crashing daemon when root lacked CAROOT env var; uninstall didn't attempt menubar removal without --with-menubar flag and had no progress output."
fix: "(1) docker_runtime: CombinedOutput→Output for stderr isolation. (2) dashboard: show actual errors, descriptive 0-route message. (3) menubar: show actual errors, remove dead code in refresh goroutine. (4) network: certificate issuance failures are non-fatal—daemon starts without HTTPS certs. (5) uninstall: always attempt menubar removal best-effort, add progress output matching install style."
verification: "go test ./... — 16/16 packages pass. Real Docker scan: 28 containers→7 routes. Git diff: 8 files, +60/-20 lines."
files_changed: ["internal/daemon/docker_runtime.go", "internal/daemon/network.go", "internal/dashboard/server.go", "internal/menubar/app.go", "internal/menubar/runtime_darwin.go", "internal/menubar/app_test.go", "internal/install/uninstall.go", "cmd/devproxy/uninstall.go"]
