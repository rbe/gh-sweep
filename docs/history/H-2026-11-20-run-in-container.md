---
id: H-2026-11-20-run-in-container
title: make run runs the program in the container
status: closed
adrs: [ADR-0002]
skills: [r-plan, r-security, r-develop, r-shell, r-review, r-project-report]
decisions: [D-9]
proposals: []
issues: []
---

# make run runs the program in the container

## Plan

The slice: `make run ARGS='...'` runs gh-sweep in the container image. The token crosses from the host keyring by name, with `-e GH_TOKEN`. [ADR-0001](../adr/ADR-0001.md) keeps `internal/gh` as the one package that starts `gh`. D-9 records the choice.

## Develop

`make run` has the recipe of the old `make container`, and it passes the token by name. `make container` is removed. The recipe uses `$(CONTAINER_ENGINE)`, as [ADR-0002](../adr/ADR-0002.md) says. No Go file changed.

## Review

Gates passed in Podman, including `make lint`, `make test`, and `make links-check`. With no token, `make run` printed `No token in the keyring. Run: make auth`. Not run here: the real image, keyring, and terminal.

## Release

The owner ran `make auth`, `make run ARGS=--summary`, and a dry run, and all worked. The slice ships with the merge of `run-in-container` into `main`.
