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

`test/run.sh` is deleted. It ran `go test ./...` on the host. `make test` runs the same tests in the Go container. D-6 states the rule that stays after K-1 is closed. [ADR-0001](../adr/ADR-0001.md) is not touched. [ADR-0002](../adr/ADR-0002.md) is kept, because `make test` runs through `$(CONTAINER_ENGINE)`.

## Review

Gates passed: `make lint`, `make test`, `make editorconfig`, `make format`, and `make integration`. Security: the change deletes one script. No listener and no secret. Finding, older than this change: [ADR-0001](../adr/ADR-0001.md) is not proven by any test. It waits as K-2.

## Release

The owner said to ship. The gates ran on this machine. The slice ships with the merge of `make-test-only` into `main`.
