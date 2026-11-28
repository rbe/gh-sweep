---
id: H-2026-11-28-delete-workflow-runs
title: The flags --delete-runs and --delete-run-logs
status: closed
adrs: [ADR-0008]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-6]
proposals: []
---

# The flags --delete-runs and --delete-run-logs

## Plan

The owner asked to delete old workflow runs and their logs, with filters such as age, branch, and status. [ADR-0008](../adr/ADR-0008.md) records the write flags and the filter flags. `jobs.go` in `main` holds the rules that every write flag shares, `runs.go` holds the two run flags, `internal/gh/runs.go` calls the endpoints, and `internal/project/runs.go` selects the runs and writes the lines. Checks: security: no listener, no secret, an explicit target, a required filter, and `--dry-run` guard each delete. privacy: a run line holds the run name and the branch, no personal data. ux-ui-design: no screen. api-design: five new flags, recorded as an ADR. observability: each error names the repository on standard error, and the exit status is 1. performance: one paginated read for each repository and one delete for each run.

## Develop

`jobs.go` adds `registerJobs`, `choose`, `checkWriteTarget`, and `runJob`. `main.go` calls them. `test/mock-gh` answers a GET with a file from `GH_MOCK_DIR` and logs each write. r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The diff coverage of the lines of this slice is above 85 percent. The missing lines are error branches of `gh` calls and flag value errors. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
