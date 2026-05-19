# Testing

## Standard checks

The repository defines these `mise` tasks in `mise.toml`:

```bash
mise run fmt
mise run test
mise run build
```

## What the checks do

| Command | Effect |
| --- | --- |
| `mise run fmt` | Fails if `gofmt -l .` reports any files. |
| `mise run test` | Runs `go test ./...`. |
| `mise run build` | Runs `go build .`. |

## CI parity

The GitHub Actions CI job on macOS-latest runs:

```bash
test -z "$(gofmt -l .)"
go test ./...
go build .
```

## Manual end-to-end validation

The repository includes `test.sh`, which exercises the full install/uninstall path, waits for the admin socket, runs `doctor`, inspects launchd, and captures daemon logs.

`run.sh` is a simpler ad hoc install helper that builds the binary, copies it to `/usr/local/bin/devproxy`, and runs uninstall/install with `--with-menubar`.
