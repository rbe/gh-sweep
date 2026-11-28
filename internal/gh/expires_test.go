// id: ADR-0005

package gh

import (
	"strings"
	"testing"
	"time"
)

func TestParseArtifactsReadsTheExpirationTime(t *testing.T) {
	raw := strings.Join([]string{
		"T 2",
		strings.Join([]string{"101", "dist", "100", "2026-11-12T12:00:00Z", "0", "main", "500", "2027-02-10T12:00:00Z"}, "\x1f"),
		strings.Join([]string{"100", "dist", "100", "2026-11-10T12:00:00Z", "0", "main", "499", ""}, "\x1f"),
	}, "\n")
	arts, _ := ParseArtifacts(raw)
	if len(arts) != 2 {
		t.Fatalf("%d artifacts", len(arts))
	}
	if !arts[0].Expires.Equal(time.Date(2027, 2, 10, 12, 0, 0, 0, time.UTC)) || !arts[1].Expires.IsZero() {
		t.Fatalf("expires %v %v", arts[0].Expires, arts[1].Expires)
	}
}

func TestArtifactQueryAsksForTheExpirationTime(t *testing.T) {
	if !strings.Contains(artifactJQ, "(.expires_at // \"\")") {
		t.Fatal("artifactJQ does not read expires_at")
	}
}
