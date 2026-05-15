package plugins

import (
	"context"

	"github.com/JoakimCarlsson/scour/query"
	"github.com/JoakimCarlsson/scour/rank"
)

type Infobox struct {
	Title   string
	Summary string
	URL     string
}

type Answer struct {
	Text   string
	Source string
}

type Context struct {
	Query   query.Query
	Ranked  []rank.Ranked
	Infobox *Infobox
	Answer  *Answer
}

type Plugin interface {
	Name() string
	Apply(ctx context.Context, c *Context) error
}

func Run(ctx context.Context, plugins []Plugin, c *Context) error {
	for _, p := range plugins {
		if err := p.Apply(ctx, c); err != nil {
			return err
		}
	}
	return nil
}
