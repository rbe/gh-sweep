---
id: H-2026-11-20-make-test-only
title: The tests run with make test only
status: closed
adrs: []
skills: [r-develop, r-review, r-shell, r-security]
decisions: [D-6]
proposals: []
issues: [K-1, K-2]
---

# The tests run with make test only

## Plan

`plan` was not invoked. The request names the known issue K-1 and the file `test/run.sh`.

## Develop

`test/run.sh` is deleted. It ran `go test ./...` on the host. `make test` runs the same tests in the Go container, and it is the one way to run them. The Tests section of `README.md` names `make test` and `make test-changed`. D-6 states the rule that stays after K-1 is closed.

The accepted ADRs for `develop` and `review` were read. [ADR-0001](../adr/ADR-0001.md) is not touched: no package starts a command. [ADR-0002](../adr/ADR-0002.md) is kept: `make test` runs through `$(CONTAINER_ENGINE)`.

No Go file changed, so the program has no behaviour change and the slice adds no unit test. `test/mock-gh` stays. The Go tests use it.

shell: the language skill for the deleted shell script. The skill says: do not add a shell-script wrapper.

`make lint` and `make test` passed. `make coverage` wrote `build/coverage/diff.md`. No Go line is in the diff.

## Review

`make lint`, `make test`, `make editorconfig`, `make format`, and `make integration` passed. `make project-report` wrote `build/project-report.md` with the requirements map and both reviews. The file was not hand-edited.

security: the change deletes one script and edits text. No listener and no secret. No input and no command of the program changed, so the checks on input and on the command do not apply.

The accepted ADRs for `develop` and `review` were read. The change breaks neither. [ADR-0002](../adr/ADR-0002.md) is a divergence, and it names the rule, what differs, and why.

`build/coverage/diff.md` has no line. The change touched no Go line.

Each test file names a Decided row. No test is skipped. No fixture has a secret or personal data.

Finding, older than this change: [ADR-0001](../adr/ADR-0001.md) says that only the package `internal/gh` starts the `gh` command. Location: `internal/`. Mechanism: no test fails when another package imports `os/exec`. Evidence: `build/project-report.md` lists [ADR-0001](../adr/ADR-0001.md) under ids with no test, and `os/exec` is imported only in `internal/gh/client.go` today. The check that proves a fix: a test that reads the imports of each package and names [ADR-0001](../adr/ADR-0001.md). The finding waits as K-2.

Left alone: [ADR-0002](../adr/ADR-0002.md) and D-6 have no test. Each is a rule about how to work, not behaviour of the program. `make image`, `make container`, and `make install` were not run. The slice does not change them.

## Release

The owner said to ship. The project has no workflow, so `make editorconfig`, `make format`, `make lint`, `make test`, `make integration`, and `make coverage` ran on this machine, and each passed. The slice ships with the merge of the branch `make-test-only` into `main`. The binary of the project is unchanged, and `make build` still writes it.
