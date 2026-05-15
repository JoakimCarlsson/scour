package plugins

import (
	"context"

	"github.com/JoakimCarlsson/scour/merge"
)

type TrackerStrip struct{}

func (TrackerStrip) Name() string { return "tracker_strip" }

func (TrackerStrip) Apply(_ context.Context, c *Context) error {
	for i := range c.Ranked {
		norm, err := merge.Normalize(c.Ranked[i].URL)
		if err == nil {
			c.Ranked[i].URL = norm
		}
	}
	return nil
}
