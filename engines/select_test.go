package engines

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

type stubEngine struct {
	name       string
	categories []query.Category
	languages  LanguageTraits
	weight     float64
}

func (s stubEngine) Name() string                 { return s.name }
func (s stubEngine) Categories() []query.Category { return s.categories }
func (s stubEngine) Languages() LanguageTraits    { return s.languages }
func (s stubEngine) Weight() float64              { return s.weight }
func (s stubEngine) Search(_ context.Context, _ query.Query) (Response, error) {
	return Response{}, nil
}

func names(es []Engine) []string {
	if len(es) == 0 {
		return nil
	}
	out := make([]string, 0, len(es))
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

func TestSelect(t *testing.T) {
	tests := []struct {
		name  string
		q     query.Query
		prefs Preferences
		want  []string
	}{
		{
			name: "default general query returns all engines alphabetically",
			q:    query.Query{Category: query.CategoryGeneral, Language: "en"},
			want: []string{
				"bing",
				"brave",
				"duckduckgo",
				"google",
				"mojeek",
				"qwant",
				"startpage",
				"yandex",
			},
		},
		{
			name: "bang pin restricts to google",
			q:    query.Query{Category: query.CategoryGeneral, Engines: []string{"google"}},
			want: []string{"google"},
		},
		{
			name:  "disabled engine is excluded",
			q:     query.Query{Category: query.CategoryGeneral},
			prefs: Preferences{DisabledEngines: []string{"bing"}},
			want: []string{
				"brave",
				"duckduckgo",
				"google",
				"mojeek",
				"qwant",
				"startpage",
				"yandex",
			},
		},
		{
			name:  "disabled wins over pinned",
			q:     query.Query{Category: query.CategoryGeneral, Engines: []string{"bing"}},
			prefs: Preferences{DisabledEngines: []string{"bing"}},
			want:  nil,
		},
		{
			name: "images category returns image-capable engines",
			q:    query.Query{Category: query.CategoryImages},
			want: []string{"bing", "duckduckgo"},
		},
		{
			name: "news category returns news-capable engines",
			q:    query.Query{Category: query.CategoryNews},
			want: []string{"bing", "brave", "google", "qwant"},
		},
		{
			name: "videos category returns videos engines",
			q:    query.Query{Category: query.CategoryVideos},
			want: []string{"sepiasearch"},
		},
		{
			name: "map category returns map-capable engines",
			q:    query.Query{Category: query.CategoryMap},
			want: []string{"nominatim", "photon"},
		},
		{
			name: "it category returns IT engines",
			q:    query.Query{Category: query.CategoryIT},
			want: []string{"hackernews", "stackexchange"},
		},
		{
			name: "science category returns science engines",
			q:    query.Query{Category: query.CategoryScience},
			want: []string{"arxiv", "openalex"},
		},
		{
			name: "social category returns social engines",
			q:    query.Query{Category: query.CategorySocial},
			want: []string{"reddit"},
		},
		{
			name: "music category returns music engines",
			q:    query.Query{Category: query.CategoryMusic},
			want: []string{"radiobrowser"},
		},
		{
			name: "wildcard language engines match any language",
			q:    query.Query{Category: query.CategoryGeneral, Language: "ja"},
			want: []string{
				"bing",
				"brave",
				"duckduckgo",
				"google",
				"mojeek",
				"qwant",
				"startpage",
				"yandex",
			},
		},
		{
			name: "empty language treated as no constraint",
			q:    query.Query{Category: query.CategoryGeneral},
			want: []string{
				"bing",
				"brave",
				"duckduckgo",
				"google",
				"mojeek",
				"qwant",
				"startpage",
				"yandex",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := names(Select(tc.q, tc.prefs))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Select() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSelectIsDeterministic(t *testing.T) {
	q := query.Query{Category: query.CategoryGeneral, Language: "en"}
	first := names(Select(q, Preferences{}))
	for i := range 5 {
		got := names(Select(q, Preferences{}))
		if !reflect.DeepEqual(got, first) {
			t.Fatalf("non-deterministic order: run %d got %v, want %v", i, got, first)
		}
	}
}

func TestSupportsLanguage(t *testing.T) {
	mk := func(all bool, langs ...string) LanguageTraits {
		m := map[string]string{}
		for _, l := range langs {
			m[strings.ToLower(l)] = l
		}
		return LanguageTraits{All: all, Supported: m}
	}
	tests := []struct {
		name     string
		traits   LanguageTraits
		queryLng string
		want     bool
	}{
		{"all-true matches any", mk(true), "en", true},
		{"empty query language matches anything", mk(false, "en"), "", true},
		{"exact match", mk(false, "en", "fr"), "fr", true},
		{"case-insensitive match", mk(false, "en-US"), "en-us", true},
		{"no match returns false", mk(false, "en", "fr"), "ja", false},
		{"empty engine map rejects non-empty query", mk(false), "en", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := stubEngine{languages: tc.traits}
			if got := supportsLanguage(e, tc.queryLng); got != tc.want {
				t.Fatalf(
					"supportsLanguage(%v, %q) = %v, want %v",
					tc.traits,
					tc.queryLng,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestRegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()
	Register(stubEngine{name: "google"})
}
