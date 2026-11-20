// id: ADR-0001

package gh

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

// commandPackage is the import path of the package that starts a command.
const commandPackage = "os/exec"

// ghPackage is the one directory that may import commandPackage.
const ghPackage = "internal/gh"

// projectRoot is the root of the project, seen from this package.
const projectRoot = "../.."

func TestOnlyInternalGHImportsOSExec(t *testing.T) {
	dirs, err := packagesImporting(projectRoot, commandPackage)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{ghPackage}; !reflect.DeepEqual(dirs, want) {
		t.Fatalf("packages that import %s = %v, want %v", commandPackage, dirs, want)
	}
}

func TestPackageOutsideInternalGHThatImportsOSExecIsNamed(t *testing.T) {
	root := t.TempDir()
	writeGoFile(t, root, "internal/gh/client.go", "package gh\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n")
	writeGoFile(t, root, "internal/ui/app.go", "package ui\n\nimport (\n\t\"os\"\n\t\"os/exec\"\n)\n\nvar _, _ = os.Args, exec.Command\n")
	writeGoFile(t, root, "internal/format/format.go", "package format\n\nimport \"os\"\n\nvar _ = os.Args\n")
	writeGoFile(t, root, "build/cache/tool.go", "package tool\n\nimport \"os/exec\"\n\nvar _ = exec.Command\n")
	dirs, err := packagesImporting(root, commandPackage)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{ghPackage, "internal/ui"}; !reflect.DeepEqual(dirs, want) {
		t.Fatalf("packages that import %s = %v, want %v", commandPackage, dirs, want)
	}
}

// writeGoFile writes one Go file below root and creates its package directory.
func writeGoFile(t *testing.T, root, name, source string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
}

// packagesImporting returns the package directories below root, sorted,
// that have a Go file which imports importPath.
func packagesImporting(root, importPath string) ([]string, error) {
	found := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return skipUnlessProjectDir(root, path)
		}
		return notePackageDir(found, root, path, importPath)
	})
	return sortedKeys(found), err
}

// skipUnlessProjectDir skips a directory that holds no package of the
// project: a hidden directory, the installed skills, and the build output.
func skipUnlessProjectDir(root, path string) error {
	name := filepath.Base(path)
	if path == root {
		return nil
	}
	if strings.HasPrefix(name, ".") || name == "apm_modules" || name == "build" || name == "bin" {
		return filepath.SkipDir
	}
	return nil
}

// goFileImports reports whether the Go file at path imports importPath.
// A file that is not a Go file imports nothing.
func goFileImports(path, importPath string) (bool, error) {
	if !strings.HasSuffix(path, ".go") {
		return false, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
	if err != nil {
		return false, err
	}
	for _, spec := range file.Imports {
		name, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			return false, err
		}
		if name == importPath {
			return true, nil
		}
	}
	return false, nil
}

// notePackageDir puts the package directory of the file at path into found,
// as a slash path relative to root, when that file imports importPath.
func notePackageDir(found map[string]bool, root, path, importPath string) error {
	imports, err := goFileImports(path, importPath)
	if err != nil || !imports {
		return err
	}
	dir, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return err
	}
	found[filepath.ToSlash(dir)] = true
	return nil
}

// sortedKeys returns the keys of found in order.
func sortedKeys(found map[string]bool) []string {
	var keys []string
	for key := range found {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
