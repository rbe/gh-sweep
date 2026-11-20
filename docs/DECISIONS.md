# Decisions

One list. An Open row is a choice that waits for an answer. A Decided row is a choice in force, in one sentence.

| Id | Status | Choice |
| --- | --- | --- |
| <a id="d-1"></a>D-1 | Decided | A size and a time are shown as short text a person reads, such as `1.5 MB` and `3d ago`. |
| <a id="d-2"></a>D-2 | Decided | The inventory of a repository is read with the `gh` CLI, and its artifacts and caches are put into groups a person can select. |
| <a id="d-3"></a>D-3 | Decided | A repository argument is parsed as `owner/name`. |
| <a id="d-4"></a>D-4 | Decided | The checklist selects a repository and a group, and it deletes the largest entry first. |
| <a id="d-5"></a>D-5 | Decided | With no repository argument, the tool lists the repositories of the logged-in user. |
| <a id="d-6"></a>D-6 | Decided | The tests run with `make test`, in the Go container. The project has no shell-script wrapper for a Make target. |
| <a id="d-7"></a>D-7 | Decided | `--summary` prints one line for each repository with its number of artifacts, its number of caches, and its size, then a total line. |
| <a id="d-8"></a>D-8 | Decided | With no repository argument, `--summary` covers the repositories of the logged-in user, and it prints a line only for a repository that has an artifact or a cache. |
