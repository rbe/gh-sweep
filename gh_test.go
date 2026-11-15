package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadProjectUsesGH(t *testing.T) {
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
	mock := filepath.Join("test", "mock-gh")
	if _, err := os.Stat(mock); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(abs(t, mock), filepath.Join(bin, "gh")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GH_MOCK_LOG", filepath.Join(root, "gh.log"))
	t.Setenv("GH_MOCK_ART", filepath.Join(root, "art"))
	t.Setenv("GH_MOCK_CACHES", filepath.Join(root, "cache"))
	t.Setenv("GH_MOCK_ART_OTHER", filepath.Join(root, "art"))

	base, err := parseRepo("acme/widgets")
	if err != nil {
		t.Fatal(err)
	}
	got, err := loadProject(base, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.ArtN != 1 || got.Items[0].Name != "notes" || got.Items[0].ID != "55" {
		t.Fatalf("%+v", got)
	}
}

func abs(t *testing.T, path string) string {
	t.Helper()
	p, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
