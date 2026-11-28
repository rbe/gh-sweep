---
id: H-2026-11-28-repository-settings
title: The flags --retention-days and --actions-pr-approval
status: closed
adrs: [ADR-0006]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-4]
proposals: []
---

# The flags --retention-days and --actions-pr-approval

## Plan

The owner asked to set the artifact and log retention and "Allow GitHub Actions to create and approve pull requests" in one or many repositories. The owner's "check, workflow run, status" retention is the same setting as the artifact and log retention: the GitHub REST API description gives one endpoint for all five. [ADR-0006](../adr/ADR-0006.md) records the flags. `main.go` reads the flags and calls `internal/gh`, and `internal/project` writes the lines. Checks: security: no listener, no secret, an explicit target and `--dry-run` guard each write. privacy: no personal data. ux-ui-design: no screen. api-design: two new flags, recorded as an ADR. observability: each error names the repository on standard error, and the exit status is 1. performance: two reads and at most two writes for each repository.

## Develop

`internal/gh/settings.go` reads and writes the two endpoints. `internal/project/settings.go` holds the types and the lines. `main.go` adds the flags and `applySettings`. `test/mock-gh` answers the settings endpoints. Two test files name [ADR-0006](../adr/ADR-0006.md). r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`, after `make host-tools`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. `make integration` runs `true` in this project. The diff coverage is 85 percent: 149 of 174 changed lines. The missing lines are error branches of `gh` calls. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
