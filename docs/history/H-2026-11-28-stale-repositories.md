---
id: H-2026-11-28-stale-repositories
title: The flags --stale-list and --archive
status: closed
adrs: [ADR-0014]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-12]
proposals: []
---

# The flags --stale-list and --archive

## Plan

The owner asked to find stale repositories by `pushedAt` and `diskUsage` and to archive chosen ones. [ADR-0014](../adr/ADR-0014.md) records the flags. `stale.go` in `main` holds the jobs, `internal/gh/stale.go` reads the states and archives, and `internal/project/stale.go` sorts and writes the lines. Checks: security: no listener, no secret, named repositories only and `--dry-run` guard each archive. privacy: no personal data. ux-ui-design: no screen. api-design: two new flags, recorded as an ADR. observability: each error names the repository, and the exit status is 1. performance: one list call, or one view call for each named repository.

## Develop

`test/mock-gh` answers `gh repo list` and `gh repo view` with the state fields. r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
