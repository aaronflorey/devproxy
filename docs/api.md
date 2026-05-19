# Admin API

DevProxy exposes a local HTTP API over the Unix socket at `/tmp/devproxy/admin.sock` by default.

The CLI uses this socket for `status`, `routes`, `logs`, `doctor`, `refresh`, `dashboard`, and `menubar`.

## Endpoints

| Method | Path | Response |
| --- | --- | --- |
| `GET` | `/status` | `{ "status": StatusView }` |
| `GET` | `/routes` | `{ "routes": []RouteView }` |
| `GET` | `/doctor` | `{ "doctor": DoctorView }` |
| `GET` | `/logs` | `{ "events": []LogEvent }` |
| `GET` | `/issues` | `{ "issues": []SessionIssue }` |
| `POST` | `/refresh` | `{ accepted, refreshed, at, error? }` |
| `POST` | `/routing/pause` | `{ paused, error? }` |
| `POST` | `/routing/resume` | `{ paused, error? }` |
| `GET` | `/startup` | `{ "roles": []StartupRoleStatus }` |
| `POST` | `/startup` | `{ role, enabled } -> { role, enabled, affected_role, error? }` |

## Request shapes

### Refresh

```json
{ "reason": "operator refresh command" }
```

### Startup toggle

```json
{ "role": "menubar", "enabled": true }
```

Only `daemon` and `menubar` are accepted.

## Notes

- Responses are JSON.
- `refresh` returns `accepted: true` even when the daemon reports an error in the payload.
- The socket is local-only; it is not exposed on a TCP port.
