# Known issues

One row for each open defect. The row is deleted when the fix lands, and the id is retired.

| Id | Defect |
| --- | --- |
| <a id="k-3"></a>K-3 | A flag after a repository argument is not read: `gh-sweep owner/repo --dry-run` takes `--dry-run` as a repository name, and the README shows that order. A test that gives `parseCLI` the arguments `acme/widgets --dry-run` and reads one repository and the dry run proves the fix. |
