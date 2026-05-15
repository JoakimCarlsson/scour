package engines

import (
	"slices"
	"sort"
	"strings"

	"github.com/JoakimCarlsson/scour/query"
)

func Select(q query.Query, prefs Preferences) []Engine {
	disabled := toSet(prefs.DisabledEngines)
	pinned := toSet(q.Engines)

	var out []Engine
	for _, e := range registry {
		if len(pinned) > 0 {
			if _, ok := pinned[e.Name()]; !ok {
				continue
			}
		}
		if _, ok := disabled[e.Name()]; ok {
			continue
		}
		if !supportsCategory(e, q.Category) {
			continue
		}
		if !supportsLanguage(e, q.Language) {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

func toSet(xs []string) map[string]struct{} {
	if len(xs) == 0 {
		return nil
	}
	s := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		s[x] = struct{}{}
	}
	return s
}

func supportsCategory(e Engine, c query.Category) bool {
	return slices.Contains(e.Categories(), c)
}

func supportsLanguage(e Engine, lang string) bool {
	if lang == "" {
		return true
	}
	want := strings.ToLower(lang)
	for _, l := range e.Languages() {
		if l == "*" {
			return true
		}
		if strings.ToLower(l) == want {
			return true
		}
	}
	return false
}
