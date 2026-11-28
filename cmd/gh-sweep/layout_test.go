// id: ADR-0016

package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// modulePath is the import path of the module.
const modulePath = "github.com/rbe/gh-sweep/"

// allowedImports names, for each package directory, the project packages
// that it may import. A package that is not listed may not exist.
var allowedImports = map[string][]string{
	"cmd/gh-sweep":     {"internal/cli"},
	"internal/cli":     {"internal/gh", "internal/project", "internal/sweep", "internal/ui"},
	"internal/ui":      {"internal/format", "internal/gh", "internal/project", "internal/sweep"},
	"internal/sweep":   {"internal/gh", "internal/project"},
	"internal/gh":      {"internal/format", "internal/project"},
	"internal/project": {"internal/format"},
	"internal/format":  {},
}

func TestEachPackageImportsOnlyThePackagesOfItsLayer(t *testing.T) {
	imports, err := projectImports("../..")
	if err != nil {
		t.Fatal(err)
	}
	for dir, got := range imports {
		want, ok := allowedImports[dir]
		if !ok {
			t.Fatalf("package %s is not in the layout of ADR-0016", dir)
		}
		for _, imp := range got {
			if !contains(want, imp) {
				t.Fatalf("package %s imports %s, which ADR-0016 does not allow", dir, imp)
			}
		}
	}
}

func TestTheRepositoryRootHoldsNoGoFile(t *testing.T) {
	files, err := filepath.Glob("../../*.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) > 0 {
		t.Fatalf("Go files at the repository root: %v", files)
	}
}

func TestAnImportAcrossTheLayersIsFound(t *testing.T) {
	root := t.TempDir()
	write(t, root, "internal/format/format.go", "package format\n")
	write(t, root, "internal/format/format_test.go", "package format\n\nimport _ \"github.com/rbe/gh-sweep/internal/ui\"\n")
	write(t, root, "internal/project/project.go", "package project\n\nimport (\n\t\"fmt\"\n\t_ \"github.com/rbe/gh-sweep/internal/ui\"\n)\n\nvar _ = fmt.Sprint\n")
	write(t, root, "build/cache/tool.go", "package tool\n\nimport _ \"github.com/rbe/gh-sweep/internal/ui\"\n")
	got, err := projectImports(root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]string{"internal/format": nil, "internal/project": {"internal/ui"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("imports %v, want %v", got, want)
	}
}

func write(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

// projectImports gives, for each package directory below root, the
// project packages that its non-test Go files import, sorted. It skips
// build, test, and hidden directories.
func projectImports(root string) (map[string][]string, error) {
	out := map[string][]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if entry.IsDir() {
			if rel != "." && (strings.HasPrefix(entry.Name(), ".") || rel == "build" || rel == "test") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		return addImports(out, path, filepath.ToSlash(filepath.Dir(rel)))
	})
	for dir := range out {
		sort.Strings(out[dir])
	}
	return out, err
}

// addImports adds the project imports of one Go file to its directory.
func addImports(out map[string][]string, path, dir string) error {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return err
	}
	if _, ok := out[dir]; !ok {
		out[dir] = nil
	}
	for _, spec := range f.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return err
		}
		if rest, ok := strings.CutPrefix(imp, modulePath); ok && !contains(out[dir], rest) {
			out[dir] = append(out[dir], rest)
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
