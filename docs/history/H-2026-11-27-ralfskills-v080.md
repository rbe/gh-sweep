---
id: H-2026-11-27-ralfskills-v080
title: Move the project to RalfSkills commit 4c028a8
status: closed
adrs: []
skills: [r-plan, r-develop, r-go, r-review, r-release]
roadmap: []
proposals: []
---

# Move the project to RalfSkills commit 4c028a8

## Plan

The owner asked to move the project to RalfSkills commit 4c028a8, the commit that becomes v0.8.0. `apm.yml` pins the commit. The four `r-upgrade-*-components` skills are gone, and their facts are in `references/upgrade.md` of the language skills.

## Develop

`apm.yml` names the commit and no longer lists the four upgrade skills. APM reinstalled `.claude/skills`, and `apm.lock.yaml` names the commit. `CLAUDE.md` no longer names the four skills. The project has no docs renderer and no `workflows.test.mjs`. `docs/pipeline-rules.md` needed no change.

## Review

`make old-board`, `make toon`, `make links`, `make ids`, and `make links-check` ran. The gate results are in the report of the task. No test changed.

## Release

Not invoked: the owner merges the pull request.
