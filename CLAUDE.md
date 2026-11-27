# gh-sweep

A terminal checklist that deletes GitHub Actions artifacts and caches.

`CLAUDE.md` is the only entry file. The skills are under `.claude/skills/`.

Stages, in order: `r-plan`, `r-develop`, `r-review`, `r-release`. A new or vague request starts at `r-plan`. A request that names a source file, a failing test, or a roadmap defect starts at `r-develop`. Other skills: `r-record`, `r-project-report`, `r-project-setup`, `r-onboard`, `r-migrate-docs`, `r-github-pipeline`, `r-system-state-report` (say `state report`), `r-board-merge-report`, `r-board-merge`, `r-skills-review`, `r-component-upgrade`. Language skills: `r-go`, `r-shell`, `r-jvm`, `r-node`, `r-python`. Subskills `r-plan` asks and `r-review` checks: `r-security`, `r-privacy`, `r-ux-ui-design`, `r-api-design`, `r-observability`, `r-performance`. A project skill is an extra name listed here. If `CLAUDE.md` lists a project skill `<project>-<stage>`, the stage reads it after its own steps. The addendum holds only what is true for that project. There is no project skill. The screen is built with Bubble Tea, Lip Gloss, and Bubbles.

`r-plan` names who owns state and what may call what. `r-develop` runs `make test-changed`. `r-review` runs `make gates`, which runs the six gates once. The linter is the one the language skill names. `make test` is the full suite. `make test-changed` runs the tests that cover the diff. `make coverage` lists the lines the change touched. `make project-report` writes the generated report. `make format-fix` rewrites the files with the formatter.

This project ships as a binary, with `make build` and `make install`, and as a container image, with a manual publish.

In-progress work is a row in `docs/ROADMAP.md`. A defect and an open question are rows there too. An architecture or development rule is an ADR in `docs/adr/`. Work for later is a file in `docs/proposals/`. A started slice is a file in `docs/history/`. A test names the ADR it proves with `id:`. An id in a document is a link. `make links` writes the links.

The general rules, the Agents section, and the communication contract stand once, in `rules.md` of `r-project-setup`. The next line imports that file. Do not copy it here.

@.claude/skills/r-project-setup/rules.md
