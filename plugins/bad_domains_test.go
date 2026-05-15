package plugins

import (
	"context"
	"slices"
	"testing"

	"github.com/JoakimCarlsson/scour/merge"
	"github.com/JoakimCarlsson/scour/rank"
)

func TestBadDomainsFlags(t *testing.T) {
	c := &Context{
		Ranked: []rank.Ranked{
			{Merged: merge.Merged{URL: "https://spam.example/"}},
			{Merged: merge.Merged{URL: "https://good.example/"}},
		},
	}
	p := BadDomains{Domains: []string{"spam.example"}, Flag: "spam"}
	if err := p.Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !slices.Contains(c.Ranked[0].Flags, "spam") {
		t.Errorf("expected spam flag, got %v", c.Ranked[0].Flags)
	}
	if len(c.Ranked[1].Flags) != 0 {
		t.Errorf("good result got flags: %v", c.Ranked[1].Flags)
	}
}

func TestBadDomainsNoConfigNoop(t *testing.T) {
	c := &Context{
		Ranked: []rank.Ranked{{Merged: merge.Merged{URL: "https://spam.example/"}}},
	}
	if err := (BadDomains{}).Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(c.Ranked[0].Flags) != 0 {
		t.Errorf("expected no flags, got %v", c.Ranked[0].Flags)
	}
}
