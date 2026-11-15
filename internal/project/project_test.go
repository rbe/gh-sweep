package project

import "testing"

func TestParse(t *testing.T) {
	p, err := Parse("acme/widgets")
	if err != nil {
		t.Fatal(err)
	}
	if p.Owner != "acme" || p.Name != "widgets" || p.NWO != "acme/widgets" || p.APIHost != "" {
		t.Fatalf("%+v", p)
	}
	p, err = Parse("github.com/acme/widgets")
	if err != nil || p.NWO != "acme/widgets" || p.APIHost != "" {
		t.Fatalf("github.com: %+v %v", p, err)
	}
	p, err = Parse("ghe.example.com/acme/widgets.git")
	if err != nil || p.APIHost != "ghe.example.com" || p.NWO != "ghe.example.com/acme/widgets" {
		t.Fatalf("enterprise: %+v %v", p, err)
	}
	if _, err := Parse("not a repo"); err == nil {
		t.Fatal("expected error")
	}
}
