package engines

import (
	"context"
	"fmt"
	"sort"

	"github.com/JoakimCarlsson/scour/query"
)

type Result struct {
	Title    string
	URL      string
	Snippet  string
	Engine   string
	Position int
}

type Engine interface {
	Name() string
	Categories() []query.Category
	Languages() []string
	Weight() float64
	Search(ctx context.Context, q query.Query) ([]Result, error)
}

type Preferences struct {
	DisabledEngines []string
}

var registry = map[string]Engine{}

func Register(e Engine) {
	name := e.Name()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("engines: duplicate registration for %q", name))
	}
	registry[name] = e
}

func All() []Engine {
	out := make([]Engine, 0, len(registry))
	for _, e := range registry {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
