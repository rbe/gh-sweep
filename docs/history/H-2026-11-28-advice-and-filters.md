---
id: H-2026-11-28-advice-and-filters
title: The advice report, the large-artifact mark, the report filters, and the workflow of an artifact
status: started
adrs: [ADR-0018]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go]
roadmap: [W-15]
proposals: []
---

# The advice report, the large-artifact mark, the report filters, and the workflow of an artifact

## Plan

The owner asked: "Implement all other ideas 2-6, maintain board". Idea 2, a report that skips a failed repository, was already done in [H-2026-11-28-skip-failed-repository](H-2026-11-28-skip-failed-repository.md). This slice builds idea 3, an advice report, idea 4, a mark for large artifacts, idea 5, the filters `--min-size` and `--top`, and idea 6, the workflow of each artifact.

The slice in one sentence: `--advice` prints numbered advice, `--artifact-list` shows the workflow and the large mark and takes `--min-size`, `--usage` takes `--top`, and the checklist draws a large size in the warning color. Out of scope: the workflow in the checklist, advice in the checklist, and `--cache-usage`, `--billing`, and `--stale-list`.

The owner gave no answer on the details. The agent chose them and wrote them in [ADR-0018](../adr/ADR-0018.md): the output, the empty case, and the error case. The owner reviews that ADR in the pull request.

Boundary and pattern: `internal/project` owns the limits, the advice, the filters, and the lines and rows. `internal/gh` reads the workflow names. `internal/cli` reads the flags and calls both. `internal/ui` reads the large limit of `internal/project`. The slice adds no package and no import.

Declined: a list of every workflow run, because a repository can have many thousands of runs. Declined: advice lines after `--usage`, because a script reads `--usage`.

security: the new inputs are two numbers and a size. Each is checked before the first gh call. No listener, no secret.
privacy: the workflow name is repository data, not personal data.
ux-ui-design: the checklist draws the size of a large artifact in the existing warning color. No new key.
api-design: not HTTP. Not a module. The public interface is the flags `--advice`, `--min-size`, and `--top`, and the new keys of [ADR-0018](../adr/ADR-0018.md).
observability: a wrong value exits with status 2 and an error with the prefix `gh-sweep: `. A failed workflow read skips the repository with its error.
performance: `--artifact-list` makes one gh call for each distinct workflow run of the listed artifacts. Every other report makes no new call.

## Develop

`internal/project` gets `limits.go` with `LargeArtifactBytes`, `OldArtifactDays`, `Item.Large`, `Item.Old`, and `WithMinSize`, `advice.go` with `AdviceLines` and `AdviceTable`, and `size.go` with `ParseSize`. `TopUsageLines` and `TopUsageTable` cut the usage to the top repositories. The artifact line and row get the workflow and the large mark. `Item` gets the field `Workflow`. `internal/gh/workflows.go` adds `AddWorkflows`. `internal/cli` reads `--advice`, `--min-size`, and `--top`, and `chooseReport` now takes `reportFlags`. `internal/ui` draws the part `large` of a row with `styleMeta`. `ParseSize` went to `internal/project`, because [ADR-0016](../adr/ADR-0016.md) does not let `internal/cli` import `internal/format`. Three old tests of `--artifact-list` now expect the part `workflow`. The README and the help text name the new flags. r-go applied. `make test-changed TOOLS=host` passed for `./internal/cli`, `./internal/gh`, `./internal/project`, and `./internal/ui`. `r-review` runs the gates.

## Review

Not invoked yet.

## Release

Not invoked yet.
