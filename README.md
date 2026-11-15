# gh-sweep

A terminal checklist for deleting GitHub Actions artifacts and caches.

The screen is built with [Charmbracelet](https://github.com/charmbracelet): Bubble Tea runs the program, Lip Gloss draws the colors, and Bubbles provides the text prompt. GitHub itself is still the `gh` CLI.

The first screen is a list of repositories. Check one or more and delete everything in them, or open a repository and check individual artifacts and cache entries.

Expired artifacts cannot be downloaded, but they still count against the Actions storage quota. A few large files usually hold most of the bytes, so deletes run largest-first. GitHub recalculates usage every 6–12 hours, which means a deploy can still report a full quota after the deletes have succeeded.

## Requirements

- macOS or Linux
- Go 1.22 or newer
- [GitHub CLI](https://cli.github.com) 2.40 or newer, logged in
- `repo` scope if the repository is private, and to delete caches

## Install

```bash
go build -o gh-sweep .
install -m 755 gh-sweep /usr/local/bin/gh-sweep
```

`gh sweep` also works when the binary is named `gh-sweep` and is on your `PATH`.

## Podman

`Containerfile` builds a small image with `gh-sweep` and the GitHub CLI. The checklist needs a terminal, so run it with `-it`.

```bash
podman build -t gh-sweep -f Containerfile .
podman run --rm -it -e GH_TOKEN="$(gh auth token)" gh-sweep -- rbe/erechnung
```

`gh auth login` does not write the token into `~/.config/gh`. Since gh 2.40 the default is the system keyring: macOS Keychain, GNOME Keyring (Secret Service) on Linux, or Windows Credential Manager. The item’s service name is `gh:github.com`, and the account is your GitHub username. `hosts.yml` only keeps the username and protocol, which is why mounting `~/.config/gh` into the container still looks logged out. The container cannot read the host keyring.

`gh auth token` prints the token from that keyring. Passing it as `GH_TOKEN` is what the container’s `gh` will use. `gh auth status` on the host should say `(keyring)` next to the account.

Mount the current directory when you want the container to use that checkout instead of an `owner/repo` argument:

```bash
podman run --rm -it \
  -e GH_TOKEN="$(gh auth token)" \
  -v "${PWD}:/work" -w /work \
  gh-sweep
```

To put the token back into `hosts.yml` instead, log in again with `gh auth login --insecure-storage`. The mount of `~/.config/gh` works after that, and the token sits in a plain-text file.

## Usage

```bash
gh-sweep                              # the checkout you are in
gh-sweep rbe/erechnung                # one repository
gh-sweep rbe/erechnung other/app      # several repositories
gh-sweep --org my-org                 # pick from an organization
gh-sweep --mine                       # pick from your repositories
gh-sweep owner/repo --artifacts-only
gh-sweep owner/repo --caches-only
gh-sweep owner/repo --dry-run
```

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
bash test/run.sh
```

## Notes

- The artifacts API returns the artifacts GitHub still lists, including expired ones. If the reported total is higher than the list, the screen says so.
- Cache deletion needs the `repo` scope.
- Deleting thousands of artifacts can hit GitHub's secondary rate limit. gh-sweep waits and retries, and it has already removed the largest files by the time that happens.
