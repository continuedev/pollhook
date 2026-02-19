---
name: pollhook
description: Set up pollhook to poll REST APIs and deliver webhooks to Continue's Mission Control. Helps agents configure polling sources for services without native webhook support.
metadata:
  author: continuedev
  version: "1.0.0"
---

# pollhook Setup

You are helping a user set up pollhook to bridge REST APIs into Continue's webhook system.

## Prerequisites

- Go 1.22+ installed
- A Continue Mission Control workflow with a webhook trigger (you'll need the workflow ID)
- API credentials for the service(s) to poll

## Step 1: Install pollhook

```bash
go install github.com/continuedev/pollhook@latest
```

Or build from source:

```bash
git clone https://github.com/continuedev/pollhook.git
cd pollhook
make build
```

## Step 2: Create config file

Create a `pollhook.yaml` with one or more sources. Each source needs:

- **name**: Unique identifier
- **command**: Shell command that outputs JSON (run via `sh -c`)
- **interval**: How often to poll (Go duration: `30s`, `5m`, `1h`)
- **items**: Dot path to the JSON array (`"."` for root, `"data.incidents"` for nested)
- **id**: Dot path to the unique ID field on each item
- **webhook.url**: The Mission Control ingest endpoint (`https://hub.continue.dev/api/webhooks/ingest/<workflow-id>`)
- **webhook.secret**: Optional secret sent as `X-Webhook-Secret` header

Example for Sentry:

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

Environment variables (`$SENTRY_TOKEN`) are expanded in the config file.

## Step 3: Test the config

Run a dry run to verify everything works:

```bash
pollhook test --config pollhook.yaml
```

This runs each command once, shows extracted items and what the webhook payloads would look like. No webhooks are sent and no state is touched.

## Step 4: Run in production

```bash
pollhook serve --config pollhook.yaml
```

State is persisted to `~/.pollhook/state.json` (customizable with `--state-dir`). The process handles SIGINT/SIGTERM gracefully.

For long-running deployment, use systemd, Docker, or a process manager:

```bash
# systemd example
pollhook serve --config /etc/pollhook/config.yaml --state-dir /var/lib/pollhook
```

## Verification

1. `pollhook version` prints a version string
2. `pollhook test --config pollhook.yaml` shows extracted items without errors
3. Check Mission Control for incoming webhook events after `pollhook serve` starts

## Troubleshooting

- **"command timed out after 60s"** — The API request is too slow. Check network connectivity or simplify the command.
- **"extract items: expected array"** — The `items` dot path doesn't point to a JSON array. Use `pollhook test` to inspect the raw output and adjust the path.
- **"webhook returned 401"** — The webhook secret doesn't match. Check the secret in your workflow configuration.
- **"webhook returned 404"** — The workflow ID in the URL is wrong. Verify it in Mission Control.
- **Items not detected as new** — pollhook deduplicates by ID. If the same IDs keep appearing, they've already been delivered. Delete `~/.pollhook/state.json` to reset.
