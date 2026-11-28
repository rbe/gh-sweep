// id: ADR-0018

package gh

import (
	"testing"

	"github.com/rbe/gh-sweep/internal/project"
)

func TestAddWorkflowsGivesDashToAnArtifactWithNoRunAndLeavesACache(t *testing.T) {
	calls := 0
	saved := runGHFunc
	runGHFunc = func(args ...string) (string, error) { calls++; return "", nil }
	t.Cleanup(func() { runGHFunc = saved })
	p := project.Project{Owner: "acme", Name: "widgets", NWO: "acme/widgets", Items: []project.Item{
		{Kind: "artifact", ID: "1", Run: "0"},
		{Kind: "cache", ID: "7"},
	}}
	if err := AddWorkflows(&p); err != nil {
		t.Fatal(err)
	}
	if p.Items[0].Workflow != "-" || p.Items[1].Workflow != "" || calls != 0 {
		t.Fatalf("items %+v, calls %d", p.Items, calls)
	}
}
