---
id: H-2026-11-28-report-formats
title: The reports write text, JSON, or CSV to standard output or to a file
status: closed
adrs: [ADR-0017]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-14]
proposals: []
---

# The reports write text, JSON, or CSV to standard output or to a file

## Plan

The owner asked for idea 1, reports that a script can read, and for idea 7 in [W-2](../ROADMAP.md#w-2). The agent chose the output shape and wrote it in [ADR-0017](../adr/ADR-0017.md), and the owner merged it. Idea 7 went into [P-1](../proposals/P-1.md). Declined: the flags `--json` and `--csv`, and a total record in CSV.

## Develop

`internal/project/table.go` builds the rows and writes JSON and CSV. `internal/cli` reads `--format` and `--output`. The CSV column `result` replaced a second column `status`. r-go applied.

## Review

`make gates TOOLS=host` passed with all six gates. The diff coverage was 85 percent, 170 of 200 lines. `make links-check` passed. The six check skills found no break.

## Release

Not invoked: the owner merged pull request 33.
