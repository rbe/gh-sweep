# gh-sweep

A terminal checklist for deleting GitHub Actions artifacts and caches.

The screen is built with [Charmbracelet](https://github.com/charmbracelet): Bubble Tea runs the program, Lip Gloss draws the colors, and Bubbles provides the text prompt. GitHub itself is still the `gh` CLI.

The first screen is a list of repositories. Check one or more and delete everything in them, or open a repository and check individual artifacts and cache entries.

Expired artifacts cannot be downloaded, but they still count against the Actions storage quota. A few large files usually hold most of the bytes, so deletes run largest-first. GitHub recalculates usage every 6–12 hours, which means a deploy can still report a full quota after the deletes have succeeded.

`main.go` reads the flags and prints the summary. `internal/ui` is the checklist, `internal/project` is a repository, the artifacts and caches in it, and the lines of the summary, `internal/gh` calls the GitHub CLI, and `internal/format` turns sizes, times, and refs into the text on screen.

## Requirements

- macOS or Linux, with GNU make (the `make` that ships with both)
- Podman, or Docker (`CONTAINER_ENGINE=docker`)
- [GitHub CLI](https://cli.github.com) 2.40 or newer, logged in, when you run the tool on the host
- a token with the permissions of the table in [Token permissions](#token-permissions)

Go does not need to be installed. `make build` and `make test` compile inside `docker.io/library/golang:1.22-bookworm` and write a binary for the machine you are on. The Go cache is kept in the container volumes `gh-sweep-gocache` and `gh-sweep-gomod`, not in the project directory. `make clean` removes `bin/`, those volumes, and a leftover `.cache` from an older build. `make clean FORCE=1` also force-removes the `gh-sweep` image and any containers still using it. On macOS that directory belongs to the container user, so `make clean` deletes it from inside the container.

## Token permissions

gh-sweep uses the token of `gh`. A fine-grained personal access token works for each flag in the table. Give the token access to the repositories you name. Log in with it as `gh auth login --with-token < token.txt`, or set `GH_TOKEN`. The classic scope is the fallback for a classic token or for the token of `gh auth login`.

| What | Fine-grained permission | Classic scope |
| --- | --- | --- |
| Checklist, `--summary`, `--usage`, `--artifact-list` | Repository: Actions read, and write to delete | `repo` |
| `--retention-days`, `--actions-pr-approval` | Repository: Administration read and write | `repo`, and admin access to the repository |
| `--delete-runs`, `--delete-run-logs` | Repository: Actions read and write | `repo` |
| `--delete-caches` | Repository: Actions read and write | `repo` |
| `--cache-usage` | Repository: Actions read, and Administration read for the retention limit | `repo`, and `admin:repository` for both limits |

Each fine-grained token also has Repository: Metadata read. The table comes from the GitHub data files `fine-grained-pat.json` and `fine-grained-pat-permissions.json` (repository github/docs) and from the REST API description (repository github/rest-api-description), read on 2026-11-28. The tests of this project do not call GitHub, so no test proves the table. `gh repo list`, which gh-sweep runs when no repository is named, uses the GraphQL API of GitHub. It is expected to list only the repositories that the fine-grained token can access, but this is not tested.

## Install

```bash
make build
make install
```

`make build` writes `bin/gh-sweep`. `make` with no target prints the targets. `make install` copies it to `~/.local/bin`, which does not need root. `PREFIX` and `DESTDIR` still work. If that directory is not on your `PATH` yet, add `export PATH="$HOME/.local/bin:$PATH"` to your shell startup file. `gh sweep` also works when that binary is named `gh-sweep` and is on your `PATH`.

## Podman

The Makefile uses Podman when it is installed, and Docker otherwise (`CONTAINER_ENGINE=docker` to force it). The checklist needs a terminal.

```bash
make auth
make run ARGS="rbe/erechnung"
```

`make auth` is `gh auth login`. Since gh 2.40 the token goes to the system keyring (macOS Keychain, or GNOME Keyring on Linux), service name `gh:github.com`, not into `~/.config/gh`. The container cannot read that keyring, so `make run` runs `gh auth token` on the host and gives the container the variable `GH_TOKEN`. The value is not on a command line. `make token` prints the same value. With no repository argument, gh-sweep lists the repositories of the logged-in user.

An existing `GH_TOKEN` in the environment is used as-is and the keyring is not read.

## Usage

```bash
gh-sweep                              # repositories of the logged-in user
gh-sweep rbe/erechnung                # one repository
gh-sweep rbe/erechnung other/app      # several repositories
gh-sweep --org my-org                 # pick from an organization
gh-sweep --mine                       # same list as with no arguments
gh-sweep owner/repo --artifacts-only
gh-sweep owner/repo --caches-only
gh-sweep owner/repo --dry-run
```

### Summary

```bash
gh-sweep --summary                    # repositories of the logged-in user
gh-sweep --summary rbe/erechnung other/app
```

`--summary` prints one line for each repository, then a total line, and exits. It opens no checklist and deletes nothing. Put `--summary` before the repository names. The output is plain text on standard output:

```text
rbe/erechnung: artifacts 42, caches 7, size 3.4 GB
other/app: artifacts 3, caches 0, size 12.0 MB
total: artifacts 45, caches 7, size 3.4 GB
```

A repository with no artifact and no cache gets no line. When no repository has an entry, the tool prints only the total line, with zeros. The total line comes last. `--org`, `--user`, `--artifacts-only`, and `--caches-only` work as they do for the checklist. `--summary` together with `--dry-run` behaves as `--summary`. If `gh` gives an error for one repository, the tool skips that repository and finishes the summary. The skipped repository gets a line before the total line, with the first line of the error. The total counts only the repositories the tool read. Standard error then says how many repositories were skipped, and the exit status is 1:

```text
rbe/erechnung: artifacts 42, caches 7, size 3.4 GB
other/app: skipped, HTTP 403: Resource not accessible by personal access token
total: artifacts 42, caches 7, size 3.4 GB
```

An error before the first repository, for example when `gh` is not logged in, stops the tool before it prints a summary.

### Storage usage

```bash
gh-sweep --usage                      # repositories of the logged-in user
gh-sweep --usage rbe/erechnung other/app
```

`--usage` prints one line for each repository, largest first, then a total line, and exits. It deletes nothing:

```text
rbe/erechnung: size 3.4 GB (3650722201 bytes), artifacts 3.1 GB, expired artifacts 1.2 GB, caches 300 MB, share 99.7%
other/app: size 12.0 MB (12582912 bytes), artifacts 12.0 MB, expired artifacts 0 B, caches 0 B, share 0.3%
total: size 3.4 GB (3663305113 bytes), artifacts 3.1 GB, expired artifacts 1.2 GB, caches 300 MB
```

The size is the sum of the artifacts and the caches that the API lists. A repository with no artifact and no cache gets no line.

### Artifact list

```bash
gh-sweep --artifact-list rbe/erechnung
```

`--artifact-list` prints one line for each artifact, then a total line, and exits. It deletes nothing and reads no cache. The artifacts of a repository stand largest first. Dates are UTC, and `-` is a date that GitHub did not give:

```text
rbe/erechnung: dist #2207435, size 512 MB, created 2026-11-20, expires 2027-02-18, status active
rbe/erechnung: coverage #2101113, size 40.0 MB, created 2026-08-01, expires 2026-10-30, status expired
total: artifacts 2, size 552 MB, expired 1, expired size 40.0 MB
```

Use only one of `--summary`, `--usage`, and `--artifact-list`. `--artifact-list` together with `--caches-only` is an error. Errors from `gh`, the skipped lines, and the other flags behave as for `--summary`.

### Repository settings

```bash
gh-sweep --retention-days 30 rbe/erechnung other/app
gh-sweep --actions-pr-approval deny --org my-org
gh-sweep --dry-run --retention-days 7 --actions-pr-approval allow --mine
```

`--retention-days N` sets the artifact and log retention to N days. This is the setting "Artifact and log retention" under Settings, Actions, General. GitHub applies the one value to checks, workflow runs, commit statuses, artifacts, and logs. A public repository accepts 1 to 90 days. The tool reads the maximum of each repository and refuses a larger value.

`--actions-pr-approval allow` or `deny` sets "Allow GitHub Actions to create and approve pull requests". The default `GITHUB_TOKEN` permission (read or write) stays as it is. GitHub says that `allow` can be a security risk.

The tool reads each setting first and prints one line for each repository and setting. It writes only a value that differs. With `--dry-run` it writes nothing and prints `would set`:

```text
rbe/erechnung: retention-days 90, set 30 (maximum 90)
other/app: retention-days 30, unchanged (maximum 90)
rbe/erechnung: actions-pr-approval allow, set deny
```

Name the repositories, or pass `--mine`, `--org`, or `--user`. With none of them, the tool changes nothing and exits with an error. Put the flags before the repository names. A settings flag together with `--summary`, `--usage`, or `--artifact-list` is an error. If one repository fails, for example because you are not an admin there, the tool prints the error, goes on with the next repository, and exits with status 1. An organization or enterprise policy can block either setting, and GitHub then refuses the change.

### Workflow runs

```bash
gh-sweep --dry-run --delete-runs --older-than 90 rbe/erechnung
gh-sweep --delete-runs --status failure --branch main rbe/erechnung
gh-sweep --delete-run-logs --older-than 30 --mine
```

`--delete-runs` deletes workflow runs. GitHub deletes the logs of a run together with the run. `--delete-run-logs` deletes only the logs and keeps the runs. Both flags take only runs with the status `completed`, so a run that still runs or waits stays. They print one line for each run, then a total line, and exit:

```text
rbe/erechnung: run 1234567 "CI", branch main, conclusion failure, created 2026-08-01, deleted
total: deleted 1 workflow runs
```

The filters select the runs. Give at least one:

- `--older-than N` takes runs created more than N days ago.
- `--branch NAME` takes runs of that branch.
- `--status VALUE` takes runs with that status or conclusion, such as `failure`, `cancelled`, or `success`.

GitHub gives at most 1,000 runs for one call with `--branch` or `--status`. Run the flag again to take more. With `--dry-run` the tool deletes nothing and prints `would delete`. Name the repositories, or pass `--mine`, `--org`, or `--user`. With none of them, the tool changes nothing and exits with an error. If one repository fails, the tool prints the error, goes on with the next repository, and exits with status 1. A deleted run cannot be restored.

### Caches

```bash
gh-sweep --dry-run --delete-caches --key-prefix node- rbe/erechnung
gh-sweep --delete-caches --branch dependabot/npm --older-than 7 --mine
gh-sweep --cache-usage rbe/erechnung other/app
```

`--delete-caches` deletes the caches that the filters select, prints one line for each cache, then a total line, and exits. Give at least one filter:

- `--key-prefix TEXT` takes caches whose key starts with TEXT.
- `--branch NAME` takes caches of the ref `refs/heads/NAME`.
- `--older-than N` takes caches last used more than N days ago.

```text
rbe/erechnung: cache 7 "node-Linux-abc", ref main, size 2.0 MB, last used 2026-11-13, deleted
total: deleted 1 caches, size 2.0 MB
```

The tool deletes each cache by its id, so a filter on the branch or the age keeps the caches of the same key on other refs. `--dry-run`, the guard against an empty target, and errors behave as for `--delete-runs`.

`--cache-usage` prints one line for each repository with the number and the size of the active caches, the storage limit, and the retention limit, then a total line. It deletes nothing and works with no repository argument:

```text
rbe/erechnung: caches 3, size 2.0 MB (2097152 bytes), storage limit 10 GB, retention limit 7 days
total: caches 3, size 2.0 MB (2097152 bytes)
```

The tool reads the limits from `actions/cache/storage-limit` and `actions/cache/retention-limit`. GitHub refuses the retention limit to a token without admin access. The line then says `unknown`, standard error gives the reason, and the exit status is 1.

### Projects

| Key | Action |
| --- | --- |
| space | Mark or unmark the repository. A marked repository is wiped entirely. |
| enter | Open it and pick individual artifacts and caches. |
| a / n | Mark all shown repositories, or clear them. |
| d | Delete everything in the marked repositories, after a confirmation. |
| s | Scan sizes without deleting. |
| u | Show the storage usage of the scanned projects, largest first. |
| + | Add `owner/repo`. |
| / | Filter the list. Type to narrow it. Esc clears it. |
| q | Quit. |

### Inside a repository

| Key | Action |
| --- | --- |
| space | Toggle the row. A group toggles every row inside it. |
| enter | Toggle an item, or open and close a group. |
| j / k | Move. Arrow keys do the same. |
| h / l | Collapse or expand the group. `e` and `c` fold all. |
| a / n | Select or clear what the filter shows. |
| A / C | Select shown artifacts, or shown caches. |
| x | Select expired artifacts. |
| o | Select artifacts older than a number of days. |
| t | Keep the newest N of each artifact name, and select the rest. |
| i | Show each artifact with its size, creation date, expiration date, and status. |
| d | Delete the checked rows. `y` confirms. |
| b | Back to the project list. |
| / | Filter by name, branch, ref, or id. |
| r | Reload from GitHub. |
| q | Quit. |

Nothing is deleted until the confirmation. One large repository asks you to type its name. Marking more than one repository asks you to type `wipe`.

Checking an artifact group selects every run with that name. Checking a cache group selects every entry with that key, which is the whole cache across refs. Open the group to keep one.

## What gets deleted

An artifact:

```bash
gh api -X DELETE repos/OWNER/REPO/actions/artifacts/ARTIFACT_ID
```

A cache entry:

```bash
gh cache delete CACHE_ID -R OWNER/REPO
```

## Tests

```bash
make test
```

The tests run inside the Go container. `make coverage` reports the lines a change touched, and `make test-changed` then runs the tests of the packages that cover the change.

A workflow run, or only its logs:

```bash
gh api -X DELETE repos/OWNER/REPO/actions/runs/RUN_ID
gh api -X DELETE repos/OWNER/REPO/actions/runs/RUN_ID/logs
```

## What gets changed

The retention:

```bash
gh api -X PUT repos/OWNER/REPO/actions/permissions/artifact-and-log-retention -F days=30
```

The pull request setting:

```bash
gh api -X PUT repos/OWNER/REPO/actions/permissions/workflow \
  -f default_workflow_permissions=read -F can_approve_pull_request_reviews=false
```

## Notes

- The artifacts API returns the artifacts GitHub still lists, including expired ones. If the reported total is higher than the list, the screen says so.
- Cache deletion needs the `repo` scope.
- Deleting thousands of artifacts can hit GitHub's secondary rate limit. gh-sweep waits and retries, and it has already removed the largest files by the time that happens.
