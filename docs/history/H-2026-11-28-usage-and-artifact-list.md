---
id: H-2026-11-28-usage-and-artifact-list
title: The flags --usage and --artifact-list, and the checklist screens u and i
status: closed
adrs: [ADR-0005]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-3]
proposals: []
---

# The flags --usage and --artifact-list, and the checklist screens u and i

## Plan

The owner chose new flags and checklist screens, and no change to `--summary`. [ADR-0005](../adr/ADR-0005.md) records the rule. `internal/project` writes the lines, and `main.go` and `internal/ui` call it. Checks: no listener, no secret, no personal data, no extra `gh` call.

## Develop

`internal/gh` reads `expires_at`. `UsageLines` and `ArtifactLines` are new in `internal/project`. `u` and `i` open the report screen. Six test files name [ADR-0005](../adr/ADR-0005.md). r-go applied.

## Review

`make format`, `make lint`, `make test`, and `make integration` passed in host mode. `make editorconfig` and `make coverage` did not run, because `editorconfig-checker` and `diff-cover` are not installed in the cloud session. CI runs all six. `make links-check` passed. [ADR-0002](../adr/ADR-0002.md) has no test, as before.

## Release

Not invoked: the owner merges the pull request.
