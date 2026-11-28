---
id: H-2026-11-28-preferences
title: The preferences screen and the preferences file
status: closed
adrs: [ADR-0019]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-16]
proposals: []
---

# The preferences screen and the preferences file

## Plan

The owner asked that the values and retention times are configurable in the checklist. The agent chose seven values, the key `p`, and a JSON file, and wrote them in [ADR-0019](../adr/ADR-0019.md). Declined: a flag for each value.

## Develop

`internal/project/preferences.go` holds the values, the checks, and the file. `internal/ui/preferences.go` is the screen. The prompts, the retention field, and the typed confirmation read the preferences. One test of the retention field now clears the default 90 first. r-go applied.

## Review

`make gates TOOLS=host` passed with all six gates. `make links-check` passed. security: the file holds no secret and gets mode 0600. privacy: no personal data. ux-ui-design: one new screen, one new key `p`. The image build did not run, because the cloud session has no Podman.

## Release

Not invoked: the owner merges the pull request.
