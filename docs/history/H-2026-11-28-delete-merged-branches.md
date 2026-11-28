---
id: H-2026-11-28-delete-merged-branches
title: The flag --delete-merged-branches
status: closed
adrs: [ADR-0012]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-10]
proposals: []
---

# The flag --delete-merged-branches

## Plan

The owner asked to delete branches whose pull request is merged or closed. [ADR-0012](../adr/ADR-0012.md) records the flag and the cases where a branch stays. `branches.go` in `main` holds the job, `internal/gh/branches.go` calls the endpoints, and `internal/project/branches.go` selects the branches. Checks: security: no listener, no secret, an explicit target and `--dry-run` guard each delete, and the branch name is escaped. privacy: a line holds the branch name and the pull request number, no personal data. ux-ui-design: no screen. api-design: one new flag, recorded as an ADR. observability: each error names the repository, and the exit status is 1. performance: three paginated reads for each repository; the pull request list grows with the repository.

## Develop

r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
