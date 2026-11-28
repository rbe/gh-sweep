// id: ADR-0018

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockArtifacts replaces the artifacts that test/mock-gh gives for the
// repository acme/widgets, or for other/lib when other is true.
func mockArtifacts(t *testing.T, other bool, lines ...string) {
	t.Helper()
	name := filepath.Join(t.TempDir(), "art")
	text := "T " + string(rune('0'+len(lines))) + "\n" + strings.Join(lines, "")
	if err := os.WriteFile(name, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if other {
		t.Setenv("GH_MOCK_ART_OTHER", name)
		return
	}
	t.Setenv("GH_MOCK_ART", name)
}

func TestArtifactListReadsEachWorkflowRunOnceAndMarksALargeArtifact(t *testing.T) {
	log := mockInventory(t)
	mockArtifacts(t, false,
		fields("201", "image", "62914560", "2026-11-12T12:00:00Z", "0", "main", "700", "2027-02-10T12:00:00Z"),
		fields("200", "dist", "1048576", "2026-11-12T12:00:00Z", "0", "main", "700", "2027-02-10T12:00:00Z"))
	mockAPI(t, "repos/acme/widgets/actions/runs/700", "release\n")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: image #201, workflow release, size 60.0 MB, created 2026-11-12, expires 2027-02-10, status active, large\n" +
		"acme/widgets: dist #200, workflow release, size 1.0 MB, created 2026-11-12, expires 2027-02-10, status active\n" +
		"total: artifacts 2, size 61.0 MB, expired 0, expired size 0 B\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if n := strings.Count(readLog(t, log), "GET repos/acme/widgets/actions/runs/700"); n != 1 {
		t.Fatalf("run 700 was read %d times", n)
	}
}

func TestArtifactListWritesTheWorkflowAndTheLargeMarkInJSON(t *testing.T) {
	mockInventory(t)
	mockArtifacts(t, false, fields("201", "image", "62914560", "2026-11-12T12:00:00Z", "0", "main", "700", "2027-02-10T12:00:00Z"))
	mockAPI(t, "repos/acme/widgets/actions/runs/700", "release\n")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "--format", "json", "acme/widgets"}, &stdout, &stderr)
	for _, part := range []string{`"id": "201",` + "\n      " + `"workflow": "release",`, `"status": "active",` + "\n      " + `"large": true`} {
		if !strings.Contains(stdout.String(), part) {
			t.Fatalf("missing %s\nstdout\n%s", part, stdout.String())
		}
	}
	if status != 0 {
		t.Fatalf("status %d\nstderr\n%s", status, stderr.String())
	}
}

func TestArtifactListSkipsARepositoryWhoseWorkflowRunFails(t *testing.T) {
	mockInventory(t)
	mockAPI(t, "repos/acme/widgets/actions/runs/500.fail", "HTTP 403: Resource not accessible by integration\n")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: skipped, HTTP 403: Resource not accessible by integration\n" +
		"total: artifacts 0, size 0 B, expired 0, expired size 0 B\n"
	if status != 1 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestArtifactListGivesTheWorkflowDashWhenTheRunIsDeleted(t *testing.T) {
	mockInventory(t)
	mockAPI(t, "repos/acme/widgets/actions/runs/500.fail", "HTTP 404: Not Found\n")
	mockAPI(t, "repos/acme/widgets/actions/runs/499", "nightly\n")

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.Contains(stdout.String(), "dist #101, workflow -,") || !strings.Contains(stdout.String(), "dist #100, workflow nightly,") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestMinSizeKeepsTheArtifactsOfThatSizeOrMoreAndCountsOnlyThem(t *testing.T) {
	mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--artifact-list", "--min-size", "1MB", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: dist #101, workflow -, size 1.0 MB, created 2026-11-12, expires 2027-02-10, status active\n" +
		"total: artifacts 1, size 1.0 MB, expired 0, expired size 0 B\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestTopShowsTheLargestRepositoriesAndTheTotalOfAll(t *testing.T) {
	mockInventory(t)
	mockArtifacts(t, true, fields("300", "dist", "1048576", "2026-11-12T12:00:00Z", "0", "main", "800", "2027-02-10T12:00:00Z"))

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--usage", "--top", "1", "other/lib", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: size 3.5 MB (3670016 bytes), artifacts 1.5 MB, expired artifacts 512 KB, caches 2.0 MB, share 77.8%\n" +
		"total: size 4.5 MB (4718592 bytes), artifacts 2.5 MB, expired artifacts 512 KB, caches 2.0 MB\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestAdviceSaysNoAdviceForARepositoryWithNoEntry(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := Run([]string{"--advice", "other/lib"}, &stdout, &stderr)
	want := "no advice: no expired, old, or large artifact, and no storage\ntotal: repositories 1, size 0 B\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestAdviceMinSizeAndTopRefuseAWrongCombinationOrValue(t *testing.T) {
	log := mockInventory(t)
	for _, args := range [][]string{
		{"--advice", "--summary", "acme/widgets"},
		{"--usage", "--min-size", "1MB", "acme/widgets"},
		{"--min-size", "1MB", "acme/widgets"},
		{"--artifact-list", "--min-size", "5XB", "acme/widgets"},
		{"--summary", "--top", "3", "acme/widgets"},
		{"--usage", "--top", "0", "acme/widgets"},
	} {
		var stdout, stderr bytes.Buffer
		if status := Run(args, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), "gh-sweep: ") {
			t.Fatalf("%v: status %d\nstdout\n%sstderr\n%s", args, status, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(log); !os.IsNotExist(err) {
		t.Fatalf("gh was called: %v", err)
	}
}
