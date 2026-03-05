<p align="center">
  <a href="https://continue.dev">
    <img src=".github/assets/continue-banner.png" width="800" alt="Continue" />
  </a>
</p>

<h1 align="center">pollhook</h1>

<p align="center">Poll REST APIs, deliver webhooks. Bridge any API into Continue's event system.</p>

<p align="center"><em>An autonomous codebase built by the <a href="https://continue.dev/blueprint">Continue Software Factory</a></em></p>

---

## Why?

Many services (Sentry, PagerDuty, Snyk, etc.) have REST APIs but no outgoing webhooks. Setting up custom integrations for each one means writing bespoke glue code, managing state, and handling retries — over and over.

**pollhook** does one thing: run a command on an interval, detect new items via ID-based dedup, and POST them to a webhook endpoint. One YAML file replaces all that glue.

## Table of Contents

- [Quick Start](#quick-start)
- [Config Format](#config-format)
- [Commands](#commands)
- [How It Works](#how-it-works)
- [Design Decisions](#design-decisions)
- [Contributing](#contributing)
- [License](#license)

## Quick Start

```bash
# Build from source
git clone https://github.com/continuedev/pollhook.git
cd pollhook
make build

# Or install directly
go install github.com/continuedev/pollhook@latest
```

Create a `pollhook.yaml`:

```yaml
sources:
  - name: sentry-issues
    command: |
      curl -s -H "Authorization: Bearer $SENTRY_TOKEN" \
        https://sentry.io/api/0/projects/my-org/my-project/issues/?query=is:unresolved
    interval: 5m
    items: "."
    id: "id"
    webhook:
      url: https://hub.continue.dev/api/webhooks/ingest/your-workflow-id
      secret: your-webhook-secret
```

```bash
# Dry run — validate config, show extracted items
pollhook test --config pollhook.yaml

# Run for real
pollhook serve --config pollhook.yaml
```

## Config Format

```yaml
sources:
  - name: sentry-issues          # Unique name for this source
    command: |                    # Shell command (run via sh -c) that outputs JSON
      curl -s -H "Authorization: Bearer $SENTRY_TOKEN" \
        https://sentry.io/api/0/projects/my-org/my-project/issues/
    interval: 5m                 # Poll interval (Go duration: 30s, 2m, 1h)
    items: "."                   # Dot path to the JSON array ("." = root)
    id: "id"                     # Dot path to unique ID on each item
    webhook:
      url: https://...           # Webhook endpoint URL
      secret: optional           # Sent as X-Webhook-Secret header
```

**Dot paths:** `"."` means the root is the array. `"data.incidents"` navigates `$.data.incidents`. Environment variables in the config are expanded via `$VAR` or `${VAR}`.

## Commands

### `pollhook serve`

Run pollers and deliver webhooks.

```bash
pollhook serve --config pollhook.yaml [--state-dir ~/.pollhook]
```

- Starts a goroutine per source with `time.Ticker`
- Polls immediately on startup, then on interval
- State persisted to `~/.pollhook/state.json` every 30s and on shutdown
- Graceful shutdown on SIGINT/SIGTERM

### `pollhook test`

Validate config, run each command once, show extracted items. No webhooks sent, no state touched.

```bash
pollhook test --config pollhook.yaml
```

### `pollhook version`

Print the version.

## How It Works

For each source, a goroutine runs on a timer:

1. **Execute command** — `sh -c <command>` with 60s timeout, capture stdout
2. **Extract items** — parse JSON, navigate dot path to array
3. **Extract ID** — get unique ID from each item via dot path
4. **Dedup** — skip if ID already seen (checked against persisted state)
5. **Deliver webhook** — POST `{"source": "name", "item": {...}, "polled_at": "..."}` to the webhook URL
6. **Mark seen** — only after successful delivery (at-least-once guarantee)

The webhook payload matches what Continue's `/api/webhooks/ingest/:workflowId` endpoint expects — any JSON body that gets stringified into the workflow prompt.

## Design Decisions

| Decision | Choice | Why |
|---|---|---|
| Dependencies | Only `gopkg.in/yaml.v3` | Everything else is stdlib |
| CLI framework | `flag.FlagSet` | 2 commands don't justify cobra |
| Change detection | ID-based dedup | Deterministic, order-independent, per-item |
| State cap | 10,000 IDs per source | Sliding window prevents unbounded growth |
| Delivery guarantee | At-least-once | Failed delivery = ID stays unseen = retried next tick |
| Retry | 1 retry on 5xx (2s delay) | 4xx = config error, 5xx = transient |
| Command execution | `sh -c` | Lets users write piped multi-line commands |
| Webhook secret | `X-Webhook-Secret` header | Matches ingest endpoint's timing-safe comparison |

## Contributing

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for development setup and guidelines.

## License

Apache-2.0 — see [LICENSE](LICENSE) for details.
Copyright (c) 2025 Continue Dev, Inc.
