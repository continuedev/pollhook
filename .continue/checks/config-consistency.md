---
name: Config Consistency
on:
  - pull_request
---

# Config Consistency Check

Review changes to ensure the config format stays consistent and documented.

## What to check

1. **If `config.go` changes the `Source` or `Webhook` struct fields**: verify that `README.md` config documentation, `testdata/example.yaml`, and `skill/SKILL.md` are updated to reflect new/changed/removed fields.

2. **If `config.go` adds new validation rules**: verify that `pollhook_test.go` has test cases for the new validation (both positive and negative cases).

3. **If the webhook payload format changes in `webhook.go`**: verify that `README.md`'s "How It Works" section and the skill's config example are updated.

## What NOT to check

- Code style or formatting
- Test coverage percentages
- Import ordering
