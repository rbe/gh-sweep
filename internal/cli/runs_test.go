// id: ADR-0008

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// mockRuns gives acme/widgets three runs: an old failed run, a new run, and
// an old run that still runs. The clock stands at 2026-11-28.
func mockRuns(t *testing.T) string {
	t.Helper()
	log := mockInventory(t)
	mockAPI(t, "repos/acme/widgets/actions/runs",
		fields("501", "CI", "main", "completed", "failure", "2026-09-01T10:00:00Z")+
			fields("502", "CI", "main", "completed", "success", "2026-11-27T10:00:00Z")+
			fields("503", "Nightly", "main", "in_progress", "", "2026-09-02T10:00:00Z"))
	day := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return day }
	t.Cleanup(func() { now = time.Now })
	return log
}

func TestDeleteRunsDeletesOnlyCompletedRunsOlderThanTheAge(t *testing.T) {
	log := mockRuns(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-runs", "--older-than", "30", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: run 501 \"CI\", branch main, conclusion failure, created 2026-09-01, deleted\n" +
		"total: deleted 1 workflow runs\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "DELETE repos/acme/widgets/actions/runs/501\n") ||
		strings.Contains(calls, "runs/502") || strings.Contains(calls, "runs/503") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteRunLogsSendsTheBranchAndStatusAndKeepsTheRun(t *testing.T) {
	log := mockRuns(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-run-logs", "--branch", "main", "--status", "failure", "acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.Contains(stdout.String(), "run 501 \"CI\", branch main, conclusion failure, created 2026-09-01, deleted logs\n") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	for _, call := range []string{
		"GET repos/acme/widgets/actions/runs?branch=main&per_page=100&status=failure\n",
		"DELETE repos/acme/widgets/actions/runs/501/logs\n",
	} {
		if !strings.Contains(calls, call) {
			t.Fatalf("no %q in the gh calls\n%s", call, calls)
		}
	}
	if strings.Contains(calls, "DELETE repos/acme/widgets/actions/runs/501\n") {
		t.Fatalf("the run was deleted\n%s", calls)
	}
}

func TestDeleteRunsWithDryRunSendsNoDeleteRequest(t *testing.T) {
	log := mockRuns(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--dry-run", "--delete-runs", "--older-than", "1", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: run 501 \"CI\", branch main, conclusion failure, created 2026-09-01, would delete\n" +
		"acme/widgets: run 502 \"CI\", branch main, conclusion success, created 2026-11-27, would delete\n" +
		"total: would delete 2 workflow runs\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteRunsErrorInOneRepositoryStillRunsTheNextAndFails(t *testing.T) {
	log := mockRuns(t)
	mockAPI(t, "repos/other/lib/actions/runs.fail", "HTTP 403: Resource not accessible by personal access token\n")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-runs", "--older-than", "30", "other/lib", "acme/widgets"}, &stdout, &stderr)
	wantErr := "gh-sweep: other/lib: HTTP 403: Resource not accessible by personal access token\n" +
		"gh-sweep: --delete-runs failed in 1 of 2 repositories\n"
	if status != 1 || stderr.String() != wantErr || !strings.HasSuffix(stdout.String(), "total: deleted 1 workflow runs\n") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); !strings.Contains(calls, "DELETE repos/acme/widgets/actions/runs/501") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteRunsRefusesNoTargetNoFilterAndABadFilter(t *testing.T) {
	log := mockRuns(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--delete-runs", "--older-than", "30"}, "gh-sweep: to use --delete-runs, name the repositories, or pass --mine, --org, or --user\n"},
		{[]string{"--delete-runs", "acme/widgets"}, "gh-sweep: --delete-runs needs at least one of --older-than, --branch, --status\n"},
		{[]string{"--delete-runs", "--status", "broken", "acme/widgets"}, "gh-sweep: --status broken is not a status of a workflow run\n"},
		{[]string{"--older-than", "30", "acme/widgets"}, "gh-sweep: --older-than selects entries for a write flag, such as --delete-runs. Pass one\n"},
		{[]string{"--delete-runs", "--delete-run-logs", "--older-than", "3", "acme/widgets"}, "gh-sweep: use only one of --delete-runs, --delete-run-logs\n"},
		{[]string{"--delete-runs", "--older-than", "3", "--summary", "acme/widgets"}, "gh-sweep: --delete-runs does not combine with a report flag or a settings flag\n"},
		{[]string{"--delete-runs", "--older-than", "0", "acme/widgets"}, ""},
	} {
		var stdout, stderr bytes.Buffer
		status := Run(c.args, &stdout, &stderr)
		if status != 2 || stdout.Len() != 0 || (c.want != "" && stderr.String() != c.want) {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", c.args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
