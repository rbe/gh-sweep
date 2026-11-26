# Pipeline rules

The rules of the pipeline stand once, in the skills `r-project-setup` (the Make contract) and `r-github-pipeline` (the GitHub contract, the publish rules with `AUTO_PUBLISH`, versions, cleanup, and retention). This file does not repeat them. It holds only the facts of this repository and its divergences.

## This repository

This repository is gh-sweep, a Go terminal program. It ships as a binary with `make install` and as a container image on GHCR.

| | |
| --- | --- |
| Image | `ghcr.io/rbe/gh-sweep` |
| Default branch | `main` |
| Toolchain | Go 1.22 in the image `docker.io/library/golang:1.22-bookworm`, run through Podman |

## Divergences

Each entry names the shared rule, what differs, and why. A toolchain difference belongs here.

- Rule: Podman is the only container engine. Here: the project Makefile runs Go and `make run` through `$(CONTAINER_ENGINE)`, which is Podman, or Docker when Podman is missing. Why: the tool is for any person with a GitHub account. The accepted [ADR-0002](adr/ADR-0002.md) states this. The targets of `container.mk` and `sdlc.mk` call `podman`.
- Rule: The image copies named files and holds only the tested build. Here: the final image also holds the GitHub CLI `gh`, because the program starts it. Why: [ADR-0001](adr/ADR-0001.md) says that only `internal/gh` reaches GitHub, through `gh`.
- Rule: `publish-github` releases a JAR or a binary. Here: there is no `publish-github` job. The binary ships with `make install`. Why: no skill target releases a Go binary.
- Rule: There is no `docs` job. Here: the project renders no docs.
