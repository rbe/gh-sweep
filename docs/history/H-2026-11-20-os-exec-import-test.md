---
id: H-2026-11-20-os-exec-import-test
title: A test proves that only internal/gh imports os/exec
status: started
adrs: [ADR-0001]
skills: [develop, go, review, security, project-report]
decisions: []
proposals: []
issues: [K-2]
---

# A test proves that only internal/gh imports os/exec

## Plan

`plan` was not invoked. The request names the known issue K-2.

## Develop

`internal/gh/os_exec_import_test.go` is new. It names ADR-0001 with `id:`. `TestOnlyInternalGHImportsOSExec` reads the imports of each Go file of the project and fails when a package other than `internal/gh` imports `os/exec`. `TestPackageOutsideInternalGHThatImportsOSExecIsNamed` builds a small tree in which `internal/ui` imports `os/exec`, and it proves that the check names that package. The check reads each package of the project, so it also covers the package `main` and a package added later. It does not read a hidden directory, `apm_modules/`, `build/`, or `bin/`.

The accepted ADRs for `develop` and `go` were read. ADR-0001 is kept: the test starts no command and imports no `os/exec`. ADR-0002 is not touched: the Makefile did not change.

go: the language skill for the Go test file. The test file names its id in a comment in the file header.

No file of the program changed, so the program has no behaviour change. The test is in the package `internal/gh`, the package that ADR-0001 names. A first version of the file was in the package `main`. The coverage map did not list that package, and `make test-changed` printed `test-changed: no test covers the changed files. make test runs the full suite.` With the file in `internal/gh`, `make test-changed` printed `test-changed: ./internal/gh`.

`make format`, `make lint`, and `make test` passed. `make coverage` wrote `build/coverage/diff.md`. It has no line, because a Go test file has no coverage information.

## Review

`make lint`, `make test`, `make editorconfig`, `make format`, `make integration`, and `make ids` passed.

security: no secret is in the tree, a log, an error, or a fixture. The change adds no listener. The program accepts no new input. The test reads only the Go files of the project and of a temporary directory. The program runs no new command, and the test now proves that only `internal/gh` can start one.

The accepted ADRs for `develop`, `review`, and `go` were read. The change breaks neither. ADR-0002 is a divergence, and it names the rule, what differs, and why.

project-report: `make project-report` wrote `build/project-report.md` with the requirements map and both reviews. The file was not hand-edited. The map names `internal/gh/os_exec_import_test.go` as the test of ADR-0001. The ids with no test are ADR-0002 and D-6.

`build/coverage/diff.md` says: `No lines with coverage information in this diff.` The change touched no line of the program.

Each test file names an ADR or a Decided row. No id is in a path. No test is skipped. No error is ignored. No fixture has a secret or personal data. Each function of the new file does one thing or only composes other functions, and each name is from the domain of ADR-0001.

No finding in this change.

Left alone: ADR-0002 and D-6 have no test. Each is a rule about how to work, not behaviour of the program. The test checks the import of `os/exec`. It does not check that no package uses a GitHub client library, the second part of ADR-0001. `go.mod` names no such library today. `make build`, `make image`, `make container`, and `make install` were not run. The slice does not change them.

## Release

Not invoked yet.
