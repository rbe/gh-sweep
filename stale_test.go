// id: ADR-0014

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

// mockStale puts the clock at 2026-11-28. The logged-in user has the
// repositories acme/old (pushed 2024-11-28, 10 MB), acme/widgets (pushed
// 2026-11-27), and acme/gone (archived, pushed 2023-01-01).
func mockStale(t *testing.T) string {
	t.Helper()
	log := mockInventory(t)
	old := fields("acme/old", "2024-11-28T12:00:00Z", "10240", "0")
	widgets := fields("acme/widgets", "2026-11-27T12:00:00Z", "2048", "0")
	gone := fields("acme/gone", "2023-01-01T12:00:00Z", "1024", "1")
	mockAPI(t, "repo_list", widgets+old+gone)
	mockAPI(t, "repo_view_acme_old", old)
	mockAPI(t, "repo_view_acme_widgets", widgets)
	mockAPI(t, "repo_view_acme_gone", gone)
	day := time.Date(2026, 11, 28, 12, 0, 0, 0, time.UTC)
	now = func() time.Time { return day }
	t.Cleanup(func() { now = time.Now })
	return log
}

func TestStaleListPrintsTheRepositoriesOldestPushFirst(t *testing.T) {
	log := mockStale(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--stale-list", "--older-than", "365"}, &stdout, &stderr)
	want := "acme/gone: pushed 2023-01-01 (1427 days ago), disk 1.0 MB, archived yes\n" +
		"acme/old: pushed 2024-11-28 (730 days ago), disk 10.0 MB, archived no\n" +
		"total: repositories 2, disk 11.0 MB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "PATCH") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestArchiveArchivesOnlyTheNamedRepositoriesAndSkipsAnArchivedOne(t *testing.T) {
	log := mockStale(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--archive", "acme/old", "acme/gone"}, &stdout, &stderr)
	want := "acme/old: pushed 2024-11-28, disk 10.0 MB, archived\n" +
		"acme/gone: pushed 2023-01-01, disk 1.0 MB, already archived\n" +
		"total: archived 1 repositories\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if !strings.Contains(calls, "PATCH repos/acme/old archived=true\n") || strings.Contains(calls, "PATCH repos/acme/gone") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestArchiveWithOlderThanRefusesARecentRepositoryAndDryRunWritesNothing(t *testing.T) {
	log := mockStale(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--dry-run", "--archive", "--older-than", "365", "acme/widgets", "acme/old"}, &stdout, &stderr)
	wantErr := "gh-sweep: acme/widgets: pushed 2026-11-27, less than 365 days ago. Not archived\n" +
		"gh-sweep: --archive failed in 1 of 2 repositories\n"
	if status != 1 || stdout.String() != "acme/old: pushed 2024-11-28, disk 10.0 MB, would archive\ntotal: would archive 1 repositories\n" || stderr.String() != wantErr {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "PATCH") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestArchiveRefusesEveryTargetButNamedRepositories(t *testing.T) {
	log := mockStale(t)
	want := "gh-sweep: --archive archives only the repositories you name. Name them, and do not pass --mine, --org, or --user\n"
	for _, args := range [][]string{{"--archive"}, {"--archive", "--mine"}, {"--archive", "--org", "acme"}} {
		var stdout, stderr bytes.Buffer
		if status := run(args, &stdout, &stderr); status != 2 || stderr.String() != want {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
