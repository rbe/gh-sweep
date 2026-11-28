---
id: H-2026-11-28-delete-caches-and-cache-usage
title: The flags --delete-caches and --cache-usage
status: closed
adrs: [ADR-0009]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-7]
proposals: []
---

# The flags --delete-caches and --cache-usage

## Plan

The owner asked to delete caches by key prefix and to show the cache usage and the cache retention and storage limits. [ADR-0009](../adr/ADR-0009.md) records the flags. `caches.go` in `main` holds the two jobs, `internal/gh/caches.go` reads the endpoints, and `internal/project/caches.go` selects the caches and writes the lines. A job gets a read-only mode and a total function. Declined: a delete by key, because it removes the key on every ref. Checks: security: no listener, no secret, an explicit target, a required filter, and `--dry-run` guard each delete. privacy: no personal data. ux-ui-design: no screen. api-design: three new flags, recorded as an ADR. observability: each error names the repository, and the exit status is 1. performance: one cache list for each repository, three reads for the usage.

## Develop

`jobs.go` gets `readOnly`, `total`, and the filter `--key-prefix`. `internal/project/job.go` gets `Total`. r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
