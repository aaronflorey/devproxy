# Configuration

## Sources and precedence

DevProxy loads config in this order:

1. Defaults from `internal/config.DefaultConfig()`.
2. YAML from the path passed with `--config`.
3. Environment variables with the `DEVPROXY_` prefix.

Viper is used for loading, so later sources override earlier ones.

## Config file

Pass a YAML file with any command:

```bash
devproxy --config ./devproxy.yaml status
```

Example:

```yaml
domain_suffix: test
root_services:
  - app
  - web
serving:
  managed_suffix: test
  redirect_http_to_https: false
overrides:
  acme-crm:
    services:
      api:
        port: 3001
        scheme: http
```

## Environment variables

The repository only documents the prefix, not a custom parser, so use YAML for complex values unless you verify the env format in code.

Example:

```bash
export DEVPROXY_DOMAIN_SUFFIX=test
```

## Defaults

| Key | Default | Consumed by |
| --- | --- | --- |
| `domain_suffix` | `test` | install, daemon, doctor, routing |
| `root_services` | `app, web, nginx, laravel.test` | routing |
| `ignored_services` | `mysql, mariadb, postgres, redis, memcached, meilisearch, selenium` | discovery in the daemon |
| `ignored_ports` | `3306, 5432, 6379, 9200, 11211` | discovery in the daemon |
| `port_preference_order` | `443, 8443, 80, 8080, 8000, 3000, 5173, 8025` | stored in config; no runtime consumer found |
| `serving.managed_suffix` | `test` | DNS/proxy/doctor/runtime health |
| `serving.redirect_http_to_https` | `false` | stored in config; no runtime consumer found |
| `overrides` | `{}` | daemon reconciler |

## Validation and failure modes

- Missing or unreadable config path: `read config: ...`
- Invalid YAML or decode failure: `decode config: ...`
- `install` refuses to continue if `domain_suffix` is empty: `config domain_suffix is required`
- Dashboard listen addresses must be `127.0.0.1` or `localhost`.

## Config fields

### Top level

| Field | Type | Notes |
| --- | --- | --- |
| `domain_suffix` | string | Managed suffix for generated hostnames. |
| `root_services` | list[string] | Services that map to `project.suffix` instead of `service.project.suffix`. |
| `ignored_services` | list[string] | Services skipped by discovery. |
| `ignored_ports` | list[int] | Published ports skipped by discovery. |
| `port_preference_order` | list[int] | Present in config, but not currently consumed by runtime code. |
| `serving.managed_suffix` | string | Managed suffix used by DNS/proxy/runtime health. |
| `serving.redirect_http_to_https` | bool | Present in config, but not currently consumed by runtime code. |
| `overrides` | map | Project-specific service overrides. |

### Override fields

`overrides.<project>.services.<service>` supports:

| Field | Type | Notes |
| --- | --- | --- |
| `enable` | bool | Optional enable/disable override. |
| `domain` | string | Explicit hostname. |
| `domains` | list[string] | Additional explicit hostnames. |
| `root` | bool | Force root/non-root hostname shape. |
| `port` | int | Override the published port. |
| `scheme` | string | Override the upstream scheme. |
| `priority` | int | Override route priority. |

For routing behavior, see [architecture](./architecture.md) and [API](./api.md).
