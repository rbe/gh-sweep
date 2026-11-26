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

The slice: `make run ARGS='...'` runs gh-sweep in the container image. The host reads the token from the keyring, and the container gets it as `GH_TOKEN`.

Out of scope: the image itself. `make install` and the binary on the host. A change to a Go file.

Questions and answers:

1. Question of the owner: can the token from the keyring of the host go to a containerized run? Answer: yes. `make container` did it. `make run` started a binary on the host instead, and the `go` skill says that `make run` runs the program in the container.
2. Question: one target or two? Options: `make run` takes the recipe of `make container` and `make container` goes away, or `make container` stays as a second name. Recommended: one target. Answer of the owner: do the slice as recommended.

security: What crosses a trust boundary: the token, from the keyring of the host into the container. Who can call it: the person who runs `make run`, with the login of `gh` on the host. What happens when the check says no: with no token the target prints `No token in the keyring. Run: make auth` and fails. With a token GitHub refuses, `gh` fails and the tool prints the error. The bound on input: `ARGS` is what the person types, and the program parses it as before. The token is given by name, with `-e GH_TOKEN`, so the value is not an argument of the engine.

Boundary: the `Makefile` owns how the program starts in development. The program does not change. `internal/gh` stays the one package that starts the `gh` command, as [ADR-0001](../adr/ADR-0001.md) says.

Pattern: one way to run. The recipe reads the secret on the host and hands it over by name.

D-9 records the choice.

## Develop

`make run` has the recipe of the old `make container`, with three changes. The token is exported and passed with `-e GH_TOKEN`. Before, the value was in the arguments of the engine, where another process of the machine can read it. The run gets `-it` only when Make has a terminal, so `--summary` works in a pipe. The recipe is not printed.

`make container` is removed. `README.md`, the comment in `Containerfile`, and `make notes` name `make run`. `bin/gh-sweep` stays the binary of the host, for `make install`.

No Go file changed, so the slice adds no unit test. D-9 is a rule about how to work, not behaviour of the program.

shell: the language skill for the recipe. The recipe is one POSIX shell, joined with backslashes.

`make lint` and `make test` passed. `make coverage` wrote `build/coverage/diff.md`. No Go line is in the diff.

## Review

`make lint`, `make test`, `make editorconfig`, `make format`, `make integration`, `make ids`, `make toon`, and `make links-check` passed. `make project-report` wrote `build/project-report.md`. The file was not hand-edited.

The recipe ran with a stand-in image. The machine of this session cannot install `gh` with apt, so `make image` fails here at that step. The stand-in has the same build stage and the same entry point, and `test/mock-gh` behind a check of `GH_TOKEN` as its `gh` command. With the right token, `make run ARGS='--summary acme/widgets'` printed the line of the repository and the total line. With a wrong token, the tool printed that `gh` is not logged in. With no token and no `gh`, the target printed `No token in the keyring. Run: make auth`. `make -n run` shows `-e GH_TOKEN` with no value.

security: the token is not in the tree, a log, an error, or a fixture. It is not an argument of the engine. No listener. The command the recipe runs is the one the slice named. Input: `ARGS`, as before.

The accepted ADRs for `develop` and `review` were read. The change breaks neither. The recipe uses `$(CONTAINER_ENGINE)`, as [ADR-0002](../adr/ADR-0002.md) says.

No finding. Left alone: the real image, the real keyring, and the screen in a terminal were not run here. The owner runs `make auth` and `make run` once on their machine.

## Release

The owner ran `make auth`, `make run ARGS=--summary`, and a dry run on their machine, with the real image, the real keyring, and a terminal. All worked. The owner said to ship. The project has no workflow, so the Make targets of the Review section ran on the machine of this session. The slice ships with the merge of the branch `run-in-container` into `main`. The binary and the image of the project are unchanged.
