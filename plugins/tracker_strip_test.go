package plugins

import (
	"context"
	"testing"

	"github.com/JoakimCarlsson/scour/merge"
	"github.com/JoakimCarlsson/scour/rank"
)

func TestTrackerStrip(t *testing.T) {
	c := &Context{
		Ranked: []rank.Ranked{
			{Merged: merge.Merged{URL: "https://example.com/?utm_source=x&q=1"}},
			{Merged: merge.Merged{URL: "https://example.com/clean"}},
		},
	}
	if err := (TrackerStrip{}).Apply(context.Background(), c); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if c.Ranked[0].URL != "https://example.com/?q=1" {
		t.Errorf("expected utm stripped, got %q", c.Ranked[0].URL)
	}
	if c.Ranked[1].URL != "https://example.com/clean" {
		t.Errorf("clean url changed: %q", c.Ranked[1].URL)
	}
}
