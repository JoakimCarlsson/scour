package engines

import (
	"context"

	"github.com/JoakimCarlsson/scour/query"
)

type stubEngine struct {
	name       string
	categories []query.Category
	languages  []string
	weight     float64
}

func (s stubEngine) Name() string                 { return s.name }
func (s stubEngine) Categories() []query.Category { return s.categories }
func (s stubEngine) Languages() []string          { return s.languages }
func (s stubEngine) Weight() float64              { return s.weight }
func (s stubEngine) Search(_ context.Context, _ query.Query) ([]Result, error) {
	return nil, ErrNotImplemented
}

func init() {
	Register(stubEngine{
		name: "google",
		categories: []query.Category{
			query.CategoryGeneral,
			query.CategoryNews,
			query.CategoryImages,
			query.CategoryVideos,
			query.CategoryMap,
		},
		languages: []string{"*"},
		weight:    1.0,
	})
	Register(stubEngine{
		name: "bing",
		categories: []query.Category{
			query.CategoryGeneral,
			query.CategoryNews,
			query.CategoryImages,
			query.CategoryVideos,
		},
		languages: []string{"*"},
		weight:    1.0,
	})
	Register(stubEngine{
		name: "duckduckgo",
		categories: []query.Category{
			query.CategoryGeneral,
			query.CategoryNews,
			query.CategoryImages,
			query.CategoryVideos,
		},
		languages: []string{"*"},
		weight:    1.0,
	})
	Register(stubEngine{
		name: "brave",
		categories: []query.Category{
			query.CategoryGeneral,
			query.CategoryNews,
			query.CategoryImages,
			query.CategoryVideos,
		},
		languages: []string{"*"},
		weight:    1.0,
	})
}
