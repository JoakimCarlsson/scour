package query

import (
	"reflect"
	"testing"
)

func TestParseSearchOperators(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		want      Filters
		wantTerms string
	}{
		{
			name:      "site positive",
			raw:       "golang site:reddit.com",
			want:      Filters{Sites: []string{"reddit.com"}},
			wantTerms: "golang",
		},
		{
			name:      "filetype",
			raw:       "filetype:pdf transformers",
			want:      Filters{FileTypes: []string{"pdf"}},
			wantTerms: "transformers",
		},
		{
			name:      "intitle",
			raw:       "intitle:tutorial generics",
			want:      Filters{InTitle: []string{"tutorial"}},
			wantTerms: "generics",
		},
		{
			name:      "inurl",
			raw:       "inurl:docs ranger",
			want:      Filters{InURL: []string{"docs"}},
			wantTerms: "ranger",
		},
		{
			name:      "negative term",
			raw:       "golang -windows",
			want:      Filters{Excluded: []string{"windows"}},
			wantTerms: "golang",
		},
		{
			name:      "negative site",
			raw:       "images -site:pinterest.com",
			want:      Filters{NotSites: []string{"pinterest.com"}},
			wantTerms: "images",
		},
		{
			name:      "multiple sites",
			raw:       "site:a.com site:b.com query",
			want:      Filters{Sites: []string{"a.com", "b.com"}},
			wantTerms: "query",
		},
		{
			name:      "case insensitive operator",
			raw:       "SITE:reddit.com mech",
			want:      Filters{Sites: []string{"reddit.com"}},
			wantTerms: "mech",
		},
		{
			name:      "mixed with category bang",
			raw:       "!news ukraine -trump",
			want:      Filters{Excluded: []string{"trump"}},
			wantTerms: "ukraine",
		},
		{
			name:      "ext alias",
			raw:       "ext:csv data",
			want:      Filters{FileTypes: []string{"csv"}},
			wantTerms: "data",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			q, err := Parse(c.raw, defaultPrefs())
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !reflect.DeepEqual(q.Filters, c.want) {
				t.Errorf("Filters = %+v, want %+v", q.Filters, c.want)
			}
			if q.Terms != c.wantTerms {
				t.Errorf("Terms = %q, want %q", q.Terms, c.wantTerms)
			}
		})
	}
}

func TestFiltersRender(t *testing.T) {
	f := Filters{
		Sites:     []string{"reddit.com"},
		NotSites:  []string{"pinterest.com"},
		FileTypes: []string{"pdf"},
		Excluded:  []string{"windows"},
	}
	got := f.Render("golang")
	want := "golang site:reddit.com -site:pinterest.com filetype:pdf -windows"
	if got != want {
		t.Fatalf("Render: %q, want %q", got, want)
	}
}

func TestFiltersRenderEmpty(t *testing.T) {
	if got := (Filters{}).Render("just terms"); got != "just terms" {
		t.Fatalf("got %q, want %q", got, "just terms")
	}
}

func TestFiltersCanonicalOrderIndependent(t *testing.T) {
	a := Filters{Sites: []string{"a.com", "b.com"}}
	b := Filters{Sites: []string{"b.com", "a.com"}}
	if a.Canonical() != b.Canonical() {
		t.Fatalf("order-dependent canonical: %q vs %q", a.Canonical(), b.Canonical())
	}
}
