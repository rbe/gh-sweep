---
id: H-2026-11-28-advice-and-filters
title: The advice report, the large-artifact mark, the report filters, and the workflow of an artifact
status: closed
adrs: [ADR-0018]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-15]
proposals: []
---

# The advice report, the large-artifact mark, the report filters, and the workflow of an artifact

## Plan

The owner asked for the ideas 2 to 6. Idea 2 was done in [H-2026-11-28-skip-failed-repository](H-2026-11-28-skip-failed-repository.md). The agent chose the details of the ideas 3 to 6 and wrote them in [ADR-0018](../adr/ADR-0018.md). Declined: a list of every workflow run, and advice lines after `--usage`.

## Develop

`internal/project` holds the limits, the advice, `ParseSize`, and the top and minimum filters. `internal/gh/workflows.go` reads the workflow names. `internal/cli` reads `--advice`, `--min-size`, and `--top`. `internal/ui` marks a large row. r-go applied.

## Review

`make gates TOOLS=host` passed with all six gates. The diff coverage is 100 percent, 256 of 256 lines. `make links-check` passed. The six check skills found no break. The image build did not run, because the cloud session has no Podman.

## Release

Not invoked: the owner merges the pull request.
