# Known issues

One row for each open defect. The row is deleted when the fix lands, and the id is retired.

| Id | Defect |
| --- | --- |
| K-2 | No test proves ADR-0001. A test that reads the imports of each package and fails when a package other than `internal/gh` imports `os/exec` proves the fix. |
