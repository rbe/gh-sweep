# Known issues

One row for each open defect. The row is deleted when the fix lands, and the id is retired.

| Id | Defect |
| --- | --- |
| K-1 | `test/run.sh` is a shell-script wrapper that runs `go test` on the host. The Make contract has no shell-script wrapper and runs Go in the container. `make test` is the target. |
| K-2 | No test proves ADR-0001. A test that reads the imports of each package and fails when a package other than `internal/gh` imports `os/exec` proves the fix. |
