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
		// www. prefix stripped (3+ labels)
		{"https://www.example.com/foo", "https://example.com/foo"},
		{"https://WWW.Example.com/foo", "https://example.com/foo"},
		// www. preserved when it's the apex (2 labels)
		{"https://www.com/foo", "https://www.com/foo"},
		// default ports stripped
		{"https://example.com:443/foo", "https://example.com/foo"},
		{"http://example.com:80/foo", "http://example.com/foo"},
		// non-default ports preserved
		{"https://example.com:8443/foo", "https://example.com:8443/foo"},
		// broader tracker list
		{"https://example.com/?ref_src=twitter&q=keep", "https://example.com/?q=keep"},
		{"https://example.com/?mkt_tok=abc&q=keep", "https://example.com/?q=keep"},
		{"https://example.com/?pk_campaign=x&q=keep", "https://example.com/?q=keep"},
		{"https://example.com/?_hsenc=x&__hstc=y&q=keep", "https://example.com/?q=keep"},
		// query params sorted by key (Go's url.Values.Encode behaviour)
		{"https://example.com/?z=1&a=2&m=3", "https://example.com/?a=2&m=3&z=1"},
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
