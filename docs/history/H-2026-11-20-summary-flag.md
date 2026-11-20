---
id: H-2026-11-20-summary-flag
title: The flag --summary prints one line for each repository
status: started
adrs: [ADR-0003]
skills: [plan, security]
decisions: [D-7, D-8]
proposals: [P-1]
issues: []
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

Not invoked yet.

## Review

Not invoked yet.

## Release

Not invoked yet.
