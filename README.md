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
- `repo` scope if the repository is private, and to delete caches
- `repo` scope and admin access to the repository, to change its settings with `--retention-days` or `--actions-pr-approval`

Go does not need to be installed. `make build` and `make test` compile inside `docker.io/library/golang:1.22-bookworm` and write a binary for the machine you are on. The Go cache is kept in the container volumes `gh-sweep-gocache` and `gh-sweep-gomod`, not in the project directory. `make clean` removes `bin/`, those volumes, and a leftover `.cache` from an older build. `make clean FORCE=1` also force-removes the `gh-sweep` image and any containers still using it. On macOS that directory belongs to the container user, so `make clean` deletes it from inside the container.

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

A repository with no artifact and no cache gets no line. When no repository has an entry, the tool prints only the total line, with zeros. The total line comes last. `--org`, `--user`, `--artifacts-only`, and `--caches-only` work as they do for the checklist. `--summary` together with `--dry-run` behaves as `--summary`. An error from `gh` is printed to standard error, the tool prints no summary, and the exit status is not zero.

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

Use only one of `--summary`, `--usage`, and `--artifact-list`. `--artifact-list` together with `--caches-only` is an error. Errors from `gh` and the other flags behave as for `--summary`.

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
