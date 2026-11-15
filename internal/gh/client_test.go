package gh

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/rbe/gh-sweep/internal/format"
	"github.com/rbe/gh-sweep/internal/project"
)

func TestLoadUsesGH(t *testing.T) {
	root := t.TempDir()
	art := strings.Join([]string{
		"T 1",
		strings.Join([]string{"55", "notes", "100", "2026-11-01T00:00:00Z", "0", "main", "1"}, "\x1f"),
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(root, "art"), []byte(art), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "cache"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(mockGH(t), filepath.Join(bin, "gh")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_MOCK_LOG", filepath.Join(root, "gh.log"))
	t.Setenv("GH_MOCK_ART", filepath.Join(root, "art"))
	t.Setenv("GH_MOCK_CACHES", filepath.Join(root, "cache"))
	t.Setenv("GH_MOCK_ART_OTHER", filepath.Join(root, "art"))

	base, err := project.Parse("acme/widgets")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load(base, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArtN != 1 || got.Items[0].Name != "notes" || got.Items[0].ID != "55" {
		t.Fatalf("%+v", got)
	}
}

func TestGroupsAndSelection(t *testing.T) {
	p := sampleProject()
	if p.ArtGroups[0] < 0 || p.Groups[p.ArtGroups[0]].Name != "dist" {
		t.Fatalf("largest artifact group = %s", p.Groups[p.ArtGroups[0]].Name)
	}
	if p.Groups[p.CacheGroups[0]].Name != "node-Linux-abc" {
		t.Fatalf("largest cache = %s", p.Groups[p.CacheGroups[0]].Name)
	}
	g := p.Groups[p.ArtGroups[0]]
	if p.Items[g.Start].ID != "101" || p.Items[g.Start+1].ID != "100" {
		t.Fatalf("order %s then %s", p.Items[g.Start].ID, p.Items[g.Start+1].ID)
	}
	p.ToggleMatching(func(it project.Item) bool { return it.Group == p.ArtGroups[0] && it.Matches("") })
	if !p.Items[g.Start].On || !p.Items[g.Start+1].On {
		t.Fatal("group toggle")
	}
	p.Items[g.Start+1].On = false
	live, on, _ := p.StateOf(func(it project.Item) bool { return it.Group == g.Start || it.Name == "dist" && it.Kind == "artifact" })
	if project.BoxState(live, on) != "some" && on != 1 {
		t.Fatalf("partial live=%d on=%d", live, on)
	}
	n := p.SelectExceptNewest(1, "")
	if n != 1 || itemOn(p, "101") || !itemOn(p, "100") || itemOn(p, "90") {
		t.Fatalf("keep newest n=%d 101=%v 100=%v 90=%v", n, itemOn(p, "101"), itemOn(p, "100"), itemOn(p, "90"))
	}
	if p.SelectExpired("") != 1 || !itemOn(p, "90") {
		t.Fatal("expired")
	}
	if p.Items[idIndex(p, "90")].Expired != true {
		t.Fatal("expired flag")
	}
	rows := p.Rows("dist[1]", true, true)
	if len(rows) != 0 {
		t.Fatal("filter brackets should be literal")
	}
	rows = p.Rows("release", true, true)
	for _, r := range rows {
		if r.Kind == project.RowGroup && p.Groups[r.Ref].Name == "coverage" {
			t.Fatal("branch filter showed coverage")
		}
	}
}

func sampleProject() project.Project {
	format.Now = func() time.Time { return time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC) }
	raw := strings.Join([]string{
		"T 4",
		strings.Join([]string{"101", "dist", "900000000", "2026-11-12T12:00:00Z", "0", "main", "500"}, "\x1f"),
		strings.Join([]string{"100", "dist", "600000000", "2026-11-10T12:00:00Z", "0", "release", "499"}, "\x1f"),
		strings.Join([]string{"90", "coverage", "12000000", "2026-10-01T12:00:00Z", "1", "", "0"}, "\x1f"),
		strings.Join([]string{"80", "Weird Name", "2048", "2026-11-14T11:00:00Z", "0", "feat/tabs", "1"}, "\x1f"),
	}, "\n")
	arts, total := ParseArtifacts(raw)
	if total != 4 || len(arts) != 4 {
		panic("parse artifacts")
	}
	caches := ParseCaches(strings.Join([]string{
		strings.Join([]string{"7", "node-Linux-abc", "2000000000", "2026-11-01T00:00:00Z", "refs/heads/main", "2026-11-13T12:00:00Z"}, "\x1f"),
		strings.Join([]string{"8", "node-Linux-abc", "100000000", "2026-10-01T00:00:00Z", "refs/pull/12/merge", "2026-10-02T00:00:00Z"}, "\x1f"),
		strings.Join([]string{"9", "pip-Linux", "50000000", "2026-09-01T00:00:00Z", "refs/heads/main", "2026-09-02T00:00:00Z"}, "\x1f"),
	}, "\n"))
	p := project.Project{Owner: "acme", Name: "widgets", NWO: "acme/widgets", On: true}
	p.Items = append(arts, caches...)
	p.RebuildGroups()
	return p
}

func itemOn(p project.Project, id string) bool {
	for _, it := range p.Items {
		if it.ID == id {
			return it.On
		}
	}
	return false
}

func idIndex(p project.Project, id string) int {
	for i, it := range p.Items {
		if it.ID == id {
			return i
		}
	}
	return -1
}

func mockGH(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	p, err := filepath.Abs(filepath.Join(filepath.Dir(file), "..", "..", "test", "mock-gh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	return p
}
