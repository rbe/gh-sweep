# Decisions

One list. An Open row is a choice that waits for an answer. A Decided row is a choice in force, in one sentence.

| Id | Status | Choice |
| --- | --- | --- |
| D-1 | Decided | A size and a time are shown as short text a person reads, such as `1.5 MB` and `3d ago`. |
| D-2 | Decided | The inventory of a repository is read with the `gh` CLI, and its artifacts and caches are put into groups a person can select. |
| D-3 | Decided | A repository argument is parsed as `owner/name`. |
| D-4 | Decided | The checklist selects a repository and a group, and it deletes the largest entry first. |
| D-5 | Decided | With no repository argument, the tool lists the repositories of the logged-in user. |
| D-6 | Decided | The tests run with `make test`, in the Go container. The project has no shell-script wrapper for a Make target. |
| D-7 | Decided | `--summary` prints one line for each repository with its number of artifacts, its number of caches, and its size, then a total line. |
| D-8 | Decided | With no repository argument, `--summary` covers the repositories of the logged-in user, and it prints a line only for a repository that has an artifact or a cache. |
