---
status: resolved
trigger: "Debug and fix the remaining runtime issue revealed by output.log.\n\nObserved on real macOS run:\n- daemon is running via launchd\n- `/tmp/devproxy/admin.sock` is owned by the GUI user and accessible\n- doctor passes launchd/admin_socket/http_listener/https_listener/proxy_http/proxy_https (placeholder skipped)\n- `managed_domain_resolution` fails with:\n  `lookup example.test failed via devproxy DNS: read udp ... ->127.0.0.1:53535: read: connection refused`\n- resolver file and scutil state are correct for 127.0.0.1:53535\n\nLikely root cause to investigate:\n- daemon reports DNS healthy / bind address 127.0.0.1:53535, but no actual DNS UDP listener is started\n- or doctor is probing the wrong transport/address relative to the real DNS server implementation\n\nPlease:\n1. Inspect the daemon/network runtime and DNS server startup path.\n2. Implement the smallest correct fix so devproxy actually serves DNS on the configured local resolver port, or correct the health/reporting path if a listener already exists elsewhere.\n3. Add focused tests covering the bug.\n4. Run targeted tests for affected packages.\n5. Commit with a clear message.\n\nReturn:\n- root cause\n- files changed\n- tests run\n- commit hash."
created: 2026-05-06T23:35:19+00:00
updated: 2026-05-06T23:45:10+00:00
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "Managed-domain DNS fails because NetworkRuntime never binds/serves UDP DNS, so resolver queries to 127.0.0.1:53535 hit no listener."
  confirming_evidence:
    - "NetworkRuntime.Start in internal/daemon/network.go binds only HTTP/HTTPS listeners and starts only HTTP servers."
    - "No other daemon startup path creates a miekg/dns server or UDP listener, while health advertises DNS bind address 127.0.0.1:53535."
  falsification_test: "After adding DNS UDP listener startup, an integration-style test query to runtime DNS address should return 127.0.0.1 for managed host; if it still refuses or returns no answer, hypothesis is wrong/incomplete."
  fix_rationale: "Starting a UDP DNS server in NetworkRuntime ties actual runtime behavior to the resolver contract and removes the connection-refused failure at 127.0.0.1:53535."
  blind_spots: "Did not test launchd/macOS socket permissions end-to-end in this environment; verification relies on targeted runtime tests."

hypothesis: fix implemented by adding DNS UDP server lifecycle in network runtime.
test: run targeted tests for internal/daemon and related doctor package.
expecting: network runtime test proves DNS bind + managed-domain DNS response; affected package tests pass.
next_action: run go test ./internal/daemon ./internal/doctor.

## Symptoms

expected: managed domains like example.test resolve via local DNS listener at 127.0.0.1:53535.
actual: lookup example.test fails with UDP connection refused to 127.0.0.1:53535 while daemon otherwise appears healthy.
errors: lookup example.test failed via devproxy DNS: read udp ... ->127.0.0.1:53535: read: connection refused
reproduction: run daemon via launchd on macOS, run doctor checks, observe managed_domain_resolution failure.
started: after prior runtime fixes; remaining runtime issue from output.log session.

## Eliminated

- hypothesis: doctor probes wrong transport/address and daemon DNS is actually serving elsewhere.
  evidence: no DNS listener startup exists in daemon runtime; there is no alternate bind path to any DNS transport/address.
  timestamp: 2026-05-06T23:40:10+00:00

## Evidence

- timestamp: 2026-05-06T23:36:30+00:00
  checked: output.log managed_domain_resolution and socket checks
  found: doctor reports managed_domain_resolution fail with UDP connection refused to 127.0.0.1:53535 while other checks pass.
  implication: resolver configuration exists but no responder is listening on expected DNS UDP endpoint at probe time.

- timestamp: 2026-05-06T23:39:40+00:00
  checked: internal/daemon/network.go, internal/daemon/app.go, internal/dns/server.go, internal/doctor/checks.go
  found: NetworkRuntime.Start binds only HTTP/HTTPS TCP listeners; no DNS UDP server is started anywhere. DNS bind address is prefilled as 127.0.0.1:53535 in health.
  implication: root issue is missing DNS listener startup (not doctor transport mismatch).

## Resolution

root_cause: "NetworkRuntime advertised DNS health and the expected bind address, but never actually bound or served a UDP DNS listener."
fix: added DNS UDP listener startup/shutdown in NetworkRuntime and wired it to miekg/dns handler for managed suffix responses.
verification: "Targeted tests passed: go test ./internal/daemon ./internal/doctor"
files_changed: [internal/daemon/network.go, internal/daemon/network_test.go]
