package engines

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/JoakimCarlsson/scour/query"
)

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
			name: "default general query returns all stubs alphabetically",
			q:    query.Query{Category: query.CategoryGeneral, Language: "en"},
			want: []string{"bing", "brave", "duckduckgo", "google"},
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
			want:  []string{"brave", "duckduckgo", "google"},
		},
		{
			name:  "disabled wins over pinned",
			q:     query.Query{Category: query.CategoryGeneral, Engines: []string{"bing"}},
			prefs: Preferences{DisabledEngines: []string{"bing"}},
			want:  nil,
		},
		{
			name: "images category returns all stubs",
			q:    query.Query{Category: query.CategoryImages},
			want: []string{"bing", "brave", "duckduckgo", "google"},
		},
		{
			name: "map category returns only google",
			q:    query.Query{Category: query.CategoryMap},
			want: []string{"google"},
		},
		{
			name: "music category returns empty slice",
			q:    query.Query{Category: query.CategoryMusic},
			want: nil,
		},
		{
			name: "wildcard language engines match any language",
			q:    query.Query{Category: query.CategoryGeneral, Language: "ja"},
			want: []string{"bing", "brave", "duckduckgo", "google"},
		},
		{
			name: "empty language treated as no constraint",
			q:    query.Query{Category: query.CategoryGeneral},
			want: []string{"bing", "brave", "duckduckgo", "google"},
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
	tests := []struct {
		name      string
		engineLng []string
		queryLng  string
		want      bool
	}{
		{"wildcard matches any", []string{"*"}, "en", true},
		{"empty query language matches anything", []string{"en"}, "", true},
		{"exact match", []string{"en", "fr"}, "fr", true},
		{"case-insensitive match", []string{"en-US"}, "en-us", true},
		{"no match returns false", []string{"en", "fr"}, "ja", false},
		{"empty engine list rejects non-empty query", []string{}, "en", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := stubEngine{languages: tc.engineLng}
			if got := supportsLanguage(e, tc.queryLng); got != tc.want {
				t.Fatalf(
					"supportsLanguage(%v, %q) = %v, want %v",
					tc.engineLng,
					tc.queryLng,
					got,
					tc.want,
				)
			}
		})
	}
}

func TestStubSearchReturnsNotImplemented(t *testing.T) {
	for _, e := range All() {
		got, err := e.Search(context.Background(), query.Query{})
		if got != nil {
			t.Errorf("%s.Search returned non-nil results: %v", e.Name(), got)
		}
		if !errors.Is(err, ErrNotImplemented) {
			t.Errorf("%s.Search err = %v, want ErrNotImplemented", e.Name(), err)
		}
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
