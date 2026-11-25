// id: ADR-0004

package ui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCollectListsLoggedInRepos(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(mockGH(t), filepath.Join(bin, "gh")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_MOCK_LOG", filepath.Join(root, "gh.log"))
	t.Setenv("GH_MOCK_REPOS", "acme/widgets\nother/lib")

	for _, o := range []Options{{}, {Mine: true}} {
		projects, _, err := collect(o)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) != 2 || projects[0].NWO != "acme/widgets" || projects[1].NWO != "other/lib" {
			t.Fatalf("options %+v listed %+v", o, projects)
		}
	}
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
