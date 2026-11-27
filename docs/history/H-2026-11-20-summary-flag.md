---
id: H-2026-11-20-summary-flag
title: The flag --summary prints one line for each repository
status: closed
adrs: [ADR-0003]
skills: [r-plan, r-security, r-develop, r-go, r-review, r-project-report, r-release]
decisions: [D-7, D-8]
proposals: [P-1]
issues: [K-3]
---

# The flag --summary prints one line for each repository

## Plan

The slice: `gh-sweep --summary` prints one line for each repository with its artifacts, caches, and size, then a total line. The confirm screen is [P-1](../proposals/P-1.md). The owner chose `--summary`, which [ADR-0003](../adr/ADR-0003.md) records. D-7 and D-8 are the Decided rows.

## Develop

`SummaryLines` in `internal/project/summary.go` is a pure function. `main.go` owns the flag. [ADR-0001](../adr/ADR-0001.md) keeps `internal/gh` as the one package that starts `gh`. [ADR-0002](../adr/ADR-0002.md) is a divergence.

## Review

Gates passed: `make lint`, `make test`, `make editorconfig`, `make format`, `make integration`, and `make coverage`. Security: no secret, no listener. Finding, older than this change: the Go `flag` package stops at the first non-flag argument. It waits as K-3. [ADR-0001](../adr/ADR-0001.md) waits as K-2.

## Release

The owner said do not merge. The slice ships with the merge of `summary-flag` into `main`, after the owner's review. [P-1](../proposals/P-1.md) stays open.
