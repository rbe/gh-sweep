---
id: H-2026-11-20-os-exec-import-test
title: A test proves that only internal/gh imports os/exec
status: closed
adrs: [ADR-0001]
skills: [r-develop, r-go, r-review, r-security, r-project-report, r-release]
decisions: []
proposals: []
issues: [K-2]
---

# A test proves that only internal/gh imports os/exec

## Plan

`plan` was not invoked. The request names the known issue K-2.

## Develop

`internal/gh/os_exec_import_test.go` is new and names [ADR-0001](../adr/ADR-0001.md) with `id:`. The test fails when a package other than `internal/gh` imports `os/exec`. A second test proves the check names the package. No program file changed. [ADR-0002](../adr/ADR-0002.md) is not touched.

## Review

Gates passed: `make lint`, `make test`, `make editorconfig`, `make format`, `make integration`, and `make ids`. `make project-report` names the new test for [ADR-0001](../adr/ADR-0001.md). Security: no secret, no listener, and no new input. Left alone: [ADR-0002](../adr/ADR-0002.md) and D-6 have no test. No finding in this change.

## Release

The owner said do not merge. The slice ships with the merge of `os-exec-import-test` into `main`, which the owner does after review. K-2 is deleted.
