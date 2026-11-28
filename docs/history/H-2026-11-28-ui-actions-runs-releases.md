---
id: H-2026-11-28-ui-actions-runs-releases
title: The action menu of the checklist, with workflow runs and releases
status: closed
adrs: [ADR-0015]
skills: [r-plan, r-ux-ui-design, r-security, r-develop, r-go, r-review]
roadmap: [W-13]
proposals: []
---

# The action menu of the checklist, with workflow runs and releases

## Plan

The owner asked for every CLI action and filter in the checklist. The work has two pull requests. This one moves the actions into `internal/sweep` and adds the action menu with `--delete-runs`, `--delete-run-logs`, `--delete-releases`, and `--delete-release-assets`. The second adds the other flags. [ADR-0015](../adr/ADR-0015.md) records the rule. `internal/sweep` owns the actions. `main.go` and `internal/ui` call it, and `internal/sweep` calls `internal/gh` and `internal/project`.

## Develop

The old job code of package `main` is gone. The CLI tests pass with no change. `internal/ui/actions.go` is the new screen. The tests in `internal/ui/actions_test.go` name [ADR-0015](../adr/ADR-0015.md) and replace the read of GitHub with fixed entries.

## Review

`make gates TOOLS=host` passed with all six gates in the cloud session. `make coverage` reports 80% of the 814 changed lines as covered. `make links-check` and `make ids` passed.

## Release

Not invoked: the owner merges the pull request.
