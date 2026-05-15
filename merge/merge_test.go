package merge

import (
	"reflect"
	"sort"
	"testing"

	"github.com/JoakimCarlsson/scour/engines"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"HTTPS://Example.com/Path/?utm_source=x&q=1#frag", "https://example.com/Path?q=1"},
		{"http://example.com/", "http://example.com/"},
		{"https://EXAMPLE.com", "https://example.com"},
		{"https://example.com/a/b/", "https://example.com/a/b"},
		{"https://example.com/?fbclid=abc&q=keep", "https://example.com/?q=keep"},
		{"https://example.com/?utm_a=1&utm_b=2", "https://example.com/"},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := Normalize(tc.in)
			if err != nil {
				t.Fatalf("Normalize: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestNormalizeMalformed(t *testing.T) {
	if _, err := Normalize("not a url"); err == nil {
		t.Fatal("expected error on missing scheme/host")
	}
	if _, err := Normalize(""); err == nil {
		t.Fatal("expected error on empty")
	}
}

func TestMergeDedup(t *testing.T) {
	in := []engines.Result{
		{Title: "Go", URL: "https://go.dev/", Snippet: "short", Engine: "duckduckgo", Position: 1},
		{
			Title:    "The Go Programming Language",
			URL:      "https://GO.dev/?utm_x=1",
			Snippet:  "longer snippet",
			Engine:   "bing",
			Position: 2,
		},
		{
			Title:    "Tutorial",
			URL:      "https://go.dev/doc/tutorial/",
			Snippet:  "tut",
			Engine:   "duckduckgo",
			Position: 2,
		},
	}
	out := Merge(in)
	if len(out) != 2 {
		t.Fatalf("expected 2 merged, got %d", len(out))
	}
	var goDev *Merged
	for i := range out {
		if out[i].URL == "https://go.dev/" {
			goDev = &out[i]
		}
	}
	if goDev == nil {
		t.Fatal("merged go.dev missing")
	}
	if goDev.Title != "The Go Programming Language" {
		t.Errorf("expected longest title, got %q", goDev.Title)
	}
	if goDev.Snippet != "longer snippet" {
		t.Errorf("expected longest snippet, got %q", goDev.Snippet)
	}
	if len(goDev.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d", len(goDev.Sources))
	}
	sort.Slice(
		goDev.Sources,
		func(i, j int) bool { return goDev.Sources[i].Engine < goDev.Sources[j].Engine },
	)
	want := []Source{{Engine: "bing", Position: 2}, {Engine: "duckduckgo", Position: 1}}
	if !reflect.DeepEqual(goDev.Sources, want) {
		t.Errorf("sources = %+v, want %+v", goDev.Sources, want)
	}
}

func TestMergeDropsMalformed(t *testing.T) {
	in := []engines.Result{
		{Title: "ok", URL: "https://go.dev/", Engine: "a"},
		{Title: "bad", URL: "not a url", Engine: "b"},
	}
	out := Merge(in)
	if len(out) != 1 {
		t.Fatalf("expected 1 result, got %d", len(out))
	}
}
