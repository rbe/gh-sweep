// id: D-1

package format

import (
	"testing"
	"time"
)

func TestHumanSizeAndTime(t *testing.T) {
	if HumanSize(0) != "0 B" || HumanSize(1536) != "1.5 KB" || HumanSize(1<<30) != "1.0 GB" {
		t.Fatalf("%q %q %q", HumanSize(0), HumanSize(1536), HumanSize(1<<30))
	}
	Now = func() time.Time { return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { Now = time.Now })
	got := HumanTime(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	if got != "2d ago" {
		t.Fatalf("age %q", got)
	}
	when := ParseGitHubTime("2020-01-10T14:59:22.0766667Z")
	if when.UTC().Format(time.RFC3339) != "2020-01-10T14:59:22Z" {
		t.Fatalf("fraction %v", when)
	}
	off := ParseGitHubTime("2020-01-10T14:59:22+02:00")
	if !off.Equal(time.Date(2020, 1, 10, 12, 59, 22, 0, time.UTC)) {
		t.Fatalf("offset %v", off.UTC())
	}
	if ShortenRef("refs/pull/12/merge") != "pr/12" || ShortenRef("refs/heads/main") != "main" {
		t.Fatal(ShortenRef("refs/pull/12/merge"), ShortenRef("refs/heads/main"))
	}
}
