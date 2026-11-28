// id: ADR-0009

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// mockCaches puts the clock at 2026-11-28. acme/widgets has the cache 7
// with the key node-Linux-abc on main, last used 2026-11-13.
func mockCaches(t *testing.T) string {
	t.Helper()
	log := mockInventory(t)
	day := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return day }
	t.Cleanup(func() { now = time.Now })
	return log
}

func TestDeleteCachesDeletesTheCachesWhoseKeyStartsWithThePrefix(t *testing.T) {
	log := mockCaches(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--delete-caches", "--key-prefix", "node-", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: cache 7 \"node-Linux-abc\", ref main, size 2.0 MB, last used 2026-11-13, deleted\n" +
		"total: deleted 1 caches, size 2.0 MB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); !strings.Contains(calls, "DELETE_CACHE acme/widgets 7") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteCachesKeepsACacheThatAFilterDoesNotSelect(t *testing.T) {
	for _, args := range [][]string{
		{"--delete-caches", "--key-prefix", "pip-", "acme/widgets"},
		{"--delete-caches", "--branch", "dev", "acme/widgets"},
		{"--delete-caches", "--older-than", "30", "acme/widgets"},
	} {
		log := mockCaches(t)
		var stdout, stderr bytes.Buffer
		if status := run(args, &stdout, &stderr); status != 0 || stdout.String() != "total: deleted 0 caches, size 0 B\n" {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", args, status, stdout.String(), stderr.String())
		}
		if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
			t.Fatalf("%v: gh calls\n%s", args, calls)
		}
	}
}

func TestDeleteCachesWithDryRunSendsNoDeleteRequest(t *testing.T) {
	log := mockCaches(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--dry-run", "--delete-caches", "--branch", "main", "--older-than", "7", "acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.HasSuffix(stdout.String(), "last used 2026-11-13, would delete\ntotal: would delete 1 caches, size 2.0 MB\n") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestDeleteCachesRefusesNoTargetAndNoFilter(t *testing.T) {
	log := mockCaches(t)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--delete-caches", "--key-prefix", "node-"}, "gh-sweep: to use --delete-caches, name the repositories, or pass --mine, --org, or --user\n"},
		{[]string{"--delete-caches", "acme/widgets"}, "gh-sweep: --delete-caches needs at least one of --key-prefix, --branch, --older-than\n"},
		{[]string{"--delete-caches", "--status", "failure", "acme/widgets"}, "gh-sweep: --delete-caches does not read --status\n"},
	} {
		var stdout, stderr bytes.Buffer
		if status := run(c.args, &stdout, &stderr); status != 2 || stdout.Len() != 0 || stderr.String() != c.want {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", c.args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}

func TestCacheUsagePrintsTheUsageAndTheLimitsAndAnUnknownLimit(t *testing.T) {
	log := mockCaches(t)
	t.Setenv("GH_MOCK_REPOS", "acme/widgets\nother/lib")
	mockAPI(t, "repos/acme/widgets/actions/cache/usage", "3 2097152\n")
	mockAPI(t, "repos/acme/widgets/actions/cache/storage-limit", "10\n")
	mockAPI(t, "repos/acme/widgets/actions/cache/retention-limit", "7\n")
	mockAPI(t, "repos/other/lib/actions/cache/usage", "0 0\n")
	mockAPI(t, "repos/other/lib/actions/cache/storage-limit", "10\n")
	mockAPI(t, "repos/other/lib/actions/cache/retention-limit.fail", "HTTP 403: Must have admin rights to Repository.\n")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--cache-usage"}, &stdout, &stderr)
	want := "acme/widgets: caches 3, size 2.0 MB (2097152 bytes), storage limit 10 GB, retention limit 7 days\n" +
		"other/lib: caches 0, size 0 B (0 bytes), storage limit 10 GB, retention limit unknown\n" +
		"total: caches 3, size 2.0 MB (2097152 bytes)\n"
	wantErr := "gh-sweep: other/lib: cache retention limit: HTTP 403: Must have admin rights to Repository.\n" +
		"gh-sweep: --cache-usage failed in 1 of 2 repositories\n"
	if status != 1 || stdout.String() != want || stderr.String() != wantErr {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}
