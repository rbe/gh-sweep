# gh-sweep

A terminal checklist that deletes GitHub Actions artifacts and caches.

`CLAUDE.md` is the only entry file. The skills are installed under `.claude/skills/` with APM.

Stages, in order: `plan`, `develop`, `review`, `release`. A new or vague request starts at `plan`. A request that names a source file, a failing test, or a known issue starts at `develop`. `record` writes every id and every board document. `project-report` generates the requirements-to-tests map and the architecture and development reviews. `project-setup` is the Make contract.

Language skills: `go`, `shell`. Subskills `plan` asks and `review` checks: `security`. A project skill is an extra name listed here. There is no project skill. The screen is built with Bubble Tea, Lip Gloss, and Bubbles.

`plan` names who owns state and what may call what. `develop` is not done until `make lint` and `make test` pass. The linter is the one the language skill names. `make test` is the full suite. `make test-changed` runs the tests that cover the diff. `make coverage` lists the lines the change touched. `make project-report` writes the generated report. `make format-fix` rewrites the files with the formatter.

This project ships as a binary, with `make build` and `make install`, and as a container image, with `make image`.

In-progress work is a line in `docs/ROADMAP.md`. A defect is a row in `docs/KNOWN_ISSUES.md`. A small choice is a row in `docs/DECISIONS.md`. An architecture or development rule is an ADR in `docs/adr/`. Work for later is a file in `docs/proposals/`. A started slice is a file in `docs/history/`. An accepted ADR is a rule for every change. A test names the ADR or the decision it proves with `id:`. Run `make toon` before reading an index. `make decision-log` writes the decisions as one document.

If `docs/BUGS.md`, `docs/WAVES.md`, `docs/FEATURES.md`, `docs/bugs/`, `docs/waves/`, or `docs/features/` exists, the only skill is `migrate-docs`.

The communication contract uses the names and concepts already in the project. Do not invent a synonym for a term the project already uses. The first sentence to the user states the understood intent, in one sentence. Stop when that sentence is wrong.

One idea per sentence. Be clear and on the point. English follows ASD-STE100. German follows "Deutsch für die Technische Kommunikation" (tekom Deutschland e.V.). Name the standards. Do not copy their text. Keep German and English separate.
