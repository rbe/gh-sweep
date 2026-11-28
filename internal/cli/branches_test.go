// id: ADR-0012

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// mockBranches gives acme/widgets these branches: main (the default),
// done (merged in #1), dropped (closed in #2), moved (merged in #3, then a
// new commit), busy (merged in #4, open again in #5), base (the base of the
// open #6), and keep (protected, merged in #7).
func mockBranches(t *testing.T) string {
	t.Helper()
	log := mockInventory(t)
	mockAPI(t, "repos/acme/widgets", "main\n")
	mockAPI(t, "repos/acme/widgets/branches",
		fields("main", "m1", "1")+fields("done", "d1", "0")+fields("dropped", "x1", "0")+
			fields("moved", "v2", "0")+fields("busy", "b1", "0")+fields("base", "s1", "0")+fields("keep", "k1", "1"))
	mockAPI(t, "repos/acme/widgets/pulls",
		fields("1", "closed", "1", "done", "d1", "acme/widgets", "main")+
			fields("2", "closed", "0", "dropped", "x1", "acme/widgets", "main")+
			fields("3", "closed", "1", "moved", "v1", "acme/widgets", "main")+
			fields("4", "closed", "1", "busy", "b1", "acme/widgets", "main")+
			fields("5", "open", "0", "busy", "b1", "acme/widgets", "main")+
			fields("6", "open", "0", "feature", "f1", "fork/widgets", "base")+
			fields("8", "closed", "1", "base", "s1", "acme/widgets", "main")+
			fields("7", "closed", "1", "keep", "k1", "acme/widgets", "main")+
			fields("9", "closed", "1", "main", "m1", "fork/widgets", "main"))
	return log
}

func TestDeleteMergedBranchesDeletesOnlyBranchesWithNoNewCommitAndNoOpenPullRequest(t *testing.T) {
	log := mockBranches(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-merged-branches", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: branch done, pull request #1 merged, deleted\n" +
		"acme/widgets: branch dropped, pull request #2 closed, deleted\n" +
		"total: deleted 2 branches\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	for _, kept := range []string{"heads/main", "heads/moved", "heads/busy", "heads/base", "heads/keep"} {
		if strings.Contains(calls, kept) {
			t.Fatalf("%s was deleted\n%s", kept, calls)
		}
	}
	if !strings.Contains(calls, "DELETE repos/acme/widgets/git/refs/heads/done\n") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteMergedBranchesWithDryRunSendsNoDeleteRequest(t *testing.T) {
	log := mockBranches(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--dry-run", "--delete-merged-branches", "acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.HasSuffix(stdout.String(), "#2 closed, would delete\ntotal: would delete 2 branches\n") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteMergedBranchesRefusesNoTargetAndAFilter(t *testing.T) {
	log := mockBranches(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--delete-merged-branches"}, "gh-sweep: to use --delete-merged-branches, name the repositories, or pass --mine, --org, or --user\n"},
		{[]string{"--delete-merged-branches", "--older-than", "3", "acme/widgets"}, "gh-sweep: --delete-merged-branches does not read --older-than\n"},
	} {
		var stdout, stderr bytes.Buffer
		if status := Run(c.args, &stdout, &stderr); status != 2 || stderr.String() != c.want {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", c.args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
