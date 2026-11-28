---
id: H-2026-11-28-deployments-and-environments
title: The flags --delete-deployments and --delete-unused-environments
status: closed
adrs: [ADR-0013]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-11]
proposals: []
---

# The flags --delete-deployments and --delete-unused-environments

## Plan

The owner asked to remove old deployments and unused environments. [ADR-0013](../adr/ADR-0013.md) records the flags. `deployments.go` in `main` holds the jobs, `internal/gh/deployments.go` calls the endpoints, and `internal/project/deployments.go` selects the entries. Checks: security: no listener, no secret, an explicit target and `--dry-run` guard each write, and the environment name is escaped. privacy: no personal data. ux-ui-design: no screen. api-design: two new flags, recorded as an ADR. observability: each error names the repository, and the exit status is 1. performance: one or two paginated reads for each repository, two writes for each deployment.

## Develop

r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
