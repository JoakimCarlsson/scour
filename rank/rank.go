package rank

import (
	"sort"

	"github.com/JoakimCarlsson/scour/merge"
)

type Ranked struct {
	merge.Merged
	Score float64
	Flags []string
}

func Rank(in []merge.Merged, weights map[string]float64) []Ranked {
	out := make([]Ranked, len(in))
	for i, m := range in {
		var score float64
		for _, s := range m.Sources {
			if s.Position <= 0 {
				continue
			}
			w, ok := weights[s.Engine]
			if !ok {
				w = 1.0
			}
			score += w * (1.0 / float64(s.Position))
		}
		out[i] = Ranked{Merged: m, Score: score}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].URL < out[j].URL
	})
	return out
}
