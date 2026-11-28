---
id: H-2026-11-28-ui-actions-all-flags
title: Every flag of the command line in the action menu of the checklist
status: closed
adrs: [ADR-0015]
skills: [r-plan, r-ux-ui-design, r-security, r-develop, r-go, r-review]
roadmap: [W-13]
proposals: []
---

# Every flag of the command line in the action menu of the checklist

## Plan

This is the second part of W-13 after [H-2026-11-28-ui-actions-runs-releases](H-2026-11-28-ui-actions-runs-releases.md). The menu gets every action of `internal/sweep`, and the two settings flags. [ADR-0015](../adr/ADR-0015.md) records the scope of the account actions.

## Develop

`internal/sweep/settings.go` holds `SetSetting` and `SettingEntries`, and `main.go` calls them. A read-only action shows its report after the read. A line with no write shows in the preview without a box, and in the result. The UI test of the retention found that `project.TotalLine` had no past tense for `change`, and `internal/project` now has it.

## Review

`make gates TOOLS=host` passed with all six gates in the cloud session. `make coverage` reports 83% of the changed lines as covered, counted against `main`. `make links-check` and `make ids` passed.

## Release

Not invoked: the owner merges the pull request.
