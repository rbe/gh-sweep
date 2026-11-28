// id: ADR-0012

package gh

import (
	"strings"
	"testing"

	"github.com/rbe/gh-sweep/internal/project"
)

func TestDeleteBranchEscapesEachPartOfTheName(t *testing.T) {
	var got []string
	runGHFunc = func(args ...string) (string, error) {
		got = args
		return "", nil
	}
	t.Cleanup(func() { runGHFunc = runGH })
	if err := DeleteBranch(project.Project{Owner: "acme", Name: "widgets"}, "feature/a b#1"); err != nil {
		t.Fatal(err)
	}
	if path := got[len(got)-1]; path != "repos/acme/widgets/git/refs/heads/feature/a%20b%231" {
		t.Fatalf("gh %s", strings.Join(got, " "))
	}
}
