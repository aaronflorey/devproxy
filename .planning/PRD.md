# PRD: DevProxy - macOS Docker Compose Vanity Domains

## 1. Summary

Build a Go CLI and macOS background agent that automatically creates local vanity domains for Docker Compose services without requiring changes to `docker-compose.yml`.

The tool watches Docker container lifecycle events, discovers containers with published ports, infers Compose project/service names, maps domains to running containers, serves local DNS, and proxies HTTP/HTTPS traffic to the correct published localhost port.

The product name for v1 is `devproxy`.

Example:

```bash
cd ~/Code/acme-crm
docker compose up
```

Automatically provides:

```text
https://acme-crm.test
https://api.acme-crm.test
https://mailpit.acme-crm.test
```

## 2. Goals

- Work with normal `docker compose up`.
- Require no Compose file modifications for common cases.
- Support optional Docker labels for overrides.
- Run natively on macOS.
- Support HTTP and HTTPS.
- Provide wildcard local DNS via a resolver + local DNS server.
- Provide a menu bar app showing active mappings.
- Handle Laravel Sail without extra config by mapping `laravel.test` to the project root domain and applying sensible defaults for common Sail companion services such as Mailpit when present.
- Prefer safe, predictable defaults by failing clearly and warning explicitly instead of silently changing ports, rewriting config, or inventing fallback behavior.
- Be easy to debug by surfacing warnings and conflicts consistently in `doctor`, `status`, the dashboard, and logs.

## 3. Non-goals

- Do not proxy containers without published ports in v1.
- Do not mutate `docker-compose.yml`.
- Do not require a proxy container.
- Do not replace Traefik/Caddy for production.
- Do not support Linux/Windows in v1.
- Do not support arbitrary TCP proxying in v1.
- Do not expose routes to the public internet.

## 4. Target users

- macOS developers using Docker Desktop.
- Laravel Sail users.
- Developers running multiple Compose projects locally.
- Developers who want nice local domains without editing `/etc/hosts` or Compose files.

## 5. Product shape

CLI binary:

```bash
devproxy install
devproxy uninstall
devproxy daemon
devproxy menubar
devproxy status
devproxy routes
devproxy refresh
devproxy doctor
devproxy logs
```

Background services:

```text
devproxy daemon

  - Docker event watcher
  - route registry
  - DNS server
  - HTTP server
  - HTTPS server
  - local API socket

devproxy menubar

  - macOS menu bar UI
  - reads state from daemon API

```

## 6. Architecture

```text
Docker Desktop
  -> Docker Engine API/events
    -> devproxy daemon
      -> Route discovery
      -> Route registry
      -> DNS server on 127.0.0.1:53535
      -> HTTP proxy on 127.0.0.1:80
      -> HTTPS proxy on 127.0.0.1:443
      -> Local admin API
        -> CLI
        -> Menu bar app
```

## 7. macOS installation

`devproxy install` should:

1. Create config directory:

```text
~/.config/devproxy/config.yml
```

2. Create state directory:

```text
~/.local/state/devproxy/
```

3. Install a LaunchAgent:

```text
~/Library/LaunchAgents/com.mochaka.devproxy.daemon.plist
```

4. Optionally install menu bar LaunchAgent:

```text
~/Library/LaunchAgents/com.mochaka.devproxy.menubar.plist
```

Install the menu bar LaunchAgent only when the user passes `--with-menubar`.

5. Configure resolver:

```text
/etc/resolver/test
```

Example:

```text
nameserver 127.0.0.1
port 53535
```

6. Install local CA/certs using `mkcert`.

7. Start services via `launchctl`.

## 8. DNS behavior

Default suffix:

```text
.test
```

Configurable:

```yaml
domain_suffix: test
```

DNS server behavior:

```text
*.test -> 127.0.0.1
```

Only the configured suffix should resolve. Any hostname under that suffix resolves to `127.0.0.1`. If the hostname does not match an active route, the proxy returns a friendly 404 explaining that no matching route exists.

## 9. Docker discovery

The daemon watches Docker events for:

```text
container start
container stop
container die
container destroy
container rename
container update
```

Docker documents `docker events` as a real-time event stream from the server. ([Docker Documentation][8])

On refresh/startup, the daemon should also inspect all currently running containers.

A container is routable when:

- it is running
- it has at least one published TCP port
- it is not explicitly disabled
- it is not ignored by service/container/image rules

`devproxy.enable=true` overrides ignore lists and makes the container eligible for routing as long as it otherwise has a usable published port.

If multiple routable containers claim the same domain, the container with the highest route priority wins. If priorities tie, devproxy applies deterministic tie-breaking by sorting on container name and selecting the first result, and logs the conflict and losing candidates.

## 10. Compose detection

Use labels when available:

```text
com.docker.compose.project
com.docker.compose.service
com.docker.compose.project.working_dir
```

Compose labels should be treated as the primary signal. If labels are unavailable, fall back to container name parsing.

When config and Docker labels both define the same route field, Docker labels win for that container route in v1.

## 11. Domain generation

Given:

```text
project = acme-crm
service = api
suffix = test
```

Default route:

```text
api.acme-crm.test
```

Root service names map to the project root:

```yaml
root_services:

  - app
  - web
  - nginx
  - laravel.test

```

So:

```text
app + acme-crm -> acme-crm.test
laravel.test + acme-crm -> acme-crm.test
```

If no root service is detected or configured, devproxy does not create the project root domain.

Laravel Sail example:

```text
service/container: laravel.test
project folder: my-shop
domain: my-shop.test
```

When common Sail companion services are present, devproxy should also generate their normal service subdomains using the standard routing rules, such as `mailpit.my-shop.test` for Mailpit.

Explicit domains from labels or config may use any local-only development suffix and are not limited to the default `domain_suffix`. DevProxy must reject obviously public internet suffixes such as `.com`, `.net`, and country-code public domains. If an explicit domain uses a suffix that devproxy DNS does not manage, devproxy still allows the route but must warn clearly that DNS for that suffix must be managed separately. Public internet hostnames are out of scope for v1.

## 12. Port selection

Priority:

1. `devproxy.port` label
2. configured service override
3. HTTPS-ish ports:

  - `443`
  - `8443`
4. HTTP-ish ports:

  - `80`
  - `8080`
  - `8000`
  - `3000`
  - `5173`
  - `8025`
5. first published TCP port

If the chosen published port has multiple host bindings, devproxy uses the first binding Docker reports.

Default ignored ports:

```yaml
ignored_ports:

  - 3306
  - 5432
  - 6379
  - 9200
  - 11211

```

Default ignored services:

```yaml
ignored_services:

  - mysql
  - mariadb
  - postgres
  - redis
  - memcached
  - meilisearch
  - selenium

```

## 13. Labels

Supported labels:

```yaml
devproxy.enable: "true"
devproxy.disable: "true"

devproxy.domain: "api.acme.test"
devproxy.domains: "api.acme.test,admin.acme.test"

devproxy.port: "8000"
devproxy.scheme: "http"
devproxy.root: "true"

devproxy.priority: "100"
```

Rules:

- `devproxy.disable=true` always wins.
- `devproxy.domain` replaces inferred domains.
- `devproxy.domains` adds multiple domains.
- `devproxy.root=true` maps service to `{project}.test`.
- `devproxy.port` must match the published host port on `127.0.0.1` that devproxy proxies to.
- `devproxy.scheme` controls upstream scheme, usually `http`.
- `devproxy.priority` is a numeric route priority. Higher values win when multiple containers claim the same domain. The default priority is `0`.
- Config may also define route overrides, but labels take precedence over config for the same field.
- Invalid label values such as malformed domains, non-numeric priorities, or unusable ports are ignored field-by-field with clear warnings in logs, `doctor`, and the dashboard when the route can still be reconciled.
- Duplicate explicit domains follow the same normal priority and deterministic tie-breaking rules as inferred domains.
- `devproxy.enable=true` can override ignored services and ignored ports, but it does not override hard requirements such as the need for a usable published port.

## 14. HTTPS

Use `mkcert`.

`devproxy install` fails if `mkcert` is not installed or the local CA cannot be set up. When `https.enabled=true`, the daemon also fails fast during startup or route discovery if required certificate prerequisites are missing.

Cert generation strategy:

For each project:

```text
project.test
*.project.test
```

Example:

```bash
mkcert acme-crm.test "*.acme-crm.test"
```

For explicit domains outside the default project wildcard pattern, devproxy generates certificates for each exact configured hostname.

DevProxy regenerates certificates when the set of hostnames served for a project changes. Otherwise it reuses existing valid local certificates.

Store certs in:

```text
~/.local/share/devproxy/certs/
```

The proxy should:

- serve HTTP on port 80
- serve HTTPS on port 443
- not redirect HTTP to HTTPS by default
- allow enabling redirect globally or per route

Config:

```yaml
https:
  enabled: true
  redirect_http: false
  cert_strategy: mkcert
```

## 15. Proxy behavior

For each incoming request:

1. Read `Host`.
2. Strip port from host.
3. Lookup route.
4. Reverse proxy to upstream:

```text
http://127.0.0.1:{published_port}
```

5. Preserve headers:

```text
Host
X-Forwarded-Host
X-Forwarded-Proto
X-Forwarded-For
X-Real-IP
```

For WebSockets, use Go reverse proxy support.

Unmapped route response:

```text
No devproxy route found for api.foo.test.
Run `devproxy routes` to see active mappings.
```

## 16. Menu bar app

Menu items:

```text
DevProxy

- Status: Running
- Routes
  - acme-crm.test -> localhost:8080
  - api.acme-crm.test -> localhost:3000
  - mailpit.acme-crm.test -> localhost:8025
- Refresh Routes
- Open Dashboard
- Open Logs
- Pause Routing
- Start at Login
- Doctor
- Quit Menu Bar

```

The menu bar app should not own routing. It should talk to the daemon over:

```text
unix socket: ~/.local/state/devproxy/devproxy.sock
```

`Open Dashboard` opens a local daemon-served status page in the browser that shows daemon health, active routes, recent conflicts, and recent errors from the current daemon session only.

The `Open route` action opens the HTTPS URL when HTTPS is enabled for that route; otherwise it opens the HTTP URL.

When routing is paused, DNS continues resolving normally, and any request for a hostname under a managed suffix returns a friendly paused response page instead of forwarding requests or returning the normal no-route 404.

## 17. Config file

Path:

```text
~/.config/devproxy/config.yml
```

Example:

```yaml
domain_suffix: test

dns:
  listen: 127.0.0.1:53535

proxy:
  http_addr: 127.0.0.1:80
  https_addr: 127.0.0.1:443
  redirect_http: false

docker:
  socket: unix:///var/run/docker.sock

root_services:

  - app
  - web
  - nginx
  - laravel.test

ignored_services:

  - mysql
  - mariadb
  - postgres
  - redis
  - memcached
  - meilisearch
  - selenium

port_preference:

  - 443
  - 8443
  - 80
  - 8080
  - 8000
  - 3000
  - 5173
  - 8025

projects:
  acme-crm:
    priority: 10
    root_domain: acme.test
    services:
      api:
        domain: api.acme.test
        port: 3000
        priority: 100
      mailpit:
        domain: mail.acme.test
        port: 8025
```

## 18. CLI commands

### `devproxy install`

Installs daemon, resolver, certs, and optionally menu bar. Installation fails with a clear error if required proxy ports cannot be bound or if HTTPS prerequisites are unavailable.

Options:

```bash
devproxy install --with-menubar
devproxy install --suffix test
devproxy install --dns-port 53535
```

### `devproxy daemon`

Runs the daemon in foreground. Startup fails with a clear error if configured listener ports such as `127.0.0.1:80` or `127.0.0.1:443` are unavailable.

### `devproxy status`

Shows daemon, DNS, HTTP proxy, and HTTPS proxy health, install state, and active route counts.

### `devproxy routes`

Shows active mappings:

```text
DOMAIN              PROJECT   SERVICE   UPSTREAM                SCHEME  STATE
acme-crm.test       acme-crm  app       http://127.0.0.1:8080  http    active
api.acme-crm.test   acme-crm  api       http://127.0.0.1:3000  http    active
```

### `devproxy refresh`

Re-inspects all running containers and rebuilds route table.

### `devproxy doctor`

Checks:

```text
Docker socket reachable
DNS server running
/etc/resolver/test configured
Port 80 available
Port 443 available
mkcert installed
Local CA installed
LaunchAgent loaded
Proxy reachable
Example domain resolves
```

### `devproxy logs`

Streams daemon logs from the current running session and may support follow mode. Persisted cross-restart log history is out of scope for v1.

### `devproxy uninstall`

Stops LaunchAgents and removes resolver config. It must interactively ask whether to keep or remove config, state, logs, and certificates instead of choosing a default deletion policy.

## 19. MVP milestones

### Milestone 1: Docker discovery

- Connect to Docker socket.
- List running containers.
- Inspect labels and published ports.
- Infer project/service/domain.
- Print route table.

### Milestone 2: Event watcher

- Watch start/stop/die/destroy events.
- Keep in-memory route table current.
- Add `devproxy routes`.

### Milestone 3: HTTP proxy

- Bind `127.0.0.1:80`.
- Route by Host header.
- Proxy to published localhost ports.

### Milestone 4: DNS

- Run DNS server on `127.0.0.1:53535`.
- Add installer for `/etc/resolver/test`.
- Resolve `*.test`.

### Milestone 5: HTTPS

- Integrate mkcert.
- Generate project certs.
- Serve HTTPS on `127.0.0.1:443`.
- Redirect HTTP to HTTPS.

### Milestone 6: macOS LaunchAgent

- Install daemon as LaunchAgent.
- Add logs.
- Add `status`, `doctor`, `uninstall`.

### Milestone 7: Menu bar

- Add systray menu.
- Show active routes.
- Refresh routes.
- Open route in browser.

## 20. Resolved decisions

- The product name and primary CLI command are `devproxy`.
- The default domain suffix is `.test`.
- HTTP-to-HTTPS redirect is disabled by default.
- Certificates are generated during route discovery, not lazily on first request.
- The menu bar app ships as a subcommand in the main binary.

[1]: https://docs.docker.com/reference/api/engine/sdk/?utm_source=chatgpt.com "Develop with Docker Engine SDKs"
[2]: https://github.com/miekg/dns?utm_source=chatgpt.com "miekg/dns: DNS library in Go"
[3]: https://github.com/filosottile/mkcert?utm_source=chatgpt.com "FiloSottile/mkcert: A simple zero-config tool to make locally ..."
[4]: https://github.com/getlantern/systray?utm_source=chatgpt.com "getlantern/systray: a cross platfrom Go library to place an ..."
[5]: https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingLaunchdJobs.html?utm_source=chatgpt.com "Creating Launch Daemons and Agents"
[6]: https://pkg.go.dev/github.com/caddyserver/certmagic?utm_source=chatgpt.com "certmagic package - github.com/caddyserver ..."
[7]: https://docs.docker.com/compose/compose-sdk/?utm_source=chatgpt.com "Using the Compose SDK"
[8]: https://docs.docker.com/reference/cli/docker/system/events/?utm_source=chatgpt.com "docker system events"

## Useful packages

Core:

| Area                | Package/tool                                         | Why                                                                                                                                        |
| ------------------- | ---------------------------------------------------- |------------------------------------------------------------------------------------------------------------------------------------------- |
| Docker API          | `github.com/moby/moby/client` / Docker Engine Go SDK | Watch events, inspect containers, read labels/ports. Docker documents Go SDK usage for Engine API integration. ([Docker Documentation][1]) |
| DNS server          | `github.com/miekg/dns`                               | Mature Go DNS server/client library. ([GitHub][2])                                                                                         |
| HTTP proxy          | Go stdlib `net/http/httputil.ReverseProxy`           | Enough for v1; avoids pulling in Caddy/Traefik complexity.                                                                                 |
| TLS certs           | `mkcert` integration                                 | Creates locally trusted development certificates and installs a local CA. ([GitHub][3])                                                    |
| Menu bar            | `github.com/getlantern/systray` or `fyne.io/systray` | Simple macOS menu bar/status item support. ([GitHub][4])                                                                                   |
| CLI                 | `github.com/spf13/cobra`                             | Standard Go CLI framework.                                                                                                                 |
| Config              | `github.com/spf13/viper`                             | YAML/env config loading.                                                                                                                   |
| Logging             | `log/slog`                                           | Built-in structured logging.                                                                                                               |
| LaunchAgent install | custom plist writer + `launchctl`                    | macOS uses `launchd`/`launchctl` to manage agents and daemons. ([Apple Developer][5])                                                      |

Potential later:

| Area            | Option                             | Notes                                                                                                                                                                                             |
| --------------- | ---------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| TLS automation  | `github.com/caddyserver/certmagic` | Powerful, but mostly aimed at ACME/public cert automation. Probably overkill for local mkcert-based certs. ([Go Packages][6])                                                                     |
| Native macOS UI | Swift wrapper                      | Better if the menu bar grows into a preferences app.                                                                                                                                              |
| Compose SDK     | `github.com/docker/compose`        | Useful later if you want to parse Compose files, but v1 can avoid it by using container labels/events. Docker has a Compose SDK for programmatic Compose integration. ([Docker Documentation][7]) |
| LaunchAgent     | `github.com/mishamyrt/go-lunch`    | Useful for abstracting launch agent control on macOS                                                                                                                                              |

---
