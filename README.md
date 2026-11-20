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
make container ARGS="rbe/erechnung"
```

`make auth` is `gh auth login`. Since gh 2.40 the token goes to the system keyring (macOS Keychain, or GNOME Keyring on Linux), service name `gh:github.com`, not into `~/.config/gh`. The container cannot read that keyring, so `make container` runs `gh auth token` on the host and passes the result as `GH_TOKEN`. `make token` prints the same value. With no repository argument, gh-sweep lists the repositories of the logged-in user.

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

### Projects

| Key | Action |
| --- | --- |
| space | Mark or unmark the repository. A marked repository is wiped entirely. |
| enter | Open it and pick individual artifacts and caches. |
| a / n | Mark all shown repositories, or clear them. |
| d | Delete everything in the marked repositories, after a confirmation. |
| s | Scan sizes without deleting. |
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

## Notes

- The artifacts API returns the artifacts GitHub still lists, including expired ones. If the reported total is higher than the list, the screen says so.
- Cache deletion needs the `repo` scope.
- Deleting thousands of artifacts can hit GitHub's secondary rate limit. gh-sweep waits and retries, and it has already removed the largest files by the time that happens.
