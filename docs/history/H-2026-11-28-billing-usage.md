---
id: H-2026-11-28-billing-usage
title: The flag --billing
status: closed
adrs: [ADR-0011]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-9]
proposals: []
---

# The flag --billing

## Plan

The owner asked for the billed Actions storage of the account, from `GET /users/{username}/settings/billing/usage/summary`. [ADR-0011](../adr/ADR-0011.md) records the flag. `billing.go` in `main` holds the job, `internal/gh/billing.go` reads the summary and the login, and `internal/project/billing.go` writes the lines. Declined: a filter on a storage SKU name, because the REST API description lists no SKU names. Checks: security: no listener, no secret, no write. privacy: the lines hold the account login and amounts of the owner, shown only to the owner. ux-ui-design: no screen. api-design: one new flag, recorded as an ADR. observability: an error names the account, and the exit status is 1. performance: two reads.

## Develop

`internal/project/job.go` gets the amounts in `Total`. `test/mock-gh` matches the path `organizations/`. r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to GitHub.

## Release

Not invoked: the owner merges the pull request.
