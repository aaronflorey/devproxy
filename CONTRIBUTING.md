# Contributing

## Development Environment

This repository uses `mise` to pin local tooling.

```bash
mise install
```

## Common Commands

```bash
mise run fmt
mise run test
mise run build
```

## Project Notes

- Keep changes focused and avoid unrelated refactors
- Preserve the macOS-only runtime assumptions unless the task explicitly expands platform support
- Prefer small, explicit failures over implicit fallback behavior

## Commits

Release automation is driven by conventional commit prefixes. Prefer commit subjects such as:

- `feat: ...`
- `fix: ...`
- `docs: ...`
- `chore: ...`

## Pull Requests

- Include a short summary of the change
- Call out user-visible behavior changes
- Mention any manual verification performed for install, launchd, DNS, proxy, or menu bar flows
