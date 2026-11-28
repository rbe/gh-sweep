---
id: H-2026-11-28-source-layout
title: The Go files move from the repository root to cmd/gh-sweep and internal/cli
status: closed
adrs: [ADR-0016]
skills: [r-plan, r-develop, r-go, r-review]
roadmap: []
proposals: []
---

# The Go files move from the repository root to cmd/gh-sweep and internal/cli

## Plan

The owner asked to organize the codebase, because all Go files of the command line were at the root. [ADR-0016](../adr/ADR-0016.md) records the layout and the import direction.

## Develop

`main.go` is split into `internal/cli/cli.go`, `report.go`, `settings.go`, and `usage.go`. `jobs.go` and the 11 test files of package `main` move to `internal/cli`. `main_test.go` is now `cli_test.go`. `cmd/gh-sweep/main.go` calls `cli.Run`. The tests changed only the package name, `run` to `Run`, and the path to `test/mock-gh`. The `Makefile` and the `Containerfile` build `./cmd/gh-sweep`.

## Review

`make gates TOOLS=host` passed with all six gates in the cloud session. `make bin/gh-sweep TOOLS=host` built the binary, and `--help` printed the usage. The `Containerfile` build did not run, because the cloud session has no Podman. CI builds the image.

## Release

Not invoked: the owner merges the pull request.
