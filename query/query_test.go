package query

import (
	"errors"
	"reflect"
	"testing"
)

func defaultPrefs() Preferences {
	return Preferences{
		DefaultLanguage:   "en",
		DefaultCategory:   CategoryGeneral,
		DefaultSafeSearch: SafeModerate,
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantTerms   string
		wantCat     Category
		wantLang    string
		wantEngines []string
		wantErr     error
	}{
		{
			name:      "plain query",
			raw:       "golang tutorials",
			wantTerms: "golang tutorials",
			wantCat:   CategoryGeneral,
			wantLang:  "en",
		},
		{
			name:        "single bang",
			raw:         "!g golang",
			wantTerms:   "golang",
			wantCat:     CategoryGeneral,
			wantLang:    "en",
			wantEngines: []string{"google"},
		},
		{
			name:        "multiple bangs preserve order",
			raw:         "!g !ddg golang",
			wantTerms:   "golang",
			wantCat:     CategoryGeneral,
			wantLang:    "en",
			wantEngines: []string{"google", "duckduckgo"},
		},
		{
			name:        "duplicate bangs dedup",
			raw:         "!g !g !ddg foo",
			wantTerms:   "foo",
			wantCat:     CategoryGeneral,
			wantLang:    "en",
			wantEngines: []string{"google", "duckduckgo"},
		},
		{
			name:      "colon category prefix",
			raw:       ":news ukraine",
			wantTerms: "ukraine",
			wantCat:   CategoryNews,
			wantLang:  "en",
		},
		{
			name:      "bang category prefix",
			raw:       "!news ukraine",
			wantTerms: "ukraine",
			wantCat:   CategoryNews,
			wantLang:  "en",
		},
		{
			name:      "language hint short",
			raw:       ":en golang",
			wantTerms: "golang",
			wantCat:   CategoryGeneral,
			wantLang:  "en",
		},
		{
			name:      "language hint region",
			raw:       ":en-US golang",
			wantTerms: "golang",
			wantCat:   CategoryGeneral,
			wantLang:  "en-US",
		},
		{
			name:      "unknown bang passthrough",
			raw:       "!unknown foo",
			wantTerms: "!unknown foo",
			wantCat:   CategoryGeneral,
			wantLang:  "en",
		},
		{
			name:        "mixed directives",
			raw:         "!g :news !ddg golang generics",
			wantTerms:   "golang generics",
			wantCat:     CategoryNews,
			wantLang:    "en",
			wantEngines: []string{"google", "duckduckgo"},
		},
		{
			name:    "all whitespace",
			raw:     "   ",
			wantErr: ErrEmptyQuery,
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: ErrEmptyQuery,
		},
		{
			name:      "internal whitespace collapse",
			raw:       "foo   bar",
			wantTerms: "foo bar",
			wantCat:   CategoryGeneral,
			wantLang:  "en",
		},
		{
			name:      "unknown colon passthrough",
			raw:       ":1foo bar",
			wantTerms: ":1foo bar",
			wantCat:   CategoryGeneral,
			wantLang:  "en",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, err := Parse(tc.raw, defaultPrefs())
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if q.Raw != tc.raw {
				t.Errorf("Raw = %q, want %q", q.Raw, tc.raw)
			}
			if q.Terms != tc.wantTerms {
				t.Errorf("Terms = %q, want %q", q.Terms, tc.wantTerms)
			}
			if q.Category != tc.wantCat {
				t.Errorf("Category = %q, want %q", q.Category, tc.wantCat)
			}
			if q.Language != tc.wantLang {
				t.Errorf("Language = %q, want %q", q.Language, tc.wantLang)
			}
			if !reflect.DeepEqual(q.Engines, tc.wantEngines) {
				t.Errorf("Engines = %v, want %v", q.Engines, tc.wantEngines)
			}
			if q.SafeSearch != SafeModerate {
				t.Errorf("SafeSearch = %v, want %v", q.SafeSearch, SafeModerate)
			}
		})
	}
}

func TestParseCategory(t *testing.T) {
	cases := map[string]Category{
		"general": CategoryGeneral,
		"News":    CategoryNews,
		"IT":      CategoryIT,
		"SCIENCE": CategoryScience,
	}
	for in, want := range cases {
		got, ok := ParseCategory(in)
		if !ok || got != want {
			t.Errorf("ParseCategory(%q) = (%q, %v), want (%q, true)", in, got, ok, want)
		}
	}
	if _, ok := ParseCategory("nope"); ok {
		t.Error("ParseCategory(\"nope\") ok = true, want false")
	}
}

func TestParseSafeLevel(t *testing.T) {
	cases := map[string]SafeLevel{
		"off":      SafeOff,
		"Moderate": SafeModerate,
		"STRICT":   SafeStrict,
	}
	for in, want := range cases {
		got, ok := ParseSafeLevel(in)
		if !ok || got != want {
			t.Errorf("ParseSafeLevel(%q) = (%v, %v), want (%v, true)", in, got, ok, want)
		}
	}
	if _, ok := ParseSafeLevel("nope"); ok {
		t.Error("ParseSafeLevel(\"nope\") ok = true, want false")
	}
	if got := SafeStrict.String(); got != "strict" {
		t.Errorf("SafeStrict.String() = %q, want %q", got, "strict")
	}
}
