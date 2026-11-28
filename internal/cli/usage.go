package cli

const usageText = `gh-sweep — delete GitHub Actions artifacts and caches

The screen is a Charmbracelet (Bubble Tea) checklist. Mark repositories to
wipe them, or open one and check individual artifacts and cache entries.

Usage:
  gh-sweep [options] [owner/repo ...]

Options:
  -repo owner/repo     repository, repeatable
  -org name            list repositories in this organization
  -user name           list repositories owned by this user
  -mine                list repositories for the logged-in user (the default)
  -artifacts-only      don't list or delete caches
  -caches-only         don't list or delete artifacts
  -filter text         start the item view with this filter
  -dry-run             confirm without deleting
  -summary             print one line for each repository and a total line, then exit
  -usage               print the storage of each repository, largest first, then exit
  -artifact-list       print one line for each artifact with size, dates, and status, then exit
  -advice             print advice on expired, old, and large artifacts, then exit
  -min-size size       with -artifact-list, only artifacts of this size or more, such as 50MB
  -top n               with -usage, only the n largest repositories
  -format text|json|csv
                       write the report of -summary, -usage, -artifact-list, or -advice in this format
  -output file         write that report to this file instead of standard output
  -retention-days n    set the artifact and log retention of each repository to n days, then exit
  -actions-pr-approval allow|deny
                       allow or deny that GitHub Actions creates and approves pull requests, then exit
  -delete-runs         delete completed workflow runs and their logs, then exit
  -delete-run-logs     delete the logs of completed workflow runs and keep the runs, then exit
  -delete-caches       delete the caches that the filters select, then exit
  -cache-usage         print the cache usage and the cache limits of each repository, then exit
  -delete-releases     delete the releases that the filters select and keep their tags, then exit
  -delete-release-assets
                       delete the assets of the releases that the filters select, then exit
  -delete-package-versions
                       delete versions of the package that -package names, then exit
  -delete-codespaces   delete your stopped codespaces with no unsaved work, then exit
  -billing             print the billed GitHub Actions usage of this month, then exit
  -delete-merged-branches
                       delete branches whose pull request is merged or closed, then exit
  -delete-deployments  delete old deployments and keep the newest of each environment, then exit
  -delete-unused-environments
                       delete environments with no deployment, then exit
  -stale-list          print each repository with its last push and disk usage, oldest first, then exit
  -archive             archive the named repositories, then exit

Filters of the write flags:
  -older-than n        only entries more than n days old
  -branch name         only workflow runs or caches of this branch
  -status value        only workflow runs with this status or conclusion, such as failure
  -key-prefix text     only caches whose key starts with this text
  -keep n              keep the newest n releases, package versions, or deployments of each environment
  -untagged            only package versions with no tag
  -package name        the package of -delete-package-versions
  -package-type type   container (the default), npm, maven, rubygems, docker, or nuget
  -h                   show this help

Project keys:
  space       mark a repository for a full wipe
  enter       open it and pick artifacts and caches
  a / n       mark all shown, or clear them
  d           delete everything marked
  s           scan sizes
  u           storage usage of the scanned projects
  m           action menu
  p           preferences, saved to the preferences file
  +           add owner/repo
  /           filter
  q           quit

Inside a repository:
  space       toggle the row (a group toggles every row in it)
  enter       toggle an item, or open and close a group
  j / k       move
  h / l       collapse or expand    e / c fold all
  x           select expired artifacts
  o           select artifacts older than N days
  t           keep the newest N of each name, select the rest
  i           details of each artifact
  d           delete the checked rows
  b           back to the project list
  q           quit
`
