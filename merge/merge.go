package merge

import "github.com/JoakimCarlsson/scour/engines"

type Source struct {
	Engine   string
	Position int
}

type Merged struct {
	Title   string
	URL     string
	Snippet string
	Sources []Source
}

func Merge(in []engines.Result) []Merged {
	byURL := map[string]*Merged{}
	order := []string{}
	for _, r := range in {
		norm, err := Normalize(r.URL)
		if err != nil {
			continue
		}
		m, ok := byURL[norm]
		if !ok {
			m = &Merged{URL: norm}
			byURL[norm] = m
			order = append(order, norm)
		}
		if len(r.Title) > len(m.Title) {
			m.Title = r.Title
		}
		if len(r.Snippet) > len(m.Snippet) {
			m.Snippet = r.Snippet
		}
		m.Sources = append(m.Sources, Source{Engine: r.Engine, Position: r.Position})
	}
	out := make([]Merged, 0, len(order))
	for _, k := range order {
		out = append(out, *byURL[k])
	}
	return out
}
