---
id: H-2026-11-28-releases-packages-codespaces
title: The flags for releases, package versions, and codespaces
status: closed
adrs: [ADR-0010]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-8]
proposals: []
---

# The flags for releases, package versions, and codespaces

## Plan

The owner asked to remove old package versions, such as untagged container images, releases and release assets, and stopped codespaces. [ADR-0010](../adr/ADR-0010.md) records the four flags. `releases.go` in `main` holds the jobs, `internal/gh/releases.go` calls the endpoints, and `internal/project` selects the entries and writes the lines. A job gets an account mode for the flags that do not run in each repository. Declined: a guard against platform versions of a multi-platform image, because it needs the container registry. Checks: security: no listener, no secret, an explicit target, a required filter, and `--dry-run` guard each delete. privacy: a codespace line holds the codespace name and the repository, no personal data. ux-ui-design: no screen. api-design: four write flags and four filter flags, recorded as an ADR. observability: each error names the repository, the owner, or the codespace, and the exit status is 1. performance: one paginated read for each repository or package.

## Develop

`jobs.go` gets the fields `account` and `target`, and `--package`, `--package-type`, `--keep`, and `--untagged` work. r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
