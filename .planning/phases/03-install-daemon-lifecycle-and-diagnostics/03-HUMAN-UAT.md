---
status: human_needed
phase: 03-install-daemon-lifecycle-and-diagnostics
source: [03-VERIFICATION.md]
started: 2026-05-06T00:00:00Z
updated: 2026-05-06T01:11:44Z
---

## Current Test

Pending full macOS UAT run in a privileged host environment.

## Tests

### 1. Install lifecycle (`devproxy install`)
expected: resolver/paths/services installed correctly; menubar installed only with `--with-menubar`
result: [issue]

```bash
# prereqs: macOS host, Docker Desktop running, mkcert installed, root privileges
sudo go run ./main.go install
sudo go run ./main.go install --with-menubar

# verify service + resolver artifacts
sudo launchctl print system/com.devproxy.daemon
sudo ls -la /etc/resolver
sudo ls -la /Library/LaunchDaemons/com.devproxy.daemon.plist
```

result

```bash
>go run ./... install --with-menubar
Error: devproxy install requires root privileges; rerun with sudo
Usage:
  devproxy install [flags]

Flags:
  -h, --help           help for install
      --with-menubar   also install optional menu bar LaunchAgent

Global Flags:
      --config string   path to config file

exit status 1

>sudo go run ./... install --with-menubar                        
Password:
Error: bootstrap certificates: mkcert not found: install mkcert before enabling HTTPS: exec: "mkcert": executable file not found in $PATH
Usage:
  devproxy install [flags]

Flags:
  -h, --help           help for install
      --with-menubar   also install optional menu bar LaunchAgent

Global Flags:
      --config string   path to config file

exit status 1
```


### 2. Doctor runtime diagnostics (`devproxy doctor`)
expected: accurate health output for docker/launchd/admin socket/listeners/scutil/mkcert/CA/domain resolution
result: [issue]

```bash
go run ./main.go doctor

# capture resolver evidence used by doctor
scutil --dns | grep -A6 "test"
```

result

```bash
>go run ./... doctor                                                                                                                                                                    
docker	ok	ok
launchd	fail	launchctl print system/com.devproxy.daemon failed: exit status 113: Bad request.
Could not find service "com.devproxy.daemon" in domain for system
admin_socket	fail	request /status: Get "http://unix/status": connect admin socket "/tmp/devproxy/admin.sock": dial unix /tmp/devproxy/admin.sock: connect: no such file or directory
resolver_state	fail	scutil --dns inspected
http_listener	fail	request /status: Get "http://unix/status": connect admin socket "/tmp/devproxy/admin.sock": dial unix /tmp/devproxy/admin.sock: connect: no such file or directory
https_listener	fail	request /status: Get "http://unix/status": connect admin socket "/tmp/devproxy/admin.sock": dial unix /tmp/devproxy/admin.sock: connect: no such file or directory
proxy_http	fail	cannot verify managed proxy reachability without daemon status: request /status: Get "http://unix/status": connect admin socket "/tmp/devproxy/admin.sock": dial unix /tmp/devproxy/admin.sock: connect: no such file or directory
proxy_https	fail	cannot verify managed proxy reachability without daemon status: request /status: Get "http://unix/status": connect admin socket "/tmp/devproxy/admin.sock": dial unix /tmp/devproxy/admin.sock: connect: no such file or directory
mkcert	fail	mkcert not found: exec: "mkcert": executable file not found in $PATH
local_ca	fail	mkcert local CA unavailable: exec: "mkcert": executable file not found in $PATH:
managed_domain_resolution	fail	lookup example.test failed: lookup example.test: no such host
```


### 3. Uninstall selective cleanup (`devproxy uninstall`)
expected: teardown first, then only selected cleanup categories removed
result: [pending]

```bash
# run once with mixed answers (example: keep config/logs, remove state/certs)
sudo go run ./main.go uninstall --with-menubar

# verify selected categories only
sudo launchctl print system/com.devproxy.daemon
sudo ls -la /usr/local/etc/devproxy
sudo ls -la /var/lib/devproxy
sudo ls -la /var/log/devproxy
```

result
```bash
sudo go run ./... uninstall                                                                                                                                                          
Password:
Remove config? [y/N]: Y
Remove state? [y/N]: y
Remove logs? [y/N]: y
Remove certificates? [y/N]: y
Error: stop daemon service: launchctl bootout system /Library/LaunchDaemons/com.devproxy.daemon.plist failed: exit status 5: Boot-out failed: 5: Input/output error
Usage:
  devproxy uninstall [flags]

Flags:
  -h, --help           help for uninstall
      --with-menubar   also unregister optional menu bar LaunchAgent

Global Flags:
      --config string   path to config file

exit status 1
```

## Summary

total: 3
passed: 0
issues: 3
pending: 0
skipped: 0
blocked: 0

## Gaps

- None yet. Awaiting human execution evidence from macOS host.
