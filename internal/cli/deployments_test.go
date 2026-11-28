// id: ADR-0013

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// mockDeployments puts the clock at 2026-11-28. acme/widgets has three
// deployments to production, one old deployment to staging, and the
// environments production, staging, preview (no deployment), and qa%1.
func mockDeployments(t *testing.T) string {
	t.Helper()
	log := mockInventory(t)
	mockAPI(t, "repos/acme/widgets/deployments",
		fields("13", "production", "main", "2026-11-20T00:00:00Z")+
			fields("12", "production", "main", "2026-10-01T00:00:00Z")+
			fields("11", "production", "v1", "2026-05-01T00:00:00Z")+
			fields("21", "staging", "dev", "2026-03-01T00:00:00Z"))
	mockAPI(t, "repos/acme/widgets/environments", "production\nstaging\npreview\nqa%1\n")
	day := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return day }
	t.Cleanup(func() { now = time.Now })
	return log
}

func TestDeleteDeploymentsKeepsTheNewestOfEachEnvironmentAndSetsInactiveFirst(t *testing.T) {
	log := mockDeployments(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-deployments", "--older-than", "30", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: deployment 12, environment production, ref main, created 2026-10-01, deleted\n" +
		"acme/widgets: deployment 11, environment production, ref v1, created 2026-05-01, deleted\n" +
		"total: deleted 2 deployments\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	inactive := strings.Index(calls, "POST repos/acme/widgets/deployments/12/statuses state=inactive\n")
	deleted := strings.Index(calls, "DELETE repos/acme/widgets/deployments/12\n")
	if inactive < 0 || deleted < inactive || strings.Contains(calls, "deployments/13") || strings.Contains(calls, "deployments/21") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteDeploymentsWithKeepAndDryRunSendsNoWrite(t *testing.T) {
	log := mockDeployments(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--dry-run", "--delete-deployments", "--keep", "2", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: deployment 11, environment production, ref v1, created 2026-05-01, would delete\n" +
		"total: would delete 1 deployments\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "POST") || strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteUnusedEnvironmentsDeletesEnvironmentsWithNoDeployment(t *testing.T) {
	log := mockDeployments(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--delete-unused-environments", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: environment preview, deployments 0, newest deployment -, deleted\n" +
		"acme/widgets: environment qa%1, deployments 0, newest deployment -, deleted\n" +
		"total: deleted 2 environments\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "DELETE repos/acme/widgets/environments/qa%251\n") || strings.Contains(calls, "environments/staging") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteUnusedEnvironmentsWithOlderThanAlsoTakesAnEnvironmentWithOnlyOldDeployments(t *testing.T) {
	mockDeployments(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--dry-run", "--delete-unused-environments", "--older-than", "180", "acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.Contains(stdout.String(), "environment staging, deployments 1, newest deployment 2026-03-01, would delete\n") ||
		strings.Contains(stdout.String(), "production") || !strings.HasSuffix(stdout.String(), "total: would delete 3 environments\n") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestDeploymentFlagsRefuseNoTargetAndNoFilter(t *testing.T) {
	log := mockDeployments(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--delete-deployments", "--keep", "1"}, "gh-sweep: to use --delete-deployments, name the repositories, or pass --mine, --org, or --user\n"},
		{[]string{"--delete-deployments", "acme/widgets"}, "gh-sweep: --delete-deployments needs at least one of --older-than, --keep\n"},
		{[]string{"--delete-unused-environments"}, "gh-sweep: to use --delete-unused-environments, name the repositories, or pass --mine, --org, or --user\n"},
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
