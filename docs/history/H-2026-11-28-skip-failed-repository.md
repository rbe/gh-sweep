---
id: H-2026-11-28-skip-failed-repository
title: A report skips a repository that fails and finishes
status: closed
adrs: [ADR-0007]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-5]
proposals: []
---

# A report skips a repository that fails and finishes

## Plan

The owner asked: "If one repository fails, mark it skipped and finish the report". Before, one error from `gh` stopped `--summary`, `--usage`, and `--artifact-list` with no output. [ADR-0007](../adr/ADR-0007.md) records the new rule. `main.go` collects the skipped repositories, and `internal/project/skipped.go` writes the skipped line. Declined: a skipped line on standard error only, because the report would not show the missing repository. Checks: security: no listener, no secret, no new write. privacy: the skipped line holds the repository name and the gh error, no personal data. ux-ui-design: no screen. api-design: the output of three flags gets a skipped line, recorded in the ADR. observability: each skipped repository names its error, and the exit status is 1. performance: no new call.

## Develop

`loadInventories` in `main.go` returns the read inventories and the skipped repositories. `printReport` puts the skipped lines before the total line. The old test `TestSummaryPrintsAnErrorFromGHAndFails` tested the old behaviour, and the owner asked to change that behaviour, so `skipped_test.go` replaces it. `test/mock-gh` gets `GH_MOCK_FAIL` and `GH_MOCK_CACHE_FAIL`. r-go applied.

## Review

`make gates` passed in host mode (`TOOLS=host`, after `make host-tools`): `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. `make integration` runs `true` in this project. The diff coverage is 100 percent: 25 of 25 changed lines. The checks of r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, and r-performance gave the lines of the Plan section. No call went to a real repository.

## Release

Not invoked: the owner merges the pull request.
