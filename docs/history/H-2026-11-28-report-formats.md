---
id: H-2026-11-28-report-formats
title: The reports write text, JSON, or CSV to standard output or to a file
status: started
adrs: [ADR-0017]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance, r-develop, r-go, r-review, r-release]
roadmap: [W-14]
proposals: []
---

# The reports write text, JSON, or CSV to standard output or to a file

## Plan

The owner asked: "Extend [W-2](../ROADMAP.md#w-2) with idea 7. Implement idea 1." Idea 1 is machine-readable output for the reports, after the `--format` and `--output` flags of github-artifacts-analyzer. Idea 7 is the grouped cleanup with a size for each group. It went into [P-1](../proposals/P-1.md) and the title of [W-2](../ROADMAP.md#w-2), with its open questions there, and this slice does not build it.

The slice in one sentence: `--format text|json|csv` and `--output <file>` write the report of `--summary`, `--usage`, or `--artifact-list` in that format to standard output or to the file. Out of scope: `--cache-usage`, `--billing`, `--stale-list`, the screens `u` and `i`, and the confirm screen of [W-2](../ROADMAP.md#w-2).

The owner gave no answer on the shape of the output, because the owner was away. The agent chose it and wrote it in [ADR-0017](../adr/ADR-0017.md): the result, the empty case, and the error case. The owner reviews that ADR in the pull request before the merge.

Boundary and pattern: `internal/project` owns the rows of each report and writes them as JSON or CSV. `internal/cli` reads the flags, loads the inventories, and writes to standard output or to the file. The slice adds no package and no import.

Declined: the flags `--json` and `--csv`, because one flag with a value needs no check for two flags. Declined: a total record in CSV, because a script that sums the rows would count it twice.

security: the new input is a file path from the person who runs the tool, and the tool writes that file with mode 0644 and runs no command with it. No listener, no secret.
privacy: the output holds repository names, artifact names, ids, sizes, dates, and gh errors, the same data as the text report. No personal data is added.
ux-ui-design: no screen changes. The help text gets two lines.
api-design: not HTTP. Not a module. The public interface is the two flags and the JSON and CSV shapes of [ADR-0017](../adr/ADR-0017.md).
observability: errors keep the prefix `gh-sweep: `. A wrong value or a missing report flag exits with status 2. A file that cannot be written exits with status 1.
performance: no new gh call. The report is built in memory once, as before.

## Develop

`internal/project/table.go` adds the type `Table`, its methods `WriteJSON` and `WriteCSV`, and the functions `SummaryTable`, `UsageTable`, and `ArtifactTable`. They reuse `usedRepositories` and `artifactsBySize`, so the rows keep the order of the text lines. `internal/cli/report.go` adds `reportOutput`, `checkReportOutput`, `writeReport`, and `renderReport`. `internal/cli/cli.go` reads `--format` and `--output`, and `internal/cli/usage.go` gets their two help lines. The README gets the section "Report formats and files". During Develop the CSV header of the artifact list had the column `status` twice, so the extra column became `result`, and [ADR-0017](../adr/ADR-0017.md) says so. r-go applied. `make test-changed TOOLS=host` passed for `./internal/cli` and `./internal/project`. `make coverage TOOLS=host` reports 170 of 200 changed lines covered. The 30 missing lines are error returns of `json.Marshal`, `json.Indent`, and writes to a `bytes.Buffer`, which these values cannot make fail. `r-review` runs the gates.

## Review

`make gates TOOLS=host` passed in the cloud session with all six gates: `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage`. `make integration` runs `true` in this project. The diff coverage is 85 percent: 170 of 200 changed lines. `make links-check` passed. `make project-report` lists [ADR-0002](../adr/ADR-0002.md) as the only id with no test. [ADR-0002](../adr/ADR-0002.md) is a rule about how to work, so it needs no test. Every new test names [ADR-0017](../adr/ADR-0017.md) in its `id:` line. The CI workflow runs only `make` targets.

security: the only new input is the path of `--output`. The tool writes that file with mode 0644 after the read and runs no command with it. No secret is in the diff.
privacy: the JSON and the CSV hold the same data as the text report. No personal data is added.
ux-ui-design: no screen changed. The help text has two new lines.
api-design: not HTTP, not a module. The flags and the keys match [ADR-0017](../adr/ADR-0017.md), and tests read the exact JSON of the summary and the exact CSV of the usage and the artifact list.
observability: an unknown format and a missing report flag exit with status 2. A file that cannot be written exits with status 1. Errors keep the prefix `gh-sweep: `.
performance: no new gh call. The report is built once in memory.

Finding, fixed in this change: the CSV header of the artifact list had the column `status` twice. The extra column is now `result`.

Left alone: `--cache-usage`, `--billing`, `--stale-list`, and the screens `u` and `i` keep text only. Roadmap row [W-1](../ROADMAP.md#w-1) still holds: a flag after a repository name is not read, so the README puts `--format` and `--output` before the names.

## Release

Not invoked: the owner merges the pull request.
