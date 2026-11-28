// id: ADR-0015

package sweep

import (
	"testing"

	"github.com/rbe/gh-sweep/internal/project"
)

func TestSetFilterParsesEachFilterOfTheCommandLine(t *testing.T) {
	var f project.Filter
	for name, value := range map[string]string{
		"older-than": "30", "keep": "0", "branch": "main", "status": "failure", "key-prefix": "node-",
		"untagged": "yes", "package": "app", "package-type": "npm",
	} {
		if err := SetFilter(&f, name, value); err != nil {
			t.Fatalf("%s=%s: %v", name, value, err)
		}
	}
	want := project.Filter{OlderThan: 30, Branch: "main", Status: "failure", KeyPrefix: "node-",
		Untagged: true, Package: "app", PackageType: "npm"}
	if f != want {
		t.Fatalf("filter %+v", f)
	}
	if err := SetFilter(&f, "untagged", "no"); err != nil || f.Untagged {
		t.Fatalf("untagged=no: %v, %+v", err, f)
	}
	for _, c := range [][3]string{
		{"older-than", "0", "give a number of days of 1 or more"},
		{"keep", "-1", "give a number of 0 or more"},
		{"untagged", "maybe", "give yes or no"},
		{"color", "red", "--color is not a filter"},
	} {
		if err := SetFilter(&f, c[0], c[1]); err == nil || err.Error() != c[2] {
			t.Fatalf("%s=%s gives %v", c[0], c[1], err)
		}
	}
}

func TestEveryFlagHasOneActionWithATitle(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range All() {
		if a.Title == "" || seen[a.Flag] || (a.List == nil) == (a.Account == nil) {
			t.Fatalf("action %s: title %q, repeated %v, list and account both set or both unset", a.Flag, a.Title, seen[a.Flag])
		}
		seen[a.Flag] = true
		if got, ok := ByFlag(a.Flag); !ok || got.Flag != a.Flag {
			t.Fatalf("ByFlag(%s)", a.Flag)
		}
	}
	if _, ok := ByFlag("--nothing"); ok {
		t.Fatal("ByFlag found --nothing")
	}
}
