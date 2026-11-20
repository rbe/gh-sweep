---
id: H-2026-11-20-summary-flag
title: The flag --summary prints one line for each repository
status: started
adrs: [ADR-0003]
skills: [plan, security, develop, go, review, project-report]
decisions: [D-7, D-8]
proposals: [P-1]
issues: [K-3]
---

# The flag --summary prints one line for each repository

## Plan

The slice: `gh-sweep --summary` prints one line for each repository with its number of artifacts, its number of caches, and its size, then a total line, and exits. It deletes nothing and opens no checklist.

Out of scope: the numbers on the confirm screen. That is proposal P-1. A machine format such as JSON. A change to `--dry-run`.

Questions and answers:

1. Question: `--dry-run` exists, and its result screen shows the number of artifacts and caches and the size. What is missing? Answer of the owner: the summary before the confirm, output without the screen, and a list for each repository.
2. Question: with no checklist the tool cannot ask what was picked. How is the output without the screen started, and what does it count? Options: a new flag `--summary`, `--dry-run` without a terminal, or `--dry-run` always prints. Recommended: the new flag. Answer of the owner: the new flag `--summary`. It counts everything in the named repositories. `--dry-run` keeps the checklist.
3. Question: the wish has three parts. Which part is this slice? Answer of the owner: `--summary` with one line for each repository. The confirm screen is a proposal for later.
4. Question: what does `--summary` do with no repository argument? Answer of the owner: it prints one line for each repository of the logged-in user that has an artifact or a cache, then the total. That matches D-5.

security: the questions were answered from the code and the board, because the slice adds no new boundary. What crosses a trust boundary: the `gh` command, as today. Who can call it: the person who runs the tool, with the login of `gh`. What happens when the check says no: `gh` fails, and the tool prints the error and exits with a failure. The bound on input: a repository argument is parsed as `owner/name`, as D-3 says. `--summary` takes no value.

Boundary: `main.go` owns the flag, the output stream, and the exit status. One new function builds the summary lines from the inventories. It owns no state. `internal/gh` stays the one package that starts the `gh` command, as ADR-0001 says. `internal/format` writes each size. `internal/ui` is not called when `--summary` is set.

Pattern: the summary is a pure function from the inventories to text. Loading and printing are at the edge, in `main.go`. The test gives the function inventories and reads the lines.

The choice of the flag is expensive to reverse. A flag is a public name of the tool, and a script can depend on it. The options are in answer 2. The owner chose `--summary`. ADR-0003 records it.

Each answer a test must prove is a Decided row: D-7 and D-8.

## Develop

`gh-sweep --summary` prints one line for each repository that has an artifact or a cache, then the total line, and exits. A line is `owner/name: artifacts 2, caches 1, size 3.5 MB`. The total line is `total: artifacts 2, caches 1, size 3.5 MB`.

`internal/project/summary.go` has the new function `SummaryLines`. It is the pure function Plan named: it reads the numbers of the inventories and returns the lines. It owns no state, and `internal/format` writes each size. The function is in `internal/project` because that package owns the inventory of a repository, and it does not import `os/exec`.

`main.go` owns the flag, the output stream, and the exit status. `run` returns the exit status for one command line. With `--summary` it calls `printSummary` and does not call `internal/ui`. `printSummary` checks the login, names the repositories, loads each inventory with `gh.Load`, and prints the lines. `internal/gh` stays the one package that starts the `gh` command.

The answers of the owner that Plan does not have: the output is plain text on standard output. The total line comes last. When no repository has an entry, the tool prints only the total line, with zeros. An error from `gh` is printed to standard error, and the exit status is not zero. `--summary` together with `--dry-run` behaves as `--summary`. There is no JSON.

An error from `gh` for a part of an inventory also stops the summary, because the numbers would be wrong. The tool then prints no line on standard output. When `gh` gives only the first 200 repositories, the tool says so on standard error, and the exit status is zero.

The accepted ADRs for `go`, `develop`, and `review` were read: ADR-0001, ADR-0002, and ADR-0003. The change breaks none.

The tests: `internal/project/summary_test.go` names D-7 and gives `SummaryLines` inventories. `main_test.go` names ADR-0003 and D-8. It runs `run` with `test/mock-gh` as the `gh` command and reads standard output, standard error, the exit status, and the list of `gh` calls.

The Usage section of `README.md` has the new part Summary. The usage text of the tool names `-summary`.

go: the language skill for the Go files. gofmt is the formatter and `go vet` is the linter.

`README.md` has no `language` row in `docs/skill-index.toon`. The file was edited as text, with the rules of `CLAUDE.md`.

`make lint` and `make test` passed. `make coverage` wrote `build/coverage/diff.md`. `make test-changed` ran the packages `.` and `./internal/project`.

## Review

`make lint`, `make test`, `make editorconfig`, `make format`, `make integration`, and `make coverage` passed. `make project-report` wrote `build/project-report.md` with the requirements map and both reviews. The file was not hand-edited. The map names `main_test.go` for ADR-0003 and D-8, and `internal/project/summary_test.go` for D-7.

project-report: `make project-report` wrote the report, and it has the three parts.

security: no secret is in the tree, a fixture, or an error text. The change adds no listener. A repository argument is parsed as `owner/name`, as D-3 says, and `--summary` takes no value. The program runs only the `gh` command, in `internal/gh`, and `--summary` sends no delete request. A test reads the list of `gh` calls and proves it.

The accepted ADRs for `go`, `develop`, and `review` were read. The change breaks none. `main.go` and `internal/project` do not import `os/exec`. ADR-0002 is a divergence, and it names the rule, what differs, and why.

`build/coverage/diff.md` lists the lines this change touched: `internal/project/summary.go` 100%, `main.go` 80.8%. The lines with no test in `main.go` are `main`, the call of the checklist, the failed login, a failed list of repositories, and the usage errors of `parseCLI`. Those usage errors are older than this change. The diff has them because the return type of `parseCLI` changed.

The pattern of Plan is kept. `SummaryLines` is a pure function from the inventories to text. Loading and printing are in `main.go`. Each new name is from the domain. Each test file names an ADR or a Decided row. No test is skipped. No catch is silent. No fixture has a secret or personal data.

Finding, older than this change: a flag after a repository argument is not read. Location: `parseCLI` in `main.go`, and the Usage section of `README.md`. Mechanism: the Go package `flag` stops at the first argument that is not a flag, so `gh-sweep owner/repo --dry-run` takes `--dry-run` as a repository name. Evidence: `run` with the arguments `acme/widgets --summary` did not print the summary. It went to the checklist and failed with `could not open a new TTY`. The README shows `gh-sweep owner/repo --dry-run`. The check that proves a fix: a test that gives `parseCLI` the arguments `acme/widgets --dry-run` and reads one repository and the dry run. The finding waits as K-3. The new part Summary of the README says to put `--summary` before the repository names.

Left alone: ADR-0001, ADR-0002, and D-6 have no test. ADR-0001 waits as K-2. ADR-0002 and D-6 are rules about how to work. `collect` in `internal/ui` and `summaryNames` in `main.go` both name the repositories from the options. They were not joined, because Plan says that `internal/ui` is not called when `--summary` is set. The summary loads one repository after the other. `--filter` has no effect on the summary. `make image`, `make container`, and `make install` were not run. The slice does not change them.

## Release

Not invoked yet.
