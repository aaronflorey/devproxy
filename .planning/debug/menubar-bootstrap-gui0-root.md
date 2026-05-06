---
status: fixing
trigger: "Investigate and fix this installer bug in /home/aaron/Code/devproxy: Observed failure: sudo go run ./... install --with-menubar -> launchctl bootstrap gui/0 /var/root/Library/LaunchAgents/com.devproxy.menubar.plist failed: ... Domain does not support specified action"
created: 2026-05-06T00:00:00Z
updated: 2026-05-06T00:31:00Z
---

## Current Focus

reasoning_checkpoint:
  hypothesis: "Menubar install/start uses process user identity (root under sudo) for both plist path and launchctl GUI domain, causing bootstrap against gui/0 and /var/root/Library/LaunchAgents instead of the logged-in desktop user."
  confirming_evidence:
    - "DefaultPaths() builds UserLibraryDir from user.Current().HomeDir; under sudo this resolves to root home, matching observed /var/root/Library/LaunchAgents path."
    - "domainTarget() uses os.Getuid() for DomainAgent and returns gui/<uid>; under sudo this becomes gui/0, matching observed launchctl bootstrap gui/0 failure."
  falsification_test: "If menubar config is built with a non-root GUI UID/home and domainTarget uses that UID, install should no longer call gui/0 or /var/root path; if it still does, hypothesis is wrong."
  fix_rationale: "Resolve active GUI user explicitly for menubar role, wire resolved UID/home into menubar service config, and fail with clear guidance when no GUI desktop session is active. This changes only agent-domain/path resolution and keeps daemon system behavior unchanged."
  blind_spots: "Exact macOS edge-cases for console-owner detection across all login states are simulated via unit tests, not validated on a live host in this run."

hypothesis: confirmed root cause in menubar identity resolution under sudo.
test: implement GUI-user resolver + agent UID-based domain target; add tests for sudo/root failure mode and no-GUI-user explicit error.
expecting: with resolved GUI user, menubar plist path targets that user's Library and domain target is gui/<gui uid>; when unavailable, installer returns actionable message before launchctl bootstrap.
next_action: run targeted go tests for internal/install and cmd/devproxy install-related paths, then validate git diff and commit.

## Symptoms

expected: install --with-menubar should bootstrap menubar LaunchAgent in the active GUI user's domain/home.
actual: installer calls launchctl bootstrap gui/0 with plist under /var/root/Library/LaunchAgents and fails with domain does not support specified action.
errors: launchctl bootstrap gui/0 /var/root/Library/LaunchAgents/com.devproxy.menubar.plist failed: Domain does not support specified action
reproduction: run `sudo go run ./... install --with-menubar` on macOS with no gui/0 bootstrap support for root context.
started: unknown

## Eliminated

## Evidence

- timestamp: 2026-05-06T00:09:00Z
  checked: .planning/debug/knowledge-base.md
  found: file does not exist
  implication: no prior known-pattern match available; proceed with fresh investigation.

- timestamp: 2026-05-06T00:09:30Z
  checked: project skills directories
  found: no .claude/skills or .agents/skills entries in repository
  implication: no project-specific skill rules required for this fix.

- timestamp: 2026-05-06T00:14:00Z
  checked: internal/install/paths.go
  found: DefaultPaths derives UserLibraryDir from user.Current().HomeDir.
  implication: running installer with sudo/root points LaunchAgent plist to root home (e.g., /var/root/Library/LaunchAgents).

- timestamp: 2026-05-06T00:15:00Z
  checked: internal/install/launchd.go
  found: domainTarget returns gui/<os.Getuid()> for DomainAgent.
  implication: under sudo/root, launchctl bootstrap targets gui/0, which matches reported domain failure.

- timestamp: 2026-05-06T00:16:00Z
  checked: internal/install/install.go
  found: with --with-menubar, installer directly uses MenubarServiceConfig(DefaultPaths()) without GUI user resolution.
  implication: installer currently cannot target logged-in desktop user's home/domain when invoked as root.

- timestamp: 2026-05-06T00:30:00Z
  checked: implementation changes in internal/install
  found: added explicit ResolveGUIUser path for menubar install, menubar config now carries AgentUID and uses resolved GUI home for plist location.
  implication: installer no longer needs to infer menubar target from root runtime identity under sudo.

## Resolution

root_cause: menubar service installation used root process identity under sudo for both LaunchAgent plist path and launchctl GUI domain (DefaultPaths via user.Current + domainTarget via os.Getuid), leading to /var/root/Library/LaunchAgents and gui/0 bootstrap failures.
fix: added GUI-user resolution for menubar install (UID + home), passed UID into LaunchdServiceConfig for DomainAgent target, and built menubar plist path from resolved GUI home; installer now returns explicit actionable error if no GUI user session is resolvable.
verification: ""
files_changed:
  - internal/install/install.go
  - internal/install/launchd.go
  - internal/install/gui_user_darwin.go
  - internal/install/gui_user_stub.go
  - internal/install/uninstall.go
  - internal/install/install_test.go
  - internal/install/launchd_test.go
