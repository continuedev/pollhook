# Contributing to pollhook

Thanks for your interest in contributing to pollhook!

## Development Setup

```bash
git clone https://github.com/continuedev/pollhook.git
cd pollhook
make build   # Build the binary
make test    # Run tests
```

Requires Go 1.22+.

## Project Structure

```
main.go           # CLI entry: serve, test, version commands
config.go         # YAML config types, loader, validation
dotpath.go        # JSON dot-path extraction (items array + ID field)
state.go          # Seen-ID tracking with 10K cap, atomic file persistence
webhook.go        # HTTP POST delivery with retry
poller.go         # Core poll loop: exec → extract → dedup → deliver
pollhook_test.go  # Unit tests
testdata/         # Example configs
```

All files are in `package main`. No `internal/` — the codebase is intentionally flat.

## Making Changes

1. Fork the repo and create a branch from `main`
2. Make your changes
3. Add or update tests as needed
4. Run `make test` to verify
5. Commit using [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `docs:`, etc.)
6. Open a PR

## Conventional Commits

We use Conventional Commits for changelog generation:

- `feat: add PagerDuty example config` — new feature
- `fix: handle empty JSON arrays` — bug fix
- `docs: update config reference` — documentation only
- `refactor: simplify state persistence` — code change that neither fixes a bug nor adds a feature
- `test: add webhook retry tests` — adding or updating tests

## Reporting Bugs

Use the [bug report template](https://github.com/continuedev/pollhook/issues/new?template=bug_report.yml) to file issues. Include:

- Your pollhook version (`pollhook version`)
- Your Go version (`go version`)
- Config file (redact secrets)
- Command output / error messages
