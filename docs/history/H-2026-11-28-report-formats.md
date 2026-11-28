---
id: H-2026-11-28-report-formats
title: The reports write text, JSON, or CSV to standard output or to a file
status: started
adrs: [ADR-0017]
skills: [r-plan, r-security, r-privacy, r-ux-ui-design, r-api-design, r-observability, r-performance]
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

Not invoked yet.

## Review

Not invoked yet.

## Release

Not invoked yet.
