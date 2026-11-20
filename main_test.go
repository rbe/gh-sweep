// id: ADR-0003
// id: D-8

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestSummaryPrintsTheLinesAndSendsNoDeleteRequest(t *testing.T) {
	log := mockInventory(t)
	t.Setenv("GH_MOCK_REPOS", "acme/widgets\nother/lib")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary", "acme/widgets"}, &stdout, &stderr)
	want := "acme/widgets: artifacts 2, caches 1, size 3.5 MB\n" +
		"total: artifacts 2, caches 1, size 3.5 MB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	if strings.Contains(calls, "DELETE") || strings.Contains(calls, "repo\tlist") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestSummaryWithDryRunBehavesAsSummary(t *testing.T) {
	log := mockInventory(t)

	var alone, both, stderr bytes.Buffer
	if status := run([]string{"--summary", "acme/widgets"}, &alone, &stderr); status != 0 {
		t.Fatalf("status %d\n%s", status, stderr.String())
	}
	if status := run([]string{"--summary", "--dry-run", "acme/widgets"}, &both, &stderr); status != 0 {
		t.Fatalf("status %d\n%s", status, stderr.String())
	}
	if both.String() != alone.String() || !strings.HasSuffix(both.String(), "size 3.5 MB\n") {
		t.Fatalf("with --dry-run\n%swithout\n%s", both.String(), alone.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "DELETE") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

func TestSummaryWithNoArgumentCoversTheLoggedInUserAndLeavesOutAnEmptyRepository(t *testing.T) {
	log := mockInventory(t)
	t.Setenv("GH_MOCK_REPOS", "acme/widgets\nother/lib")

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary"}, &stdout, &stderr)
	want := "acme/widgets: artifacts 2, caches 1, size 3.5 MB\n" +
		"total: artifacts 2, caches 1, size 3.5 MB\n"
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	calls := readLog(t, log)
	for _, call := range []string{"CALL\trepo\tlist\t--limit", "LIST_ARTIFACTS other/lib", "LIST_CACHES other/lib"} {
		if !strings.Contains(calls, call) {
			t.Fatalf("no %q in the gh calls\n%s", call, calls)
		}
	}
}

func TestSummaryPrintsOnlyTheTotalWhenNoRepositoryHasAnEntry(t *testing.T) {
	mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary", "other/lib"}, &stdout, &stderr)
	if status != 0 || stdout.String() != "total: artifacts 0, caches 0, size 0 B\n" || stderr.Len() != 0 {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestSummarySaysWhenGHGaveOnlyTheFirstRepositories(t *testing.T) {
	mockInventory(t)
	names := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		names = append(names, "acme/widgets-"+strconv.Itoa(i))
	}
	t.Setenv("GH_MOCK_REPOS", strings.Join(names, "\n"))

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary"}, &stdout, &stderr)
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if status != 0 || len(lines) != 201 || lines[200] != "total: artifacts 400, caches 200, size 700 MB" {
		t.Fatalf("status %d, %d lines, last %q", status, len(lines), lines[len(lines)-1])
	}
	if stderr.String() != "gh-sweep: The summary covers the first 200 repositories.\n" {
		t.Fatalf("stderr\n%s", stderr.String())
	}
}

func TestSummaryPrintsAnErrorFromGHAndFails(t *testing.T) {
	// gh fails for every command after the login check, then only for the caches.
	for _, passes := range []string{"auth", "auth|api"} {
		root := t.TempDir()
		script := "#!/bin/sh\ncase \"$1\" in " + passes + ") exit 0 ;; esac\nprintf '%s\\n' 'gh: HTTP 404: Not Found' >&2\nexit 1\n"
		if err := os.WriteFile(filepath.Join(root, "gh"), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))

		var stdout, stderr bytes.Buffer
		status := run([]string{"--summary", "acme/widgets"}, &stdout, &stderr)
		if status == 0 || stdout.Len() != 0 || stderr.String() != "gh-sweep: acme/widgets: HTTP 404: Not Found\n" {
			t.Fatalf("gh passes %s: status %d\nstdout\n%sstderr\n%s", passes, status, stdout.String(), stderr.String())
		}
	}
}

func TestSummaryCountsARepeatedRepositoryOnce(t *testing.T) {
	mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary", "acme/widgets", "github.com/acme/widgets"}, &stdout, &stderr)
	if status != 0 || !strings.HasSuffix(stdout.String(), "\ntotal: artifacts 2, caches 1, size 3.5 MB\n") {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
}

func TestSummaryRefusesANameThatIsNotARepository(t *testing.T) {
	log := mockInventory(t)

	var stdout, stderr bytes.Buffer
	status := run([]string{"--summary", "not a repository"}, &stdout, &stderr)
	if status == 0 || stdout.Len() != 0 || stderr.String() != "gh-sweep: not a repository name: not a repository\n" {
		t.Fatalf("status %d\nstdout\n%sstderr\n%s", status, stdout.String(), stderr.String())
	}
	if calls := readLog(t, log); strings.Contains(calls, "LIST_") {
		t.Fatalf("gh calls\n%s", calls)
	}
}

// mockInventory puts test/mock-gh on PATH as gh. acme/widgets has two
// artifacts and one cache. other/lib has no artifact and no cache.
// It returns the file that lists the gh calls.
func mockInventory(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	artifacts := strings.Join([]string{
		"T 2",
		strings.Join([]string{"101", "dist", "1048576", "2026-11-12T12:00:00Z", "0", "main", "500"}, "\x1f"),
		strings.Join([]string{"100", "dist", "524288", "2026-11-10T12:00:00Z", "0", "main", "499"}, "\x1f"),
	}, "\n") + "\n"
	caches := strings.Join([]string{"7", "node-Linux-abc", "2097152", "2026-11-01T00:00:00Z", "refs/heads/main", "2026-11-13T12:00:00Z"}, "\x1f") + "\n"
	for name, text := range map[string]string{"art": artifacts, "cache": caches, "none": ""} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(mockGH(t), filepath.Join(bin, "gh")); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(root, "gh.log")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_MOCK_LOG", log)
	t.Setenv("GH_MOCK_ART", filepath.Join(root, "art"))
	t.Setenv("GH_MOCK_CACHES", filepath.Join(root, "cache"))
	t.Setenv("GH_MOCK_ART_OTHER", filepath.Join(root, "none"))
	return log
}

func readLog(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mockGH(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	p, err := filepath.Abs(filepath.Join(filepath.Dir(file), "test", "mock-gh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	return p
}
